package agents

import (
	"context"
	"encoding/json"
	"fmt"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	agentctx "github.com/thinktwiceco/agent-forge/src/agents/context"
	"github.com/thinktwiceco/agent-forge/src/agents/prompts"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/history"
	"github.com/thinktwiceco/agent-forge/src/llms"
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
	llmEngine llms.LLMEngine
	// Response channel for the agent.
	responseCh *core.ResponseCh
	// Tools available to the agent.
	tools []llms.Tool
	// Subsystem of agents
	subAgents []core.SubAgent
	// System Prompt as a final system prompt
	// Agent context built once at initialization
	agentContext *core.AgentContext
	// System Prompt as a final system prompt
	systemPrompt string
	// Hooks for the agent
	hooks HookRegistry
	// Extracted components (execution, prompts, context)
	// Using interfaces for loose coupling and testability
	executor      ExecutionEngine
	promptBuilder PromptBuilder
	contextMgr    ContextManager
	// Context window management (prepared for future token counting implementation)
	maxContextTokens     int
	reservedOutputTokens int
	minRecentMessages    int
	enableSummarization  bool
	tokenCounter         llms.TokenCounter
	// History factory creates a new history Manager for each ChatStream call
	historyFactory func(chatId string) history.Manager
}

// ===== Constructor =====

