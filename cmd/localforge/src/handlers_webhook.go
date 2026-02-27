package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/queue"
)

// WebhookRequest represents a generic webhook payload
type WebhookRequest struct {
	Provider string                 `json:"provider"`
	Event    string                 `json:"event"`
	Data     map[string]interface{} `json:"data"`
}

// handleWebhook processes incoming webhook requests from various providers
// POST /api/webhooks/:provider
func (s *Server) handleWebhook(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is required"})
		return
	}

	// Read raw body for signature verification
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// Verify webhook signature if secret is configured
	if err := s.verifyWebhookSignature(provider, bodyBytes, c.Request.Header); err != nil {
		agentforge.Debug("Webhook signature verification failed for %s: %v", provider, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "signature verification failed"})
		return
	}

	// Parse webhook payload
	var payload map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	agent := s.agentMgr.GetAgent()
	if agent == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "agent not ready"})
		return
	}

	// Format webhook payload as a message to the agent
	message := s.formatWebhookMessage(provider, payload)
	
	// Use a dedicated conversation ID for webhooks (or create a new one per webhook)
	// Option 1: Shared webhook conversation
	conversationID := fmt.Sprintf("webhook-%s", provider)
	// Option 2: New conversation per webhook (comment line above, uncomment below)
	// conversationID := ""

	agentforge.Debug("Processing webhook from %s: %s", provider, message)

	// Create a context for this webhook processing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Enrich message with metadata
	enriched := queue.FormatHeaders(message, map[string]string{
		"sender":   "webhook",
		"provider": provider,
	})

	// Start agent processing in the background
	// For webhooks, we typically don't stream back to the caller
	// Instead, we acknowledge receipt and process asynchronously
	go func() {
		providerInst := s.providerRegistry.Get(provider)
		if providerInst == nil {
			agentforge.Debug("No provider registered for %s", provider)
			// Still process the message but don't send a response
			responseCh := agent.ChatStream(ctx, enriched, conversationID)
			stream := responseCh.Start()
			for range stream {
				// Drain the stream
			}
			return
		}

		recipientID, err := providerInst.ExtractRecipient(payload)
		if err != nil {
			agentforge.Debug("Failed to extract recipient for %s: %v", provider, err)
			return
		}

		responseCh := agent.ChatStream(ctx, enriched, conversationID)
		stream := responseCh.Start()
		
		var fullResponse strings.Builder
		for chunk := range stream {
			if chunk.Status == "error" {
				agentforge.Debug("Webhook processing error: %s", chunk.Content)
				continue
			}
			// Accumulate content chunks only
			if chunk.Content != "" && chunk.Status != "tool_call" && chunk.Status != "tool_executing" && chunk.Status != "tool_result" {
				fullResponse.WriteString(chunk.Content)
			}
		}

		// Send accumulated response back via provider
		if fullResponse.Len() > 0 {
			err := providerInst.SendMessage(ctx, recipientID, fullResponse.String())
			if err != nil {
				agentforge.Debug("Failed to send message via %s: %v", provider, err)
			} else {
				agentforge.Debug("Successfully sent response to %s via %s", recipientID, provider)
			}
		}
	}()

	// Return immediate acknowledgment
	c.JSON(http.StatusOK, gin.H{
		"status":  "accepted",
		"message": "webhook received and processing",
		"provider": provider,
	})
}

// verifyWebhookSignature verifies the webhook signature based on provider
func (s *Server) verifyWebhookSignature(provider string, body []byte, headers http.Header) error {
	// Get webhook secret from environment
	secretEnvVar := fmt.Sprintf("WEBHOOK_SECRET_%s", strings.ToUpper(provider))
	secret := os.Getenv(secretEnvVar)
	
	// If no secret is configured, skip verification (insecure, but allows testing)
	if secret == "" {
		agentforge.Debug("No webhook secret configured for %s (set %s to enable verification)", provider, secretEnvVar)
		return nil
	}

	switch provider {
	case "github":
		return verifyGitHubSignature(body, headers.Get("X-Hub-Signature-256"), secret)
	case "stripe":
		return verifyStripeSignature(body, headers.Get("Stripe-Signature"), secret)
	case "instagram":
		return verifyInstagramSignature(body, headers.Get("X-Hub-Signature-256"), secret)
	case "telegram":
		return verifyTelegramSignature(headers.Get("X-Telegram-Bot-Api-Secret-Token"), secret)
	default:
		// Generic HMAC-SHA256 verification
		signature := headers.Get("X-Webhook-Signature")
		if signature == "" {
			return fmt.Errorf("no signature header found")
		}
		return verifyHMACSignature(body, signature, secret)
	}
}

