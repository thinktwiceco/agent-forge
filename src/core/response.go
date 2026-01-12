package core

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/thinktwice/agentForge/src/llms"
)

// ExtendedChunkResponse extends ChunkResponse with agent-specific information.
//
// This struct includes all properties from ChunkResponse plus agentName and trace
// fields for enhanced context in multi-agent scenarios.
type ExtendedChunkResponse struct {
	Content          string            `json:"content"`                    // Current chunk content
	Delta            string            `json:"delta"`                      // Incremental delta
	FullContent      string            `json:"fullContent"`                // Accumulated full content
	Status           string            `json:"status"`                     // Status: see llms.Status* constants (StatusStreaming, StatusCompleted, etc.)
	Type             string            `json:"type"`                       // Response type: see llms.Type* constants (TypeContent, TypeCompletion, etc.)
	ToolCalls        []llms.ToolCall   `json:"toolCalls,omitempty"`        // Tool calls (when Type is "tool-call")
	ToolExecuting    *llms.ToolCall    `json:"toolExecuting,omitempty"`    // Tool being executed (when Status is "tool-executing")
	ToolResults      []llms.ToolResult `json:"toolResults,omitempty"`      // Tool execution results (when Status is "tool-result")
	PromptTokens     int               `json:"promptTokens,omitempty"`     // Input tokens consumed
	CompletionTokens int               `json:"completionTokens,omitempty"` // Output tokens generated
	TotalTokens      int               `json:"totalTokens,omitempty"`      // Total tokens used
	AgentName        string            `json:"agentName"`                  // Name of the agent producing this chunk
	Trace            string            `json:"trace"`                      // Trace information (e.g., "thinking", "response")
}

// ChunkReadHook is a callback function called when a chunk is read from the channel
// It receives the ExtendedChunkResponse and can be used to trigger hooks
type ChunkReadHook func(extendedChunk *ExtendedChunkResponse) error

// ResponseCh manages channels for streaming responses and errors at the Agent level.
//
// This struct provides a channel-based API for receiving streaming responses
// from the agent. The Start() method returns a channel that can be ranged over.
type ResponseCh struct {
	Response chan []byte // Channel for JSON-serialized ChunkResponse
	Error    chan error  // Channel for errors

	agentName string // Name of the agent associated with this response channel
	trace     string // Trace information for this response channel
	onChunkRead ChunkReadHook // Hook called when chunks are read from the channel

	started bool
	closed  bool
	mu      sync.Mutex
}

// NewResponseCh creates a new ResponseCh instance.
//
// Parameters:
//   - agentName: Name of the agent associated with this response channel
//   - trace: Optional trace information (e.g., "thinking", "response")
//   - onChunkRead: Optional hook function called when chunks are read from the channel
//
// Returns:
//   - *ResponseCh: A new ResponseCh instance
func NewResponseCh(agentName string, trace string, onChunkRead ChunkReadHook) *ResponseCh {
	return &ResponseCh{
		Response:    make(chan []byte, 10), // Buffered channel
		Error:       make(chan error, 1),   // Buffered channel for errors
		agentName:   agentName,
		trace:       trace,
		onChunkRead: onChunkRead,
		started:     false,
	}
}

// Start begins listening to the response and error channels and returns a channel
// of ExtendedChunkResponse that can be ranged over.
//
// This method reads from the internal Response and Error channels, deserializes
// JSON chunks, and sends ExtendedChunkResponse structs to the returned channel.
// Errors are converted to ExtendedChunkResponse with Status=llms.StatusError.
//
// Usage:
//
//	for chunk := range responseCh.Start() {
//	    // Process chunk
//	}
//
// Returns:
//   - <-chan ExtendedChunkResponse: A receive-only channel of ExtendedChunkResponse that can be ranged over
func (arc *ResponseCh) Start() <-chan ExtendedChunkResponse {
	chunkChan := make(chan ExtendedChunkResponse)

	go func() {
		defer close(chunkChan)

		for {
			select {
			case chunkBytes, ok := <-arc.Response:
				if !ok {
					// Response channel closed, streaming complete
					return
				}

				// Try to deserialize as ExtendedChunkResponse first (may have AgentName/Trace already)
				var extendedChunk ExtendedChunkResponse
				if err := json.Unmarshal(chunkBytes, &extendedChunk); err != nil {
					// Send error as extended chunk
					chunkChan <- ExtendedChunkResponse{
						Status:    llms.StatusError,
						Type:      llms.TypeContent, // Error chunks use TypeContent
						Content:   fmt.Sprintf("Error deserializing chunk: %v", err),
						AgentName: arc.agentName,
						Trace:     arc.trace,
					}
					continue
				}

				// Only set AgentName and Trace if they're not already set
				// (they might be set if this chunk came from a delegated agent)
				if extendedChunk.AgentName == "" {
					extendedChunk.AgentName = arc.agentName
				}
				if extendedChunk.Trace == "" {
					extendedChunk.Trace = arc.trace
				}

				// Trigger hook when chunk is read from channel
				if arc.onChunkRead != nil {
					if err := arc.onChunkRead(&extendedChunk); err != nil {
						// Log hook error but continue processing
						// Hook errors shouldn't stop chunk processing
					}
				}

				// Send chunk
				chunkChan <- extendedChunk

			case err := <-arc.Error:
				if err != nil {
					// Send error as extended chunk
					chunkChan <- ExtendedChunkResponse{
						Content:   err.Error(),
						Status:    llms.StatusError,
						Type:      llms.TypeContent, // Error chunks use TypeContent
						AgentName: arc.agentName,
						Trace:     arc.trace,
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
func (arc *ResponseCh) Close() {
	arc.mu.Lock()
	defer arc.mu.Unlock()

	if arc.closed {
		return
	}

	close(arc.Response)
	close(arc.Error)
	arc.closed = true
}

// GetResponseChan returns the response channel for sending chunks.
// This method is used by tools to send custom chunks during execution.
// This implements IParentResponseCh.
func (arc *ResponseCh) GetResponseChan() chan<- []byte {
	return arc.Response
}

// GetErrorChan returns the error channel for sending errors.
// This method is used by tools to report errors during execution.
// This implements IParentResponseCh.
func (arc *ResponseCh) GetErrorChan() chan<- error {
	return arc.Error
}

// TrySend safely sends a chunk to the response channel.
// Returns false if the channel is closed, true if the send succeeded.
// This prevents panics when trying to send to a closed channel.
func (arc *ResponseCh) TrySend(chunkBytes []byte) bool {
	arc.mu.Lock()
	closed := arc.closed
	arc.mu.Unlock()

	if closed {
		return false
	}

	// Use recover to catch panic if channel closes between check and send
	sent := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Panic occurred, channel was closed
				sent = false
			}
		}()
		arc.Response <- chunkBytes
		sent = true
	}()

	return sent
}
