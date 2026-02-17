package telemetry

import (
	"context"
	"time"
)

// Tracer provides structured observability for agent behavior.
//
// Implementations can log, emit metrics, or integrate with external
// systems (OpenTelemetry, Prometheus, etc.). Use NoopTracer as the
// default when no observability is needed.
type Tracer interface {
	// TraceToolExecution records a tool execution event.
	TraceToolExecution(ctx context.Context, event ToolExecutionEvent)

	// TraceTokenUsage records token consumption from an LLM response.
	TraceTokenUsage(ctx context.Context, event TokenUsageEvent)

	// TraceHistoryTruncation records when message history was truncated.
	TraceHistoryTruncation(ctx context.Context, event TruncationEvent)

	// TraceAgentStart records the start of an agent execution.
	TraceAgentStart(ctx context.Context, agentName string)

	// TraceAgentComplete records the completion of an agent execution.
	TraceAgentComplete(ctx context.Context, agentName string, duration time.Duration, err error)
}

// ToolExecutionEvent captures tool execution metrics.
type ToolExecutionEvent struct {
	AgentName string        // Name of the agent executing the tool
	ToolName  string        // Name of the tool executed
	Duration  time.Duration // Execution duration
	Success   bool          // Whether the tool succeeded
	Error     string        // Error message if failed
	Iteration int           // Tool call iteration within the chat loop
}

// TokenUsageEvent captures token consumption from an LLM response.
type TokenUsageEvent struct {
	AgentName        string // Name of the agent
	PromptTokens     int    // Input tokens
	CompletionTokens int    // Output tokens
	TotalTokens      int    // Total tokens
	Iteration        int    // Iteration within the chat loop
	HadToolCalls     bool   // Whether the response included tool calls
}

// TruncationEvent captures history truncation metrics.
type TruncationEvent struct {
	AgentName      string // Name of the agent
	MessagesBefore int    // Message count before truncation
	MessagesAfter  int    // Message count after truncation
	TokensBefore   int    // Estimated tokens before truncation
	TokensAfter    int    // Estimated tokens after truncation
	TokensRemoved  int    // Tokens removed by truncation
}
