package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/queue"
)

func (s *Server) handleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if strings.TrimSpace(req.Message) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}

	agent := s.agentMgr.GetAgent()
	if agent == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "agent not ready"})
		return
	}

	conversationID := c.Query("conversationId")

	// Log conversation ID for debugging context loss issues
	if conversationID == "" {
		agentforge.Debug("No conversationID provided - starting new conversation")
	} else {
		agentforge.Debug("Continuing conversation with ID: %s", conversationID)
	}

	// Create a context with cancel for this request
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Register cancel function if we have a conversation ID
	// Note: For new conversations, we'll get the ID from the first chunk
	var chatReg *ChatRegistration

	writer := NewSSEWriter(c)
	writer.SetHeaders()

	enriched := queue.FormatHeaders(req.Message, map[string]string{"sender": "user"})
	responseCh := agent.ChatStream(ctx, enriched, conversationID)
	stream := responseCh.Start()

	defer func() {
		if chatReg != nil {
			s.convRegistry.Unregister(chatReg)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			errorChunk := core.ExtendedChunkResponse{
				Content: "Agent stopped",
				Status:  "error",
			}
			_ = writer.WriteEvent("error", errorChunk)
			return
		case chunk, ok := <-stream:
			if !ok {
				return
			}

			if chatReg == nil && chunk.ChatId != "" {
				chatReg = s.convRegistry.Register(chunk.ChatId, cancel)
			}

			eventType := EventTypeFromChunk(chunk)
			if err := writer.WriteEvent(eventType, chunk); err != nil {
				return
			}
		}
	}
}
