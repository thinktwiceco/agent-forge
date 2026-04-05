package telegram

import (
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

const toolName = "telegram"

type telegramTool struct {
	port string // default local port for ngrok tunneling
}

// NewTelegramTool creates the telegram tool.
// port is the local server port ngrok should tunnel (e.g. "8080").
// An empty string defaults to "8080".
func NewTelegramTool(port string) llms.Tool {
	if port == "" {
		port = "8080"
	}
	t := &telegramTool{port: port}
	return core.NewTool(core.ToolConfig{
		Name:        toolName,
		Description: "Set up a Telegram bot for Localforge: validate the bot token, tunnel with ngrok, register the webhook with a required secret, rotate the secret without respawning ngrok, or report health.",
		AdvanceDesc: `Instructions:
  Localforge serves POST /api/webhooks/telegram. Telegram must call that URL; ngrok exposes your local
  server. Localforge always requires WEBHOOK_SECRET_TELEGRAM (or the same value as webhook_secret) to
  accept webhooks: Telegram sends X-Telegram-Bot-Api-Secret-Token and it must match.

  Suggested order:
  1) If setup may be incomplete, ask the user whether they want Telegram configured before changing anything.
  2) Optionally run health_status to see ngrok local API, TELEGRAM_BOT_TOKEN, and WEBHOOK_SECRET_TELEGRAM
     (reported as set/missing only; no secret values).
  3) Ensure WEBHOOK_SECRET_TELEGRAM is set in the host environment (e.g. .env loaded by Localforge) or
     pass webhook_secret on start_ngrok / set_webhook.
  4) register_token with token from @BotFather (sets TELEGRAM_BOT_TOKEN in this process).
  5) start_ngrok to spawn ngrok and call Telegram setWebhook with secret_token, or set_webhook if you
     only need to apply or rotate the secret (same public HTTPS base URL is fine).

Actions:
  register_token    getMe check; sets TELEGRAM_BOT_TOKEN. params: token (required)

  start_ngrok       Starts ngrok http on the tool port (or port param), then setWebhook to
                    {publicUrl}/api/webhooks/telegram. Requires webhook_secret or WEBHOOK_SECRET_TELEGRAM.
                    params: port (optional), webhook_secret (optional if env set)

  set_webhook       setWebhook only; no new ngrok. For new secret or after editing .env: pass
                    webhook_public_url (https base, e.g. https://xyz.ngrok.io) or rely on ngrok local
                    API at 127.0.0.1:4040 if a tunnel is already up.
                    params: webhook_secret (optional if env set), webhook_public_url (optional)

  health_status     Read-only probe; no secrets in output. When not fully healthy, the result includes
                    [BRAIN] nudges: ask about setup help, reminder preference (then save_short_term_memory),
                    and if token+secret are set but the HTTPS tunnel is missing, ask permission to run
                    start_ngrok without asking each time (then save_short_term_memory).

Requirements:
  - ngrok binary in PATH for start_ngrok only.
  - Do not print bot tokens or webhook secrets in chat.`,
		TroubleshootingInfo: `Troubleshooting:
- 401 on webhook: Localforge secret and Telegram secret_token must match; after changing .env restart Localforge.
- register_token: "token is required" — pass the token from @BotFather
- register_token: "Telegram API error" — token invalid or revoked
- start_ngrok: "TELEGRAM_BOT_TOKEN not set" — call register_token or set env
- start_ngrok: "TELEGRAM_BOT_TOKEN is not set" — same as above
- start_ngrok: "ngrok not found" — install ngrok and ensure PATH
- start_ngrok: "timed out waiting for ngrok tunnel" — ngrok failed; check its logs
- start_ngrok: "setWebhook failed" — token, URL, or network issue
- start_ngrok / set_webhook: "WEBHOOK_SECRET_TELEGRAM is required" — set env or pass webhook_secret
- set_webhook: "Could not discover public URL" — pass webhook_public_url or start ngrok so 127.0.0.1:4040 lists a tunnel
- health_status: "ngrok local API: down" — ngrok not running or 4040 blocked
- health_status: "TELEGRAM_BOT_TOKEN: missing" — register_token or env
- health_status: "WEBHOOK_SECRET_TELEGRAM: missing" — set in .env and restart host, or pass webhook_secret on actions
- start_ngrok spawns a new OS process each time; stale tunnels: pkill ngrok`,
		Parameters: []core.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Required:    true,
				Description: "register_token | start_ngrok | set_webhook | health_status",
			},
			{
				Name:        "token",
				Type:        "string",
				Description: "@BotFather bot token. Required for register_token only",
			},
			{
				Name:        "port",
				Type:        "string",
				Description: "Local HTTP port Localforge listens on (ngrok tunnels this). start_ngrok only; default from tool config",
			},
			{
				Name:        "webhook_secret",
				Type:        "string",
				Description: "Same value as WEBHOOK_SECRET_TELEGRAM / Telegram secret_token. Required for start_ngrok and set_webhook if env unset",
			},
			{
				Name:        "webhook_public_url",
				Type:        "string",
				Description: "HTTPS origin only (no path), e.g. https://abc.ngrok.io. set_webhook only; omit to read from ngrok 127.0.0.1:4040",
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			return t.handler(agentContext, args)
		},
	})
}

func (t *telegramTool) handler(_ map[string]any, args map[string]any) llms.ToolReturn {
	action, _ := args["action"].(string)
	switch action {
	case "register_token":
		return actRegisterToken(args)
	case "start_ngrok":
		return t.actStartNgrok(args)
	case "set_webhook":
		return t.actSetWebhook(args)
	case "health_status":
		return actHealthStatus()
	default:
		return core.NewErrorResponse("unknown action; valid actions: register_token, start_ngrok, set_webhook, health_status")
	}
}
