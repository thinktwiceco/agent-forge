# Plugin Structure

This document describes how plugins are built in the agentForge framework, based on the knowledge plugin implementation.

## Plugin Interface

All plugins must implement the `core.Plugin` interface:

```go
type Plugin interface {
    Name() string
    On(event Event) AgentHookFn
    Tools() []llms.Tool
}
```

### Required Methods

1. **`Name() string`**: Returns a unique identifier for the plugin
2. **`On(event Event) AgentHookFn`**: Returns a hook function for a specific event, or `nil` if the plugin doesn't handle that event
3. **`Tools() []llms.Tool`**: Returns a list of tools that the plugin provides to agents

## Plugin Structure

A plugin typically consists of:

1. **Plugin struct**: Main plugin type that implements the `Plugin` interface
2. **Constructor function**: `NewPlugin(...)` that initializes the plugin
3. **Hook handlers**: Methods that handle agent lifecycle events
4. **Tool definitions**: Functions that create tools for the agent
5. **Internal implementation**: Supporting types and functions for plugin functionality

## Example: Knowledge Plugin Structure

### 1. Plugin Struct

```go
type KnowledgePlugin struct {
    knowledgeIdentifier string
    documentPaths       []string
    dbPath              string
    vectorDB            *vectorDB
    embeddingService    *embeddingService
    qdrantDocker        *qdrantDocker
    chunkSize           int
    chunkOverlap        int
    loadCtx             context.Context
    loadCancel          context.CancelFunc
    maxWorkers          int
    batchSize           int
}
```

### 2. Constructor Function

```go
func NewPlugin(documentPaths []string, dbPath string, knowledgeIdentifier string) (*KnowledgePlugin, error) {
    // Initialize dependencies (vector DB, embedding service, etc.)
    // Set up context and signal handling
    // Return initialized plugin
}
```

### 3. Interface Implementation

#### Name() Method

```go
func (p *KnowledgePlugin) Name() string {
    return "knowledge"
}
```

#### On() Method - Event Handling

```go
func (p *KnowledgePlugin) On(event core.Event) core.AgentHookFn {
    switch event {
    case core.EventAgentInitialization:
        fn := agents.OnAgentInitializationHook(p.handleAgentInitialization)
        return fn
    case core.EventAgentInitialized:
        fn := agents.OnAgentInitializedHook(p.handleAgentInitialized)
        return fn
    }
    return nil
}
```

#### Tools() Method

```go
func (p *KnowledgePlugin) Tools() []llms.Tool {
    return []llms.Tool{NewSearchTool(p)}
}
```

### 4. Hook Handlers

Hook handlers are methods that get called during agent lifecycle events:

```go
func (p *KnowledgePlugin) handleAgentInitialization(a *agents.Agent, config *agents.AgentConfig) error {
    // Perform initialization tasks (e.g., load documents)
    return nil
}

func (p *KnowledgePlugin) handleAgentInitialized(a *agents.Agent) error {
    // Perform post-initialization tasks
    return nil
}
```

### 5. Tool Creation

Tools are created using the `core.Tool` struct:

```go
func NewSearchTool(kp *KnowledgePlugin) llms.Tool {
    return &core.Tool{
        Name:        "semantic-search-tool",
        Description: "Search for semantic information in documents",
        AdvanceDesc: "Advanced Details: ...",
        Parameters: []core.Parameter{
            {
                Name:        "query",
                Type:        "string",
                Description: "The query to search the knowledge base for",
                Required:    true,
            },
            {
                Name:        "limit",
                Type:        "int",
                Description: "The maximum number of results to return",
                Required:    false,
            },
        },
        Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
            query := args["query"].(string)
            limit := args["limit"].(int)
            // Execute tool logic
            return core.NewSuccessResponse(results)
        },
    }
}
```

## Available Events

Plugins can hook into the following agent lifecycle events:

- `EventAgentInitialization`: Called before agent initialization
- `EventAgentInitialized`: Called after agent initialization
- `EventContextBuild`: Called when building agent context
- `EventBeforeToolExecution`: Called before tool execution
- `EventToolExecution`: Called after tool execution
- `EventNewUserMessage`: Called when a new user message is received
- `EventAddSystemAgent`: Called before adding a system agent
- `EventAddedSystemAgent`: Called after adding a system agent
- `EventNewAssistantMessage`: Called when a new assistant message is created
- `EventNewAssistantMessageWithToolCalls`: Called when assistant message includes tool calls
- `EventAddedTools`: Called when tools are added to the agent

