// ─── Agent Layer: Initialization ─────────────────────────────────────────────
//
// The agent layer is purely a runtime concern. It derives all of its state from
// agents.AgentConfig (produced by builder.AgentFactory.Build()) and holds no
// durable state of its own. On every AgentManager.Reload() the entire agent is
// discarded and rebuilt from scratch — a safe operation because:
//
//  1. In-flight chats hold a pointer to the old agent (captured before the swap)
//     and complete normally; the new agent only serves subsequent requests.
//  2. The RWMutex in AgentManager guarantees a clean pointer swap without races.
//
// Plugin wiring is deferred to EventAgentInitialization so that all agent
// infrastructure (hooks, inbox, tools slice) is ready before plugins attach to
// it. The initialization handler:
//
//   - Injects WorkingDir into WorkingDirAware plugins
//   - Injects the agent turn inbox into InboxAware plugins (allows async injection)
//   - Registers lifecycle hooks from HookProvider plugins
//   - Appends tools from ToolProvider plugins to the agent's tool list
//   - Appends system-prompt fragments from PromptProvider plugins
//
// Nothing in this file writes to disk or persists across restarts.

package agents

import (
	"fmt"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	agentctx "github.com/thinktwiceco/agent-forge/src/agents/context"
	"github.com/thinktwiceco/agent-forge/src/agents/execution"
	"github.com/thinktwiceco/agent-forge/src/agents/handlers"
	"github.com/thinktwiceco/agent-forge/src/agents/prompts"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/tools/expand"
	"github.com/thinktwiceco/agent-forge/src/tools/meta"
	"github.com/thinktwiceco/agent-forge/src/tools/searchlogs"
	"github.com/thinktwiceco/agent-forge/src/tools/spawn"
)

// ==============================
// ===== Initialization Methods
// ==============================

// ensureConfig validates and sets default configuration values
func (a *Agent) ensureConfig() {
	if err := a.config.validate(); err != nil {
		panic(fmt.Errorf("invalid AgentConfig: %w", err))
	}

	if a.config.MaxToolIterations <= 0 {
		a.config.MaxToolIterations = 10
	}

	a.llmEngine = a.config.LLMEngine

	// Initialize tools from config (if provided)
	if a.config.Tools != nil {
		a.tools = make([]llms.Tool, len(a.config.Tools))
		copy(a.tools, a.config.Tools)
	}

	a.systemPrompt = a.config.SystemPrompt

	// Initialize context window management fields
	a.maxContextTokens = a.config.MaxContextTokens
	if a.maxContextTokens == 0 {
		if info := a.llmEngine.ModelInfo(); info.ContextWindow > 0 {
			a.maxContextTokens = info.ContextWindow
			agentforge.Debug("MaxContextTokens auto-set from ModelInfo: %d", a.maxContextTokens)
		}
	}
	a.reservedOutputTokens = a.config.ReservedOutputTokens
	a.minRecentMessages = a.config.MinRecentMessages
	a.enableSummarization = a.config.EnableSummarization

	if a.maxContextTokens > 0 {
		agentforge.Debug("Context window management enabled: maxTokens=%d, reserved=%d, minRecent=%d, summarization=%v",
			a.maxContextTokens, a.reservedOutputTokens, a.minRecentMessages, a.enableSummarization)

		// Initialize token counter if context window management is enabled
		a.tokenCounter = llms.NewTokenCounter("approximate")
		agentforge.Debug("Token counter initialized for context window management")

		// Initialize truncation strategy if not set
		if a.config.TruncationStrategy == nil {
			a.config.TruncationStrategy = agentctx.NewSlidingWindow(
				a.minRecentMessages,
				a.tokenCounter,
			)
			agentforge.Debug("Using default SlidingWindowStrategy for truncation")
		} else {
			agentforge.Debug("Using custom TruncationStrategy")
		}
	}

	// Set up history factory for per-request history creation
	a.historyFactory = a.createHistoryFactory()
}

// prepare the callbacks for the agent
func (a *Agent) ensureHooks() {
	if a.hooks == nil {
		a.hooks = newAgentHooks()
	}
}

