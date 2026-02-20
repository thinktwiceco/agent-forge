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
	"github.com/thinktwiceco/agent-forge/src/tools/delegate"
	"github.com/thinktwiceco/agent-forge/src/tools/expand"
	"github.com/thinktwiceco/agent-forge/src/tools/meta"
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
	a.subAgents = a.config.SubAgents

	// Initialize tools from config (if provided)
	if a.config.Tools != nil {
		a.tools = make([]llms.Tool, len(a.config.Tools))
		copy(a.tools, a.config.Tools)
	}

	a.systemPrompt = a.config.SystemPrompt

	// Initialize context window management fields
	a.maxContextTokens = a.config.MaxContextTokens
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

	// Expand Tool - add only if can expand
	if a.config.CanExpand {
		et := expand.NewExpandTool()
		a.tools = append(a.tools, et)
	}
}

// initResponseCh initializes the response channel.
// This method is not used - setResponseCh() is used instead.
// Keeping for backward compatibility if needed.
//
//nolint:unused // Reserved for backward compatibility
func (a *Agent) initResponseCh() {
	a.responseCh = core.NewResponseCh(a.Name(), a.Trace(), "", nil)
	a.responseCh.Start()
}

func (a *Agent) loadDelegateTool() {
	if len(a.subAgents) == 0 {
		return
	}

	// Look if a delegate tool already exists
	for _, tool := range a.tools {
		if tool.GetName() == delegate.DELEGATE_TOOL {
			a.tools = a.removeDelegateTool(a.tools)
		}
	}

	dt := delegate.NewDelegateTool(a.subAgents, a.inbox)
	a.tools = append(a.tools, dt)
}

// initAgentContext rebuilds the agent context and syncs all components.
// Called when tools or sub-agents change (e.g. AddTools, AddSystemAgent).
func (a *Agent) initAgentContext() {
	a.contextMgr.UpdateConfig(agentctx.Config{
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
	a.executor.UpdateAgentContext(a.agentContext)
	a.executor.UpdateTools(a.tools)
}

// ensureSystemPrompt rebuilds the system prompt and updates promptBuilder.
// Called when tools or sub-agents change.
func (a *Agent) ensureSystemPrompt() {
	a.promptBuilder.UpdateConfig(prompts.Config{
		SystemPrompt: a.config.SystemPrompt,
		MainAgent:    a.config.MainAgent,
		Tone:         a.config.Tone,
		Tools:        a.tools,
		SubAgents:    a.subAgents,
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
	return execution.NewExecutor(
		a.llmEngine,
		a.tools,
		a.agentContext,
		execution.Config{
			MaxToolIterations: a.config.MaxToolIterations,
			AgentName:         a.config.AgentName,
			Tracer:            a.config.Tracer,
		},
		hooks,
	)
}

func (a *Agent) addSystemAgents() {
	var systemAgents []core.SubAgent

	if a.config.Reasoning {
		// Create reasoning agent from template
		raAsSubAgent := ReasoningAgent(a.config.LLMEngine)
		systemAgents = append(systemAgents, raAsSubAgent)
	}

	// Append system agents to subagents
	a.subAgents = append(a.subAgents, systemAgents...)
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
	// Create system handlers and register them with the hook system
	systemHandlers := handlers.NewSystemHandlers()

	// Create adapter functions that bridge between the hook system and the interface-based handlers
	registerSystemAgentHook := func(handler func(handlers.AgentOperations, core.SubAgent) error) {
		// Create a hook with the correct signature for the hook system
		hook := OnAddedSystemAgentHook(func(agent *Agent, subAgent core.SubAgent) error {
			return handler(agent, subAgent)
		})
		a.hooks.on(core.EventAddedSystemAgent, hook)
	}

	registerToolsHook := func(handler func(handlers.AgentOperations, []llms.Tool) error) {
		// Create a hook with the correct signature for the hook system
		hook := OnAddedToolsHook(func(agent *Agent, tools []llms.Tool) error {
			return handler(agent, tools)
		})
		a.hooks.on(core.EventAddedTools, hook)
	}

	systemHandlers.RegisterWith(registerSystemAgentHook, registerToolsHook)
}
