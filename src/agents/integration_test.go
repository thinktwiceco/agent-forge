//go:build integration

package agents

import (
	"context"
	"fmt"
	"testing"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/tools"
)

// hasAPIKey checks if an API key is available for the given provider.
func hasAPIKey(provider string) bool {
	config, err := agentforge.NewConfig()
	if err != nil {
		return false
	}

	switch provider {
	case "togetherai":
		return config.AFTogetherAIAPIKey != ""
	case "deepseek":
		return config.AFDeepSeekAPIKey != ""
	case "openai":
		return config.AFOpenAIAPIKey != ""
	case "openrouter":
		return config.APOpenRouterAPIKey != ""
	default:
		return false
	}
}

// TestAgent_fooTool_WithRealLLM tests the agent with automatic tool execution using TogetherAI.
func TestAgent_fooTool_WithRealLLM(t *testing.T) {
	// Skip if API key is not available
	if !hasAPIKey("togetherai") {
		t.Skip("Skipping test - TogetherAI API key not available")
	}

	// Use TogetherAI's model
	multiModelLLM, err := llms.NewOpenAILLMBuilder("togetherai").
		SetModel(llms.TOGETHERAI_Llama3170BInstructTurbo).
		Build()
	if err != nil {
		t.Skipf("Skipping real LLM test - TogetherAI API not available: %v", err)
	}
	llm := multiModelLLM.MainModel()

	// Create agent with the LLM and foo tool
	agent := NewAgent(&AgentConfig{
		LLMEngine: llm,
		AgentName: "test agent",
		Trace:     "testing",
		Tools:     []llms.Tool{tools.NewFooTool()},
	})

	responseCh := agent.ChatStream(context.Background(), `Use the foo tool to echo back exactly "Hello, test!"`, "")

	var (
		sawToolCall      bool
		sawToolExecuting bool
		sawToolResult    bool
		sawFinalContent  bool
		toolCallID       string
		executingToolID  string
		resultToolCallID string
		resultData       string
	)

	fmt.Println("\n=== Streaming Response ===")
	// No casting needed - Start() returns the concrete channel type
	for chunk := range responseCh.Start() {
		if chunk.Status == llms.StatusToolCall && len(chunk.ToolCalls) > 0 {
			sawToolCall = true
			toolCallID = chunk.ToolCalls[0].ID
			fmt.Printf("  ✓ Tool call: name=%s, args=%+v, id=%s\n",
				chunk.ToolCalls[0].Name,
				chunk.ToolCalls[0].Arguments,
				chunk.ToolCalls[0].ID)

			// Verify tool call structure
			if chunk.ToolCalls[0].Name != "foo" {
				t.Errorf("Expected tool name 'foo', got '%s'", chunk.ToolCalls[0].Name)
			}
		}

		if chunk.Status == llms.StatusToolExecuting && chunk.ToolExecuting != nil {
			sawToolExecuting = true
			executingToolID = chunk.ToolExecuting.ID
			fmt.Printf("  ✓ Executing tool: %s (ID: %s)\n", chunk.ToolExecuting.Name, chunk.ToolExecuting.ID)

			// Verify executing tool matches the tool call
			if chunk.ToolExecuting.Name != "foo" {
				t.Errorf("Expected executing tool name 'foo', got '%s'", chunk.ToolExecuting.Name)
			}
		}

		if chunk.Status == llms.StatusToolResult && len(chunk.ToolResults) > 0 {
			sawToolResult = true
			resultToolCallID = chunk.ToolResults[0].ToolCallID
			resultData = chunk.ToolResults[0].Result
			fmt.Printf("  ✓ Tool result: success=%v, data=%s\n",
				chunk.ToolResults[0].Success,
				chunk.ToolResults[0].Result)

			// Verify tool result
			if !chunk.ToolResults[0].Success {
				t.Errorf("Expected tool to succeed, but it failed: %s", chunk.ToolResults[0].Error)
			}
			if chunk.ToolResults[0].ToolName != "foo" {
				t.Errorf("Expected tool name 'foo', got '%s'", chunk.ToolResults[0].ToolName)
			}
		}

		if chunk.Status == llms.StatusStreaming && chunk.Content != "" {
			// Check for any final content (not just specific text)
			fmt.Print(chunk.Content)
		}

		if chunk.Status == llms.StatusCompleted {
			fmt.Println("  ✓ Completed")
			// Expect the final content to be the same as the accumulatd content
		}
	}

	// Verify streaming works (minimum requirement)
	if !sawFinalContent && !sawToolCall && !sawToolExecuting && !sawToolResult {
		t.Error("Expected to see either content streaming or tool execution")
	}

	// Check for tool execution (optional - depends on model's function calling support)
	if sawToolCall && sawToolExecuting && sawToolResult {
		fmt.Println("\n✅ Model properly supports OpenAI function calling!")

		// Verify tool call IDs match across chunks
		if toolCallID != "" && executingToolID != "" && toolCallID != executingToolID {
			t.Errorf("Tool call ID mismatch: tool-call=%s, executing=%s", toolCallID, executingToolID)
		}
		if toolCallID != "" && resultToolCallID != "" && toolCallID != resultToolCallID {
			t.Errorf("Tool call ID mismatch: tool-call=%s, result=%s", toolCallID, resultToolCallID)
		}
	} else if !sawToolCall {
		fmt.Println("\nℹ️  Model did not use OpenAI function calling format.")
		fmt.Println("   This is expected for some TogetherAI models (e.g., GPTOSS120B).")
		fmt.Println("   The model may output tool calls as text instead.")
		fmt.Println("   For proper function calling, use DeepSeek or OpenAI GPT-4.")
	}

	// Log summary
	fmt.Println("\n=== Test Summary ===")
	fmt.Printf("✓ Saw tool call: %v\n", sawToolCall)
	fmt.Printf("✓ Saw tool executing: %v\n", sawToolExecuting)
	fmt.Printf("✓ Saw tool result: %v\n", sawToolResult)
	fmt.Printf("✓ Saw final content: %v\n", sawFinalContent)
	if resultData != "" {
		fmt.Printf("✓ Tool result data: %s\n", resultData)
	}
}