// verifyGitHubSignature verifies GitHub webhook signature
func verifyGitHubSignature(body []byte, signature, secret string) error {
	if signature == "" {
		return fmt.Errorf("missing X-Hub-Signature-256 header")
	}

	// Remove "sha256=" prefix
	signature = strings.TrimPrefix(signature, "sha256=")
	
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedMAC)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

// verifyStripeSignature verifies Stripe webhook signature
func verifyStripeSignature(body []byte, signature, secret string) error {
	if signature == "" {
		return fmt.Errorf("missing Stripe-Signature header")
	}

	// Stripe signature format: t=timestamp,v1=signature
	// For simplicity, we'll extract v1 and verify
	parts := strings.Split(signature, ",")
	var timestamp, sig string
	for _, part := range parts {
		if strings.HasPrefix(part, "t=") {
			timestamp = strings.TrimPrefix(part, "t=")
		} else if strings.HasPrefix(part, "v1=") {
			sig = strings.TrimPrefix(part, "v1=")
		}
	}

	if timestamp == "" || sig == "" {
		return fmt.Errorf("invalid Stripe signature format")
	}

	// Create signed payload: timestamp.body
	signedPayload := fmt.Sprintf("%s.%s", timestamp, string(body))
	
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expectedMAC)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

// verifyHMACSignature verifies generic HMAC-SHA256 signature
func verifyHMACSignature(body []byte, signature, secret string) error {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedMAC)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

// verifyInstagramSignature verifies Instagram webhook signature
// Instagram uses the same HMAC-SHA256 signature scheme as GitHub
func verifyInstagramSignature(body []byte, signature, secret string) error {
	if signature == "" {
		return fmt.Errorf("missing X-Hub-Signature-256 header")
	}

	// Remove "sha256=" prefix if present
	signature = strings.TrimPrefix(signature, "sha256=")
	
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedMAC)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

// verifyTelegramSignature verifies Telegram webhook signature
// Telegram uses a simple secret token comparison
func verifyTelegramSignature(token, secret string) error {
	if token == "" {
		return fmt.Errorf("missing X-Telegram-Bot-Api-Secret-Token header")
	}

	if token != secret {
		return fmt.Errorf("token mismatch")
	}
	return nil
}

// formatWebhookMessage formats the webhook payload into a human-readable message
func (s *Server) formatWebhookMessage(provider string, payload map[string]interface{}) string {
	// Try to extract common fields
	eventType, _ := payload["event"].(string)
	action, _ := payload["action"].(string)
	
	var message strings.Builder
	message.WriteString(fmt.Sprintf("[Webhook from %s]\n", provider))
	
	if eventType != "" {
		message.WriteString(fmt.Sprintf("Event: %s\n", eventType))
	}
	if action != "" {
		message.WriteString(fmt.Sprintf("Action: %s\n", action))
	}

	// Provider-specific formatting
	switch provider {
	case "github":
		message.WriteString(s.formatGitHubWebhook(payload))
	case "stripe":
		message.WriteString(s.formatStripeWebhook(payload))
	case "instagram":
		message.WriteString(s.formatInstagramWebhook(payload))
	case "telegram":
		message.WriteString(s.formatTelegramWebhook(payload))
	default:
		// Generic formatting - pretty print the entire payload
		jsonBytes, err := json.MarshalIndent(payload, "", "  ")
		if err == nil {
			message.WriteString(fmt.Sprintf("\nPayload:\n%s", string(jsonBytes)))
		}
	}

	return message.String()
}

// formatGitHubWebhook formats GitHub-specific webhook events
func (s *Server) formatGitHubWebhook(payload map[string]interface{}) string {
	var msg strings.Builder
	
	// Extract common GitHub fields
	if repo, ok := payload["repository"].(map[string]interface{}); ok {
		if name, ok := repo["full_name"].(string); ok {
			msg.WriteString(fmt.Sprintf("Repository: %s\n", name))
		}
	}
	
	if sender, ok := payload["sender"].(map[string]interface{}); ok {
		if login, ok := sender["login"].(string); ok {
			msg.WriteString(fmt.Sprintf("Sender: %s\n", login))
		}
	}

	// Handle specific event types
	if commits, ok := payload["commits"].([]interface{}); ok {
		msg.WriteString(fmt.Sprintf("\nCommits (%d):\n", len(commits)))
		for i, c := range commits {
			if commit, ok := c.(map[string]interface{}); ok {
				if message, ok := commit["message"].(string); ok {
					msg.WriteString(fmt.Sprintf("  %d. %s\n", i+1, message))
				}
			}
		}
	}

	if issue, ok := payload["issue"].(map[string]interface{}); ok {
		if title, ok := issue["title"].(string); ok {
			msg.WriteString(fmt.Sprintf("Issue: %s\n", title))
		}
	}

	if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
		if title, ok := pr["title"].(string); ok {
			msg.WriteString(fmt.Sprintf("Pull Request: %s\n", title))
		}
	}

	return msg.String()
}

