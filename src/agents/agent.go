package agents

import (
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// Agent represents an advanced agent with an LLM engine.
//
// This struct wraps an LLM engine (OpenAI-compatible) and provides
// a higher-level agent interface.
type Agent struct {
	// Configurations.
	// Used at initialization time
	config *AgentConfig
	// llmEngine is the underlying LLM engine that handles streaming responses.
	// It implements the llms.Agent interface which provides ChatStream method.
	llmEngine *llms.LLMEngine
	// Response channel for the agent.
	responseCh *core.ResponseCh
	// Tools available to the agent.
	tools []llms.Tool
	// Deal with message history.
	history *History
	// Subsystem of agents
	subAgents []*core.SubAgent
	// If this is a main agent of a team of agents.
	// Extra engine configurations for subsystems of agents.
	extraEngines map[string]llms.LLMEngine
	// System Prompt as a final system prompt
	// Agent context built once at initialization
	agentContext *core.AgentContext
	// System Prompt as a final system prompt
	systemPrompt string
	// Hooks for the agent
	hooks *AgentHooks
}

// ===== Constructor =====

// NewAgent creates a new Agent instance with the provided configuration.
//
// Tools can be set on the LLM engine before or after creating the Agent.
// Use GetTools() and SetTools() methods to manage tools via the Agent interface.
//
// This function validates that all required fields are set before creating the Agent.
// It panics if validation fails to ensure invalid agents are never created.
//
// Parameters:
//   - config: AgentConfig struct containing all agent configuration parameters
//
// Returns:
//   - *Agent: A new Agent instance
//
// Panics:
//   - If required fields (LLMEngine or AgentName) are missing
func NewAgent(config *AgentConfig) *Agent {

	a := &Agent{
		config: config,
	}

	a.ensureConfig()
	a.ensureHooks()
	a.addSystemAgents()
	a.initSystemTools()
	a.loadDelegateTool()
	a.setResponseCh()
	a.initAgentContext()
	a.registerSystemCallbacks()
	a.ensureSystemPrompt()

	return a
}

// ===== Public Methods =====

// ChatStream sends a message to the underlying LLM engine and returns a response channel
// for streaming responses with agent-specific context.
//
// This method creates a ResponseCh with channels for receiving streaming chunks
// and errors. The chunks are forwarded from the underlying LLM's ResponseCh and enriched
// with agent name and trace information.
//
// This method implements the core.SubAgent interface.
//
// Parameters:
//   - message: The user message to send
//
// Returns:
//   - *core.ResponseCh: Response channel that can be used to receive streaming chunks
func (a *Agent) ChatStream(message string) *core.ResponseCh {
	errs := a.hooks.newUserMessageEvent(a, message)
	logHookErrors(errs)

	// Start the tool execution loop in a goroutine
	go func() {
		defer a.responseCh.Close()

		if err := a.executeChatWithTools(); err != nil {
			a.responseCh.Error <- err
		}
	}()

	return a.responseCh
}

// GetTools returns the list of tools currently configured for this agent.
//
// Returns:
//   - []llms.Tool: Slice of tools (empty slice if no tools configured, never nil)
func (a *Agent) GetTools() []llms.Tool {
	if a.tools == nil {
		return []llms.Tool{}
	}
	return a.tools
}

// SetTools sets the tools available to this agent.
//
// Tools can be set at any time and will be used in subsequent ChatStream calls.
//
// Parameters:
//   - tools: Slice of tools to configure (can be nil or empty)
func (a *Agent) SetTools(tools []llms.Tool) {
	a.tools = tools
}

// ===== Sub Agent Interface =====

func (a *Agent) Name() string {
	return a.config.AgentName
}

func (a *Agent) Description() string {
	return a.config.Description
}

func (a *Agent) BasicDescription() string {
	return a.config.Description
}

func (a *Agent) AdvanceDescription() string {
	return a.config.AdvanceDescription
}

func (a *Agent) Trace() string {
	return a.config.Trace
}

func (a *Agent) SystemPrompt() string {
	return a.config.SystemPrompt
}

// Troubleshooting returns information about common issues, debugging tips,
// and configuration guidance for this agent.
// This implements the agentforge.Discoverable interface.
func (a *Agent) Troubleshooting() string {
	return a.config.Troubleshooting
}

// AddSystemAgent adds a system agent (sub-agent) to this agent's list of sub-agents.
//
// This method allows dynamic addition of sub-agents after agent initialization.
// It updates the delegate tool and agent context to include the new sub-agent.
// The system prompt will be automatically updated on the next message to include
// the new sub-agent in the list of available sub-agents.
//
// Parameters:
//   - subAgent: The sub-agent to add (must implement core.SubAgent interface)
//
// Example:
//
//	evalAgent := agents.EvalAgent(llmEngine, "/path/to/root")
//	mainAgent.AddSystemAgent(evalAgent)
func (a *Agent) AddSystemAgent(subAgent *core.SubAgent) {
	errs := a.hooks.addSystemAgentEvent(a, subAgent)
	logHookErrors(errs)

	// Add sub-agent to the list
	a.subAgents = append(a.subAgents, subAgent)

	errs = a.hooks.addedSystemAgentEvent(a, subAgent)
	logHookErrors(errs)
}

func (a *Agent) AgentAsSubAgent() *core.SubAgent {
	a.config.MainAgent = false
	sa := core.SubAgent(a)
	return &sa
}
