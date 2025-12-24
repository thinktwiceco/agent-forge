package agents

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
type extendedChunkResponse struct {
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

// ResponseCh manages channels for streaming responses and errors at the Agent level.
//
// This struct provides a channel-based API for receiving streaming responses
// from the agent. The Start() method returns a channel that can be ranged over.
type responseCh struct {
	Response chan []byte // Channel for JSON-serialized ChunkResponse
	Error    chan error  // Channel for errors

	agentName string // Name of the agent associated with this response channel
	trace     string // Trace information for this response channel

	started bool
	closed  bool
	mu      sync.Mutex
}

// newResponseCh creates a new ResponseCh instance.
//
// Parameters:
//   - agentName: Name of the agent associated with this response channel
//   - trace: Optional trace information (e.g., "thinking", "response")
//
// Returns:
//   - *ResponseCh: A new ResponseCh instance
func newResponseCh(agentName string, trace string) *responseCh {
	return &responseCh{
		Response:  make(chan []byte, 10), // Buffered channel
		Error:     make(chan error, 1),   // Buffered channel for errors
		agentName: agentName,
		trace:     trace,
		started:   false,
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
//	for chunk := range agentResponseCh.Start() {
//	    // Process chunk
//	}
//
// Returns:
//   - <-chan ExtendedChunkResponse: A receive-only channel of ExtendedChunkResponse that can be ranged over
func (arc *responseCh) Start() <-chan extendedChunkResponse {
	chunkChan := make(chan extendedChunkResponse)

	go func() {
		defer close(chunkChan)

		for {
			select {
			case chunkBytes, ok := <-arc.Response:
				if !ok {
					// Response channel closed, streaming complete
					return
				}

				// Try to deserialize as extendedChunkResponse first (may have AgentName/Trace already)
				var extendedChunk extendedChunkResponse
				if err := json.Unmarshal(chunkBytes, &extendedChunk); err != nil {
					// Send error as extended chunk
					chunkChan <- extendedChunkResponse{
						Status:    llms.StatusError,
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

				// Send chunk
				chunkChan <- extendedChunk

			case err := <-arc.Error:
				if err != nil {
					// Send error as extended chunk
					chunkChan <- extendedChunkResponse{
						Content:   err.Error(),
						Status:    llms.StatusError,
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
func (arc *responseCh) Close() {
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
func (arc *responseCh) GetResponseChan() chan<- []byte {
	return arc.Response
}

// GetErrorChan returns the error channel for sending errors.
// This method is used by tools to report errors during execution.
func (arc *responseCh) GetErrorChan() chan<- error {
	return arc.Error
}

// responseChAdapter wraps responseCh to provide a generic Start() method
// that returns interface{} instead of the concrete channel type.
// This allows responseCh to be used polymorphically in tools.
type responseChAdapter struct {
	*responseCh
}

// Start returns the channel as interface{} for polymorphic use
func (rca *responseChAdapter) Start() interface{} {
	return rca.responseCh.Start()
}

// AsGeneric wraps this responseCh in an adapter that provides generic Start() method
func (arc *responseCh) AsGeneric() *responseChAdapter {
	return &responseChAdapter{responseCh: arc}
}
