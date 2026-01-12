package agents

import (
	"fmt"

	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/tools/delegate"
	"github.com/thinktwice/agentForge/src/tools/expand"
	"github.com/thinktwice/agentForge/src/tools/meta"
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
		// Convert ExtendedChunkResponse to ChunkResponse for hooks
		chunk := &llms.ChunkResponse{
			Content:          extendedChunk.Content,
			Delta:            extendedChunk.Delta,
			FullContent:      extendedChunk.FullContent,
			Status:           extendedChunk.Status,
			Type:             extendedChunk.Type,
			ToolCalls:        extendedChunk.ToolCalls,
			ToolExecuting:    extendedChunk.ToolExecuting,
			ToolResults:      extendedChunk.ToolResults,
			PromptTokens:     extendedChunk.PromptTokens,
			CompletionTokens: extendedChunk.CompletionTokens,
			TotalTokens:      extendedChunk.TotalTokens,
		}
		// Store agent name/trace in logger plugin if available
		// We need to find the logger plugin from config
		for _, plugin := range a.config.Plugins {
			if lp, ok := plugin.(interface{ SetCurrentChunkInfo(string, string) }); ok {
				lp.SetCurrentChunkInfo(extendedChunk.AgentName, extendedChunk.Trace)
				break
			}
		}
		// Trigger hooks
		errs := a.hooks.newChunkEvent(a, chunk)
		for _, err := range errs {
			if err != nil {
				return err
			}
		}
		return nil
	}
	a.responseCh = core.NewResponseCh(a.config.AgentName, a.config.Trace, onChunkRead)
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

func (a *Agent) initResponseCh() {
	// This method is not used - setResponseCh() is used instead
	// Keeping for backward compatibility if needed
	a.responseCh = core.NewResponseCh(a.Name(), a.Trace(), nil)
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

	dt := delegate.NewDelegateTool(a.subAgents)
	a.tools = append(a.tools, dt)
}

// initAgentContext builds the agent context struct with static fields
// that don't change during the agent's lifetime.
func (a *Agent) initAgentContext() {
	a.agentContext = &core.AgentContext{
		AgentName: a.Name(),
		Trace:     a.Trace(),
		Model:     fmt.Sprintf("%s-%s", a.config.LLMEngine.Provider(), a.config.LLMEngine.Model()),
		Tools:     a.tools,
		SubAgents: a.subAgents,
	}
}

// addSystemAgents adds system agents based on configs
func (a *Agent) addSystemAgents() {
	var systemAgents []*core.SubAgent

	if a.config.Reasoning {
		// Create reasoning agent from template
		// Check if a specific engine is configured for this sub agent
		var engineForReasoning llms.LLMEngine
		if a.config.ExtraEngines != nil {
			if engine, ok := a.config.ExtraEngines[AgentNameSystemReasoning]; ok && engine != nil {
				engineForReasoning = engine
			} else {
				engineForReasoning = a.config.LLMEngine
			}
		} else {
			engineForReasoning = a.config.LLMEngine
		}
		raAsSubAgent := ReasoningAgent(engineForReasoning)
		systemAgents = append(systemAgents, raAsSubAgent)
	}

	// Append system agents to subagents
	a.subAgents = append(a.subAgents, systemAgents...)
}

// / ========= SYSTEM HOOKS ========= ///
func (a *Agent) registerSystemCallbacks() {
	a.hooks.on(core.EventNewUserMessage, handleNewUserMessage)
	a.hooks.on(core.EventAddedSystemAgent, handleNewSystemAgentAdded)
	a.hooks.on(core.EventAddedTools, handleNewToolsAdded)
	a.hooks.on(core.EventNewAssistantMessage, handleNewAssistantMessage)
	a.hooks.on(core.EventNewAssistantMessageWithToolCalls, handleNewAssistantMessageWithToolCalls)
	a.hooks.on(core.EventAgentInitialization, handlePluginInitialization)
}

// / ========= PLUGIN HOOKS ========= ///
func (a *Agent) registerPluginCallbacks() {
	a.hooks.registerPlugins(a.config.Plugins)
}
