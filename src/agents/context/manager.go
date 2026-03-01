package context

import (
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// Ensure Manager implements agents.ContextManager interface at compile time
// This is a forward declaration - the actual interface check is done
// in the agents package to avoid circular dependencies

// Config holds configuration for context management.
type Config struct {
	AgentName          string
	Trace              string
	Model              string
	Tools              []llms.Tool
	SubAgents          []core.SubAgent
	TokenCounter       llms.TokenCounter  // Optional: for token counting
	TruncationStrategy TruncationStrategy // Optional: for history truncation
	MaxContextTokens   int                // Maximum tokens allowed in context
	ReservedTokens     int                // Reserved tokens for output
	WorkingDir         string             // Agent's working directory
}

// Manager manages agent context lifecycle and operations.
//
// The Manager is responsible for:
// - Creating and maintaining AgentContext
// - Building context maps for tool execution
// - Syncing changes back after tool execution
// - Token counting (if configured)
// - History truncation (if configured)
type Manager struct {
	agentContext       *core.AgentContext
	config             Config
	tokenCounter       llms.TokenCounter
	truncationStrategy TruncationStrategy
	maxContextTokens   int
	reservedTokens     int
}

// NewManager creates a new context Manager.
func NewManager(config Config) *Manager {
	m := &Manager{
		config:             config,
		tokenCounter:       config.TokenCounter,
		truncationStrategy: config.TruncationStrategy,
		maxContextTokens:   config.MaxContextTokens,
		reservedTokens:     config.ReservedTokens,
	}
	m.initAgentContext()
	return m
}

// Context returns the current AgentContext.
func (m *Manager) Context() *core.AgentContext {
	return m.agentContext
}

// UpdateConfig updates the manager configuration and rebuilds context.
// Call when tools or sub-agents change.
func (m *Manager) UpdateConfig(cfg interface{}) {
	if config, ok := cfg.(Config); ok {
		m.config = config
		m.tokenCounter = config.TokenCounter
		m.truncationStrategy = config.TruncationStrategy
		m.maxContextTokens = config.MaxContextTokens
		m.reservedTokens = config.ReservedTokens
		m.initAgentContext()
	}
}

// BuildContext converts the AgentContext to a map for tool execution.
// This method merges the static context with the session-specific responseCh.
//
// Parameters:
//   - responseCh: The response channel for the current chat session
//
// Returns:
//   - map[string]any: The complete agent context map ready for tool execution
func (m *Manager) BuildContext(responseCh *core.ResponseCh) map[string]any {
	// Delegate to AgentContext's BuildContext method
	// This maintains the existing serialization logic
	return m.agentContext.BuildContext(responseCh)
}

// SyncFromMap syncs mutable fields from the context map back to AgentContext.
// This ensures changes made by tools persist across tool calls.
//
// Parameters:
//   - contextMap: The context map that may have been modified during tool execution
//
// Returns:
//   - error: An error if sync fails (should not happen in normal operation)
func (m *Manager) SyncFromMap(contextMap map[string]any) error {
	// Delegate to AgentContext's SyncFromMap method
	// This maintains the existing deserialization logic
	return m.agentContext.SyncFromMap(contextMap)
}

// TruncateHistory applies truncation to message history if configured.
// Returns the truncated history, or the original if truncation is not configured.
//
// Parameters:
//   - messages: The full message history to potentially truncate
//
// Returns:
//   - Truncated message slice that fits within token budget
func (m *Manager) TruncateHistory(messages []*llms.UnifiedMessage) []*llms.UnifiedMessage {
	// Skip truncation if not configured
	if m.truncationStrategy == nil || m.maxContextTokens <= 0 {
		return messages
	}

	// Calculate available token budget (reserve space for completion)
	maxAllowed := m.maxContextTokens - m.reservedTokens
	if maxAllowed <= 0 {
		// If no budget available, return empty slice or minimal messages
		return messages
	}

	// Delegate to configured strategy
	return m.truncationStrategy.Truncate(messages, maxAllowed)
}

// UpdateTools updates the tools in the context.
// This is a convenience method that updates just the tools without rebuilding entire config.
func (m *Manager) UpdateTools(tools []llms.Tool) {
	m.config.Tools = tools
	m.initAgentContext()
}

// UpdateSubAgents updates the sub-agents in the context.
// This is a convenience method that updates just the sub-agents without rebuilding entire config.
func (m *Manager) UpdateSubAgents(agents []core.SubAgent) {
	m.config.SubAgents = agents
	m.initAgentContext()
}

func (m *Manager) initAgentContext() {
	var existingSessionStorage map[string]any
	if m.agentContext != nil && m.agentContext.SessionStorage != nil {
		existingSessionStorage = m.agentContext.SessionStorage
	} else {
		existingSessionStorage = make(map[string]any)
	}

	var existingPluginFields map[string]any
	if m.agentContext != nil && m.agentContext.PluginFields != nil {
		existingPluginFields = m.agentContext.PluginFields
	} else {
		existingPluginFields = make(map[string]any)
	}

	m.agentContext = &core.AgentContext{
		AgentName:      m.config.AgentName,
		Trace:          m.config.Trace,
		Model:          m.config.Model,
		Tools:          m.config.Tools,
		WorkingDir:     m.config.WorkingDir,
		SubAgents:      m.config.SubAgents,
		SessionStorage: existingSessionStorage,
		PluginFields:   existingPluginFields,
	}
}
