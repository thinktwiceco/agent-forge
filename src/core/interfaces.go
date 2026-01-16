package core

import (
	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/llms"
)

// SubAgent represents an agent that can be used as a sub-agent
// for delegation. This interface defines the minimal contract
// that any agent must satisfy to participate in delegation.
type SubAgent interface {
	agentforge.Discoverable
	// ChatStream initiates a streaming chat interaction with the agent
	// Returns a ResponseCh that can be used to consume streaming responses
	ChatStream(message string) *ResponseCh

	// Name returns the unique identifier of the agent
	Name() string

	// BasicDescription returns a short one-line description of the agent
	BasicDescription() string

	// AdvanceDescription returns detailed information about the agent's
	// capabilities, tools, sub-agents, and usage patterns
	AdvanceDescription() string

	// Troubleshooting returns information about common issues, debugging tips,
	// and configuration guidance for this agent
	Troubleshooting() string
}

// maybeEphemeralToolCall is a tool call that may be ephemeral.
// It is used to represent a tool call that may be ephemeral.
// This is to avoid cluttering the history with too many tool call messages.
type MaybeEphemeralToolCall interface {
	Ephemeral() bool
	ToolCall() *llms.ToolResult
}

type AgentHookFn any

type Plugin interface {
	Name() string
	On(event Event) AgentHookFn
	Tools() []llms.Tool
	SystemPrompt() string
}

// Event represents an agent lifecycle event
type Event string

const (
	EventAgentInitialization              Event = "agentInitialization"
	EventAgentInitialized                 Event = "agentInitialized"
	EventContextBuild                     Event = "contextBuild"
	EventBeforeToolExecution              Event = "beforeToolExecution"
	EventToolExecution                    Event = "toolExecution"
	EventNewUserMessage                   Event = "newUserMessage"
	EventAddSystemAgent                   Event = "addSystemAgent"
	EventAddedSystemAgent                 Event = "addedSystemAgent"
	EventNewAssistantMessage              Event = "newAssistantMessage"
	EventNewAssistantMessageWithToolCalls Event = "newAssistantMessageWithToolCalls"
	EventAddedTools                       Event = "addedTools"
	EventNewChunk                         Event = "newChunk"
)

var Events = []Event{
	EventAgentInitialization,
	EventAgentInitialized,
	EventContextBuild,
	EventBeforeToolExecution,
	EventToolExecution,
	EventNewUserMessage,
	EventAddSystemAgent,
	EventAddedSystemAgent,
	EventNewAssistantMessage,
	EventNewAssistantMessageWithToolCalls,
	EventAddedTools,
	EventNewChunk,
}
