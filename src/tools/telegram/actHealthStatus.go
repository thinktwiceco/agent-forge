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
	allowlistLine := describeAllowlistEnv()
	note := "\n(Env checks reflect the process environment; if you rely on a .env file, the host must load it — e.g. Localforge loads appDir/.env at startup.)"

	out := ngrokLine + "\n" + tokenLine + "\n" + secretLine + "\n" + allowlistLine + note
	if n := brainNudgesForTelegramHealth(ngrokLine, tokenLine, secretLine, allowlistLine, hasHTTPS); n != "" {
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

func brainNudgesForTelegramHealth(ngrokLine, tokenLine, secretLine, allowlistLine string, hasHTTPS bool) string {
	if telegramWebhookHealthOK(ngrokLine, tokenLine, secretLine, allowlistLine, hasHTTPS) {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString("[BRAIN]: Ask the user whether you should help set up Telegram (token, webhook secret, ngrok HTTPS tunnel, webhook registration).\n")
	b.WriteString("[BRAIN]: Ask the user whether they want to be reminded when Telegram is down so they can set it up; if they answer, persist their preference with save_short_term_memory(topic, fact).\n")
	if secretsEnvSet(tokenLine, secretLine) && !hasHTTPS {
		b.WriteString("[BRAIN]: TELEGRAM_BOT_TOKEN and WEBHOOK_SECRET_TELEGRAM are set but the ngrok HTTPS tunnel is not available. Ask the user whether you may run telegram action start_ngrok without asking for permission each time; if they answer, record that in short-term memory with save_short_term_memory.\n")
	}
	if strings.Contains(allowlistLine, "not set") {
		b.WriteString("[BRAIN]: TELEGRAM_ALLOWED_USER_IDS is not set — all incoming Telegram webhooks are blocked. Ask the user to set it to their Telegram username (e.g. TELEGRAM_ALLOWED_USER_IDS=@yourname) in .env.\n")
	}
	return b.String()
}

func telegramWebhookHealthOK(ngrokLine, tokenLine, secretLine, allowlistLine string, hasHTTPS bool) bool {
	if strings.Contains(tokenLine, "missing") || strings.Contains(secretLine, "missing") {
		return false
	}
	if strings.Contains(allowlistLine, "not set") {
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

func describeAllowlistEnv() string {
	raw := strings.TrimSpace(os.Getenv("TELEGRAM_ALLOWED_USER_IDS"))
	if raw == "" {
		return "TELEGRAM_ALLOWED_USER_IDS: not set (all webhooks blocked)"
	}
	var usernames []string
	for _, entry := range strings.Split(raw, ",") {
		u := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(entry), "@"))
		if u != "" {
			usernames = append(usernames, "@"+u)
		}
	}
	if len(usernames) == 0 {
		return "TELEGRAM_ALLOWED_USER_IDS: not set (all webhooks blocked)"
	}
	return fmt.Sprintf("TELEGRAM_ALLOWED_USER_IDS: %s", strings.Join(usernames, ", "))
}
