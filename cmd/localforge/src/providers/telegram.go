package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// TelegramProvider implements the Provider interface for Telegram messaging
type TelegramProvider struct {
	botToken         string
	client           *http.Client
	allowedUsernames map[string]struct{} // lowercase, no @; empty = deny all (TELEGRAM_ALLOWED_USER_IDS required)
}

// NewTelegramProvider creates a new Telegram provider.
// allowlistEntries (TELEGRAM_ALLOWED_USER_IDS) must contain at least one Telegram username
// (with or without @), matched case-insensitively against message.from.username.
// An empty list blocks all incoming webhooks.
func NewTelegramProvider(botToken string, allowlistEntries []string) *TelegramProvider {
	allowed := make(map[string]struct{}, len(allowlistEntries))
	for _, raw := range allowlistEntries {
		u := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "@")))
		if u != "" {
			allowed[u] = struct{}{}
		}
	}
	return &TelegramProvider{
		botToken:         botToken,
		client:           &http.Client{Timeout: 30 * time.Second},
		allowedUsernames: allowed,
	}
}

// AllowlistPermits returns true when the sender's username (message.from.username /
// callback_query.from.username) is listed in TELEGRAM_ALLOWED_USER_IDS.
// Matching is case-insensitive; leading @ is ignored.
// Returns false when the allowlist is empty (TELEGRAM_ALLOWED_USER_IDS not set).
func (p *TelegramProvider) AllowlistPermits(payload map[string]interface{}) bool {
	if len(p.allowedUsernames) == 0 {
		return false
	}
	from, ok := telegramFromPayload(payload)
	if !ok {
		return false
	}
	username, _ := from["username"].(string)
	if username == "" {
		return false
	}
	_, ok = p.allowedUsernames[strings.ToLower(username)]
	return ok
}

func telegramFromPayload(payload map[string]interface{}) (map[string]interface{}, bool) {
	if cbq, ok := payload["callback_query"].(map[string]interface{}); ok {
		if from, ok := cbq["from"].(map[string]interface{}); ok {
			return from, true
		}
	}
	var msg map[string]interface{}
	if m, ok := payload["message"].(map[string]interface{}); ok {
		msg = m
	} else if m, ok := payload["edited_message"].(map[string]interface{}); ok {
		msg = m
	}
	if msg == nil {
		return nil, false
	}
	if from, ok := msg["from"].(map[string]interface{}); ok {
		return from, true
	}
	return nil, false
}

// Name returns the provider name
func (p *TelegramProvider) Name() string {
	return "telegram"
}

// formatTelegramScalarID formats a user or chat id from JSON-decoded Update objects.
func formatTelegramScalarID(id interface{}) (string, error) {
	switch v := id.(type) {
	case float64:
		return fmt.Sprintf("%.0f", v), nil
	case int:
		return fmt.Sprintf("%d", v), nil
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("invalid id type: %T", id)
	}
}

// extractChatID parses a chat.id value from a Telegram message map.
func extractChatID(message map[string]interface{}) (string, error) {
	chat, ok := message["chat"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("missing 'chat' field")
	}
	chatID, ok := chat["id"]
	if !ok {
		return "", fmt.Errorf("missing chat 'id'")
	}
	return formatTelegramScalarID(chatID)
}

// ExtractRecipient extracts the chat ID from a Telegram webhook payload.
// Handles message, edited_message, and callback_query update types.
func (p *TelegramProvider) ExtractRecipient(payload map[string]interface{}) (string, error) {
	if message, ok := payload["message"].(map[string]interface{}); ok {
		return extractChatID(message)
	}
	if message, ok := payload["edited_message"].(map[string]interface{}); ok {
		return extractChatID(message)
	}
	// callback_query carries the originating message inside cbq.message
	if cbq, ok := payload["callback_query"].(map[string]interface{}); ok {
		if message, ok := cbq["message"].(map[string]interface{}); ok {
			return extractChatID(message)
		}
	}
	return "", fmt.Errorf("unsupported update type: no message, edited_message, or callback_query found")
}

// TelegramMessageText returns the text of a regular or edited message, if any.
func TelegramMessageText(payload map[string]interface{}) (string, bool) {
	var msg map[string]interface{}
	if m, ok := payload["message"].(map[string]interface{}); ok {
		msg = m
	} else if m, ok := payload["edited_message"].(map[string]interface{}); ok {
		msg = m
	} else {
		return "", false
	}
	text, ok := msg["text"].(string)
	return text, ok && strings.TrimSpace(text) != ""
}

// IsTelegramNewConversationCommand reports whether text is only /new_conversation
// with an optional @BotUsername suffix.
func IsTelegramNewConversationCommand(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	fields := strings.Fields(text)
	if len(fields) != 1 {
		return false
	}
	cmd := fields[0]
	if i := strings.IndexByte(cmd, '@'); i >= 0 {
		cmd = cmd[:i]
	}
	return strings.EqualFold(cmd, "/new_conversation")
}

