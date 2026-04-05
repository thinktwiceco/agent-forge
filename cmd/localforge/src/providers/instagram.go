package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// InstagramProvider implements the Provider interface for Instagram messaging
type InstagramProvider struct {
	accessToken string
	apiVersion  string
	client      *http.Client
}

// NewInstagramProvider creates a new Instagram provider
func NewInstagramProvider(accessToken string) *InstagramProvider {
	return &InstagramProvider{
		accessToken: accessToken,
		apiVersion:  "v18.0",
		client:      &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the provider name
func (p *InstagramProvider) Name() string {
	return "instagram"
}

// ExtractRecipient extracts the sender ID from Instagram webhook payload
func (p *InstagramProvider) ExtractRecipient(payload map[string]interface{}) (string, error) {
	// Instagram webhook structure: entry[0].messaging[0].sender.id
	entry, ok := payload["entry"].([]interface{})
	if !ok || len(entry) == 0 {
		return "", fmt.Errorf("missing or invalid 'entry' field")
	}

	firstEntry, ok := entry[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid entry format")
	}

	messaging, ok := firstEntry["messaging"].([]interface{})
	if !ok || len(messaging) == 0 {
		return "", fmt.Errorf("missing or invalid 'messaging' field")
	}

	firstMessage, ok := messaging[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid messaging format")
	}

	sender, ok := firstMessage["sender"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("missing 'sender' field")
	}

	senderID, ok := sender["id"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid sender 'id'")
	}

	return senderID, nil
}

// SendMessage sends a message to an Instagram user via the Graph API
func (p *InstagramProvider) SendMessage(ctx context.Context, recipient string, message string) error {
	url := fmt.Sprintf("https://graph.facebook.com/%s/me/messages", p.apiVersion)

	payload := map[string]interface{}{
		"recipient": map[string]string{
			"id": recipient,
		},
		"message": map[string]string{
			"text": message,
		},
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.accessToken))

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var errorBody bytes.Buffer
		_, _ = errorBody.ReadFrom(resp.Body)
		return fmt.Errorf("instagram API error (status %d): %s", resp.StatusCode, errorBody.String())
	}

	return nil
}
