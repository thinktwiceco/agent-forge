package agents

import (
	"fmt"

	"github.com/thinktwice/agentForge/src/llms"
)

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
	// tools, sub-agents, and usage patterns.
	AdvanceDescription string

	// Troubleshooting provides common issues, debugging tips, and configuration guidance.
	Troubleshooting string

	// Trace is optional trace information (e.g., "thinking", "response").
	Trace string

	// Reasoning indicates whether reasoning mode is enabled.
	// This parameter is reserved for future use.
	Reasoning bool

	// SystemPrompt is the system prompt to use for the agent.
	SystemPrompt string

	// Tools is the list of tools available to the agent.
	// Can be nil or empty if no tools are needed.
	Tools []llms.Tool

	// MaxToolIterations is the maximum number of tool execution iterations
	// to prevent infinite loops. Defaults to 10 if not set.
	MaxToolIterations int

	// ToolExecutionContext is custom context passed to tool execution.
	// Can be nil if no custom context is needed.
	ToolExecutionContext map[string]any

	// MainAgent indicates whether the agent is the main agent.
	// This parameter is reserved for future use.
	MainAgent bool

	// ExtraEngines allows specifying different LLM engines for team agents.
	// Key is the agent name (e.g., "system-reasoning"), value is the LLM engine to use.
	// If nil or if a team agent's name is not found, the default LLMEngine is used.
	ExtraEngines map[string]llms.LLMEngine

	// Persistence specifies the persistence layer type for conversation history.
	// Supported values: "" (none), "json"
	// If empty or not set, no persistence is used.
	Persistence string
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
	return nil
}