const telegramMaxLen = 4096

// splitMessage splits text into chunks that respect Telegram's 4096-character
// limit. It prefers splitting on newline boundaries to avoid mid-sentence cuts.
func splitMessage(text string) []string {
	if len(text) <= telegramMaxLen {
		return []string{text}
	}

	var chunks []string
	for len(text) > 0 {
		if len(text) <= telegramMaxLen {
			chunks = append(chunks, text)
			break
		}

		// Find the last newline within the allowed window.
		window := text[:telegramMaxLen]
		splitAt := strings.LastIndex(window, "\n")
		if splitAt <= 0 {
			// No newline found; hard-cut at the limit.
			splitAt = telegramMaxLen
		} else {
			splitAt++ // include the newline in the current chunk
		}

		chunks = append(chunks, text[:splitAt])
		text = text[splitAt:]
	}
	return chunks
}

// TelegramResponse represents the Telegram API response structure
type TelegramResponse struct {
	Ok     bool `json:"ok"`
	Result struct {
		MessageID int `json:"message_id"`
	} `json:"result"`
}

// SendMessage sends a message to a Telegram chat via the Bot API.
// Long messages are automatically split into multiple messages to respect
// Telegram's 4096-character limit.
func (p *TelegramProvider) SendMessage(ctx context.Context, chatID string, message string) error {
	for _, chunk := range splitMessage(message) {
		if _, err := p.SendMessageWithID(ctx, chatID, chunk); err != nil {
			return err
		}
	}
	return nil
}

// SendMessageWithID sends a message and returns the message ID
func (p *TelegramProvider) SendMessageWithID(ctx context.Context, chatID string, message string) (int, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", p.botToken)

	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var errorBody bytes.Buffer
		_, _ = errorBody.ReadFrom(resp.Body)
		return 0, fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, errorBody.String())
	}

	var telegramResp TelegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	if !telegramResp.Ok {
		return 0, fmt.Errorf("telegram API returned ok=false")
	}

	return telegramResp.Result.MessageID, nil
}

// EditMessage edits an existing message
func (p *TelegramProvider) EditMessage(ctx context.Context, chatID string, messageID int, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", p.botToken)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       message,
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

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var errorBody bytes.Buffer
		_, _ = errorBody.ReadFrom(resp.Body)
		return fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, errorBody.String())
	}

	return nil
}

// SendInitialMessage implements EditableProvider: sends a placeholder message
// and returns the message ID as the opaque msgRef string.
func (p *TelegramProvider) SendInitialMessage(ctx context.Context, recipient string, text string) (string, error) {
	id, err := p.SendMessageWithID(ctx, recipient, text)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", id), nil
}

// UpdateMessage implements EditableProvider: edits an existing message identified
// by msgRef (the string message ID returned by SendInitialMessage). If the
// response exceeds Telegram's 4096-character limit it edits the placeholder with
// the first chunk and sends the remaining chunks as new messages. Falls back to
// SendMessage when msgRef is invalid.
func (p *TelegramProvider) UpdateMessage(ctx context.Context, recipient string, msgRef string, text string) error {
	var msgID int
	if _, err := fmt.Sscanf(msgRef, "%d", &msgID); err != nil || msgID == 0 {
		return p.SendMessage(ctx, recipient, text)
	}

	chunks := splitMessage(text)
	if err := p.EditMessage(ctx, recipient, msgID, chunks[0]); err != nil {
		return err
	}
	for _, chunk := range chunks[1:] {
		if _, err := p.SendMessageWithID(ctx, recipient, chunk); err != nil {
			return err
		}
	}
	return nil
}

// SendTypingAction implements EditableProvider: broadcasts a "typing" indicator.
func (p *TelegramProvider) SendTypingAction(ctx context.Context, recipient string) error {
	return p.SendChatAction(ctx, recipient, "typing")
}

// SendChatAction sends a chat action (e.g., "typing") to show activity
func (p *TelegramProvider) SendChatAction(ctx context.Context, chatID string, action string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendChatAction", p.botToken)

	payload := map[string]interface{}{
		"chat_id": chatID,
		"action":  action,
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

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var errorBody bytes.Buffer
		_, _ = errorBody.ReadFrom(resp.Body)
		return fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, errorBody.String())
	}

	return nil
}

// AnswerCallbackQuery dismisses the loading spinner on the client after an
// inline keyboard button tap. callbackQueryID is taken from callback_query.id
// in the webhook payload.
func (p *TelegramProvider) AnswerCallbackQuery(ctx context.Context, callbackQueryID string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", p.botToken)

	payload := map[string]interface{}{
		"callback_query_id": callbackQueryID,
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

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var errorBody bytes.Buffer
		_, _ = errorBody.ReadFrom(resp.Body)
		return fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, errorBody.String())
	}

	return nil
}
