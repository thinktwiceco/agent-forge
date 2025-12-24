package llms

// ChunkResponse Status Constants
//
// These constants define the possible status values for ChunkResponse.Status field.
// The Status field indicates the current state of the streaming response.
const (
	// StatusStreaming indicates that content is actively being streamed from the LLM.
	// This status is used during the content generation phase when the LLM is producing text.
	//
	// When to expect:
	//   - During active content generation
	//   - Multiple chunks with this status may be received sequentially
	//   - Each chunk contains incremental content in the Delta field
	//
	// Associated fields:
	//   - Content: Current chunk's content
	//   - Delta: Same as Content (incremental text)
	//   - FullContent: Accumulated content so far
	//   - Type: Usually "content"
	StatusStreaming = "streaming"

	// StatusCompleted indicates that the streaming response has finished successfully.
	// This is the final status sent when the LLM has completed generating its response
	// and no further chunks will be sent (unless tool calls require another iteration).
	//
	// When to expect:
	//   - After all content has been streamed
	//   - As the last chunk in a successful response
	//   - Before tool execution begins (if tools are called)
	//
	// Associated fields:
	//   - FullContent: Complete accumulated content
	//   - Type: Usually "completion"
	StatusCompleted = "completed"

	// StatusError indicates that an error occurred during the streaming process.
	// This status is used to signal failures in deserialization, network issues,
	// or other errors that prevent normal response processing.
	//
	// When to expect:
	//   - When chunk deserialization fails
	//   - When the LLM stream encounters an error
	//   - When network or API errors occur
	//
	// Associated fields:
	//   - Content: Error message describing what went wrong
	StatusError = "error"

	// StatusToolCall indicates that the LLM is requesting one or more tool executions.
	// This status signals that the model wants to use external tools to gather information
	// or perform actions before continuing its response.
	//
	// When to expect:
	//   - When the LLM determines it needs external tool capabilities
	//   - After content streaming has completed (if any)
	//   - Before StatusToolExecuting chunks
	//
	// Associated fields:
	//   - ToolCalls: Array of tool calls the LLM wants to execute
	//   - FullContent: Any content generated before the tool calls
	//   - Type: Usually "tool-call"
	//
	// Flow:
	//   1. LLM generates StatusToolCall with tool requests
	//   2. Agent executes each tool (StatusToolExecuting)
	//   3. Agent returns results (StatusToolResult)
	//   4. LLM generates new response with tool context
	StatusToolCall = "tool-call"

	// StatusToolExecuting indicates that a specific tool is currently being executed.
	// This status is emitted by the agent to provide real-time feedback about
	// which tool is running, allowing consumers to track execution progress.
	//
	// When to expect:
	//   - After receiving StatusToolCall
	//   - Before StatusToolResult for the same tool
	//   - One chunk per tool being executed
	//
	// Associated fields:
	//   - ToolExecuting: Pointer to the ToolCall currently being executed
	//   - Type: Usually "tool-executing"
	//
	// Use cases:
	//   - Showing progress indicators in UIs
	//   - Logging tool execution for debugging
	//   - Tracking which tools are being used
	StatusToolExecuting = "tool-executing"

	// StatusToolResult indicates that tool execution has completed and results are available.
	// This status is emitted after each tool finishes executing, containing the
	// success/failure status and the tool's output or error message.
	//
	// When to expect:
	//   - After StatusToolExecuting for the same tool
	//   - Before the next LLM response iteration
	//   - One chunk per completed tool execution
	//
	// Associated fields:
	//   - ToolResults: Array of ToolResult objects with execution outcomes
	//   - Type: Usually "tool-result"
	//
	// ToolResult fields:
	//   - Success: Whether the tool executed successfully
	//   - Result: Tool output data (if successful)
	//   - Error: Error message (if failed)
	StatusToolResult = "tool-result"
)

// ChunkResponse Type Constants
//
// These constants define the possible type values for ChunkResponse.Type field.
// The Type field categorizes the kind of response chunk being sent.
const (
	// TypeContent indicates a chunk containing regular content being streamed from the LLM.
	// This is the most common type during normal text generation.
	//
	// When to expect:
	//   - During active content streaming
	//   - With Status: StatusStreaming
	//
	// Associated data:
	//   - Content: The actual text content
	//   - Delta: Incremental text for this chunk
	//   - FullContent: All content accumulated so far
	TypeContent = "content"

	// TypeToolCall indicates a chunk containing tool call requests from the LLM.
	// The LLM has decided to use external tools and is providing the tool names
	// and arguments needed to execute them.
	//
	// When to expect:
	//   - When the LLM needs external tool capabilities
	//   - With Status: StatusToolCall
	//
	// Associated data:
	//   - ToolCalls: Array of requested tool executions
	//   - Each ToolCall includes: ID, Name, and Arguments
	TypeToolCall = "tool-call"

	// TypeToolExecuting indicates a chunk signaling that a tool is currently being executed.
	// This type is used for progress tracking and allows consumers to monitor
	// which tools are running in real-time.
	//
	// When to expect:
	//   - Between tool call request and tool result
	//   - With Status: StatusToolExecuting
	//
	// Associated data:
	//   - ToolExecuting: The specific ToolCall being executed right now
	TypeToolExecuting = "tool-executing"

	// TypeToolResult indicates a chunk containing the results of a tool execution.
	// This includes both successful results and error information if the tool failed.
	//
	// When to expect:
	//   - After a tool has finished executing
	//   - With Status: StatusToolResult
	//
	// Associated data:
	//   - ToolResults: Array of execution results
	//   - Each result includes: ToolCallID, ToolName, Success, Result, Error
	TypeToolResult = "tool-result"

	// TypeCompletion indicates the final chunk signaling that the response is complete.
	// This type marks the end of the streaming response when no more data will be sent.
	//
	// When to expect:
	//   - As the last chunk in a response
	//   - With Status: StatusCompleted
	//
	// Associated data:
	//   - FullContent: The complete accumulated response
	TypeCompletion = "completion"
)

// Status and Type Relationship
//
// The Status and Type fields work together to provide complete context about a chunk:
//
// Common Combinations:
//   - Status: StatusStreaming,  Type: TypeContent        → Regular content streaming
//   - Status: StatusCompleted,  Type: TypeCompletion     → Response finished
//   - Status: StatusToolCall,   Type: TypeToolCall       → LLM requesting tools
//   - Status: StatusToolExecuting, Type: TypeToolExecuting → Tool is running
//   - Status: StatusToolResult, Type: TypeToolResult     → Tool results available
//   - Status: StatusError,      Type: (any)              → Error occurred
//
// Typical Flow (without tools):
//   1. Status: StatusStreaming, Type: TypeContent (multiple chunks)
//   2. Status: StatusCompleted, Type: TypeCompletion (final chunk)
//
// Typical Flow (with tools):
//   1. Status: StatusStreaming, Type: TypeContent (optional initial content)
//   2. Status: StatusToolCall, Type: TypeToolCall (LLM requests tools)
//   3. Status: StatusToolExecuting, Type: TypeToolExecuting (for each tool)
//   4. Status: StatusToolResult, Type: TypeToolResult (for each tool)
//   5. Status: StatusStreaming, Type: TypeContent (LLM continues with tool context)
//   6. Status: StatusCompleted, Type: TypeCompletion (final chunk)
