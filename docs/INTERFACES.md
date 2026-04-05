# Agent Interfaces

This document describes the internal interfaces used in the agent system.

## Overview

The agent system uses well-defined interfaces to:
- **Decouple components**: Each major component has a clear contract
- **Enable testing**: Mock implementations allow isolated unit tests
- **Support flexibility**: Swap implementations without changing consumers
- **Document APIs**: Interfaces serve as clear contracts between modules

## Core Interfaces

### HistoryManager

Manages conversation history lifecycle including persistence and message management.

```go
type HistoryManager interface {
    AddUserMessage(msg string)
    AddSystemMessage(msg string)
    AddAssistantMessage(msg string, tokens TokenUsage)
    AddAssistantMessageWithToolCalls(content string, reasoningContent string, toolCalls []llms.ToolCall, tokens TokenUsage)
    AddToolMessage(toolCallID, result string, ephemeral bool)
    Messages() []*llms.UnifiedMessage
    SetMessages(messages []*llms.UnifiedMessage)
    ChatId() string
    Save() string
    Load(chatId string) error
}
```

**Implementation**: `history.ConversationHistory`  
**Mock**: `mocks.MockHistoryManager`

**Responsibilities**:
- Adding messages (user, assistant, system, tool)
- Managing chat sessions
- Persisting and loading history
- Token tracking for completions

### PromptBuilder

Constructs system prompts from configuration.

```go
type PromptBuilder interface {
    Build() string
    UpdateConfig(config interface{})
}
```

**Implementation**: `prompts.Builder`  
**Mock**: `mocks.MockPromptBuilder`

**Responsibilities**:
- Building the base system prompt
- Adding concise tool-use guidance when tools are configured
- Applying tone and style configurations
- Normalizing prompt formatting

### ExecutionEngine

Handles chat execution and tool invocation.

```go
type ExecutionEngine interface {
    ExecuteChatWithTools(ctx context.Context, hm history.Manager, responseCh *core.ResponseCh) error
    ExecuteTool(toolCall llms.ToolCall, responseCh *core.ResponseCh) llms.ToolResult
    UpdateTools(tools []llms.Tool)
    UpdateAgentContext(agentContext *core.AgentContext)
}
```

**Implementation**: `execution.Executor`  
**Mock**: `mocks.MockExecutionEngine`

**Responsibilities**:
- Managing the chat loop with automatic tool execution
- Streaming responses from the LLM
- Executing tool calls
- Managing iteration limits
- Triggering execution hooks

### ContextManager

Manages agent context lifecycle and operations.

```go
type ContextManager interface {
    // Context returns the current AgentContext
    Context() *core.AgentContext
    
    // BuildContext converts the AgentContext to a map for tool execution
    BuildContext(responseCh *core.ResponseCh) map[string]any
    
    // SyncFromMap syncs mutable fields from the context map back to AgentContext
    SyncFromMap(contextMap map[string]any) error
    
    // TruncateHistory applies truncation to message history if configured
    TruncateHistory(messages []*llms.UnifiedMessage) []*llms.UnifiedMessage
    
    // UpdateConfig updates the manager configuration and rebuilds context
    UpdateConfig(config interface{})
    
    // UpdateTools updates just the tools without rebuilding entire config
    UpdateTools(tools []llms.Tool)
}
```

**Implementation**: `context.Manager`  
**Mock**: `mocks.MockContextManager`

**Responsibilities**:
- Creating and maintaining the AgentContext
- Building context maps for tool execution
- Syncing changes back after tool execution
- Truncating message history (if configured with token counter and strategy)
- Updating context when tools change
- Managing session storage across requests
- Managing plugin fields
- Preserving context state during updates

### HookRegistry

Manages agent lifecycle hooks. This is an **internal** interface with unexported methods — plugins interact with it indirectly via `core.HookProvider.Hooks()`.

**Implementation**: `AgentHooks`

**Responsibilities**:
- Event registration (type-safe via typed hook constructors like `OnToolExecutionHook`)
- Event triggering at lifecycle points
- Error collection from hooks

### Executable (core package)

`core.Executable` is implemented by `*agents.Agent` for streaming chat. Use `core.Identifier` (`Name()`) when only identification is needed.

## Usage Patterns

### Testing with Mocks

```go
func TestMyComponent(t *testing.T) {
    // Create mock dependencies
    mockHistory := &mocks.MockHistoryManager{
        MessagesFunc: func() []*llms.UnifiedMessage {
            return []*llms.UnifiedMessage{
                llms.UserMessage("test"),
            }
        },
    }
    
    mockExecutor := &mocks.MockExecutionEngine{
        ExecuteToolFunc: func(toolCall llms.ToolCall, responseCh *core.ResponseCh) llms.ToolResult {
            return llms.ToolResult{Success: true, Result: "ok"}
        },
    }
    
    // Test your component with controlled behavior
    // ...
}
```

### Swapping Implementations

```go
type Agent struct {
    executor      ExecutionEngine  // Interface, not concrete type
    promptBuilder PromptBuilder    // Interface, not concrete type
    contextMgr    ContextManager   // Interface, not concrete type
}

// Easy to swap implementations
agent.executor = newCustomExecutor()
agent.promptBuilder = newCustomPromptBuilder()
```

### Interface Composition

```go
// Create custom interfaces combining multiple capabilities
type AgentCore interface {
    ExecutionEngine
    PromptBuilder
}

// Use for specific scenarios
func ProcessWithCore(core AgentCore) {
    prompt := core.Build()
    // ... use core.ExecuteChatWithTools()
}
```

## Benefits

### Loose Coupling
Components depend on interfaces, not concrete implementations. This reduces ripple effects when changing implementations.

### Testability
Mock implementations allow testing in isolation without:
- Real LLM API calls
- Database connections
- External dependencies

### Flexibility
Swap implementations for different scenarios:
- Production vs. test environments
- Different storage backends
- Alternative execution strategies

### Documentation
Interfaces serve as contracts, documenting:
- What methods are available
- What parameters are required
- What behavior is expected

## Design Principles

1. **Single Responsibility**: Each interface has one clear purpose
2. **Interface Segregation**: Interfaces are focused, not bloated
3. **Dependency Inversion**: Depend on abstractions, not concretions
4. **Explicit Contracts**: Method signatures are clear and well-documented

## Compile-Time Safety

Interface implementations are verified at compile time:

```go
// Ensures ConversationHistory implements HistoryManager
var _ HistoryManager = (*history.ConversationHistory)(nil)
```

If an implementation doesn't satisfy its interface, compilation fails immediately.

## Adding New Interfaces

When adding a new interface:

1. **Define the interface** in `interfaces.go` with clear documentation
2. **Create the implementation** in the appropriate package
3. **Add compile-time assertion** to verify implementation
4. **Create a mock** in `mocks/` for testing
5. **Add examples** to `interfaces_example_test.go`
6. **Update this documentation**

## Related Documentation

- [Testing Guide](agents/testing.md) - How to test with interfaces
- [Architecture](agents/architecture.md) - Overall system design
- [Mocks](../src/agents/mocks/README.md) - Mock implementations