// NewAgent creates a new Agent instance with the provided configuration.
//
// Tools can be set on the LLM engine before or after creating the Agent.
// Use GetTools() and AddTools() methods to manage tools via the Agent interface.
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

	// Initialize prompt builder early (before plugins) with minimal config
	// Plugins may need to update the system prompt during initialization
	a.promptBuilder = prompts.NewBuilder(prompts.Config{
		SystemPrompt: config.SystemPrompt,
		MainAgent:    config.MainAgent,
		Tone:         config.Tone,
		Tools:        []llms.Tool{},     // Will be populated later
		SubAgents:    []core.SubAgent{}, // Will be populated later
	})

	a.ensureHooks()
	a.registerPlugins()

	errs := a.hooks.agentInitializationEvent(a, config)
	logHookErrors(errs)

	// Data consistency step.
	// Loads the configuration, setup the system agents,
	// And initializes the system tools.
	a.ensureConfig()
	a.addSystemAgents()
	a.initSystemTools()
	a.loadDelegateTool()
	a.setResponseCh()

	// Create context manager and set agentContext
	// Pass truncation strategy and token counter to the manager
	a.contextMgr = agentctx.NewManager(agentctx.Config{
		AgentName:          a.Name(),
		Trace:              a.Trace(),
		Model:              fmt.Sprintf("%s-%s", a.config.LLMEngine.Provider(), a.config.LLMEngine.Model()),
		Tools:              a.tools,
		SubAgents:          a.subAgents,
		TokenCounter:       a.tokenCounter,
		TruncationStrategy: a.config.TruncationStrategy,
		MaxContextTokens:   a.maxContextTokens,
		ReservedTokens:     a.reservedOutputTokens,
	})
	a.agentContext = a.contextMgr.Context()

	// Update prompt builder with final tools and subagents, then build system prompt
	a.promptBuilder.UpdateConfig(prompts.Config{
		SystemPrompt: a.config.SystemPrompt,
		MainAgent:    a.config.MainAgent,
		Tone:         a.config.Tone,
		Tools:        a.tools,
		SubAgents:    a.subAgents,
	})
	a.systemPrompt = a.promptBuilder.Build()

	// Create executor
	a.executor = a.createExecutor()

	// System Callbacks are defined in systemHandlers.go
	a.registerSystemCallbacks()

	errs = a.hooks.agentInitializedEvent(a)

	logHookErrors(errs)

	// Log the system prompt if it's the main agent
	if a.config.MainAgent {
		agentforge.Debug("========================================================")
		agentforge.Debug("System prompt for agent %s: %s", a.config.AgentName, a.systemPrompt)
		agentforge.Debug("========================================================")
	}

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
//   - ctx: Context for cancellation and deadline control
//   - message: The user message to send
//   - chatId: The conversation ID (empty string for new conversations)
//
// Returns:
//   - *core.ResponseCh: Response channel that can be used to receive streaming chunks
func (a *Agent) ChatStream(ctx context.Context, message string, chatId string) *core.ResponseCh {
	// Create a new History instance for this request (per-request history)
	// This eliminates concurrency issues with shared state
	hm := a.createHistory(chatId)
	a.injectSystemPrompt(hm)
	hm.AddUserMessage(message)
	errs := a.hooks.newUserMessageEvent(a, message)
	logHookErrors(errs)

	// Create a new response channel for this ChatStream call
	// The hook callback triggers agent hooks when chunks are read
	onChunkRead := func(extendedChunk *core.ExtendedChunkResponse) error {
		// Trigger hooks with extended chunk (includes AgentName and Trace)
		errs := a.hooks.newChunkEvent(a, extendedChunk)
		for _, err := range errs {
			if err != nil {
				return err
			}
		}
		return nil
	}
	responseCh := core.NewResponseCh(a.config.AgentName, a.config.Trace, chatId, onChunkRead)

	// Store the response channel temporarily for use in executeChatWithTools
	oldResponseCh := a.responseCh
	a.responseCh = responseCh

	// Start the tool execution loop in a goroutine
	go func() {
		defer responseCh.Close()
		// Restore the old responseCh after this call completes
		defer func() {
			a.responseCh = oldResponseCh
		}()

		if err := a.executor.ExecuteChatWithTools(ctx, hm, responseCh); err != nil {
			responseCh.Error <- err
		}
		// Save history and get the final chatId (generated if it was empty)
		finalChatId := hm.Save()
		responseCh.SetChatId(finalChatId)

		// Send final chunk with chatId so frontend receives it before stream closes.
		// Without this, new conversations never propagate the generated ID to the client.
		if finalChatId != "" {
			chatIdChunk := core.ExtendedChunkResponse{
				ChatId:    finalChatId,
				Status:    llms.StatusCompleted,
				Type:      llms.TypeCompletion,
				Content:   "",
				AgentName: a.config.AgentName,
				Trace:     a.config.Trace,
			}
			if chunkBytes, err := json.Marshal(chatIdChunk); err == nil {
				responseCh.TrySend(chunkBytes)
			}
		}
	}()

	return responseCh
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

// AddTools adds tools to the existing tools available to this agent.
//
// Tools can be added at any time and will be used in subsequent ChatStream calls.
// This method appends the provided tools to the existing tools slice, triggering
// a refresh of the system prompt and agent context.
//
// Parameters:
//   - tools: Slice of tools to add (can be nil or empty)
func (a *Agent) AddTools(tools []llms.Tool) {
	if len(tools) == 0 {
		return
	}

	// Ensure tools slice exists
	if a.tools == nil {
		a.tools = []llms.Tool{}
	}

	// Append new tools to existing tools
	a.tools = append(a.tools, tools...)

	// Trigger the addedToolsEvent hook
	errs := a.hooks.addedToolsEvent(a, tools)
	logHookErrors(errs)
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

func (a *Agent) Context() *core.AgentContext {
	return a.agentContext
}

func (a *Agent) ResponseCh() *core.ResponseCh {
	return a.responseCh
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
func (a *Agent) AddSystemAgent(subAgent core.SubAgent) {
	errs := a.hooks.addSystemAgentEvent(a, subAgent)
	logHookErrors(errs)

	// Add sub-agent to the list
	a.subAgents = append(a.subAgents, subAgent)

	errs = a.hooks.addedSystemAgentEvent(a, subAgent)
	logHookErrors(errs)
}

func (a *Agent) AgentAsSubAgent() core.SubAgent {
	a.config.MainAgent = false
	return a
}

// ===== AgentOperations Interface Implementation =====
//
// These methods implement the handlers.AgentOperations interface,
// providing the minimal operations that system handlers need.

// LoadDelegateTool rebuilds the delegate tool when sub-agents change.
// This is part of the handlers.AgentOperations interface.
func (a *Agent) LoadDelegateTool() {
	a.loadDelegateTool()
}

// EnsureSystemPrompt rebuilds the system prompt when configuration changes.
// This is part of the handlers.AgentOperations interface.
func (a *Agent) EnsureSystemPrompt() {
	a.ensureSystemPrompt()
}

// InitAgentContext rebuilds the agent context when tools or sub-agents change.
// This is part of the handlers.AgentOperations interface.
func (a *Agent) InitAgentContext() {
	a.initAgentContext()
}

// ===== Compile-time Interface Assertions =====
//
// Ensure Agent implements the SubAgent composition and its constituent interfaces.
var _ core.SubAgent = (*Agent)(nil)
var _ core.Executable = (*Agent)(nil)
var _ core.Identifier = (*Agent)(nil)
