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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thinktwiceco/agent-forge/cmd/localforge/src/providers"
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

	agentforge.Debug("Processing webhook from %s: %s", provider, message)

	// Start agent processing in the background.
	// context.Background() is used inside the goroutine so that the HTTP
	// handler returning does not cancel the ongoing agent processing.
	go func() {
		ctx := context.Background()
		providerInst := s.providerRegistry.Get(provider)

		conversationID := fmt.Sprintf("webhook-%s", provider)

		enriched := queue.FormatHeaders(message, map[string]string{
			"sender":   "webhook",
			"provider": provider,
		})

		if providerInst == nil {
			agentforge.Debug("No provider registered for %s", provider)
			responseCh := agent.ChatStream(ctx, enriched, conversationID)
			stream := responseCh.Start()
			for range stream {
			}
			return
		}

		recipientID, err := providerInst.ExtractRecipient(payload)
		if err != nil {
			agentforge.Debug("Failed to extract recipient for %s: %v", provider, err)
			return
		}

		if tp, ok := providerInst.(*providers.TelegramProvider); ok {
			if !tp.AllowlistPermits(payload) {
				agentforge.Debug("Blocked webhook from telegram: set TELEGRAM_ALLOWED_USER_IDS or sender username not in allowlist")
				return
			}
		} else if ap, ok := providerInst.(AllowlistProvider); ok && !ap.IsAllowed(recipientID) {
			agentforge.Debug("Blocked webhook from %s (recipient %s not in allowlist)", provider, recipientID)
			return
		}

		if provider == "telegram" {
			if text, ok := providers.TelegramMessageText(payload); ok && providers.IsTelegramNewConversationCommand(text) {
				s.telegramThreads.NewSession(recipientID)
				if err := providerInst.SendMessage(ctx, recipientID, "Started a new conversation. Your next message will use a fresh thread."); err != nil {
					agentforge.Debug("Failed to send new-conversation ack via %s: %v", provider, err)
				}
				return
			}
			conversationID = s.telegramThreads.ResolveConversationID(recipientID)
		} else {
			conversationID = fmt.Sprintf("webhook-%s-%s", provider, recipientID)
		}

		// For Telegram callback_query updates, immediately answer the callback to
		// dismiss the loading spinner on the client side.
		if tp, ok := providerInst.(*providers.TelegramProvider); ok {
			if cbq, ok := payload["callback_query"].(map[string]interface{}); ok {
				if cbqID, ok := cbq["id"].(string); ok && cbqID != "" {
					if err := tp.AnswerCallbackQuery(ctx, cbqID); err != nil {
						agentforge.Debug("Failed to answer callback query: %v", err)
					}
				}
			}
		}

		// For providers that support editable messages, send an immediate
		// placeholder and start a typing indicator loop, then replace the
		// placeholder with the final response. Plain providers fall through
		// to a simple SendMessage.
		var msgRef string
		var typingCancel context.CancelFunc

		if ep, ok := providerInst.(EditableProvider); ok {
			ref, err := ep.SendInitialMessage(ctx, recipientID, "⏳ Processing your request...")
			if err != nil {
				agentforge.Debug("Failed to send initial message via %s: %v", provider, err)
			} else {
				msgRef = ref
				agentforge.Debug("Sent initial message via %s (ref=%s)", provider, msgRef)
			}

			// Typing indicator: Telegram expires it after ~5 s, so refresh every 4 s.
			typingCtx, cancel := context.WithCancel(ctx)
			typingCancel = cancel
			go func() {
				ticker := time.NewTicker(4 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-typingCtx.Done():
						return
					case <-ticker.C:
						_ = ep.SendTypingAction(typingCtx, recipientID)
					}
				}
			}()
		}

		responseCh := agent.ChatStream(ctx, enriched, conversationID)
		stream := responseCh.Start()

		var fullResponse strings.Builder
		for chunk := range stream {
			if chunk.Status == "error" {
				agentforge.Debug("Webhook processing error: %s", chunk.Content)
				continue
			}
			if chunk.Content != "" && chunk.Status != "tool_call" && chunk.Status != "tool_executing" && chunk.Status != "tool_result" {
				fullResponse.WriteString(chunk.Content)
			}
		}

		if typingCancel != nil {
			typingCancel()
		}

		if fullResponse.Len() == 0 {
			return
		}

		response := fullResponse.String()

		if ep, ok := providerInst.(EditableProvider); ok && msgRef != "" {
			if err := ep.UpdateMessage(ctx, recipientID, msgRef, response); err != nil {
				agentforge.Debug("Failed to update message via %s: %v", provider, err)
				// Fallback to a new message
				if err := providerInst.SendMessage(ctx, recipientID, response); err != nil {
					agentforge.Debug("Failed to send fallback message via %s: %v", provider, err)
				}
			} else {
				agentforge.Debug("Successfully updated message via %s (ref=%s)", provider, msgRef)
			}
			return
		}

		if err := providerInst.SendMessage(ctx, recipientID, response); err != nil {
			agentforge.Debug("Failed to send message via %s: %v", provider, err)
		} else {
			agentforge.Debug("Successfully sent response to %s via %s", recipientID, provider)
		}
	}()

	// Return immediate acknowledgment
	c.JSON(http.StatusOK, gin.H{
		"status":   "accepted",
		"message":  "webhook received and processing",
		"provider": provider,
	})
}

