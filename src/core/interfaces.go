package core

// SubAgent represents an agent that can be used as a sub-agent
// for delegation. This interface defines the minimal contract
// that any agent must satisfy to participate in delegation.
type SubAgent interface {
	// ChatStream initiates a streaming chat interaction with the agent
	// Returns a ResponseCh that can be used to consume streaming responses
	ChatStream(message string) *ResponseCh

	// Name returns the unique identifier of the agent
	Name() string

	// BasicDescription returns a short one-line description of the agent
	BasicDescription() string

	// AdvanceDescription returns detailed information about the agent's
	// capabilities, tools, sub-agents, and usage patterns
	AdvanceDescription() string

	// Troubleshooting returns information about common issues, debugging tips,
	// and configuration guidance for this agent
	Troubleshooting() string
}