## Hook Function Types

Each event has a corresponding hook function type:

- `OnAgentInitializationHook func(a *Agent, config *AgentConfig) error`
- `OnAgentInitializedHook func(a *Agent) error`
- `OnContextBuildHook func(a *Agent, agentContext *core.AgentContext) error`
- `BeforeToolExecutionHook func(a *Agent, toolCall *llms.ToolCall) error`
- `OnToolExecutionHook func(a *Agent, toolResult *llms.ToolResult) error`
- `OnNewUserMessageHook func(a *Agent, message string) error`
- `OnAddSystemAgentHook func(a *Agent, subAgent *core.SubAgent) error`
- `OnAddedSystemAgentHook func(a *Agent, subAgent *core.SubAgent) error`
- `OnNewAssistantMessageHook func(a *Agent, message string, promptTokens, completionTokens, totalTokens int) error`
- `OnNewAssistantMessageWithToolCallsHook func(a *Agent, message string, toolCalls []llms.ToolCall, promptTokens, completionTokens, totalTokens int) error`
- `OnAddedToolsHook func(a *Agent, tools []llms.Tool) error`

## Tool Structure

Tools must implement the `llms.Tool` interface:

```go
type Tool interface {
    GetName() string
    Call(agentContext map[string]any, args map[string]any) ToolReturn
    GetFunctionDefinition() FunctionDefinition
}
```

The `core.Tool` struct provides a convenient implementation that also satisfies the `Discoverable` interface.

### Tool Parameters

Parameters are defined using `core.Parameter`:

```go
type Parameter struct {
    Name        string
    Type        string // "string", "number", "boolean", "object", "array"
    Description string
    Required    bool
    Validator   func(value any) error // Optional custom validation
}
```

### Tool Return Values

Tools return `llms.ToolReturn` values. Use helper functions:

- `core.NewSuccessResponse(data string)`: Returns a successful tool execution
- `core.NewErrorResponse(error string)`: Returns an error response

## Plugin Registration

Plugins are registered through the `AgentConfig`:

```go
config := agents.AgentConfig{
    LLMEngine:   llmEngine,
    AgentName:   "Assistant",
    Plugins:     []core.Plugin{knowledgePlugin},
    // ... other config
}

agent := agents.NewAgent(&config)
```

During agent initialization, plugins are registered:

1. For each plugin, iterate through all available events
2. Call `plugin.On(event)` for each event
3. If a hook is returned (not `nil`), register it with the agent's hook system
4. Call `plugin.Tools()` and register tools during `EventAgentInitialization`

## Tool Registration Flow

1. Plugin's `Tools()` method is called during plugin registration
2. Tools are added to a hook that runs during `EventAgentInitialization`
3. Tools are appended to the agent's tool list before initialization completes
4. Tools become available in the agent context and system prompt

## Best Practices

1. **Initialization**: Perform heavy initialization in `handleAgentInitialization` hook
2. **Error Handling**: Always return errors from hook handlers - they will be logged
3. **Resource Cleanup**: Implement cleanup methods (e.g., `Close()`) for resources
4. **Context Handling**: Use context for cancellation support in long-running operations
5. **Tool Descriptions**: Provide clear, descriptive tool names and descriptions
6. **Parameter Validation**: Use parameter validators for input validation
7. **Tool Responses**: Return meaningful success/error responses

## File Organization

A typical plugin package structure:

```
plugins/
  knowledge/
    plugin.go      # Main plugin struct and interface implementation
    tools.go       # Tool definitions
    knowledge.go   # Core plugin functionality
    vector_db.go   # Supporting implementation
    embeddings.go  # Supporting implementation
    docker.go      # Supporting implementation
    chunking.go    # Supporting implementation
```

## Example Usage

```go
// Create plugin
knowledgePlugin, err := knowledge.NewPlugin(
    []string{"/path/to/documents"},
    "/path/to/knowledge.db",
    "codebase",
)

// Configure agent with plugin
config := agents.AgentConfig{
    LLMEngine: llmEngine,
    AgentName: "Assistant",
    Plugins:   []core.Plugin{knowledgePlugin},
}

// Create agent
agent := agents.NewAgent(&config)
```

The plugin will automatically:
- Register hooks for events it handles
- Register tools during agent initialization
- Execute hook handlers at appropriate lifecycle events