// verifyWebhookSignature verifies the webhook signature based on provider
func (s *Server) verifyWebhookSignature(provider string, body []byte, headers http.Header) error {
	// Telegram always requires a shared secret; unauthenticated webhooks are rejected.
	if provider == "telegram" {
		secret := strings.TrimSpace(os.Getenv("WEBHOOK_SECRET_TELEGRAM"))
		if secret == "" {
			return fmt.Errorf("WEBHOOK_SECRET_TELEGRAM is required for Telegram webhooks")
		}
		return verifyTelegramSignature(headers.Get("X-Telegram-Bot-Api-Secret-Token"), secret)
	}

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
	fmt.Fprintf(&message, "[Webhook from %s]\n", provider)

	if eventType != "" {
		fmt.Fprintf(&message, "Event: %s\n", eventType)
	}
	if action != "" {
		fmt.Fprintf(&message, "Action: %s\n", action)
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
			fmt.Fprintf(&message, "\nPayload:\n%s", string(jsonBytes))
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
			fmt.Fprintf(&msg, "Repository: %s\n", name)
		}
	}

	if sender, ok := payload["sender"].(map[string]interface{}); ok {
		if login, ok := sender["login"].(string); ok {
			fmt.Fprintf(&msg, "Sender: %s\n", login)
		}
	}

	// Handle specific event types
	if commits, ok := payload["commits"].([]interface{}); ok {
		fmt.Fprintf(&msg, "\nCommits (%d):\n", len(commits))
		for i, c := range commits {
			if commit, ok := c.(map[string]interface{}); ok {
				if message, ok := commit["message"].(string); ok {
					fmt.Fprintf(&msg, "  %d. %s\n", i+1, message)
				}
			}
		}
	}

	if issue, ok := payload["issue"].(map[string]interface{}); ok {
		if title, ok := issue["title"].(string); ok {
			fmt.Fprintf(&msg, "Issue: %s\n", title)
		}
	}

	if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
		if title, ok := pr["title"].(string); ok {
			fmt.Fprintf(&msg, "Pull Request: %s\n", title)
		}
	}

	return msg.String()
}

