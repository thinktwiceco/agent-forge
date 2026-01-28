# Plugins

## Table of Contents

- [How Plugins Work](#how-plugins-work)
- [Plugin Interface](#plugin-interface)
- [Available Plugins](#available-plugins)

## How Plugins Work

Plugins extend agent functionality by hooking into lifecycle events and providing tools. They are registered during agent initialization and automatically integrate with the agent's hook system.

### Registration Flow

1. Plugin implements `core.Plugin` interface
2. Agent calls `plugin.On(event)` for each lifecycle event
3. Non-nil hooks are registered with the agent
4. Tools from `plugin.Tools()` are added during initialization

### Lifecycle Events

Plugins can hook into:
- `EventAgentInitialization` - Before agent initialization
- `EventAgentInitialized` - After agent initialization
- `EventContextBuild` - When building agent context
- `EventBeforeToolExecution` - Before tool execution
- `EventToolExecution` - After tool execution
- `EventNewUserMessage` - New user message received
- `EventNewChunk` - New streaming chunk created
- `EventNewAssistantMessage` - New assistant message
- `EventNewAssistantMessageWithToolCalls` - Assistant message with tool calls
- `EventAddSystemAgent` - Before adding system agent
- `EventAddedSystemAgent` - After adding system agent
- `EventAddedTools` - When tools are added

## Plugin Interface

```go
type Plugin interface {
    Name() string
    On(event Event) AgentHookFn
    Tools() []llms.Tool
    SystemPrompt() string
}
```

- **Name()**: Returns unique plugin identifier
- **On()**: Returns hook function for event, or `nil` if not handled
- **Tools()**: Returns list of tools provided to agents
- **SystemPrompt()**: Returns system prompt instructions that are automatically appended to the agent's system prompt (empty string if no prompt needed)

## Available Plugins

- [Logger](./logger/README.md) - Configurable output formatting for agent responses
- [Todo](./todo/README.md) - Task management and todo list functionality for agents