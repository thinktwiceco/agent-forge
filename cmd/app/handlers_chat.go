package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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

	writer := NewSSEWriter(c)
	writer.SetHeaders()

	responseCh := agent.ChatStream(req.Message, conversationID)
	stream := responseCh.Start()

	for {
		select {
		case <-c.Request.Context().Done():
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
