package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricsTracer implements Tracer by emitting OpenTelemetry metrics.
// Pass a metric.Meter from your MeterProvider; if nil, all methods no-op.
type MetricsTracer struct {
	meter metric.Meter

	// Cached instruments (lazily created)
	toolExecutions  metric.Int64Counter
	toolDuration    metric.Float64Histogram
	tokenUsage      metric.Int64Counter
	truncations     metric.Int64Counter
	agentExecutions metric.Int64Counter
	agentDuration   metric.Float64Histogram
}

// Ensure MetricsTracer implements Tracer at compile time.
var _ Tracer = (*MetricsTracer)(nil)

// NewMetricsTracer returns a tracer that emits OpenTelemetry metrics.
// If meter is nil, returns a no-op tracer that ignores all events.
func NewMetricsTracer(meter metric.Meter) *MetricsTracer {
	if meter == nil {
		return &MetricsTracer{}
	}
	t := &MetricsTracer{meter: meter}
	t.initInstruments()
	return t
}

func (m *MetricsTracer) initInstruments() {
	if m.meter == nil {
		return
	}
	var err error
	m.toolExecutions, err = m.meter.Int64Counter(
		"agentforge.tool_executions",
		metric.WithDescription("Number of tool executions"),
	)
	if err != nil {
		return
	}
	m.toolDuration, err = m.meter.Float64Histogram(
		"agentforge.tool_execution_duration_seconds",
		metric.WithDescription("Tool execution duration in seconds"),
	)
	if err != nil {
		return
	}
	m.tokenUsage, err = m.meter.Int64Counter(
		"agentforge.token_usage_total",
		metric.WithDescription("Total tokens consumed"),
	)
	if err != nil {
		return
	}
	m.truncations, err = m.meter.Int64Counter(
		"agentforge.history_truncations_total",
		metric.WithDescription("Number of history truncation events"),
	)
	if err != nil {
		return
	}
	m.agentExecutions, err = m.meter.Int64Counter(
		"agentforge.agent_executions_total",
		metric.WithDescription("Number of agent execution completions"),
	)
	if err != nil {
		return
	}
	m.agentDuration, err = m.meter.Float64Histogram(
		"agentforge.agent_execution_duration_seconds",
		metric.WithDescription("Agent execution duration in seconds"),
	)
	if err != nil {
		return
	}
}

// TraceToolExecution records tool execution metrics.
func (m *MetricsTracer) TraceToolExecution(ctx context.Context, event ToolExecutionEvent) {
	if m.toolExecutions == nil {
		return
	}
	attrs := attribute.NewSet(
		attribute.String("agent", event.AgentName),
		attribute.String("tool", event.ToolName),
		attribute.Bool("success", event.Success),
	)
	m.toolExecutions.Add(ctx, 1, metric.WithAttributeSet(attrs))
	if m.toolDuration != nil {
		m.toolDuration.Record(ctx, event.Duration.Seconds(), metric.WithAttributeSet(attrs))
	}
}

// TraceTokenUsage records token consumption.
func (m *MetricsTracer) TraceTokenUsage(ctx context.Context, event TokenUsageEvent) {
	if m.tokenUsage == nil {
		return
	}
	attrs := attribute.NewSet(
		attribute.String("agent", event.AgentName),
		attribute.Bool("tool_calls", event.HadToolCalls),
	)
	m.tokenUsage.Add(ctx, int64(event.TotalTokens), metric.WithAttributeSet(attrs))
}

// TraceHistoryTruncation records truncation events.
func (m *MetricsTracer) TraceHistoryTruncation(ctx context.Context, event TruncationEvent) {
	if m.truncations == nil {
		return
	}
	attrs := attribute.NewSet(
		attribute.String("agent", event.AgentName),
	)
	m.truncations.Add(ctx, 1, metric.WithAttributeSet(attrs))
}

// TraceAgentStart is a no-op for metrics (we record on complete).
func (m *MetricsTracer) TraceAgentStart(ctx context.Context, agentName string) {}

// TraceAgentComplete records agent execution metrics.
func (m *MetricsTracer) TraceAgentComplete(ctx context.Context, agentName string, duration time.Duration, err error) {
	if m.agentExecutions == nil {
		return
	}
	attrs := attribute.NewSet(
		attribute.String("agent", agentName),
		attribute.Bool("error", err != nil),
	)
	m.agentExecutions.Add(ctx, 1, metric.WithAttributeSet(attrs))
	if m.agentDuration != nil {
		m.agentDuration.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
	}
}
