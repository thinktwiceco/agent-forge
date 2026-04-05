package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
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

	// toolsMu protects read/write access to tools
	toolsMu sync.RWMutex
	// Tools available to the agent.
	tools []llms.Tool
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
	inbox *queue.Queue
	// inboxCancel cancels the current background inbox drain goroutine.
	// Called when Drain replaces the internal queue with an external one.
	inboxCancel context.CancelFunc
	// chunkRouter is an optional callback that receives each chunk produced by the background drain.
	// The HTTP layer sets this to route background responses to the appropriate push SSE connection.
	chunkRouter func(chatId string, chunk core.ExtendedChunkResponse)
	// turnCompleteRouter is called after each background drain turn with the accumulated full content.
	// It is not called when content is empty (e.g. suppressed HEARTBEAT_OK turns).
	turnCompleteRouter func(chatId string, fullContent string)
	// memoryPrefixProvider is optional; when set, injectSystemPrompt prepends its return value
	// to the system message each ChatStream (e.g. brain/MEMORY.md). Empty string means no prefix.
	memoryPrefixProvider func() string
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
		Tools:        []llms.Tool{}, // Will be populated later
	})

	a.ensureHooks()
	a.inbox = queue.New(64)
	a.registerPlugins()

	errs := a.hooks.agentInitializationEvent(a, config)
	logHookErrors(errs)

	// Data consistency step: loads configuration and initializes system tools.
	a.ensureConfig()
	a.initSystemTools()
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
		TokenCounter:       a.tokenCounter,
		TruncationStrategy: a.config.TruncationStrategy,
		MaxContextTokens:   a.maxContextTokens,
		ReservedTokens:     a.reservedOutputTokens,
		WorkingDir:         a.config.WorkingDir,
	})
	a.agentContext = a.contextMgr.Context()

	// Update prompt builder with final tools, then build system prompt
	a.promptBuilder.UpdateConfig(prompts.Config{
		SystemPrompt: a.config.SystemPrompt,
		MainAgent:    a.config.MainAgent,
		Tone:         a.config.Tone,
		Tools:        a.tools,
	})
	a.systemPrompt = a.promptBuilder.Build()

	// Create executor
	a.executor = a.createExecutor()

	a.executor.UpdateTools(a.tools)
	a.startBackgroundDrain()

	// System callbacks: registerSystemCallbacks in agentInit.go (handlers package).
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
// Parameters:
//   - ctx: Context for cancellation and deadline control
//   - message: The user message to send
//   - chatId: The conversation ID (empty string for new conversations)
//
// Returns:
//   - *core.ResponseCh: Response channel that can be used to receive streaming chunks
func (a *Agent) ChatStream(ctx context.Context, message string, chatId string) *core.ResponseCh {
	// Pre-generate a UUID for new conversations so the chatId is known for the
	// entire duration of this turn — including during tool execution. Without this,
	// plugins (e.g. brain) and memory tools cannot create per-conversation files on
	// the first message because the chatId is only assigned by hm.Save() at the end.
	// hm.Save() will reuse this ID rather than generating a new one.
	if chatId == "" {
		chatId = uuid.NewString()
	}

	// Create a new History instance for this request (per-request history)
	// This eliminates concurrency issues with shared state
	hm := a.createHistory(chatId)
	a.injectSystemPrompt(hm)
	prevMessages := append([]*llms.UnifiedMessage(nil), hm.Messages()...)
	hm.AddUserMessage(message)
	errs := a.hooks.newUserMessageEvent(a, message)
	logHookErrors(errs)

	// Notify plugins that a chat turn has started with this conversation ID.
	errs = a.hooks.chatStartEvent(a, chatId)
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

		execResult, err := a.executor.ExecuteChatWithTools(ctx, hm, responseCh)
		if err != nil {
			responseCh.Error <- err
		} else if execResult.HeartbeatAckSuppressed {
			hm.SetMessages(prevMessages)
		}

		var finalChatId string
		if err == nil && !execResult.HeartbeatAckSuppressed {
			saveId, saveErr := hm.Save()
			if saveErr != nil {
				agentforge.Error("Failed to save history: %v", saveErr)
			}
			finalChatId = saveId
			responseCh.SetChatId(finalChatId)
		}

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
	a.toolsMu.RLock()
	defer a.toolsMu.RUnlock()
	if a.tools == nil {
		return []llms.Tool{}
	}
	res := make([]llms.Tool, len(a.tools))
	copy(res, a.tools)
	return res
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

	a.toolsMu.Lock()
	if a.tools == nil {
		a.tools = []llms.Tool{}
	}
	a.tools = append(a.tools, tools...)
	a.toolsMu.Unlock()

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

// DetailsAbout returns per-item details when DetailsAboutFunc is set; otherwise
// "Nothing to add about <item>". Implements the agentforge.Discoverable interface.
func (a *Agent) DetailsAbout(item string) string {
	if a.config.DetailsAboutFunc != nil {
		return a.config.DetailsAboutFunc(item)
	}
	return fmt.Sprintf("Nothing to add about %s", item)
}

// ===== AgentOperations Interface Implementation =====
//
// These methods implement the handlers.AgentOperations interface,
// providing the minimal operations that system handlers need.

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

// SetMemoryPrefixProvider registers a function called on each ChatStream to prepend
// per-turn content to the system message (after trimming empty). Plugins such as brain
// use this to inject brain/MEMORY.md without importing plugin code into the agent package.
func (a *Agent) SetMemoryPrefixProvider(f func() string) {
	a.memoryPrefixProvider = f
}

// InitAgentContext rebuilds the agent context when tools change.
// This is part of the handlers.AgentOperations interface.
func (a *Agent) InitAgentContext() {
	a.initAgentContext()
}

// SetTurnCompleteRouter sets a callback that is invoked after each background drain turn
// with the accumulated full content. Not called for suppressed turns (empty content).
func (a *Agent) SetTurnCompleteRouter(fn func(chatId string, fullContent string)) {
	a.turnCompleteRouter = fn
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
				var sb strings.Builder
				finalChatId := msg.ChatId
				for chunk := range responseCh.Start() {
					a.routeChunk(chunk)
					if chunk.Content != "" {
						sb.WriteString(chunk.Content)
					}
					if chunk.ChatId != "" {
						finalChatId = chunk.ChatId
					}
				}
				if a.turnCompleteRouter != nil && sb.Len() > 0 {
					a.turnCompleteRouter(finalChatId, sb.String())
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
	a.inbox = q
	a.toolsMu.RLock()
	a.executor.UpdateTools(a.tools)
	a.toolsMu.RUnlock()

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
// Ensure Agent implements Executable and Identifier for streaming and identification.
var _ core.Executable = (*Agent)(nil)
var _ core.Identifier = (*Agent)(nil)
