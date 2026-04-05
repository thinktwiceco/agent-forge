package telegram

import (
	"fmt"
	"os"
	"strings"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// actSetWebhook calls Telegram setWebhook with the current token and secret without starting ngrok.
// Use this to apply or rotate WEBHOOK_SECRET_TELEGRAM after the tunnel URL is already known.
func (t *telegramTool) actSetWebhook(args map[string]any) llms.ToolReturn {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return core.NewErrorResponse(
			"TELEGRAM_BOT_TOKEN is not set. Call register_token first or set the env var manually.",
		)
	}

	secret := webhookSecretFromArgsOrEnv(args)
	if secret == "" {
		return core.NewErrorResponse(
			"WEBHOOK_SECRET_TELEGRAM is required. Set it in the environment (e.g. .env) or pass webhook_secret.",
		)
	}

	base := ""
	if u, _ := args["webhook_public_url"].(string); strings.TrimSpace(u) != "" {
		base = strings.TrimSpace(u)
		base = strings.TrimRight(base, "/")
	}
	if base == "" {
		var err error
		base, err = fetchNgrokHTTPSPublicURL()
		if err != nil {
			return core.NewErrorResponse(
				fmt.Sprintf("Could not discover public URL: %v. Pass webhook_public_url (HTTPS base, e.g. https://xyz.ngrok.io).", err),
			)
		}
	}

	if !strings.HasPrefix(base, "https://") {
		return core.NewErrorResponse("webhook_public_url must be an https:// URL")
	}

	webhookURL := base + "/api/webhooks/telegram"
	if err := registerWebhook(token, webhookURL, secret); err != nil {
		return core.NewErrorResponse(fmt.Sprintf("setWebhook failed: %v", err))
	}

	msg := fmt.Sprintf(
		"Webhook updated with secret_token.\nRegistered URL: %s\n\nLocalforge will verify X-Telegram-Bot-Api-Secret-Token against WEBHOOK_SECRET_TELEGRAM.",
		webhookURL,
	)
	return core.NewSuccessResponse(msg)
}
