package core

import (
	"context"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// Identifier provides minimal identification for an agent or component.
// Use this when only the name is needed (e.g., lookup, display).
type Identifier interface {
	Name() string
}

// Executable represents an agent that can execute chat requests.
// Use this when only execution is needed (e.g., delegation).
type Executable interface {
	ChatStream(ctx context.Context, message string, chatId string) *ResponseCh
}

// SubAgent represents an agent that can be used as a sub-agent
// for delegation. This interface composes smaller focused interfaces:
// - Identifier: for lookup and display
// - Discoverable: for progressive discovery (descriptions, troubleshooting)
// - Executable: for chat execution
//
// Use SubAgent when the full contract is needed. Prefer Identifier,
// agentforge.Discoverable, or Executable where only a subset is required.
type SubAgent interface {
	Identifier
	agentforge.Discoverable
	Executable
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

// LegacyPlugin wraps old-style plugins for backward compatibility.
// This adapter converts the old On(event) pattern to the new Hooks() map pattern.
type LegacyPlugin struct {
	plugin interface {
		Plugin
		On(event Event) AgentHookFn
		Tools() []llms.Tool
		SystemPrompt() string
	}
}

// NewLegacyPlugin creates a backward-compatibility adapter for old-style plugins.
func NewLegacyPlugin(plugin interface {
	Plugin
	On(event Event) AgentHookFn
	Tools() []llms.Tool
	SystemPrompt() string
}) *LegacyPlugin {
	return &LegacyPlugin{plugin: plugin}
}

// Name implements Plugin interface
func (lp *LegacyPlugin) Name() string {
	return lp.plugin.Name()
}

// Hooks implements HookProvider interface by adapting On() to Hooks()
func (lp *LegacyPlugin) Hooks() map[Event]AgentHookFn {
	hooks := make(map[Event]AgentHookFn)
	for _, event := range Events {
		hook := lp.plugin.On(event)
		if hook != nil {
			hooks[event] = hook
		}
	}
	return hooks
}

// Tools implements ToolProvider interface
func (lp *LegacyPlugin) Tools() []llms.Tool {
	return lp.plugin.Tools()
}

// SystemPrompt implements PromptProvider interface
func (lp *LegacyPlugin) SystemPrompt() string {
	return lp.plugin.SystemPrompt()
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
