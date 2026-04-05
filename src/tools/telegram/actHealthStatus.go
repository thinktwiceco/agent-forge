package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

const healthHTTPTimeout = 5 * time.Second

func actHealthStatus() llms.ToolReturn {
	ngrokLine, hasHTTPS := probeNgrokHealth()
	tokenLine := describeTelegramTokenEnv()
	secretLine := describeWebhookSecretEnv()
	note := "\n(Env checks reflect the process environment; if you rely on a .env file, the host must load it — e.g. Localforge loads appDir/.env at startup.)"

	out := ngrokLine + "\n" + tokenLine + "\n" + secretLine + note
	if n := brainNudgesForTelegramHealth(ngrokLine, tokenLine, secretLine, hasHTTPS); n != "" {
		out += n
	}

	return core.NewSuccessResponse(out)
}

// probeNgrokHealth performs one GET to the ngrok local API. hasHTTPS is true only when at least one
// tunnel has an https:// public URL (Telegram webhooks require HTTPS).
func probeNgrokHealth() (line string, hasHTTPS bool) {
	client := &http.Client{Timeout: healthHTTPTimeout}
	resp, err := client.Get(ngrokAPIBase + "/api/tunnels")
	if err != nil {
		return fmt.Sprintf("ngrok local API: down (%v)", err), false
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("ngrok local API: down (read body: %v)", err), false
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("ngrok local API: down (HTTP %d)", resp.StatusCode), false
	}

	var tunnels ngrokTunnelsResponse
	if err := json.Unmarshal(body, &tunnels); err != nil {
		return fmt.Sprintf("ngrok local API: down (invalid JSON: %v)", err), false
	}

	for _, tun := range tunnels.Tunnels {
		if strings.HasPrefix(tun.PublicURL, "https://") {
			return "ngrok local API: up", true
		}
	}
	return "ngrok local API: up (no HTTPS tunnel)", false
}

func brainNudgesForTelegramHealth(ngrokLine, tokenLine, secretLine string, hasHTTPS bool) string {
	if telegramWebhookHealthOK(ngrokLine, tokenLine, secretLine, hasHTTPS) {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString("[BRAIN]: Ask the user whether you should help set up Telegram (token, webhook secret, ngrok HTTPS tunnel, webhook registration).\n")
	b.WriteString("[BRAIN]: Ask the user whether they want to be reminded when Telegram is down so they can set it up; if they answer, persist their preference with save_short_term_memory(topic, fact).\n")
	if secretsEnvSet(tokenLine, secretLine) && !hasHTTPS {
		b.WriteString("[BRAIN]: TELEGRAM_BOT_TOKEN and WEBHOOK_SECRET_TELEGRAM are set but the ngrok HTTPS tunnel is not available. Ask the user whether you may run telegram action start_ngrok without asking for permission each time; if they answer, record that in short-term memory with save_short_term_memory.\n")
	}
	return b.String()
}

func telegramWebhookHealthOK(ngrokLine, tokenLine, secretLine string, hasHTTPS bool) bool {
	if strings.Contains(tokenLine, "missing") || strings.Contains(secretLine, "missing") {
		return false
	}
	if strings.Contains(ngrokLine, "down") {
		return false
	}
	return hasHTTPS
}

func secretsEnvSet(tokenLine, secretLine string) bool {
	return strings.Contains(tokenLine, ": set") && strings.Contains(secretLine, ": set")
}

func describeTelegramTokenEnv() string {
	if strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")) != "" {
		return "TELEGRAM_BOT_TOKEN: set"
	}
	return "TELEGRAM_BOT_TOKEN: missing"
}

func describeWebhookSecretEnv() string {
	if strings.TrimSpace(os.Getenv("WEBHOOK_SECRET_TELEGRAM")) != "" {
		return "WEBHOOK_SECRET_TELEGRAM: set"
	}
	return "WEBHOOK_SECRET_TELEGRAM: missing"
}
