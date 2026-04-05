package agents

import (
	"fmt"

	agentctx "github.com/thinktwiceco/agent-forge/src/agents/context"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/telemetry"
)

const defaultMaxToolIterations = 30

// AgentConfig holds configuration parameters for creating a new Agent.
//
// This struct encapsulates all parameters needed to create an Agent instance,
// making it easier to add new parameters without changing function signatures.
type AgentConfig struct {
	// LLMEngine is the underlying LLM engine that handles streaming responses.
	// It implements the llms.LLMEngine interface.
	LLMEngine llms.LLMEngine

	// AgentName is the name of the agent (e.g., "reasoning", "test agent").
	AgentName string

	// Description is the description of the agent.
	Description string

	// AdvanceDescription is detailed information about the agent's capabilities,
	// tools, and usage patterns.
	AdvanceDescription string

	// Troubleshooting provides common issues, debugging tips, and configuration guidance.
	Troubleshooting string

	// Trace is optional trace information (e.g., "thinking", "response").
	Trace string

	// Can Expand indicates whether the agent can exapnd description of tools
	// or other agents. Each agent and tool has some more detailed descriptions
	// that are not included in the basic description.
	// This parameter will default to true
	CanExpand bool

	// CanSpawnSubagent enables the spawn_subagent built-in tool.
	// When true, the agent can delegate tasks to ephemeral subagents that share
	// the same LLM engine and a subset of the parent's tools.
	// Defaults to false.
	CanSpawnSubagent bool

	// SystemPrompt is the system prompt to use for the agent.
	SystemPrompt string

	// Tools is the list of tools available to the agent.
	// Can be nil or empty if no tools are needed.
	Tools []llms.Tool

	// MaxToolIterations is the maximum number of tool execution iterations
	// to prevent infinite loops. Defaults to 10 if not set.
	MaxToolIterations int

	// MainAgent indicates whether the agent is the main agent.
	// This parameter is reserved for future use.
	MainAgent bool

	// Tone specifies the response tone for the main agent.
	// Supported values: ToneDefault (""), ToneKeepItShort, ToneSystemAgent (non-main agents only)
	// Use constants from the agents package (e.g., agents.ToneKeepItShort)
	Tone string

	// Persistence specifies the persistence layer type for conversation history.
	// Supported values: "" (none), "json"
	// If empty or not set, no persistence is used.
	Persistence string

	// WorkingDir is the agent's working directory. Tool and plugin paths are relative to it.
	// When set, conversation persistence (json) stores history in WorkingDir/data/conversations/{agentName}.
	// When empty, persistence uses data/conversations/{agentName} relative to process CWD.
	WorkingDir string

	// DetailsAboutFunc optionally provides per-item discovery. If nil, DetailsAbout returns "Nothing to add about <item>".
	DetailsAboutFunc func(item string) string

	// Plugins is the list of plugins to use for the agent
	Plugins []core.Plugin

	// MaxContextTokens is the model's context window size in tokens.
	// Used for intelligent history truncation. Set to 0 to disable truncation.
	// Example: 128000 for models like Kimi-K2.5
	MaxContextTokens int

	// ReservedOutputTokens reserves token space for LLM completion.
	// Deducted from MaxContextTokens when truncating history.
	// Defaults to 4000 if MaxContextTokens > 0
	ReservedOutputTokens int

	// MinRecentMessages ensures the last N messages are always kept.
	// Prevents over-aggressive truncation of recent conversation context.
	// Defaults to 10 if MaxContextTokens > 0
	MinRecentMessages int

	// EnableSummarization generates summaries of truncated messages
	// instead of discarding them entirely. Requires MaxContextTokens > 0.
	// Defaults to false
	EnableSummarization bool

	// TruncationStrategy determines how history is truncated when it exceeds MaxContextTokens.
	// If nil and MaxContextTokens > 0, defaults to SlidingWindowStrategy.
	// Set to NoTruncationStrategy to disable truncation.
	TruncationStrategy agentctx.TruncationStrategy

	// Tracer provides structured observability (tool execution, token usage, truncation).
	// If nil, defaults to telemetry.NoopTracer.
	Tracer telemetry.Tracer

	// HeartbeatAckMaxChars matches heartbeat plugin ack_max_chars when the heartbeat plugin is enabled (0 = default 300).
	// Used by the executor to detect HEARTBEAT_OK ack-only replies.
	HeartbeatAckMaxChars int
}

// validate validates that all required fields in AgentConfig are set.
//
// Required fields:
//   - LLMEngine: Must not be nil
//   - AgentName: Must not be empty
//
// Returns:
//   - error: An error describing which required field is missing, or nil if validation passes
func (c *AgentConfig) validate() error {
	if c.LLMEngine == nil {
		return fmt.Errorf("LLMEngine is required but was nil")
	}
	if c.AgentName == "" {
		return fmt.Errorf("AgentName is required but was empty")
	}

	if c.Trace == "" {
		c.Trace = fmt.Sprintf("%s-trace", c.AgentName)
	}

	if c.Tools == nil {
		c.Tools = []llms.Tool{}
	}

	if c.MaxToolIterations <= 0 {
		c.MaxToolIterations = defaultMaxToolIterations
	}

	// Set defaults for context window management
	if c.MaxContextTokens > 0 {
		if c.ReservedOutputTokens <= 0 {
			c.ReservedOutputTokens = 4000
		}
		if c.MinRecentMessages <= 0 {
			c.MinRecentMessages = 10
		}
	}

	// Default to no-op tracer if not set
	if c.Tracer == nil {
		c.Tracer = telemetry.NewNoopTracer()
	}

	return nil
}
