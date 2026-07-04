// ─── Plugin Capability Interfaces ────────────────────────────────────────────
//
// Plugins are opt-in capability bundles. The agent layer discovers what a
// plugin can contribute via interface assertion during EventAgentInitialization
// (see agents/agentInit.go: createPluginInitializationHandler).
//
// Interface segregation keeps the agent decoupled from any specific plugin:
//
//	Plugin          — base: every plugin must have a name
//	HookProvider    — opt-in: receive agent lifecycle events
//	ToolProvider    — opt-in: expose additional tools to the LLM
//	PromptProvider  — opt-in: append instructions to the system prompt
//	WorkingDirAware — opt-in: receive the agent's working directory on init
//	InboxAware      — opt-in: receive the agent's inbox queue so the plugin
//	                  can inject autonomous messages (e.g. scheduler, webhooks)
//
// Adding a new capability = add a new interface here + assert it in
// createPluginInitializationHandler. No other agent code needs to change.

package core

import (
	"context"

	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/queue"
)

// Identifier provides minimal identification for an agent or component.
// Use this when only the name is needed (e.g., lookup, display).
type Identifier interface {
	Name() string
}

// Executable represents an agent that can execute chat requests.
type Executable interface {
	ChatStream(ctx context.Context, message string, chatId string) *ResponseCh
}

// maybeEphemeralToolCall is a tool call that may be ephemeral.
// It is used to represent a tool call that may be ephemeral.
// This is to avoid cluttering the history with too many tool call messages.
type MaybeEphemeralToolCall interface {
	Ephemeral() bool
	ToolCall() *llms.ToolResult
}

type AgentHookFn any

// Plugin is the base interface that all plugins must implement.
// It provides the basic identification for a plugin.
type Plugin interface {
	Name() string
}

// HookProvider is an optional interface that plugins can implement
// to provide event hooks. Plugins only need to implement this if they
// want to respond to agent lifecycle events.
type HookProvider interface {
	Plugin
	Hooks() map[Event]AgentHookFn
}

// ToolProvider is an optional interface that plugins can implement
// to provide tools. Plugins only need to implement this if they
// want to add tools to the agent.
type ToolProvider interface {
	Plugin
	Tools() []llms.Tool
}

// PromptProvider is an optional interface that plugins can implement
// to provide system prompt additions. Plugins only need to implement
// this if they want to add instructions to the system prompt.
type PromptProvider interface {
	Plugin
	SystemPrompt() string
}

// WorkingDirAware is an optional interface for plugins and tools
// that need to know the agent's working directory.
type WorkingDirAware interface {
	SetWorkingDir(dir string)
}

// InboxAware is an optional interface for plugins that need a reference
// to the agent's turn inbox so they can inject messages autonomously
// (e.g. scheduled tasks, heartbeat).
type InboxAware interface {
	SetInbox(q queue.Inbox)
}

// LLMEngineAware is an optional interface for plugins that need direct LLM
// access for background tasks. The engine is injected during
// EventAgentInitialization alongside WorkingDir and Inbox.
// The canonical use-case is the brain plugin's DreamingRunner, which makes
// its own LLM calls to distil conversation notes without involving the main agent.
type LLMEngineAware interface {
	SetLLMEngine(engine llms.LLMEngine)
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
	EventNewAssistantMessage              Event = "newAssistantMessage"
	EventNewAssistantMessageWithToolCalls Event = "newAssistantMessageWithToolCalls"
	EventAddedTools                       Event = "addedTools"
	EventNewChunk                         Event = "newChunk"
	// EventChatStart fires at the beginning of every ChatStream call, after the
	// chatId is known and before the executor runs. It carries the conversation ID,
	// which is pre-generated for new conversations so that plugins (e.g. brain) can
	// initialise per-conversation resources (files, graph nodes) before any tool calls.
	EventChatStart Event = "chatStart"
)

var Events = []Event{
	EventAgentInitialization,
	EventAgentInitialized,
	EventContextBuild,
	EventBeforeToolExecution,
	EventToolExecution,
	EventNewUserMessage,
	EventNewAssistantMessage,
	EventNewAssistantMessageWithToolCalls,
	EventAddedTools,
	EventNewChunk,
	EventChatStart,
}
