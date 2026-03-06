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
	"github.com/thinktwiceco/agent-forge/src/queue"
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
	// inbox is the agent's message queue.
	// Created automatically at construction time; replaced when Drain is called with an external queue.
	// The delegate tool uses this to enqueue sub-agent responses for async correlation.
	inbox *queue.Queue
	// inboxCancel cancels the current background inbox drain goroutine.
	// Called when Drain replaces the internal queue with an external one.
	inboxCancel context.CancelFunc
	// chunkRouter is an optional callback that receives each chunk produced by the background drain.
	// The HTTP layer sets this to route background responses to the appropriate push SSE connection.
	chunkRouter func(chatId string, chunk core.ExtendedChunkResponse)
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

	// Inject working directory to all initially registered tools
	if a.config.WorkingDir != "" {
		for _, tool := range a.tools {
			if wda, ok := tool.(core.WorkingDirAware); ok {
				wda.SetWorkingDir(a.config.WorkingDir)
			}
		}
	}

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
		WorkingDir:         a.config.WorkingDir,
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

	// Initialize the internal inbox queue and start a background drain goroutine.
	// This ensures the delegate tool always has a queue available, even when the agent
	// is used via HTTP without an explicit Drain call.
	a.inbox = queue.New(64)
	a.loadDelegateTool()
	a.executor.UpdateTools(a.tools)
	a.startBackgroundDrain()

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

	// Start the tool execution loop in a goroutine
	go func() {
		defer responseCh.Close()

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

	// Inject working directory to newly added tools
	if a.config.WorkingDir != "" {
		for _, tool := range tools {
			if wda, ok := tool.(core.WorkingDirAware); ok {
				wda.SetWorkingDir(a.config.WorkingDir)
			}
		}
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

// DetailsAbout returns "Nothing to add about <item>" for agents, as agents do not
// expose per-item detail. This implements the agentforge.Discoverable interface.
func (a *Agent) DetailsAbout(item string) string {
	return fmt.Sprintf("Nothing to add about %s", item)
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

// AppendSystemPrompt appends text to the agent's system prompt and rebuilds it.
// Plugins that load data asynchronously (e.g. during EventAgentInitialized) can
// call this after their data is ready to inject their prompt contribution.
func (a *Agent) AppendSystemPrompt(text string) {
	a.config.SystemPrompt += text
	a.ensureSystemPrompt()
}

// InitAgentContext rebuilds the agent context when tools or sub-agents change.
// This is part of the handlers.AgentOperations interface.
func (a *Agent) InitAgentContext() {
	a.initAgentContext()
}

// SetChunkRouter sets a callback that receives each chunk produced by the background drain.
// The HTTP layer uses this to forward background responses to the appropriate push SSE connection.
func (a *Agent) SetChunkRouter(fn func(chatId string, chunk core.ExtendedChunkResponse)) {
	a.chunkRouter = fn
}

// routeChunk forwards a chunk through the chunk router if one is set and the chunk has a chatId.
func (a *Agent) routeChunk(chunk core.ExtendedChunkResponse) {
	if a.chunkRouter != nil && chunk.ChatId != "" {
		a.chunkRouter(chunk.ChatId, chunk)
	}
}

// startBackgroundDrain starts an internal goroutine that drains a.inbox.
// It is cancelled by calling a.inboxCancel(). Used for the automatic internal queue
// created at agent construction time.
func (a *Agent) startBackgroundDrain() {
	ctx, cancel := context.WithCancel(context.Background())
	a.inboxCancel = cancel
	q := a.inbox
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-q.C():
				if !ok {
					return
				}
				responseCh := a.ChatStream(ctx, msg.Format(), msg.ChatId)
				for chunk := range responseCh.Start() {
					a.routeChunk(chunk)
				}
			}
		}
	}()
}

// Drain replaces the internal inbox queue with q and drains it in the calling goroutine,
// blocking until ctx is cancelled or q is closed.
//
// Each message is routed to its own conversation via the ChatId embedded in the message.
// The formatted message content (headers + body) is what the agent receives, allowing it
// to observe metadata such as the sender identity, timestamp, and reqId.
//
// Calling Drain stops the internal background drain goroutine (started automatically at
// construction) and switches to the provided queue. This is useful when the caller needs
// direct control over the inbox, e.g. to inject messages from the web UI.
//
// Example:
//
//	q := queue.New(32)
//	go agent.Drain(ctx, q)
//	q.Enqueue("Ciao!", "conv-1", map[string]string{"sender": "user"})
func (a *Agent) Drain(ctx context.Context, q *queue.Queue) {
	// Cancel the background drain goroutine that was started at construction.
	if a.inboxCancel != nil {
		a.inboxCancel()
		a.inboxCancel = nil
	}
	// Replace inbox and sync the delegate tool with the new queue reference.
	a.inbox = q
	a.loadDelegateTool()
	a.executor.UpdateTools(a.tools)

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-q.C():
			if !ok {
				return
			}
			responseCh := a.ChatStream(ctx, msg.Format(), msg.ChatId)
			for chunk := range responseCh.Start() {
				a.routeChunk(chunk)
			}
		}
	}
}

// ===== Compile-time Interface Assertions =====
//
// Ensure Agent implements the SubAgent composition and its constituent interfaces.
var _ core.SubAgent = (*Agent)(nil)
var _ core.Executable = (*Agent)(nil)
var _ core.Identifier = (*Agent)(nil)
