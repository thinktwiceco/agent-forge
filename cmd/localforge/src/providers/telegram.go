package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// TelegramProvider implements the Provider interface for Telegram messaging
type TelegramProvider struct {
	botToken string
}

// NewTelegramProvider creates a new Telegram provider
func NewTelegramProvider(botToken string) *TelegramProvider {
	return &TelegramProvider{
		botToken: botToken,
	}
}

// Name returns the provider name
func (p *TelegramProvider) Name() string {
	return "telegram"
}

// ExtractRecipient extracts the chat ID from Telegram webhook payload
func (p *TelegramProvider) ExtractRecipient(payload map[string]interface{}) (string, error) {
	// Telegram webhook structure: message.chat.id
	message, ok := payload["message"].(map[string]interface{})
	if !ok {
		// Try edited_message as fallback
		message, ok = payload["edited_message"].(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("missing 'message' or 'edited_message' field")
		}
	}

	chat, ok := message["chat"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("missing 'chat' field")
	}

	// Chat ID can be either number or string
	chatID, ok := chat["id"]
	if !ok {
		return "", fmt.Errorf("missing chat 'id'")
	}

	// Convert to string if it's a number
	switch v := chatID.(type) {
	case float64:
		return fmt.Sprintf("%.0f", v), nil
	case int:
		return fmt.Sprintf("%d", v), nil
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("invalid chat id type: %T", chatID)
	}
}

// SendMessage sends a message to a Telegram chat via the Bot API
func (p *TelegramProvider) SendMessage(ctx context.Context, chatID string, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", p.botToken)

	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorBody bytes.Buffer
		errorBody.ReadFrom(resp.Body)
		return fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, errorBody.String())
	}

	return nil
}
