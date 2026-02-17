package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
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
	var registeredConvID string

	writer := NewSSEWriter(c)
	writer.SetHeaders()

	responseCh := agent.ChatStream(ctx, req.Message, conversationID)
	stream := responseCh.Start()

	// Clean up function
	defer func() {
		if registeredConvID != "" {
			s.convRegistry.Unregister(registeredConvID)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			// Context cancelled - send error event and return
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

			// Extract conversation ID from chunk if we haven't registered yet
			if registeredConvID == "" && chunk.ChatId != "" {
				registeredConvID = chunk.ChatId
				s.convRegistry.Register(chunk.ChatId, cancel)
			}

			eventType := EventTypeFromChunk(chunk)
			if err := writer.WriteEvent(eventType, chunk); err != nil {
				return
			}
		}
	}
}
