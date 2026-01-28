package agents

import (
	"fmt"

	agentforge "github.com/thinktwice/agentForge/src"
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

	dt := delegate.NewDelegateTool(a.subAgents)
	a.tools = append(a.tools, dt)
}

// initAgentContext builds the agent context struct with static fields
// that don't change during the agent's lifetime.
// Preserves SessionStorage and PluginFields if they already exist.
func (a *Agent) initAgentContext() {
	// Preserve existing SessionStorage when reinitializing, or create new if doesn't exist
	var existingSessionStorage map[string]any
	if a.agentContext != nil && a.agentContext.SessionStorage != nil {
		existingSessionStorage = a.agentContext.SessionStorage
	} else {
		// Always initialize SessionStorage, never leave it nil
		existingSessionStorage = make(map[string]any)
	}

	// Preserve existing PluginFields when reinitializing, or create new if doesn't exist
	var existingPluginFields map[string]any
	if a.agentContext != nil && a.agentContext.PluginFields != nil {
		existingPluginFields = a.agentContext.PluginFields
	} else {
		// Always initialize PluginFields, never leave it nil
		existingPluginFields = make(map[string]any)
	}

	a.agentContext = &core.AgentContext{
		AgentName:      a.Name(),
		Trace:          a.Trace(),
		Model:          fmt.Sprintf("%s-%s", a.config.LLMEngine.Provider(), a.config.LLMEngine.Model()),
		Tools:          a.tools,
		SubAgents:      a.subAgents,
		SessionStorage: existingSessionStorage, // Always a valid map, never nil
		PluginFields:   existingPluginFields,   // Always a valid map, never nil
	}
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
	a.hooks.on(core.EventAgentInitialization, handlePluginInitialization)
	agentforge.Debug("🔌 Plugin registration hook added successfully 🔌")
}

// / ========= SYSTEM HOOKS ========= ///
func (a *Agent) registerSystemCallbacks() {
	// a.hooks.on(core.EventNewUserMessage, handleNewUserMessage)
	a.hooks.on(core.EventAddedSystemAgent, handleNewSystemAgentAdded)
	a.hooks.on(core.EventAddedTools, handleNewToolsAdded)
	// a.hooks.on(core.EventNewAssistantMessage, handleNewAssistantMessage)
}
