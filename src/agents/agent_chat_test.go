package agents

import (
	"context"
	"testing"
	"time"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/history"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/mocks"
)

// MockTool implements llms.Tool for testing
type MockTool struct {
	Name         string
	LastArgs     map[string]any
	Called       bool
	ReturnError  bool
	ReturnResult string
}

func (m *MockTool) GetName() string {
	return m.Name
}

func (m *MockTool) GetFunctionDefinition() llms.FunctionDefinition {
	return llms.FunctionDefinition{Name: m.Name}
}

// Call must match llms.Tool interface: Call(map[string]any, map[string]any) llms.ToolReturn
func (m *MockTool) Call(ctx map[string]any, args map[string]any) llms.ToolReturn {
	m.Called = true
	m.LastArgs = args
	if m.ReturnError {
		return core.NewErrorResponse(m.ReturnResult)
	}
	return core.NewSuccessResponse(m.ReturnResult)
}

func TestAgent_ExecuteTool(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	agent := NewAgent(&AgentConfig{
		AgentName: "TestAgent",
		Trace:     "test",
		LLMEngine: mockLLM,
	})

	// Create context manually
	agent.agentContext = &core.AgentContext{
		AgentName:      "TestAgent",
		Trace:          "test",
		Tools:          []llms.Tool{},
		SessionStorage: make(map[string]any),
	}

	mockTool := &MockTool{Name: "test_tool", ReturnResult: "success"}
	agent.AddTools([]llms.Tool{mockTool})

	// Test successful execution
	call := llms.ToolCall{
		ID:        "call-1",
		Name:      "test_tool",
		Arguments: map[string]any{"arg": "value"},
	}

	result := agent.executor.ExecuteTool(call, agent.responseCh)

	if !mockTool.Called {
		t.Error("Tool should have been called")
	}
	if !result.Success {
		t.Error("Result should be success")
	}
	if result.Result != "success" {
		t.Errorf("Expected result 'success', got %s", result.Result)
	}

	// Test tool not found
	callNotFound := llms.ToolCall{
		ID:   "call-2",
		Name: "unknown_tool",
	}
	resultNotFound := agent.executor.ExecuteTool(callNotFound, agent.responseCh)
	if resultNotFound.Success {
		t.Error("Should fail for unknown tool")
	}
}

func TestAgent_ChatWithTools(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	agent := NewAgent(&AgentConfig{
		AgentName:         "TestAgent",
		Trace:             "test",
		LLMEngine:         mockLLM,
		MaxToolIterations: 2,
	})

	mockTool := &MockTool{Name: "test_tool", ReturnResult: "tool_output"}
	agent.AddTools([]llms.Tool{mockTool})

	// Initialize context and response channel manually
	agent.responseCh = core.NewResponseCh("TestAgent", "test", "chat-id", nil)
	agent.agentContext = &core.AgentContext{
		AgentName:      "TestAgent",
		Trace:          "test",
		ResponseCh:     agent.responseCh,
		Tools:          agent.tools,
		SessionStorage: make(map[string]any),
	}

	// Create a history instance for this request (per-request pattern)
	hm := history.NewConversationHistory()

	// Verify channels are ready
	if agent.responseCh == nil {
		t.Fatal("ResponseCh not initialized")
	}

	// Setup LLM to return a tool call then a completion
	// 1. Tool Call Chunk via ToolCalls field in MockLLM (which sends it first)
	toolCall := llms.ToolCall{
		ID:        "call-1",
		Name:      "test_tool",
		Arguments: map[string]any{"key": "val"},
	}
	mockLLM.ToolCalls = []llms.ToolCall{toolCall}

	// 2. Mock response after tool execution
	mockLLM.Responses = []string{"Final answer after tool"}

	// Start consuming output to prevent blocking
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		for {
			select {
			case <-agent.responseCh.Start():
				// consume
			case <-ctx.Done():
				return
			}
		}
	}()

	// Trigger the chat with the history instance
	_, err := agent.executor.ExecuteChatWithTools(context.Background(), hm, agent.responseCh)

	_ = err // Ignore error in this test context as we consume output in goroutine

	// Assert tool was called
	if !mockTool.Called {
		t.Error("Tool should have been called during chat")
	}

	// Verify history contains tool result
	hist := hm.Messages()
	foundToolResult := false
	for _, msg := range hist {
		// Check for Tool role
		if msg.Role() == llms.MessageRoleTool && msg.Content() == "tool_output" {
			foundToolResult = true
			break
		}
	}
	if !foundToolResult {
		t.Error("History should contain tool result")
	}
}
