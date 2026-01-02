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

type OnAddSystemAgentHook func(a *Agent, subAgent *core.SubAgent) error

type OnAddedSystemAgentHook func(a *Agent, subAgent *core.SubAgent) error

type OnNewAssistantMessageHook func(a *Agent, message string, promptTokens, completionTokens, totalTokens int) error

type OnNewAssistantMessageWithToolCallsHook func(a *Agent, message string, toolCalls []llms.ToolCall, promptTokens, completionTokens, totalTokens int) error

type OnAddedToolsHook func(a *Agent, tools []llms.Tool) error

type AgentHooks struct {
	onContextBuild                     []OnContextBuildHook
	beforeToolExecution                []BeforeToolExecutionHook
	onToolExecution                    []OnToolExecutionHook
	onNewUserMessage                   []OnNewUserMessageHook
	onAddSystemAgent                   []OnAddSystemAgentHook
	onAddedSystemAgent                 []OnAddedSystemAgentHook
	onNewAssistantMessage              []OnNewAssistantMessageHook
	onNewAssistantMessageWithToolCalls []OnNewAssistantMessageWithToolCallsHook
	onAddedTools                       []OnAddedToolsHook
}

type HookExecutionError struct {
	message string
	hookId  string
}

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

func (ah *AgentHooks) addSystemAgentEvent(a *Agent, subAgent *core.SubAgent) []error {
	agentforge.Debug("Triggering addSystemAgentEvent for agent %s", a.Name())
	var errors []error
	for _, hook := range ah.onAddSystemAgent {
		if err := hook(a, subAgent); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (ah *AgentHooks) addedSystemAgentEvent(a *Agent, subAgent *core.SubAgent) []error {
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

type Event string

const (
	EventContextBuild                     Event = "contextBuild"
	EventBeforeToolExecution              Event = "beforeToolExecution"
	EventToolExecution                    Event = "toolExecution"
	EventNewUserMessage                   Event = "newUserMessage"
	EventAddSystemAgent                   Event = "addSystemAgent"
	EventAddedSystemAgent                 Event = "addedSystemAgent"
	EventNewAssistantMessage              Event = "newAssistantMessage"
	EventNewAssistantMessageWithToolCalls Event = "newAssistantMessageWithToolCalls"
	EventAddedTools                       Event = "addedTools"
)

func (a *Agent) on(event Event, hook any) {
	agentforge.Debug("Registering hook for event %s", event)
	agentforge.Debug("Hook: %+v", hook)
	switch event {
	case EventContextBuild:
		typedHook := hook.(OnContextBuildHook)
		a.hooks.onContextBuild = append(a.hooks.onContextBuild, typedHook)
	case EventBeforeToolExecution:
		typedHook := hook.(BeforeToolExecutionHook)
		a.hooks.beforeToolExecution = append(a.hooks.beforeToolExecution, typedHook)
	case EventToolExecution:
		typedHook := hook.(OnToolExecutionHook)
		a.hooks.onToolExecution = append(a.hooks.onToolExecution, typedHook)
	case EventNewUserMessage:
		typedHook := hook.(OnNewUserMessageHook)
		a.hooks.onNewUserMessage = append(a.hooks.onNewUserMessage, typedHook)
	case EventAddSystemAgent:
		typedHook := hook.(OnAddSystemAgentHook)
		a.hooks.onAddSystemAgent = append(a.hooks.onAddSystemAgent, typedHook)
	case EventAddedSystemAgent:
		typedHook := hook.(OnAddedSystemAgentHook)
		a.hooks.onAddedSystemAgent = append(a.hooks.onAddedSystemAgent, typedHook)
	case EventAddedTools:
		typedHook := hook.(OnAddedToolsHook)
		a.hooks.onAddedTools = append(a.hooks.onAddedTools, typedHook)
	default:
		panic(fmt.Sprintf("unknown event: %s", event))
	}
}

func NewAgentHooks() *AgentHooks {
	return &AgentHooks{
		onContextBuild:      []OnContextBuildHook{},
		beforeToolExecution: []BeforeToolExecutionHook{},
		onToolExecution:     []OnToolExecutionHook{},
		onNewUserMessage:    []OnNewUserMessageHook{},
		onAddSystemAgent:    []OnAddSystemAgentHook{},
		onAddedSystemAgent:  []OnAddedSystemAgentHook{},
		onAddedTools:        []OnAddedToolsHook{},
	}
}
