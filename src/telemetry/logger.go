package telemetry

import (
	"context"
	"fmt"
	"time"

	agentforge "github.com/thinktwiceco/agent-forge/src"
)

// LogTracer implements Tracer by logging events at debug level.
// Useful for debugging agent behavior without external observability infrastructure.
type LogTracer struct {
	logger *agentforge.Logger
}

// Ensure LogTracer implements Tracer at compile time.
var _ Tracer = (*LogTracer)(nil)

// NewLogTracer returns a tracer that logs events using the provided logger.
// If logger is nil, uses the global agentforge logger.
func NewLogTracer(logger *agentforge.Logger) *LogTracer {
	if logger == nil {
		logger = agentforge.GetLogger()
	}
	return &LogTracer{logger: logger}
}

// TraceToolExecution logs tool execution metrics.
func (l *LogTracer) TraceToolExecution(ctx context.Context, event ToolExecutionEvent) {
	status := "success"
	if !event.Success {
		status = "failed"
	}
	msg := fmt.Sprintf("[telemetry] tool_execution agent=%s tool=%s duration=%v iteration=%d status=%s",
		event.AgentName, event.ToolName, event.Duration, event.Iteration, status)
	if event.Error != "" {
		msg += fmt.Sprintf(" error=%q", event.Error)
	}
	l.logger.Debug("%s", msg)
}

// TraceTokenUsage logs token consumption.
func (l *LogTracer) TraceTokenUsage(ctx context.Context, event TokenUsageEvent) {
	toolCalls := ""
	if event.HadToolCalls {
		toolCalls = " tool_calls=true"
	}
	l.logger.Debug("[telemetry] token_usage agent=%s prompt=%d completion=%d total=%d iteration=%d%s",
		event.AgentName, event.PromptTokens, event.CompletionTokens, event.TotalTokens, event.Iteration, toolCalls)
}

// TraceHistoryTruncation logs truncation events.
func (l *LogTracer) TraceHistoryTruncation(ctx context.Context, event TruncationEvent) {
	l.logger.Debug("[telemetry] history_truncation agent=%s messages_before=%d messages_after=%d tokens_removed=%d",
		event.AgentName, event.MessagesBefore, event.MessagesAfter, event.TokensRemoved)
}

// TraceAgentStart logs agent execution start.
func (l *LogTracer) TraceAgentStart(ctx context.Context, agentName string) {
	l.logger.Debug("[telemetry] agent_start agent=%s", agentName)
}

// TraceAgentComplete logs agent execution completion.
func (l *LogTracer) TraceAgentComplete(ctx context.Context, agentName string, duration time.Duration, err error) {
	errStr := ""
	if err != nil {
		errStr = fmt.Sprintf(" error=%q", err.Error())
	}
	l.logger.Debug("[telemetry] agent_complete agent=%s duration=%v%s", agentName, duration, errStr)
}
