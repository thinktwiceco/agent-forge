package llms

import (
	"encoding/json"
	"fmt"
	"sync"
)

// ToolCall represents a tool call request from the LLM.
type ToolCall struct {
	ID        string         `json:"id"`        // Tool call ID
	Name      string         `json:"name"`      // Tool name
	Arguments map[string]any `json:"arguments"` // Tool arguments
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	ToolCallID string `json:"toolCallId"` // ID of the tool call this result is for
	ToolName   string `json:"toolName"`   // Name of the tool that was executed
	Success    bool   `json:"success"`    // Whether the tool executed successfully
	Result     string `json:"result"`     // Result data from the tool
	Error      string `json:"error"`      // Error message if tool failed
}

// ChunkResponse represents a streaming response chunk.
//
// This struct is serialized to JSON bytes and sent through channels
// during streaming responses.
type ChunkResponse struct {
	Content          string       `json:"content"`                    // Current chunk content
	Delta            string       `json:"delta"`                      // Incremental delta
	FullContent      string       `json:"fullContent"`                // Accumulated full content
	Status           string       `json:"status"`                     // Status: see Status* constants (StatusStreaming, StatusCompleted, etc.)
	Type             string       `json:"type"`                       // Response type: see Type* constants (TypeContent, TypeCompletion, etc.)
	ToolCalls        []ToolCall   `json:"toolCalls,omitempty"`        // Tool calls (when Type is "tool-call")
	ToolExecuting    *ToolCall    `json:"toolExecuting,omitempty"`    // Tool being executed (when Status is "tool-executing")
	ToolResults      []ToolResult `json:"toolResults,omitempty"`      // Tool execution results (when Status is "tool-result")
	PromptTokens     int          `json:"promptTokens,omitempty"`     // Input tokens consumed
	CompletionTokens int          `json:"completionTokens,omitempty"` // Output tokens generated
	TotalTokens      int          `json:"totalTokens,omitempty"`      // Total tokens used
}

// ResponseCh manages channels for streaming responses and errors.
//
// This struct provides a channel-based API for receiving streaming responses
// from the agent. The Start() method returns a channel that can be ranged over.
type responseCh struct {
	Response chan []byte // Channel for JSON-serialized ChunkResponse
	Error    chan error  // Channel for errors

	started bool
	closed  bool
	mu      sync.Mutex
}

// NewResponseCh creates a new ResponseCh instance.
func newResponseCh() *responseCh {
	return &responseCh{
		Response: make(chan []byte, 10), // Buffered channel
		Error:    make(chan error, 1),   // Buffered channel for errors
		started:  false,
	}
}

// Start begins listening to the response and error channels and returns a channel
// of ChunkResponse that can be ranged over.
//
// This method reads from the internal Response and Error channels, deserializes
// JSON chunks, and sends ChunkResponse structs to the returned channel.
// Errors are converted to ChunkResponse with Status="error".
//
// Usage:
//
//	for chunk := range responseCh.Start() {
//	    // Process chunk
//	}
//
// Returns:
//   - <-chan ChunkResponse: A receive-only channel of ChunkResponse that can be ranged over
func (rc *responseCh) Start() <-chan ChunkResponse {
	chunkChan := make(chan ChunkResponse)

	go func() {
		defer close(chunkChan)

		for {
			select {
			case chunkBytes, ok := <-rc.Response:
				if !ok {
					// Response channel closed, streaming complete
					return
				}

				// Deserialize chunk
				var chunk ChunkResponse
				if err := json.Unmarshal(chunkBytes, &chunk); err != nil {
					// Send error as chunk
					chunkChan <- ChunkResponse{
						Status:  StatusError,
						Content: fmt.Sprintf("Error deserializing chunk: %v", err),
					}
					continue
				}

				// Send chunk
				chunkChan <- chunk

				// If completed, we're done
				if chunk.Status == StatusCompleted {
					return
				}

			case err := <-rc.Error:
				if err != nil {
					// Send error as chunk
					chunkChan <- ChunkResponse{
						Status:  StatusError,
						Content: err.Error(),
					}
				}
				return
			}
		}
	}()

	return chunkChan
}

// Close closes both channels.
//
// This should be called when done listening to clean up resources.
// Safe to call multiple times - will only close channels once.
func (rc *responseCh) Close() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.closed {
		return
	}

	close(rc.Response)
	close(rc.Error)
	rc.closed = true
}

// serializeChunk serializes a ChunkResponse to JSON bytes.
func serializeChunk(chunk ChunkResponse) ([]byte, error) {
	return json.Marshal(chunk)
}
