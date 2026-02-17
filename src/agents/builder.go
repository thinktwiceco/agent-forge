package agents

import (
	"fmt"

	agentctx "github.com/thinktwiceco/agent-forge/src/agents/context"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/telemetry"
)

// Builder provides a fluent API for constructing Agent instances.
//
// Use NewBuilder to create a builder, chain optional configuration methods,
// then call Build() to create the Agent. This pattern is preferred over
// manual AgentConfig construction for programmatic agent creation.
//
// Example:
//
//	agent, err := agents.NewBuilder(llmEngine, "assistant").
//	    WithDescription("Helpful assistant").
//	    WithSystemPrompt("You are helpful").
//	    WithTools(tool1, tool2).
//	    WithContextWindow(128000).
//	    Build()
//
// For advanced use cases with full control, continue using NewAgent(&AgentConfig{...}).
type Builder struct {
	config *AgentConfig
}

// NewBuilder creates a new Builder with required LLM engine and agent name.
//
// Parameters:
//   - llm: The LLM engine for the agent (required)
//   - name: The agent name (required)
//
// Returns a Builder with sensible defaults: CanExpand=true, MaxToolIterations=30.
func NewBuilder(llm llms.LLMEngine, name string) *Builder {
	return &Builder{
		config: &AgentConfig{
			LLMEngine:         llm,
			AgentName:         name,
			CanExpand:         true,
			MaxToolIterations: defaultMaxToolIterations,
		},
	}
}

// WithDescription sets the agent description.
func (b *Builder) WithDescription(desc string) *Builder {
	b.config.Description = desc
	return b
}

// WithAdvanceDescription sets detailed capability information.
func (b *Builder) WithAdvanceDescription(desc string) *Builder {
	b.config.AdvanceDescription = desc
	return b
}

// WithTroubleshooting sets troubleshooting guidance.
func (b *Builder) WithTroubleshooting(troubleshooting string) *Builder {
	b.config.Troubleshooting = troubleshooting
	return b
}

// WithSystemPrompt sets the system prompt.
func (b *Builder) WithSystemPrompt(prompt string) *Builder {
	b.config.SystemPrompt = prompt
	return b
}

// WithTrace sets the trace identifier.
func (b *Builder) WithTrace(trace string) *Builder {
	b.config.Trace = trace
	return b
}

// WithReasoning enables reasoning mode.
func (b *Builder) WithReasoning() *Builder {
	b.config.Reasoning = true
	return b
}

// WithCanExpand enables or disables expand capability for tools/subagents.
func (b *Builder) WithCanExpand(canExpand bool) *Builder {
	b.config.CanExpand = canExpand
	return b
}

// WithTools adds tools to the agent. Can be called multiple times.
func (b *Builder) WithTools(tools ...llms.Tool) *Builder {
	if b.config.Tools == nil {
		b.config.Tools = []llms.Tool{}
	}
	b.config.Tools = append(b.config.Tools, tools...)
	return b
}

// WithMaxToolIterations sets the maximum tool execution iterations.
func (b *Builder) WithMaxToolIterations(max int) *Builder {
	b.config.MaxToolIterations = max
	return b
}

// AsMainAgent marks the agent as the main agent.
func (b *Builder) AsMainAgent() *Builder {
	b.config.MainAgent = true
	return b
}

// WithTone sets the response tone (e.g., ToneKeepItShort, ToneSystemAgent).
func (b *Builder) WithTone(tone string) *Builder {
	b.config.Tone = tone
	return b
}

// WithPersistence sets the persistence layer ("", "json").
func (b *Builder) WithPersistence(persistence string) *Builder {
	b.config.Persistence = persistence
	return b
}

// WithSubAgents adds sub-agents for delegation.
func (b *Builder) WithSubAgents(subAgents ...core.SubAgent) *Builder {
	if b.config.SubAgents == nil {
		b.config.SubAgents = []core.SubAgent{}
	}
	b.config.SubAgents = append(b.config.SubAgents, subAgents...)
	return b
}

// WithPlugins adds plugins to the agent.
func (b *Builder) WithPlugins(plugins ...core.Plugin) *Builder {
	if b.config.Plugins == nil {
		b.config.Plugins = []core.Plugin{}
	}
	b.config.Plugins = append(b.config.Plugins, plugins...)
	return b
}

// WithContextWindow sets the context window size and reasonable defaults
// for ReservedOutputTokens (4000) and MinRecentMessages (10).
func (b *Builder) WithContextWindow(maxTokens int) *Builder {
	b.config.MaxContextTokens = maxTokens
	if maxTokens > 0 {
		b.config.ReservedOutputTokens = 4000
		b.config.MinRecentMessages = 10
	}
	return b
}

// WithReservedOutputTokens sets token reserve for LLM completion.
func (b *Builder) WithReservedOutputTokens(tokens int) *Builder {
	b.config.ReservedOutputTokens = tokens
	return b
}

// WithMinRecentMessages sets the minimum recent messages to keep during truncation.
func (b *Builder) WithMinRecentMessages(n int) *Builder {
	b.config.MinRecentMessages = n
	return b
}

// WithEnableSummarization enables summarization of truncated messages.
func (b *Builder) WithEnableSummarization(enabled bool) *Builder {
	b.config.EnableSummarization = enabled
	return b
}

// WithTruncationStrategy sets a custom truncation strategy.
func (b *Builder) WithTruncationStrategy(strategy agentctx.TruncationStrategy) *Builder {
	b.config.TruncationStrategy = strategy
	return b
}

// WithTracer sets the telemetry tracer for observability.
func (b *Builder) WithTracer(tracer telemetry.Tracer) *Builder {
	b.config.Tracer = tracer
	return b
}

// Build creates the Agent from the built configuration.
//
// Validates required fields (LLMEngine, AgentName) before construction.
// Returns an error if validation fails; otherwise returns the Agent.
func (b *Builder) Build() (*Agent, error) {
	if err := b.config.validate(); err != nil {
		return nil, fmt.Errorf("agent builder validation: %w", err)
	}
	return NewAgent(b.config), nil
}

// BuildConfig returns the built AgentConfig without creating an Agent.
// Useful when you need to customize the config further or pass it elsewhere.
func (b *Builder) BuildConfig() (*AgentConfig, error) {
	if err := b.config.validate(); err != nil {
		return nil, fmt.Errorf("agent builder validation: %w", err)
	}
	return b.config, nil
}
