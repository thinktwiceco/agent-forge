package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// handlePush is a persistent SSE endpoint that streams background-drain chunks to the client.
// The client subscribes once per conversation and receives agent replies that arrive
// via the inbox queue (e.g. sub-agent responses delivered asynchronously).
//
// GET /api/chat/push?conversationId=<id>
func (s *Server) handlePush(c *gin.Context) {
	chatId := c.Query("conversationId")
	if chatId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversationId required"})
		return
	}

	reg := s.pushRegistry.Register(chatId)
	defer s.pushRegistry.Unregister(reg)
	ch := reg.Channel()

	writer := NewSSEWriter(c)
	writer.SetHeaders()

	ctx := c.Request.Context()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case chunk, ok := <-ch:
			if !ok {
				return
			}
			eventType := EventTypeFromChunk(chunk)
			if err := writer.WriteEvent(eventType, chunk); err != nil {
				return
			}
		case <-ticker.C:
			// Send a keep-alive comment so the connection stays open through proxies.
			_, _ = fmt.Fprintf(c.Writer, ": ping\n\n")
			if flusher, ok := c.Writer.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}