// formatStripeWebhook formats Stripe-specific webhook events
func (s *Server) formatStripeWebhook(payload map[string]interface{}) string {
	var msg strings.Builder

	if eventType, ok := payload["type"].(string); ok {
		fmt.Fprintf(&msg, "Event Type: %s\n", eventType)
	}

	if data, ok := payload["data"].(map[string]interface{}); ok {
		if obj, ok := data["object"].(map[string]interface{}); ok {
			if id, ok := obj["id"].(string); ok {
				fmt.Fprintf(&msg, "Object ID: %s\n", id)
			}
			if amount, ok := obj["amount"].(float64); ok {
				fmt.Fprintf(&msg, "Amount: $%.2f\n", amount/100)
			}
			if status, ok := obj["status"].(string); ok {
				fmt.Fprintf(&msg, "Status: %s\n", status)
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
			fmt.Fprintf(&msg, "From: %s\n", senderID)
		}
	}

	// Extract message text
	if message, ok := firstMessage["message"].(map[string]interface{}); ok {
		if text, ok := message["text"].(string); ok {
			fmt.Fprintf(&msg, "Message: %s\n", text)
		}
	}

	return msg.String()
}

// formatTelegramSender writes the sender's name/username from a Telegram
// message map into msg.
func formatTelegramSender(msg *strings.Builder, message map[string]interface{}) {
	from, ok := message["from"].(map[string]interface{})
	if !ok {
		return
	}
	if username, ok := from["username"].(string); ok {
		fmt.Fprintf(msg, "From: @%s\n", username)
	} else if firstName, ok := from["first_name"].(string); ok {
		fmt.Fprintf(msg, "From: %s", firstName)
		if lastName, ok := from["last_name"].(string); ok {
			fmt.Fprintf(msg, " %s", lastName)
		}
		msg.WriteString("\n")
	}
}

// formatTelegramWebhook formats Telegram-specific webhook events
func (s *Server) formatTelegramWebhook(payload map[string]interface{}) string {
	var msg strings.Builder

	// callback_query — inline keyboard button tap
	if cbq, ok := payload["callback_query"].(map[string]interface{}); ok {
		msg.WriteString("(Callback) ")
		formatTelegramSender(&msg, cbq)
		if data, ok := cbq["data"].(string); ok {
			fmt.Fprintf(&msg, "Callback data: %s\n", data)
		}
		// Include the original message text for context if present
		if origMsg, ok := cbq["message"].(map[string]interface{}); ok {
			if text, ok := origMsg["text"].(string); ok {
				fmt.Fprintf(&msg, "Original message: %s\n", text)
			}
		}
		return msg.String()
	}

	// Regular message or edited message
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

	formatTelegramSender(&msg, message)

	// Text message
	if text, ok := message["text"].(string); ok {
		fmt.Fprintf(&msg, "Message: %s\n", text)
		return msg.String()
	}

	// Media / attachment types — describe them so the agent has context
	if caption, ok := message["caption"].(string); ok && caption != "" {
		fmt.Fprintf(&msg, "Caption: %s\n", caption)
	}

	if _, ok := message["photo"]; ok {
		msg.WriteString("Message: [user sent a photo]\n")
	} else if doc, ok := message["document"].(map[string]interface{}); ok {
		if name, ok := doc["file_name"].(string); ok && name != "" {
			fmt.Fprintf(&msg, "Message: [user sent a file: %s]\n", name)
		} else {
			msg.WriteString("Message: [user sent a file]\n")
		}
	} else if _, ok := message["voice"]; ok {
		msg.WriteString("Message: [user sent a voice message]\n")
	} else if _, ok := message["video"]; ok {
		msg.WriteString("Message: [user sent a video]\n")
	} else if _, ok := message["audio"]; ok {
		msg.WriteString("Message: [user sent an audio file]\n")
	} else if _, ok := message["sticker"]; ok {
		msg.WriteString("Message: [user sent a sticker]\n")
	} else if loc, ok := message["location"].(map[string]interface{}); ok {
		lat, _ := loc["latitude"].(float64)
		lon, _ := loc["longitude"].(float64)
		fmt.Fprintf(&msg, "Message: [user shared a location: %.6f, %.6f]\n", lat, lon)
	} else if _, ok := message["contact"]; ok {
		msg.WriteString("Message: [user shared a contact]\n")
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

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	message := s.formatWebhookMessage(provider, payload)
	conversationID := fmt.Sprintf("webhook-%s", provider)

	providerInst := s.providerRegistry.Get(provider)
	if providerInst != nil {
		recipientID, err := providerInst.ExtractRecipient(payload)
		if err != nil {
			agentforge.Debug("handleWebhookSync: extract recipient: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "could not extract recipient"})
			return
		}
		if tp, ok := providerInst.(*providers.TelegramProvider); ok {
			if !tp.AllowlistPermits(payload) {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
		} else if ap, ok := providerInst.(AllowlistProvider); ok && !ap.IsAllowed(recipientID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if provider == "telegram" {
			if text, ok := providers.TelegramMessageText(payload); ok && providers.IsTelegramNewConversationCommand(text) {
				s.telegramThreads.NewSession(recipientID)
				if err := providerInst.SendMessage(ctx, recipientID, "Started a new conversation. Your next message will use a fresh thread."); err != nil {
					agentforge.Debug("handleWebhookSync: send ack: %v", err)
				}
				c.JSON(http.StatusOK, gin.H{"status": "accepted", "event": "new_conversation"})
				return
			}
			conversationID = s.telegramThreads.ResolveConversationID(recipientID)
		} else {
			conversationID = fmt.Sprintf("webhook-%s-%s", provider, recipientID)
		}
	}

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
