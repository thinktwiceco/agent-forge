package llms

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// ToolCall represents a tool call request from the LLM.
type ToolCall struct {
	ID        string         `json:"id"`        // Tool call ID
	Name      string         `json:"name"`      // Tool name
	Arguments map[string]any `json:"arguments"` // Tool arguments
}

// ResolveToolNameForTools maps a model-emitted tool name to a registered tool name.
// Some providers (e.g. Groq) stream names in fragments; a stray trailing ">" can appear
// and must match the real tool (e.g. memory_read_short_term> -> memory_read_short_term).
func ResolveToolNameForTools(requested string, tools []Tool) string {
	if requested == "" || len(tools) == 0 {
		return requested
	}
	for _, t := range tools {
		if t.GetName() == requested {
			return requested
		}
	}
	trimmed := strings.TrimSuffix(requested, ">")
	if trimmed != requested {
		for _, t := range tools {
			if t.GetName() == trimmed {
				return trimmed
			}
		}
	}
	return requested
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	ToolCallID string `json:"toolCallId"` // ID of the tool call this result is for
	ToolName   string `json:"toolName"`   // Name of the tool that was executed
	Success    bool   `json:"success"`    // Whether the tool executed successfully
	Result     string `json:"result"`     // Result data from the tool
	Error      string `json:"error"`      // Error message if tool failed
	Ephemeral  bool   `json:"ephemeral"`  // Whether the result is ephemeral
	Cleanup    func() `json:"-"`          // Optional cleanup function (not serialized)
}

// ChunkResponse represents a streaming response chunk.
//
// This struct is serialized to JSON bytes and sent through channels
// during streaming responses.
type ChunkResponse struct {
	Content          string       `json:"content"`                    // Current chunk content
	Delta            string       `json:"delta"`                      // Incremental delta
	FullContent      string       `json:"fullContent"`                // Accumulated full content
	ReasoningContent string       `json:"reasoningContent,omitempty"` // Accumulated reasoning/thinking content (e.g. DeepSeek Reasoner)
	Status           string       `json:"status"`                     // Status: see Status* constants (StatusStreaming, StatusCompleted, etc.)
	Type             string       `json:"type"`                       // Response type: see Type* constants (TypeContent, TypeCompletion, etc.)
	ToolCalls        []ToolCall   `json:"toolCalls,omitempty"`        // Tool calls (when Type is "tool-call")
	ToolExecuting    *ToolCall    `json:"toolExecuting,omitempty"`    // Tool being executed (when Status is "tool-executing")
	ToolResults      []ToolResult `json:"toolResults,omitempty"`      // Tool execution results (when Status is "tool-result")
	PromptTokens     int          `json:"promptTokens,omitempty"`     // Input tokens consumed
	CompletionTokens int          `json:"completionTokens,omitempty"` // Output tokens generated
	TotalTokens      int          `json:"totalTokens,omitempty"`      // Total tokens used
	Iteration        int          `json:"iteration,omitempty"`        // Tool iteration number (for tracking response turns)
}

// ResponseCh manages channels for streaming responses and errors.
//
// This struct provides a channel-based API for receiving streaming responses
// from the agent. The Start() method returns a channel that can be ranged over.
type ResponseCh struct {
	Response chan []byte // Channel for JSON-serialized ChunkResponse
	Error    chan error  // Channel for errors

	started bool
	closed  bool
	mu      sync.Mutex
}

// NewResponseCh creates a new ResponseCh instance.
func NewResponseCh() *ResponseCh {
	return &ResponseCh{
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
func (rc *ResponseCh) Start() <-chan ChunkResponse {
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

			case err, ok := <-rc.Error:
				if ok && err != nil {
					chunkChan <- ChunkResponse{
						Status:  StatusError,
						Content: err.Error(),
					}
					return
				}
				if !ok {
					// Error channel closed; Response may still have chunks (Close() closes both).
					// Do not exit here or a racing select can skip the last Response payload.
					continue
				}
				// ok && err == nil — ignore and keep draining Response
			}
		}
	}()

	return chunkChan
}

// Close closes both channels.
//
// This should be called when done listening to clean up resources.
// Safe to call multiple times - will only close channels once.
func (rc *ResponseCh) Close() {
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
