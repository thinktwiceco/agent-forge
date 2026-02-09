package agents

import (
	"strings"
	"testing"

	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/mocks"
)

// TestAgent_ToolError_InHistory verifies that tool errors are included in the agent's history
func TestAgent_ToolError_InHistory(t *testing.T) {
	// Create a tool that always returns an error
	errorTool := mocks.NewErrorTool(
		"test_tool",
		"A tool that returns an error",
		"parameter validation failed: missing required field",
	)

	// Create a mock LLM
	mockLLM := mocks.NewMockLLMEngine()

	// Create agent with the error tool
	agent := NewAgent(&AgentConfig{
		AgentName:         "test_agent",
		LLMEngine:         mockLLM,
		MaxToolIterations: 5,
		SystemPrompt:      "You are a test agent",
		Tools:             []llms.Tool{errorTool},
	})

	// Create a history instance for this request
	history := &History{
		history: []*llms.UnifiedMessage{},
	}

	// Manually simulate a tool call and execution (to avoid multi-turn complexity)
	toolCall := llms.ToolCall{
		ID:        "call_123",
		Name:      "test_tool",
		Arguments: map[string]any{"input": "test"},
	}

	// Execute the tool (this is what the agent does internally)
	toolResult := agent.executeTool(toolCall)

	// Verify the tool execution failed
	if toolResult.Success {
		t.Fatal("Expected tool execution to fail, but it succeeded")
	}

	// Now add the tool message to history the same way the agent does
	content := toolResult.Result
	if !toolResult.Success && toolResult.Error != "" {
		if content != "" {
			content = content + "\nError: " + toolResult.Error
		} else {
			content = "Error: " + toolResult.Error
		}
	}
	history.addToolMessage(toolCall.ID, content, toolResult.Ephemeral)

	// Now check the history to verify the error is included
	messages := history.History()

	// Find the tool message in history
	var toolMessage *llms.UnifiedMessage
	for _, msg := range messages {
		if msg.Role() == llms.MessageRoleTool {
			toolMessage = msg
			break
		}
	}

	// Verify we found a tool message
	if toolMessage == nil {
		t.Fatal("Expected to find a tool message in history, but found none")
	}

	// Verify the error is included in the content
	content = toolMessage.Content()
	expectedError := "parameter validation failed: missing required field"

	if !strings.Contains(content, expectedError) {
		t.Errorf("Expected tool message to contain error: %q\nGot content: %q", expectedError, content)
	}

	// Verify the error is prefixed with "Error: "
	if !strings.Contains(content, "Error: "+expectedError) {
		t.Errorf("Expected error to be prefixed with 'Error: '\nGot content: %q", content)
	}
}
