package telemetry

import (
	"context"
	"testing"
	"time"
)

func TestNoopTracer_ImplementsInterface(t *testing.T) {
	var _ Tracer = (*NoopTracer)(nil)
}

func TestNoopTracer_DoesNotPanic(t *testing.T) {
	tracer := NewNoopTracer()
	ctx := context.Background()

	tracer.TraceToolExecution(ctx, ToolExecutionEvent{
		AgentName: "test",
		ToolName:  "example",
		Duration:  100 * time.Millisecond,
		Success:   true,
	})
	tracer.TraceTokenUsage(ctx, TokenUsageEvent{
		AgentName:   "test",
		TotalTokens: 100,
	})
	tracer.TraceHistoryTruncation(ctx, TruncationEvent{
		AgentName:      "test",
		MessagesBefore: 10,
		MessagesAfter:  5,
		TokensRemoved:  100,
	})
	tracer.TraceAgentStart(ctx, "test")
	tracer.TraceAgentComplete(ctx, "test", 5*time.Second, nil)
}
