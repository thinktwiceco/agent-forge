package agents

import (
	"errors"

	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

var handleNewSystemAgentAdded = OnAddedSystemAgentHook(func(a *Agent, subAgent *core.SubAgent) error {
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

var handleNewUserMessage = OnNewUserMessageHook(func(a *Agent, message string) error {
	// ResponseCh is already set up in setResponseCh() with hook callback
	// Don't recreate it here as that would lose the hook callback
	// Ensure the history is loaded
	a.ensureHistory()
	// Add the user message to the history if not already present
	a.handleSystemPromptInjection()
	// Add the user message to the history
	a.history.addUserMessage(message)
	// Save the history
	a.history.save()
	// Get the history
	a.history.get()
	return nil
})

var handleNewAssistantMessage = OnNewAssistantMessageHook(func(a *Agent, message string, promptTokens, completionTokens, totalTokens int) error {
	// Add the assistant message to the history
	a.history.addAssistantMessage(message, promptTokens, completionTokens, totalTokens)
	// Save the history
	a.history.save()
	return nil
})

var handleNewAssistantMessageWithToolCalls = OnNewAssistantMessageWithToolCallsHook(func(a *Agent, message string, toolCalls []llms.ToolCall, promptTokens, completionTokens, totalTokens int) error {
	// Add the assistant message with tool calls to the history
	a.history.addAssistantMessageWithToolCalls(message, toolCalls, promptTokens, completionTokens, totalTokens)
	// Save the history
	a.history.save()
	return nil
})

var handlePluginInitialization = OnAgentInitializationHook(func(a *Agent, config *AgentConfig) error {
	// Register all the plugins
	for _, plugin := range a.config.Plugins {
		for _, event := range core.Events {
			hook := plugin.On(event)
			if hook != nil {
				a.hooks.on(event, hook)
			}
		}
	}
	return nil
})
