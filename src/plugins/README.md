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
3. Plugin self-registers via `registry.Register(name, factory)` in its `init()` function. The factory receives `workingDir` and returns a `core.Plugin`.
4. [allplugins.go](../builder/allplugins.go) imports each plugin with a blank import to trigger `init()` when the builder loads.
5. Config lists plugin names (e.g. `plugins: ["todo", "vault"]`). The builder fetches plugins from the registry and instantiates them with the agent's working directory.
6. During `EventAgentInitialization`, the agent registers hooks, tools, and prompts from each plugin.

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

### Best Practices

- **Folder-based plugins**: If a plugin operates inside a folder (e.g. `vault/`, `procedures/`), it should auto-create that folder at initialization time (e.g. in `EventAgentInitialized` or when the plugin is constructed). This avoids errors when the agent first uses the plugin.
- **Paths are relative to working_dir**: All plugin folder paths are relative to the agent's `working_dir`. The factory receives `workingDir` and should use it as the base (e.g. `filepath.Join(workingDir, "vault")`).

## Available Plugins

- [Logger](./logger/README.md) - Configurable output formatting for agent responses
- [Todo](./todo/README.md) - Task management and todo list functionality for agents
- [Procedures](./procedures/plugin.go) - Structured multi-phase procedures from `working_dir/procedures/`
- [Vault](./vault/plugin.go) - Encrypted secret storage in `working_dir/vault/` with `saveSecret`, `listSecrets`, and `resolveSecret` for tools