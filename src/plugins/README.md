# Plugins

## Table of Contents

- [How Plugins Work](#how-plugins-work)
- [Plugin Interface](#plugin-interface)
- [Available Plugins](#available-plugins)

## How Plugins Work

Plugins extend agent functionality by hooking into lifecycle events, providing tools, and adding system prompt instructions. They use a composable interface design following the Interface Segregation Principle - plugins only implement the capabilities they need.

### Registration Flow

1. Plugin implements `core.Plugin` interface (required)
2. Plugin optionally implements one or more provider interfaces:
   - `HookProvider` for event hooks
   - `ToolProvider` for tools
   - `PromptProvider` for system prompt additions
3. During initialization, the agent checks which interfaces each plugin implements
4. Hooks, tools, and prompts are registered automatically based on implemented interfaces

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

Plugins use a composable interface design where you only implement what you need:

### Base Interface (Required)

```go
type Plugin interface {
    Name() string
}
```

- **Name()**: Returns unique plugin identifier

### Optional Provider Interfaces

#### HookProvider - For Event Hooks

```go
type HookProvider interface {
    Plugin
    Hooks() map[Event]AgentHookFn
}
```

- **Hooks()**: Returns a map of event hooks that the plugin provides
- Only implement this if your plugin needs to respond to agent lifecycle events

**Example:**
```go
func (p *LoggerPlugin) Hooks() map[core.Event]core.AgentHookFn {
    return map[core.Event]core.AgentHookFn{
        core.EventNewChunk: agents.OnNewChunkHook(p.handleNewChunk),
    }
}
```

#### ToolProvider - For Adding Tools

```go
type ToolProvider interface {
    Plugin
    Tools() []llms.Tool
}
```

- **Tools()**: Returns list of tools provided to agents
- Only implement this if your plugin provides tools

**Example:**
```go
func (p *TodoPlugin) Tools() []llms.Tool {
    return []llms.Tool{
        newTodoHandlerTool(p),
    }
}
```

#### PromptProvider - For System Prompts

```go
type PromptProvider interface {
    Plugin
    SystemPrompt() string
}
```

- **SystemPrompt()**: Returns system prompt instructions that are automatically appended to the agent's system prompt
- Only implement this if your plugin needs to add instructions to the system prompt

**Example:**
```go
func (p *TodoPlugin) SystemPrompt() string {
    return `Use the todo_handler tool to manage tasks...`
}
```

### Benefits

- **Interface Segregation**: Plugins only implement capabilities they actually use
- **No Empty Methods**: No need to return empty values for unused features
- **Clear Intent**: Interface implementation clearly shows plugin capabilities
- **Flexible**: Mix and match interfaces as needed

## Available Plugins

- [Logger](./logger/README.md) - Configurable output formatting for agent responses
- [Todo](./todo/README.md) - Task management and todo list functionality for agents