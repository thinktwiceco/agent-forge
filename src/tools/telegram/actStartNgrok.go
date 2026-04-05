package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// ngrokAPIBase is the base URL for ngrok's local management API.
// Overridable in tests.
var ngrokAPIBase = "http://127.0.0.1:4040"

const (
	ngrokPollAttempts = 10
	ngrokPollInterval = time.Second
)

type ngrokTunnelsResponse struct {
	Tunnels []struct {
		PublicURL string `json:"public_url"`
		Proto     string `json:"proto"`
	} `json:"tunnels"`
}

type setWebhookResponse struct {
	Ok          bool   `json:"ok"`
	Description string `json:"description"`
}

func (t *telegramTool) actStartNgrok(args map[string]any) llms.ToolReturn {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return core.NewErrorResponse(
			"TELEGRAM_BOT_TOKEN is not set. Call register_token first or set the env var manually.",
		)
	}

	port := t.port
	if p, _ := args["port"].(string); p != "" {
		port = p
	}

	if _, err := exec.LookPath("ngrok"); err != nil {
		return core.NewErrorResponse(
			"ngrok not found in PATH. Install it from https://ngrok.com/download and ensure it is in PATH.",
		)
	}

	cmd := exec.Command("ngrok", "http", port)
	if err := cmd.Start(); err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to start ngrok: %v", err))
	}

	publicURL, err := waitForNgrokTunnel()
	if err != nil {
		_ = cmd.Process.Kill()
		return core.NewErrorResponse(err.Error())
	}

	webhookURL := publicURL + "/api/webhooks/telegram"

	webhookSecret := webhookSecretFromArgsOrEnv(args)
	if webhookSecret == "" {
		_ = cmd.Process.Kill()
		return core.NewErrorResponse(
			"WEBHOOK_SECRET_TELEGRAM is required. Set it in the environment (e.g. .env) or pass webhook_secret.",
		)
	}

	if err := registerWebhook(token, webhookURL, webhookSecret); err != nil {
		_ = cmd.Process.Kill()
		return core.NewErrorResponse(fmt.Sprintf("setWebhook failed: %v", err))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "ngrok tunnel started.\nPublic URL: %s\nWebhook registered: %s\n", publicURL, webhookURL)
	sb.WriteString("\nWebhook secret: set (Telegram setWebhook secret_token)")
	sb.WriteString("\n\nNote: calling start_ngrok again will spawn another ngrok process. Kill stale ones with: pkill ngrok")
	sb.WriteString("\nTo apply or rotate the secret without starting a new ngrok process, use action: set_webhook")

	return core.NewSuccessResponse(sb.String())
}

// waitForNgrokTunnel polls the ngrok local API until an HTTPS tunnel is available.
func waitForNgrokTunnel() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	for i := 0; i < ngrokPollAttempts; i++ {
		time.Sleep(ngrokPollInterval)

		resp, err := client.Get(ngrokAPIBase + "/api/tunnels")
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			continue
		}

		var tunnels ngrokTunnelsResponse
		if err := json.Unmarshal(body, &tunnels); err != nil {
			continue
		}

		for _, tun := range tunnels.Tunnels {
			if strings.HasPrefix(tun.PublicURL, "https://") {
				return tun.PublicURL, nil
			}
		}
	}
	return "", fmt.Errorf(
		"timed out waiting for ngrok tunnel after %d seconds. Check ngrok logs for errors",
		ngrokPollAttempts,
	)
}

// fetchNgrokHTTPSPublicURL returns the first HTTPS tunnel URL from ngrok's local API (single request).
func fetchNgrokHTTPSPublicURL() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(ngrokAPIBase + "/api/tunnels")
	if err != nil {
		return "", fmt.Errorf("ngrok local API unreachable: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read ngrok response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ngrok local API returned HTTP %d", resp.StatusCode)
	}

	var tunnels ngrokTunnelsResponse
	if err := json.Unmarshal(body, &tunnels); err != nil {
		return "", fmt.Errorf("invalid ngrok JSON: %w", err)
	}
	for _, tun := range tunnels.Tunnels {
		if strings.HasPrefix(tun.PublicURL, "https://") {
			return tun.PublicURL, nil
		}
	}
	return "", fmt.Errorf("no HTTPS tunnel found; start ngrok or pass webhook_public_url")
}

// webhookSecretFromArgsOrEnv returns webhook_secret param or WEBHOOK_SECRET_TELEGRAM (trimmed).
func webhookSecretFromArgsOrEnv(args map[string]any) string {
	if s, _ := args["webhook_secret"].(string); strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(os.Getenv("WEBHOOK_SECRET_TELEGRAM"))
}

// registerWebhook calls the Telegram setWebhook Bot API endpoint (secret_token is required).
func registerWebhook(token, webhookURL, secret string) error {
	if strings.TrimSpace(secret) == "" {
		return fmt.Errorf("webhook secret is required")
	}
	payload := map[string]interface{}{
		"url":          webhookURL,
		"secret_token": secret,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/setWebhook", telegramAPIBase, token)
	client := &http.Client{Timeout: 15 * time.Second}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var setWebhookResp setWebhookResponse
	if err := json.Unmarshal(body, &setWebhookResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !setWebhookResp.Ok {
		desc := setWebhookResp.Description
		if desc == "" {
			desc = string(body)
		}
		return fmt.Errorf("telegram API returned ok=false: %s", desc)
	}

	return nil
}
