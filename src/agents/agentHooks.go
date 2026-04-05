package agents

import (
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// BASE HOOK INTERFACE //
// HookId is a unique identifier for the hook
// Critical is a flag to indicate if the hook is critical to the execution of the agent
// If a critical hook fails, the execution of the agent should be aborted

type OnContextBuildHook func(a *Agent, agentContext *core.AgentContext) error

type BeforeToolExecutionHook func(a *Agent, toolCall *llms.ToolCall) error

type OnToolExecutionHook func(a *Agent, toolResult *llms.ToolResult) error

type OnNewUserMessageHook func(a *Agent, message string) error

type OnNewAssistantMessageHook func(a *Agent, message string, promptTokens, completionTokens, totalTokens int) error

type OnNewAssistantMessageWithToolCallsHook func(a *Agent, message string, toolCalls []llms.ToolCall, promptTokens, completionTokens, totalTokens int) error

type OnAddedToolsHook func(a *Agent, tools []llms.Tool) error

type OnAgentInitializationHook func(a *Agent, config *AgentConfig) error

type OnAgentInitializedHook func(a *Agent) error

type OnNewChunkHook func(a *Agent, chunk *core.ExtendedChunkResponse) error

// OnChatStartHook is called at the beginning of every ChatStream call with the
// resolved chatId. For new conversations the chatId is pre-generated before this
// hook fires, so plugins can safely create per-conversation resources immediately.
type OnChatStartHook func(a *Agent, chatId string) error

type AgentHooks struct {
	onAgentInitialization              []OnAgentInitializationHook
	onAgentInitialized                 []OnAgentInitializedHook
	onContextBuild                     []OnContextBuildHook
	beforeToolExecution                []BeforeToolExecutionHook
	onToolExecution                    []OnToolExecutionHook
	onNewUserMessage                   []OnNewUserMessageHook
	onNewAssistantMessage              []OnNewAssistantMessageHook
	onNewAssistantMessageWithToolCalls []OnNewAssistantMessageWithToolCallsHook
	onAddedTools                       []OnAddedToolsHook
	onNewChunk                         []OnNewChunkHook
	onChatStart                        []OnChatStartHook
}

// HOOK TRIGGERS //

// runHooks is a generic helper to execute a list of hooks and collect their errors.
func runHooks[T any](hooks []T, caller func(T) error) []error {
	var errors []error
	for _, hook := range hooks {
		if err := caller(hook); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (ah *AgentHooks) contextBuildEvent(a *Agent, agentContext *core.AgentContext) []error {
	agentforge.Debug("Triggering contextBuildEvent for agent %s", a.Name())
	return runHooks(ah.onContextBuild, func(hook OnContextBuildHook) error {
		return hook(a, agentContext)
	})
}

func (ah *AgentHooks) beforeToolExecutionEvent(a *Agent, toolCall *llms.ToolCall) []error {
	agentforge.Debug("Triggering beforeToolExecutionEvent for agent %s", a.Name())
	return runHooks(ah.beforeToolExecution, func(hook BeforeToolExecutionHook) error {
		return hook(a, toolCall)
	})
}

func (ah *AgentHooks) toolExecutionEvent(a *Agent, toolResult *llms.ToolResult) []error {
	agentforge.Debug("Triggering toolExecutionEvent for agent %s", a.Name())
	return runHooks(ah.onToolExecution, func(hook OnToolExecutionHook) error {
		return hook(a, toolResult)
	})
}

func (ah *AgentHooks) newUserMessageEvent(a *Agent, message string) []error {
	agentforge.Debug("Triggering newUserMessageEvent for agent %s", a.Name())
	return runHooks(ah.onNewUserMessage, func(hook OnNewUserMessageHook) error {
		return hook(a, message)
	})
}

func (ah *AgentHooks) newAssistantMessageEvent(a *Agent, message string, promptTokens, completionTokens, totalTokens int) []error {
	agentforge.Debug("Triggering newAssistantMessageEvent for agent %s", a.Name())
	return runHooks(ah.onNewAssistantMessage, func(hook OnNewAssistantMessageHook) error {
		return hook(a, message, promptTokens, completionTokens, totalTokens)
	})
}

func (ah *AgentHooks) newAssistantMessageWithToolCallsEvent(a *Agent, message string, toolCalls []llms.ToolCall, promptTokens, completionTokens, totalTokens int) []error {
	agentforge.Debug("Triggering newAssistantMessageWithToolCallsEvent for agent %s", a.Name())
	return runHooks(ah.onNewAssistantMessageWithToolCalls, func(hook OnNewAssistantMessageWithToolCallsHook) error {
		return hook(a, message, toolCalls, promptTokens, completionTokens, totalTokens)
	})
}

func (ah *AgentHooks) addedToolsEvent(a *Agent, tools []llms.Tool) []error {
	agentforge.Debug("Triggering addedToolsEvent for agent %s", a.Name())
	return runHooks(ah.onAddedTools, func(hook OnAddedToolsHook) error {
		return hook(a, tools)
	})
}

func (ah *AgentHooks) agentInitializationEvent(a *Agent, config *AgentConfig) []error {
	agentforge.Debug("🪄 AGENT INITIALIZATION PHASE %s 🪄", a.Name())
	errs := runHooks(ah.onAgentInitialization, func(hook OnAgentInitializationHook) error {
		err := hook(a, config)
		if err == nil {
			agentforge.Debug("\t✅ Hook executed successfully")
		}
		return err
	})
	agentforge.Debug("🪄 AGENT INITIALIZATION PHASE COMPLETED 🪄")
	return errs
}

func (ah *AgentHooks) agentInitializedEvent(a *Agent) []error {
	agentforge.Debug("Triggering agentInitializedEvent for agent %s", a.Name())
	return runHooks(ah.onAgentInitialized, func(hook OnAgentInitializedHook) error {
		agentforge.Debug("Triggering agentInitializedEvent hook: %+v", hook)
		return hook(a)
	})
}

func (ah *AgentHooks) newChunkEvent(a *Agent, chunk *core.ExtendedChunkResponse) []error {
	return runHooks(ah.onNewChunk, func(hook OnNewChunkHook) error {
		return hook(a, chunk)
	})
}

// chatStartEvent fires EventChatStart handlers with the resolved conversation ID.
// Errors are collected and returned; non-critical hooks should not abort the turn.
func (ah *AgentHooks) chatStartEvent(a *Agent, chatId string) []error {
	agentforge.Debug("Triggering chatStartEvent for agent %s (chatId=%s)", a.Name(), chatId)
	return runHooks(ah.onChatStart, func(hook OnChatStartHook) error {
		return hook(a, chatId)
	})
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
		if typedHook, ok := hook.(OnAgentInitializationHook); ok {
			ah.onAgentInitialization = append(ah.onAgentInitialization, typedHook)
		} else {
			agentforge.Error("Invalid hook type for EventAgentInitialization")
		}
	case core.EventAgentInitialized:
		if typedHook, ok := hook.(OnAgentInitializedHook); ok {
			ah.onAgentInitialized = append(ah.onAgentInitialized, typedHook)
		} else {
			agentforge.Error("Invalid hook type for EventAgentInitialized")
		}
	case core.EventContextBuild:
		if typedHook, ok := hook.(OnContextBuildHook); ok {
			ah.onContextBuild = append(ah.onContextBuild, typedHook)
		} else {
			agentforge.Error("Invalid hook type for EventContextBuild")
		}
	case core.EventBeforeToolExecution:
		if typedHook, ok := hook.(BeforeToolExecutionHook); ok {
			ah.beforeToolExecution = append(ah.beforeToolExecution, typedHook)
		} else {
			agentforge.Error("Invalid hook type for EventBeforeToolExecution")
		}
	case core.EventToolExecution:
		if typedHook, ok := hook.(OnToolExecutionHook); ok {
			ah.onToolExecution = append(ah.onToolExecution, typedHook)
		} else {
			agentforge.Error("Invalid hook type for EventToolExecution")
		}
	case core.EventNewUserMessage:
		if typedHook, ok := hook.(OnNewUserMessageHook); ok {
			ah.onNewUserMessage = append(ah.onNewUserMessage, typedHook)
		} else {
			agentforge.Error("Invalid hook type for EventNewUserMessage")
		}
	case core.EventNewAssistantMessage:
		if typedHook, ok := hook.(OnNewAssistantMessageHook); ok {
			ah.onNewAssistantMessage = append(ah.onNewAssistantMessage, typedHook)
		} else {
			agentforge.Error("Invalid hook type for EventNewAssistantMessage")
		}
	case core.EventNewAssistantMessageWithToolCalls:
		if typedHook, ok := hook.(OnNewAssistantMessageWithToolCallsHook); ok {
			ah.onNewAssistantMessageWithToolCalls = append(ah.onNewAssistantMessageWithToolCalls, typedHook)
		} else {
			agentforge.Error("Invalid hook type for EventNewAssistantMessageWithToolCalls")
		}
	case core.EventAddedTools:
		if typedHook, ok := hook.(OnAddedToolsHook); ok {
			ah.onAddedTools = append(ah.onAddedTools, typedHook)
		} else {
			agentforge.Error("Invalid hook type for EventAddedTools")
		}
	case core.EventNewChunk:
		if typedHook, ok := hook.(OnNewChunkHook); ok {
			ah.onNewChunk = append(ah.onNewChunk, typedHook)
		} else {
			agentforge.Error("Invalid hook type for EventNewChunk")
		}
	case core.EventChatStart:
		if typedHook, ok := hook.(OnChatStartHook); ok {
			ah.onChatStart = append(ah.onChatStart, typedHook)
		} else {
			agentforge.Error("Invalid hook type for EventChatStart")
		}
	default:
		agentforge.Error("unknown event: %s", event)
	}
}

func newAgentHooks() *AgentHooks {
	return &AgentHooks{
		onContextBuild:                     []OnContextBuildHook{},
		beforeToolExecution:                []BeforeToolExecutionHook{},
		onToolExecution:                    []OnToolExecutionHook{},
		onNewUserMessage:                   []OnNewUserMessageHook{},
		onNewAssistantMessage:              []OnNewAssistantMessageHook{},
		onNewAssistantMessageWithToolCalls: []OnNewAssistantMessageWithToolCallsHook{},
		onAddedTools:                       []OnAddedToolsHook{},
		onNewChunk:                         []OnNewChunkHook{},
		onChatStart:                        []OnChatStartHook{},
	}
}
