# Agent Mocks

This package provides mock implementations of the agent's internal interfaces for testing purposes.

## Available Mocks

### MockHistoryManager
Mock implementation of `history.Manager` for testing conversation history operations.

```go
mock := &mocks.MockHistoryManager{
    MessagesFunc: func() []*llms.UnifiedMessage {
        return []*llms.UnifiedMessage{
            llms.UserMessage("test message"),
        }
    },
    SaveFunc: func() string {
        return "test-chat-id"
    },
}
```

### MockPromptBuilder
Mock implementation of `agents.PromptBuilder` for testing prompt construction.

```go
mock := &mocks.MockPromptBuilder{
    BuildFunc: func() string {
        return "custom test prompt"
    },
}
```

### MockExecutionEngine
Mock implementation of `agents.ExecutionEngine` for testing tool execution.

```go
mock := &mocks.MockExecutionEngine{
    ExecuteToolFunc: func(toolCall llms.ToolCall, responseCh *core.ResponseCh) llms.ToolResult {
        return llms.ToolResult{
            ToolCallID: toolCall.ID,
            ToolName:   toolCall.Name,
            Success:    true,
            Result:     "custom test result",
        }
    },
}
```

### MockContextManager
Mock implementation of `agents.ContextManager` for testing context operations.

```go
mock := &mocks.MockContextManager{
    ContextFunc: func() *core.AgentContext {
        return &core.AgentContext{
            AgentName: "test-agent",
            Tools:     []interface{}{},
        }
    },
}
```

## Usage in Tests

Mocks allow you to test agent components in isolation by replacing real dependencies with controlled implementations:

```go
func TestAgentComponent(t *testing.T) {
    // Create mock dependencies
    mockHistory := &mocks.MockHistoryManager{
        MessagesFunc: func() []*llms.UnifiedMessage {
            return []*llms.UnifiedMessage{}
        },
    }
    
    mockExecutor := &mocks.MockExecutionEngine{
        ExecuteChatWithToolsFunc: func(ctx context.Context, hm history.Manager, responseCh *core.ResponseCh) error {
            // Simulate tool execution
            return nil
        },
    }
    
    // Use mocks in your test
    // ...
}
```

## Benefits

- **Isolated Testing**: Test components without external dependencies
- **Controlled Behavior**: Define exact behavior for each test case
- **Fast Execution**: No real LLM calls or database operations
- **Predictable Results**: Consistent test outcomes
- **Easy Debugging**: Simple implementations reveal issues quickly
