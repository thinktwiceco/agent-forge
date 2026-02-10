package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type StopChatRequest struct {
	ConversationID string `json:"conversationId"`
}

type StopChatResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (s *Server) handleStopChat(c *gin.Context) {
	var req StopChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, StopChatResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if req.ConversationID == "" {
		c.JSON(http.StatusBadRequest, StopChatResponse{
			Success: false,
			Message: "conversationId is required",
		})
		return
	}

	// Try to cancel the conversation
	cancelled := s.convRegistry.Cancel(req.ConversationID)

	if !cancelled {
		// Conversation not found - might have already completed
		c.JSON(http.StatusOK, StopChatResponse{
			Success: false,
			Message: "Conversation not found or already completed",
		})
		return
	}

	c.JSON(http.StatusOK, StopChatResponse{
		Success: true,
		Message: "Conversation stopped",
	})
}
