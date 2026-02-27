package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

type SSEWriter struct {
	ctx *gin.Context
}

func NewSSEWriter(c *gin.Context) *SSEWriter {
	return &SSEWriter{ctx: c}
}

func (w *SSEWriter) SetHeaders() {
	w.ctx.Header("Content-Type", "text/event-stream")
	w.ctx.Header("Cache-Control", "no-cache")
	w.ctx.Header("Connection", "keep-alive")
	w.ctx.Header("X-Accel-Buffering", "no")
	w.ctx.Status(http.StatusOK)
}

func (w *SSEWriter) WriteEvent(eventType string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w.ctx.Writer, "event: %s\ndata: %s\n\n", eventType, payload); err != nil {
		return err
	}

	if flusher, ok := w.ctx.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func EventTypeFromChunk(chunk core.ExtendedChunkResponse) string {
	switch chunk.Status {
	case llms.StatusToolCall:
		return SSEEventToolCall
	case llms.StatusToolExecuting:
		return SSEEventToolExecuting
	case llms.StatusToolResult:
		return SSEEventToolResult
	case llms.StatusCompleted:
		return SSEEventCompleted
	case llms.StatusError:
		return SSEEventError
	default:
		if chunk.Type == llms.TypeContent {
			return SSEEventContent
		}
		return SSEEventContent
	}
}
