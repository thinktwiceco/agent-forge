package agents

import (
	"errors"
	"fmt"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

var handleNewSystemAgentAdded = OnAddedSystemAgentHook(func(a *Agent, subAgent core.SubAgent) error {
	if subAgent == nil {
		return errors.New("sub agent is nil")
	}

	// A new system agent will trigger a reload of the delegate tool
	a.loadDelegateTool()
	// A rebuild of the system prompt
	a.ensureSystemPrompt()
	// A rebuild of the agent context
	a.initAgentContext()

	return nil
})

var handleNewToolsAdded = OnAddedToolsHook(func(a *Agent, tools []llms.Tool) error {
	if tools == nil {
		return errors.New("tools slice is nil")
	}

	// New tools will trigger a rebuild of the system prompt
	a.ensureSystemPrompt()
	// A rebuild of the agent context
	a.initAgentContext()

	return nil
})

// var handleNewUserMessage = OnNewUserMessageHook(func(a *Agent, message string) error {
// 	// ResponseCh is already set up in setResponseCh() with hook callback
// 	// Don't recreate it here as that would lose the hook callback
// 	// Ensure the history is loaded
// 	a.ensureHistory()
// 	// Add the user message to the history if not already present
// 	a.handleSystemPromptInjection()
// 	// Add the user message to the history
// 	a.history.addUserMessage(message)
// 	// Save the history
// 	a.history.save()
// 	// Get the history
// 	a.history.get()
// 	return nil
// })

// var handleNewAssistantMessage = OnNewAssistantMessageHook(func(a *Agent, message string, promptTokens, completionTokens, totalTokens int) error {
// 	// Add the assistant message to the history
// 	a.history.addAssistantMessage(message, promptTokens, completionTokens, totalTokens)
// 	// Save the history
// 	a.history.save()
// 	return nil
// })

var handlePluginInitialization = OnAgentInitializationHook(func(a *Agent, config *AgentConfig) error {
	agentforge.Debug("🔌 [handlePluginInitialization] START for agent %s", a.config.AgentName)
	agentforge.Debug("🔌 [handlePluginInitialization] Plugins count: %d", len(a.config.Plugins))

	if len(a.config.Plugins) == 0 {
		agentforge.Debug("🔌 [handlePluginInitialization] No plugins to register")
		return nil
	}

	systemPromptAdded := false

	// Register all the plugins
	for i, plugin := range a.config.Plugins {
		agentforge.Debug("🔌 [handlePluginInitialization] Processing plugin %d: %s", i+1, plugin.Name())
		// Register the events
		for _, event := range core.Events {
			hook := plugin.On(event)
			if hook != nil {
				agentforge.Debug("🔌 [handlePluginInitialization] Registering hook for event: %s", event)
				a.hooks.on(event, hook)
			}
		}
		// Register the tools
		tools := plugin.Tools()
		for _, tool := range tools {
			agentforge.Debug("🔌 [handlePluginInitialization] Registering tool: %s", tool.GetName())
			a.config.Tools = append(a.config.Tools, tool)
		}

		// Update the system prompt
		sp := plugin.SystemPrompt()
		if sp != "" {
			if !systemPromptAdded {
				a.config.SystemPrompt += "[PLUGIN TOOLS INSTRUCTIONS]"
			}
			agentforge.Debug("🔌 [handlePluginInitialization] Adding system prompt from plugin")
			a.config.SystemPrompt += fmt.Sprintf("\n [%s plugin]:\n%s\n\n", plugin.Name(), sp)
			a.ensureSystemPrompt()
			systemPromptAdded = true
		}
	}

	agentforge.Debug("🔌 [handlePluginInitialization] COMPLETED")
	return nil
})
