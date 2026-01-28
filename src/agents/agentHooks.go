package agents

import (
	"fmt"

	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// type OnContextBuildHook func(agentContext *core.AgentContext) error
// type BeforeToolExecutionHook func(toolCall llms.ToolCall) error
// type OnToolExecutionHook func(toolCall llms.ToolCall) error

// BASE HOOK INTERFACE //
// HookId is a unique identifier for the hook
// Critical is a flag to indicate if the hook is critical to the execution of the agent
// If a critical hook fails, the execution of the agent should be aborted

type OnContextBuildHook func(a *Agent, agentContext *core.AgentContext) error

type BeforeToolExecutionHook func(a *Agent, toolCall *llms.ToolCall) error

type OnToolExecutionHook func(a *Agent, toolResult *llms.ToolResult) error

type OnNewUserMessageHook func(a *Agent, message string) error

type OnAddSystemAgentHook func(a *Agent, subAgent core.SubAgent) error

type OnAddedSystemAgentHook func(a *Agent, subAgent core.SubAgent) error

type OnNewAssistantMessageHook func(a *Agent, message string, promptTokens, completionTokens, totalTokens int) error

type OnNewAssistantMessageWithToolCallsHook func(a *Agent, message string, toolCalls []llms.ToolCall, promptTokens, completionTokens, totalTokens int) error

type OnAddedToolsHook func(a *Agent, tools []llms.Tool) error

type OnAgentInitializationHook func(a *Agent, config *AgentConfig) error

type OnAgentInitializedHook func(a *Agent) error

type OnNewChunkHook func(a *Agent, chunk *core.ExtendedChunkResponse) error

type AgentHooks struct {
	onAgentInitialization              []OnAgentInitializationHook
	onAgentInitialized                 []OnAgentInitializedHook
	onContextBuild                     []OnContextBuildHook
	beforeToolExecution                []BeforeToolExecutionHook
	onToolExecution                    []OnToolExecutionHook
	onNewUserMessage                   []OnNewUserMessageHook
	onAddSystemAgent                   []OnAddSystemAgentHook
	onAddedSystemAgent                 []OnAddedSystemAgentHook
	onNewAssistantMessage              []OnNewAssistantMessageHook
	onNewAssistantMessageWithToolCalls []OnNewAssistantMessageWithToolCallsHook
	onAddedTools                       []OnAddedToolsHook
	onNewChunk                         []OnNewChunkHook
}

// Hook Handle plugins
// func (ah *AgentHooks) registerPlugins(plugins []core.Plugin) {
// 	agentforge.Info("🔌 Starting Plugin Registration")
// 	for _, plugin := range plugins {
// 		agentforge.Info("♻️ Registering plugin: %s....", plugin.Name())
// 		for _, event := range core.Events {
// 			hook := plugin.On(event)
// 			if hook != nil {
// 				ah.on(event, hook)
// 				agentforge.Info("\t ✅ Hook registered for event: %s", event)
// 			}
// 		}

// 		tools := plugin.Tools()

// 		if len(tools) > 0 {
// 			agentforge.Info("\t 🧰 Registering %d tools for plugin: %s", len(tools), plugin.Name())
// 			// Will add a hook to add the tools to the agent
// 			// Before agent initialization. Init this way the tools
// 			// will be included in the agent context and system prompt.
// 			toolRegistrationHook := OnAgentInitializationHook(func(sa *Agent, config *AgentConfig) error {
// 				sa.tools = append(sa.tools, tools...)
// 				return nil
// 			})

// 			ah.on(core.EventAgentInitialization, toolRegistrationHook)
// 		}

// 		// Check if the plugin has a system prompt
// 		sp := plugin.SystemPrompt()

// 		if sp != "" {
// 			agentforge.Info("\t 📝 Registering system prompt for plugin: %s", plugin.Name())
// 			systemPromptRegistrationHook := OnAgentInitializedHook(func(sa *Agent) error {
// 				sa.systemPrompt += sp
// 				return nil
// 			})

// 			ah.on(core.EventAgentInitialized, systemPromptRegistrationHook)
// 		}
// 		agentforge.Info("🎉 Plugin registration completed\n")
// 	}
// }

// HOOK TRIGGERS //

func (ah *AgentHooks) contextBuildEvent(a *Agent, agentContext *core.AgentContext) []error {
	agentforge.Debug("Triggering contextBuildEvent for agent %s", a.Name())
	var errors []error
	for _, hook := range ah.onContextBuild {
		if err := hook(a, agentContext); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (ah *AgentHooks) beforeToolExecutionEvent(a *Agent, toolCall *llms.ToolCall) []error {
	agentforge.Debug("Triggering beforeToolExecutionEvent for agent %s", a.Name())
	var errors []error
	for _, hook := range ah.beforeToolExecution {
		if err := hook(a, toolCall); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (ah *AgentHooks) toolExecutionEvent(a *Agent, toolResult *llms.ToolResult) []error {
	agentforge.Debug("Triggering toolExecutionEvent for agent %s", a.Name())
	var errors []error
	for _, hook := range ah.onToolExecution {
		if err := hook(a, toolResult); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (ah *AgentHooks) addSystemAgentEvent(a *Agent, subAgent core.SubAgent) []error {
	agentforge.Debug("Triggering addSystemAgentEvent for agent %s", a.Name())
	var errors []error
	for _, hook := range ah.onAddSystemAgent {
		if err := hook(a, subAgent); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (ah *AgentHooks) addedSystemAgentEvent(a *Agent, subAgent core.SubAgent) []error {
	agentforge.Debug("Triggering addedSystemAgentEvent for agent %s", a.Name())
	var errors []error
	for _, hook := range ah.onAddedSystemAgent {
		if err := hook(a, subAgent); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (ah *AgentHooks) newUserMessageEvent(a *Agent, message string) []error {
	agentforge.Debug("Triggering newUserMessageEvent for agent %s", a.Name())
	var errors []error
	for _, hook := range ah.onNewUserMessage {
		if err := hook(a, message); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (ah *AgentHooks) newAssistantMessageEvent(a *Agent, message string, promptTokens, completionTokens, totalTokens int) []error {
	agentforge.Debug("Triggering newAssistantMessageEvent for agent %s", a.Name())
	var errors []error
	for _, hook := range ah.onNewAssistantMessage {
		if err := hook(a, message, promptTokens, completionTokens, totalTokens); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (ah *AgentHooks) newAssistantMessageWithToolCallsEvent(a *Agent, message string, toolCalls []llms.ToolCall, promptTokens, completionTokens, totalTokens int) []error {
	agentforge.Debug("Triggering newAssistantMessageWithToolCallsEvent for agent %s", a.Name())
	var errors []error
	for _, hook := range ah.onNewAssistantMessageWithToolCalls {
		if err := hook(a, message, toolCalls, promptTokens, completionTokens, totalTokens); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (ah *AgentHooks) addedToolsEvent(a *Agent, tools []llms.Tool) []error {
	agentforge.Debug("Triggering addedToolsEvent for agent %s", a.Name())
	var errors []error
	for _, hook := range ah.onAddedTools {
		if err := hook(a, tools); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (ah *AgentHooks) agentInitializationEvent(a *Agent, config *AgentConfig) []error {
	agentforge.Debug("🪄 AGENT INITIALIZATION PHASE %s 🪄", a.Name())
	var errors []error
	for _, hook := range ah.onAgentInitialization {
		if err := hook(a, config); err != nil {
			errors = append(errors, err)
		}
		agentforge.Debug("\t✅ Hook executed successfully")
	}
	agentforge.Debug("🪄 AGENT INITIALIZATION PHASE COMPLETED 🪄")
	return errors
}

func (ah *AgentHooks) agentInitializedEvent(a *Agent) []error {
	agentforge.Debug("Triggering agentInitializedEvent for agent %s", a.Name())
	var errors []error
	for _, hook := range ah.onAgentInitialized {
		agentforge.Debug("Triggering agentInitializedEvent hook: %+v", hook)
		if err := hook(a); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (ah *AgentHooks) newChunkEvent(a *Agent, chunk *core.ExtendedChunkResponse) []error {
	var errors []error
	for _, hook := range ah.onNewChunk {
		if err := hook(a, chunk); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// Utility function to log all the hooks errors

func logHookErrors(errors []error) {
	if len(errors) == 0 {
		return
	}

	for idx, err := range errors {
		agentforge.Debug("Error in hook %d: %s", idx, err.Error())
	}
}

// HOOK REGISTRATION METHODS //

func (ah *AgentHooks) on(event core.Event, hook any) {
	agentforge.Debug("Registering hook for event %s", event)
	agentforge.Debug("Hook: %+v", hook)
	switch event {
	case core.EventAgentInitialization:
		typedHook := hook.(OnAgentInitializationHook)
		ah.onAgentInitialization = append(ah.onAgentInitialization, typedHook)
	case core.EventAgentInitialized:
		typedHook := hook.(OnAgentInitializedHook)
		ah.onAgentInitialized = append(ah.onAgentInitialized, typedHook)
	case core.EventContextBuild:
		typedHook := hook.(OnContextBuildHook)
		ah.onContextBuild = append(ah.onContextBuild, typedHook)
	case core.EventBeforeToolExecution:
		typedHook := hook.(BeforeToolExecutionHook)
		ah.beforeToolExecution = append(ah.beforeToolExecution, typedHook)
	case core.EventToolExecution:
		typedHook := hook.(OnToolExecutionHook)
		ah.onToolExecution = append(ah.onToolExecution, typedHook)
	case core.EventNewUserMessage:
		typedHook := hook.(OnNewUserMessageHook)
		ah.onNewUserMessage = append(ah.onNewUserMessage, typedHook)
	case core.EventAddSystemAgent:
		typedHook := hook.(OnAddSystemAgentHook)
		ah.onAddSystemAgent = append(ah.onAddSystemAgent, typedHook)
	case core.EventAddedSystemAgent:
		typedHook := hook.(OnAddedSystemAgentHook)
		ah.onAddedSystemAgent = append(ah.onAddedSystemAgent, typedHook)
	case core.EventNewAssistantMessage:
		typedHook := hook.(OnNewAssistantMessageHook)
		ah.onNewAssistantMessage = append(ah.onNewAssistantMessage, typedHook)
	case core.EventNewAssistantMessageWithToolCalls:
		typedHook := hook.(OnNewAssistantMessageWithToolCallsHook)
		ah.onNewAssistantMessageWithToolCalls = append(ah.onNewAssistantMessageWithToolCalls, typedHook)
	case core.EventAddedTools:
		typedHook := hook.(OnAddedToolsHook)
		ah.onAddedTools = append(ah.onAddedTools, typedHook)
	case core.EventNewChunk:
		typedHook := hook.(OnNewChunkHook)
		ah.onNewChunk = append(ah.onNewChunk, typedHook)
	default:
		panic(fmt.Sprintf("unknown event: %s", event))
	}
}

func newAgentHooks() *AgentHooks {
	return &AgentHooks{
		onContextBuild:                     []OnContextBuildHook{},
		beforeToolExecution:                []BeforeToolExecutionHook{},
		onToolExecution:                    []OnToolExecutionHook{},
		onNewUserMessage:                   []OnNewUserMessageHook{},
		onAddSystemAgent:                   []OnAddSystemAgentHook{},
		onAddedSystemAgent:                 []OnAddedSystemAgentHook{},
		onNewAssistantMessage:              []OnNewAssistantMessageHook{},
		onNewAssistantMessageWithToolCalls: []OnNewAssistantMessageWithToolCallsHook{},
		onAddedTools:                       []OnAddedToolsHook{},
		onNewChunk:                         []OnNewChunkHook{},
	}
}
