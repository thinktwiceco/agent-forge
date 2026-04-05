package agents_test

import (
	"context"
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/agents/execution"
	"github.com/thinktwiceco/agent-forge/src/agents/mocks"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/history"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// ExampleHistoryManager demonstrates using the HistoryManager interface.
func ExampleHistoryManager() {
	// Using the real implementation
	hm := history.NewConversationHistory()

	hm.AddUserMessage("Hello, world!")
	hm.AddAssistantMessage("Hi there!", history.TokenUsage{
		PromptTokens:     10,
		CompletionTokens: 5,
		TotalTokens:      15,
	})

	messages := hm.Messages()
	fmt.Printf("Conversation has %d messages\n", len(messages))
	// Output: Conversation has 2 messages
}

// ExampleMockHistoryManager demonstrates using a mock for testing.
func ExampleMockHistoryManager() {
	// Create a mock with controlled behavior
	mock := &mocks.MockHistoryManager{
		MessagesFunc: func() []*llms.UnifiedMessage {
			return []*llms.UnifiedMessage{
				llms.UserMessage("test message"),
			}
		},
		ChatIdFunc: func() string {
			return "test-chat-123"
		},
	}

	// Use the mock in tests
	messages := mock.Messages()
	chatId := mock.ChatId()

	fmt.Printf("Mock has %d messages, chatId: %s\n", len(messages), chatId)
	// Output: Mock has 1 messages, chatId: test-chat-123
}

// ExamplePromptBuilder demonstrates using the PromptBuilder interface.
func ExamplePromptBuilder() {
	// Using a mock for testing
	mock := &mocks.MockPromptBuilder{
		BuildFunc: func() string {
			return "Custom system prompt for testing"
		},
	}

	prompt := mock.Build()
	fmt.Printf("Prompt: %s\n", prompt)
	// Output: Prompt: Custom system prompt for testing
}

// ExampleExecutionEngine demonstrates using the ExecutionEngine interface.
func ExampleExecutionEngine() {
	// Using a mock for testing
	mock := &mocks.MockExecutionEngine{
		ExecuteToolFunc: func(toolCall llms.ToolCall, responseCh *core.ResponseCh) llms.ToolResult {
			return llms.ToolResult{
				ToolCallID: toolCall.ID,
				ToolName:   toolCall.Name,
				Success:    true,
				Result:     "Tool executed successfully",
			}
		},
	}

	// Simulate tool execution
	result := mock.ExecuteTool(llms.ToolCall{
		ID:   "call-123",
		Name: "test-tool",
	}, nil)

	fmt.Printf("Tool result: %s\n", result.Result)
	// Output: Tool result: Tool executed successfully
}

// ExampleContextManager demonstrates using the ContextManager interface.
func ExampleContextManager() {
	// Using a mock for testing
	mock := &mocks.MockContextManager{
		ContextFunc: func() *core.AgentContext {
			return &core.AgentContext{
				AgentName: "test-agent",
				Model:     "gpt-4",
				Tools:     []llms.Tool{},
			}
		},
	}

	ctx := mock.Context()
	fmt.Printf("Agent: %s, Model: %s\n", ctx.AgentName, ctx.Model)
	// Output: Agent: test-agent, Model: gpt-4
}

// Example_interfaceBasedTesting demonstrates how interfaces enable better testing.
func Example_interfaceBasedTesting() {
	// This example shows how using interfaces allows you to test components
	// in isolation without needing real LLM engines or databases.

	// Create a mock execution engine that simulates successful tool execution
	mockExecutor := &mocks.MockExecutionEngine{
		ExecuteChatWithToolsFunc: func(ctx context.Context, hm history.Manager, responseCh *core.ResponseCh) (execution.ExecuteResult, error) {
			// Simulate a simple response without actual LLM call
			hm.AddAssistantMessage("Test response", history.TokenUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			})
			return execution.ExecuteResult{}, nil
		},
	}

	// Create a mock history manager
	messages := []*llms.UnifiedMessage{}
	mockHistory := &mocks.MockHistoryManager{
		AddAssistantMessageFunc: func(msg string, tokens history.TokenUsage) {
			messages = append(messages, llms.AssistantMessage(msg, tokens.PromptTokens, tokens.CompletionTokens, tokens.TotalTokens))
		},
		MessagesFunc: func() []*llms.UnifiedMessage {
			return messages
		},
	}

	// Test the executor with mock dependencies
	ctx := context.Background()
	_, err := mockExecutor.ExecuteChatWithTools(ctx, mockHistory, nil)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Test completed, messages: %d\n", len(mockHistory.Messages()))
	// Output: Test completed, messages: 1
}

// Example_interfaceComposition demonstrates how interfaces can be composed.
func Example_interfaceComposition() {
	// Interfaces allow you to create custom compositions for specific use cases

	// Custom interface combining multiple capabilities
	//nolint:unused // Demonstrates interface composition pattern
	type AgentComponents interface {
		GetExecutor() agents.ExecutionEngine
		GetPromptBuilder() agents.PromptBuilder
		GetContextManager() agents.ContextManager
	}

	// This pattern allows for flexible component swapping and testing
	fmt.Println("Interfaces enable flexible component composition")
	// Output: Interfaces enable flexible component composition
}