// formatStripeWebhook formats Stripe-specific webhook events
func (s *Server) formatStripeWebhook(payload map[string]interface{}) string {
	var msg strings.Builder
	
	if eventType, ok := payload["type"].(string); ok {
		msg.WriteString(fmt.Sprintf("Event Type: %s\n", eventType))
	}

	if data, ok := payload["data"].(map[string]interface{}); ok {
		if obj, ok := data["object"].(map[string]interface{}); ok {
			if id, ok := obj["id"].(string); ok {
				msg.WriteString(fmt.Sprintf("Object ID: %s\n", id))
			}
			if amount, ok := obj["amount"].(float64); ok {
				msg.WriteString(fmt.Sprintf("Amount: $%.2f\n", amount/100))
			}
			if status, ok := obj["status"].(string); ok {
				msg.WriteString(fmt.Sprintf("Status: %s\n", status))
			}
		}
	}

	return msg.String()
}

// formatInstagramWebhook formats Instagram-specific webhook events
func (s *Server) formatInstagramWebhook(payload map[string]interface{}) string {
	var msg strings.Builder

	// Extract entry and messaging data
	entry, ok := payload["entry"].([]interface{})
	if !ok || len(entry) == 0 {
		return "Invalid Instagram payload format"
	}

	firstEntry, ok := entry[0].(map[string]interface{})
	if !ok {
		return "Invalid entry format"
	}

	messaging, ok := firstEntry["messaging"].([]interface{})
	if !ok || len(messaging) == 0 {
		return "Invalid messaging format"
	}

	firstMessage, ok := messaging[0].(map[string]interface{})
	if !ok {
		return "Invalid message format"
	}

	// Extract sender info
	if sender, ok := firstMessage["sender"].(map[string]interface{}); ok {
		if senderID, ok := sender["id"].(string); ok {
			msg.WriteString(fmt.Sprintf("From: %s\n", senderID))
		}
	}

	// Extract message text
	if message, ok := firstMessage["message"].(map[string]interface{}); ok {
		if text, ok := message["text"].(string); ok {
			msg.WriteString(fmt.Sprintf("Message: %s\n", text))
		}
	}

	return msg.String()
}

// formatTelegramWebhook formats Telegram-specific webhook events
func (s *Server) formatTelegramWebhook(payload map[string]interface{}) string {
	var msg strings.Builder

	// Try to get message or edited_message
	var message map[string]interface{}
	if m, ok := payload["message"].(map[string]interface{}); ok {
		message = m
	} else if m, ok := payload["edited_message"].(map[string]interface{}); ok {
		message = m
		msg.WriteString("(Edited) ")
	}

	if message == nil {
		return "Invalid Telegram payload format"
	}

	// Extract sender info
	if from, ok := message["from"].(map[string]interface{}); ok {
		if username, ok := from["username"].(string); ok {
			msg.WriteString(fmt.Sprintf("From: @%s\n", username))
		} else if firstName, ok := from["first_name"].(string); ok {
			msg.WriteString(fmt.Sprintf("From: %s", firstName))
			if lastName, ok := from["last_name"].(string); ok {
				msg.WriteString(fmt.Sprintf(" %s", lastName))
			}
			msg.WriteString("\n")
		}
	}

	// Extract message text
	if text, ok := message["text"].(string); ok {
		msg.WriteString(fmt.Sprintf("Message: %s\n", text))
	}

	return msg.String()
}

// handleWebhookSync is a synchronous webhook handler that streams the response back
// POST /api/webhooks/:provider/sync
func (s *Server) handleWebhookSync(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is required"})
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if err := s.verifyWebhookSignature(provider, bodyBytes, c.Request.Header); err != nil {
		agentforge.Debug("Webhook signature verification failed for %s: %v", provider, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "signature verification failed"})
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	agent := s.agentMgr.GetAgent()
	if agent == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "agent not ready"})
		return
	}

	message := s.formatWebhookMessage(provider, payload)
	conversationID := fmt.Sprintf("webhook-%s", provider)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	writer := NewSSEWriter(c)
	writer.SetHeaders()

	enriched := queue.FormatHeaders(message, map[string]string{
		"sender":   "webhook",
		"provider": provider,
	})
	
	responseCh := agent.ChatStream(ctx, enriched, conversationID)
	stream := responseCh.Start()

	for {
		select {
		case <-ctx.Done():
			errorChunk := core.ExtendedChunkResponse{
				Content: "Processing stopped",
				Status:  "error",
			}
			_ = writer.WriteEvent("error", errorChunk)
			return
		case chunk, ok := <-stream:
			if !ok {
				return
			}
			eventType := EventTypeFromChunk(chunk)
			if err := writer.WriteEvent(eventType, chunk); err != nil {
				return
			}
		}
	}
}
