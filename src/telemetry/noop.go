package telemetry

import (
	"context"
	"time"
)

// NoopTracer is a no-op implementation of Tracer.
// It ignores all events and is used as the default when no observability is configured.
type NoopTracer struct{}

// Ensure NoopTracer implements Tracer at compile time.
var _ Tracer = (*NoopTracer)(nil)

// NewNoopTracer returns a no-op tracer that discards all events.
func NewNoopTracer() *NoopTracer {
	return &NoopTracer{}
}

// TraceToolExecution is a no-op.
func (n *NoopTracer) TraceToolExecution(ctx context.Context, event ToolExecutionEvent) {}

// TraceTokenUsage is a no-op.
func (n *NoopTracer) TraceTokenUsage(ctx context.Context, event TokenUsageEvent) {}

// TraceHistoryTruncation is a no-op.
func (n *NoopTracer) TraceHistoryTruncation(ctx context.Context, event TruncationEvent) {}

// TraceAgentStart is a no-op.
func (n *NoopTracer) TraceAgentStart(ctx context.Context, agentName string) {}

// TraceAgentComplete is a no-op.
func (n *NoopTracer) TraceAgentComplete(ctx context.Context, agentName string, duration time.Duration, err error) {
}
