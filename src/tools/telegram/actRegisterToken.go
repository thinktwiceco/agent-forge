package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// telegramAPIBase is the Telegram Bot API root URL.
// Overridable in tests.
var telegramAPIBase = "https://api.telegram.org"

type getMeResponse struct {
	Ok     bool `json:"ok"`
	Result struct {
		ID        int    `json:"id"`
		IsBot     bool   `json:"is_bot"`
		FirstName string `json:"first_name"`
		Username  string `json:"username"`
	} `json:"result"`
	Description string `json:"description"`
}

func actRegisterToken(args map[string]any) llms.ToolReturn {
	token, _ := args["token"].(string)
	if token == "" {
		return core.NewErrorResponse("token is required for register_token")
	}

	url := fmt.Sprintf("%s/bot%s/getMe", telegramAPIBase, token)
	client := &http.Client{Timeout: 15 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to reach Telegram API: %v", err))
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to read Telegram API response: %v", err))
	}

	var getMeResp getMeResponse
	if err := json.Unmarshal(body, &getMeResp); err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to parse Telegram API response: %v", err))
	}

	if !getMeResp.Ok {
		desc := getMeResp.Description
		if desc == "" {
			desc = string(body)
		}
		return core.NewErrorResponse(fmt.Sprintf("Telegram API error: %s", desc))
	}

	if err := os.Setenv("TELEGRAM_BOT_TOKEN", token); err != nil {
		return core.NewErrorResponse(fmt.Sprintf("token is valid but failed to set env var: %v", err))
	}

	result := fmt.Sprintf(
		"Token registered successfully.\nBot ID: %d\nUsername: @%s\nName: %s\n\nTELEGRAM_BOT_TOKEN is now set in the current process.",
		getMeResp.Result.ID,
		getMeResp.Result.Username,
		getMeResp.Result.FirstName,
	)
	return core.NewSuccessResponse(result)
}