func (a *Agent) setResponseCh() {
	// Create hook callback that triggers agent hooks when chunks are read
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
	a.responseCh = core.NewResponseCh(a.config.AgentName, a.config.Trace, "", onChunkRead)
}

func (a *Agent) initSystemTools() {
	// Ensure tools slice exists
	if a.tools == nil {
		a.tools = []llms.Tool{}
	}

	// Meta Tool - always add
	mt := meta.NewMetaTool()
	a.tools = append(a.tools, mt)

	// Session log search - always add
	a.tools = append(a.tools, searchlogs.NewSearchLogsTool())

	// Expand Tool - add only if can expand
	if a.config.CanExpand {
		et := expand.NewExpandTool()
		a.tools = append(a.tools, et)
	}

	// Spawn Subagent Tool - add only if enabled
	if a.config.CanSpawnSubagent {
		a.tools = append(a.tools, spawn.NewSpawnSubagentTool(a.newAsyncSubagentSpawner()))
	}

	a.tools = dedupeToolsByName(a.tools)
}

// initAgentContext rebuilds the agent context and syncs all components.
// Called when tools change (e.g. AddTools).
func (a *Agent) initAgentContext() {
	a.contextMgr.UpdateConfig(agentctx.Config{
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
	a.executor.UpdateAgentContext(a.agentContext)
	a.executor.UpdateTools(a.tools)
}

// ensureSystemPrompt rebuilds the system prompt and updates promptBuilder.
// Called when tools change.
func (a *Agent) ensureSystemPrompt() {
	a.promptBuilder.UpdateConfig(prompts.Config{
		SystemPrompt:     a.config.SystemPrompt,
		MainAgent:        a.config.MainAgent,
		Tone:             a.config.Tone,
		Tools:            a.tools,
		CanSpawnSubagent: a.config.CanSpawnSubagent,
	})
	a.systemPrompt = a.promptBuilder.Build()
}

// createExecutor creates the execution Executor with hooks and dependencies.
func (a *Agent) createExecutor() ExecutionEngine {
	hooks := &execution.HooksRunner{
		OnContextBuild: func(agentContext *core.AgentContext) []error {
			return a.hooks.contextBuildEvent(a, agentContext)
		},
		OnBeforeToolExecution: func(toolCall *llms.ToolCall) []error {
			return a.hooks.beforeToolExecutionEvent(a, toolCall)
		},
		OnToolExecution: func(toolResult *llms.ToolResult) []error {
			return a.hooks.toolExecutionEvent(a, toolResult)
		},
		OnNewAssistantMessage: func(message string, promptTokens, completionTokens, totalTokens int) []error {
			return a.hooks.newAssistantMessageEvent(a, message, promptTokens, completionTokens, totalTokens)
		},
		OnNewAssistantMessageWithTools: func(message string, toolCalls []llms.ToolCall, promptTokens, completionTokens, totalTokens int) []error {
			return a.hooks.newAssistantMessageWithToolCallsEvent(a, message, toolCalls, promptTokens, completionTokens, totalTokens)
		},
		LogHookErrors: logHookErrors,
	}
	var truncateHistory func([]*llms.UnifiedMessage) []*llms.UnifiedMessage
	if a.maxContextTokens > 0 {
		truncateHistory = a.contextMgr.TruncateHistory
	}

	return execution.NewExecutor(
		a.llmEngine,
		a.tools,
		a.agentContext,
		execution.Config{
			MaxToolIterations:    a.config.MaxToolIterations,
			AgentName:            a.config.AgentName,
			Tracer:               a.config.Tracer,
			TruncateHistory:      truncateHistory,
			HeartbeatAckMaxChars: a.config.HeartbeatAckMaxChars,
		},
		hooks,
	)
}

func (a *Agent) registerPlugins() {
	agentforge.Debug("🔌 [registerPlugins] START 🔌")

	if a.config == nil {
		agentforge.Debug("⚠️ WARNING: config is nil in registerPlugins")
		return
	}
	if a.hooks == nil {
		agentforge.Debug("⚠️ WARNING: hooks is nil in registerPlugins")
		return
	}
	agentforge.Debug("🔌 ADDING PLUGIN REGISTRATION HOOK FOR AGENT %s 🔌", a.config.AgentName)
	agentforge.Debug("🔌 Plugins count: %d 🔌", len(a.config.Plugins))
	a.hooks.on(core.EventAgentInitialization, a.createPluginInitializationHandler())
	agentforge.Debug("🔌 Plugin registration hook added successfully 🔌")
}

// createPluginInitializationHandler creates the handler for plugin initialization
func (a *Agent) createPluginInitializationHandler() OnAgentInitializationHook {
	return func(agent *Agent, config *AgentConfig) error {
		agentforge.Debug("🔌 [handlePluginInitialization] START for agent %s", agent.config.AgentName)
		agentforge.Debug("🔌 [handlePluginInitialization] Plugins count: %d", len(agent.config.Plugins))

		if len(agent.config.Plugins) == 0 {
			agentforge.Debug("🔌 [handlePluginInitialization] No plugins to register")
			return nil
		}

		systemPromptAdded := false

		// Register all the plugins using interface segregation
		for i, plugin := range agent.config.Plugins {
			agentforge.Debug("🔌 [handlePluginInitialization] Processing plugin %d: %s", i+1, plugin.Name())

			// Inject working directory if the plugin supports it
			if wda, ok := plugin.(core.WorkingDirAware); ok {
				if agent.config.WorkingDir != "" {
					agentforge.Debug("🔌 [handlePluginInitialization] Injecting working directory %s to plugin", agent.config.WorkingDir)
					wda.SetWorkingDir(agent.config.WorkingDir)
				}
			}

			// Inject turn inbox if the plugin supports it
			if ia, ok := plugin.(core.InboxAware); ok {
				agentforge.Debug("🔌 [handlePluginInitialization] Injecting turn inbox to plugin %s", plugin.Name())
				ia.SetInbox(agent.turnQueue)
			}

			// Inject LLM engine if the plugin supports it (e.g. brain dreaming runner).
			// Use config.LLMEngine: ensureConfig() has not run yet, so agent.llmEngine is still nil.
			if la, ok := plugin.(core.LLMEngineAware); ok {
				agentforge.Debug("🔌 [handlePluginInitialization] Injecting LLM engine to plugin %s", plugin.Name())
				la.SetLLMEngine(agent.config.LLMEngine)
			}

			// Check if plugin provides hooks
			if hp, ok := plugin.(core.HookProvider); ok {
				hooks := hp.Hooks()
				for event, hook := range hooks {
					if hook != nil {
						agentforge.Debug("🔌 [handlePluginInitialization] Registering hook for event: %s", event)
						agent.hooks.on(event, hook)
					}
				}
			}

			// Check if plugin provides tools
			if tp, ok := plugin.(core.ToolProvider); ok {
				tools := tp.Tools()
				for _, tool := range tools {
					agentforge.Debug("🔌 [handlePluginInitialization] Registering tool: %s", tool.GetName())
					agent.config.Tools = append(agent.config.Tools, tool)
				}
			}

			// Check if plugin provides system prompt
			if pp, ok := plugin.(core.PromptProvider); ok {
				sp := pp.SystemPrompt()
				if sp != "" {
					if !systemPromptAdded {
						agent.config.SystemPrompt += "[PLUGIN TOOLS INSTRUCTIONS]"
					}
					agentforge.Debug("🔌 [handlePluginInitialization] Adding system prompt from plugin")
					agent.config.SystemPrompt += fmt.Sprintf("\n [%s plugin]:\n%s\n\n", plugin.Name(), sp)
					agent.ensureSystemPrompt()
					systemPromptAdded = true
				}
			}
		}

		agentforge.Debug("🔌 [handlePluginInitialization] COMPLETED")
		return nil
	}
}

// / ========= SYSTEM HOOKS ========= ///
func (a *Agent) registerSystemCallbacks() {
	systemHandlers := handlers.NewSystemHandlers()

	registerToolsHook := func(handler func(handlers.AgentOperations, []llms.Tool) error) {
		hook := OnAddedToolsHook(func(agent *Agent, tools []llms.Tool) error {
			return handler(agent, tools)
		})
		a.hooks.on(core.EventAddedTools, hook)
	}

	systemHandlers.RegisterWith(registerToolsHook)
}
