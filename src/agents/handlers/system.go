package handlers

import (
	"errors"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// AgentOperations defines the minimal interface that system handlers need to interact with an Agent.
//
// This interface enables:
// - Loose coupling between handlers and Agent implementation
// - Easy unit testing with mock implementations
// - Clear documentation of handler dependencies
//
// The interface only exposes the specific operations that system handlers require,
// following the Interface Segregation Principle.
type AgentOperations interface {
	// LoadDelegateTool rebuilds the delegate tool when sub-agents change
	LoadDelegateTool()

	// EnsureSystemPrompt rebuilds the system prompt when configuration changes
	EnsureSystemPrompt()

	// InitAgentContext rebuilds the agent context when tools or sub-agents change
	InitAgentContext()
}

// SystemHandlers encapsulates all system-level event handlers.
//
// This struct provides a clean separation between system handlers and the agent implementation,
// making handlers easier to test and maintain. Instead of package-level variables,
// handlers are methods on this struct that can be injected with dependencies.
//
// System handlers respond to key agent lifecycle events:
// - New system agents being added
// - New tools being registered
// - Agent initialization and plugin setup
type SystemHandlers struct {
}

// AgentProvider defines the minimal interface needed to convert an agent to AgentOperations.
// This is needed because the hook system passes concrete *Agent types, but our handlers
// use the AgentOperations interface for testability.
type AgentProvider interface {
	AgentOperations
}

// NewSystemHandlers creates a new SystemHandlers instance.
//
// This constructor creates a handlers instance. The handlers are not stored internally
// but are accessed through methods for testing purposes.
//
// Returns:
//   - *SystemHandlers: A new instance
func NewSystemHandlers() *SystemHandlers {
	return &SystemHandlers{}
}

// RegisterWith registers all system handlers with the provided hook registration function.
//
// This method connects the system handlers to the agent's hook system,
// allowing them to respond to lifecycle events. The registration is done through
// adapters provided by the caller since this package cannot depend on the concrete
// Agent type.
//
// Parameters:
//   - registerSystemAgentHook: Adapter function for registering the system agent added hook
//   - registerToolsHook: Adapter function for registering the tools added hook
func (h *SystemHandlers) RegisterWith(
	registerSystemAgentHook func(handler func(AgentOperations, core.SubAgent) error),
	registerToolsHook func(handler func(AgentOperations, []llms.Tool) error),
) {
	registerSystemAgentHook(h.handleSystemAgentAdded)
	registerToolsHook(h.handleToolsAdded)
}

// ===== Handler Implementations =====
//
// These methods implement the actual handler logic using the AgentOperations interface.
// This enables easy unit testing with mock implementations.

// handleSystemAgentAdded handles the addition of a new system agent.
//
// When a new system agent is added, this handler:
// 1. Reloads the delegate tool to include the new sub-agent
// 2. Rebuilds the system prompt to document the new sub-agent
// 3. Rebuilds the agent context with updated sub-agent information
func (h *SystemHandlers) handleSystemAgentAdded(a AgentOperations, subAgent core.SubAgent) error {
	if subAgent == nil {
		return errors.New("sub agent is nil")
	}

	// A new system agent will trigger a reload of the delegate tool
	a.LoadDelegateTool()
	// A rebuild of the system prompt
	a.EnsureSystemPrompt()
	// A rebuild of the agent context
	a.InitAgentContext()

	return nil
}

// handleToolsAdded handles the addition of new tools.
//
// When new tools are added, this handler:
// 1. Rebuilds the system prompt to include the new tool descriptions
// 2. Rebuilds the agent context with updated tool information
func (h *SystemHandlers) handleToolsAdded(a AgentOperations, tools []llms.Tool) error {
	if tools == nil {
		return errors.New("tools slice is nil")
	}

	// New tools will trigger a rebuild of the system prompt
	a.EnsureSystemPrompt()
	// A rebuild of the agent context
	a.InitAgentContext()

	return nil
}
