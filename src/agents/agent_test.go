package agents

import (
	"fmt"
	"strings"
	"testing"

	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/tools"
)

// TestAgent_fooTool_WithRealLLM tests the agent with automatic tool execution using TogetherAI.
// This test uses TogetherAI's GPT-OSS-120B model.
//
// Note: While TogetherAI uses the OpenAI SDK, not all models properly implement
// the OpenAI function calling API format. Some models output tool calls as text.
// This test will verify streaming works and check if tool execution happens.
//
// For guaranteed tool execution testing, use a model like:
// - OpenAI GPT-5 (requires OpenAI API key)
//
// Set TOGETHERAI_API_KEY environment variable to run this test.
func TestAgent_fooTool_WithRealLLM(t *testing.T) {
	// Use TogetherAI's model
	llm, err := llms.NewOpenAILLMBuilder("togetherai").
		SetModel(llms.TOGETHERAI_Llama3170BInstructTurbo).
		Build()
	if err != nil {
		t.Skipf("Skipping real LLM test - TogetherAI API not available: %v", err)
	}

	// Create agent with the LLM and foo tool
	agent := NewAgent(&AgentConfig{
		LLMEngine: llm,
		AgentName: "test agent",
		Trace:     "testing",
		Tools:     []llms.Tool{tools.NewFooTool()},
	})

	responseCh := agent.ChatStream(`Use the foo tool to echo back exactly "Hello, test!"`)

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
	if !sawFinalContent && !(sawToolCall || sawToolExecuting || sawToolResult) {
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

// TestAgent_fooTool tests the agent with a tool using TogetherAI.
// This is a basic streaming test. For comprehensive tool execution testing with
// all chunk verification, see TestAgent_fooTool_WithRealLLM.
func TestAgent_fooTool(t *testing.T) {
	llm, err := llms.NewOpenAILLMBuilder("togetherai").
		SetModel(llms.TOGETHERAI_Llama3170BInstructTurbo).
		Build()
	if err != nil {
		t.Fatalf("failed to get together llm: %v", err)
	}
	agent := NewAgent(&AgentConfig{
		LLMEngine: llm,
		AgentName: "test agent",
		Trace:     "testing",
		Tools:     []llms.Tool{tools.NewFooTool()},
	})
	responseCh := agent.ChatStream(
		`Use the foo tool to echo back the message "Hello, world!"`,
	)

	var sawContent bool

	// No casting needed - Start() returns the concrete channel type
	for chunk := range responseCh.Start() {
		if chunk.Content != "" {
			sawContent = true
		}
	}

	// At minimum, we should see some content streaming
	if !sawContent {
		t.Error("Expected to see content chunks")
	}
}

// TestAgentConfig_validate tests the validation of AgentConfig.
func TestAgentConfig_validate(t *testing.T) {
	llm, err := llms.NewOpenAILLMBuilder("togetherai").
		SetModel(llms.TOGETHERAI_Llama3170BInstructTurbo).
		Build()
	if err != nil {
		t.Skipf("skipping validation test: failed to get together llm: %v", err)
	}

	tests := []struct {
		name    string
		config  AgentConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: AgentConfig{
				LLMEngine: llm,
				AgentName: "test agent",
			},
			wantErr: false,
		},
		{
			name: "missing LLMEngine",
			config: AgentConfig{
				LLMEngine: nil,
				AgentName: "test agent",
			},
			wantErr: true,
			errMsg:  "LLMEngine is required",
		},
		{
			name: "missing AgentName",
			config: AgentConfig{
				LLMEngine: llm,
				AgentName: "",
			},
			wantErr: true,
			errMsg:  "AgentName is required",
		},
		{
			name: "missing both required fields",
			config: AgentConfig{
				LLMEngine: nil,
				AgentName: "",
			},
			wantErr: true,
			errMsg:  "LLMEngine is required", // Should fail on first check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("validate() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validate() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestNewAgent_validation tests that NewAgent panics on invalid config.
func TestNewAgent_validation(t *testing.T) {
	llm, err := llms.NewOpenAILLMBuilder("togetherai").
		SetModel(llms.TOGETHERAI_Llama3170BInstructTurbo).
		Build()
	if err != nil {
		t.Skipf("skipping NewAgent validation test: failed to get together llm: %v", err)
	}

	tests := []struct {
		name        string
		config      AgentConfig
		shouldPanic bool
		panicMsg    string
	}{
		{
			name: "valid config does not panic",
			config: AgentConfig{
				LLMEngine: llm,
				AgentName: "test agent",
			},
			shouldPanic: false,
		},
		{
			name: "nil LLMEngine panics",
			config: AgentConfig{
				LLMEngine: nil,
				AgentName: "test agent",
			},
			shouldPanic: true,
			panicMsg:    "LLMEngine is required",
		},
		{
			name: "empty AgentName panics",
			config: AgentConfig{
				LLMEngine: llm,
				AgentName: "",
			},
			shouldPanic: true,
			panicMsg:    "AgentName is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tt.shouldPanic {
					if r == nil {
						t.Errorf("NewAgent() expected panic but did not panic")
						return
					}
					errStr := fmt.Sprintf("%v", r)
					if tt.panicMsg != "" && !strings.Contains(errStr, tt.panicMsg) {
						t.Errorf("NewAgent() panic = %v, want panic containing %q", r, tt.panicMsg)
					}
				} else {
					if r != nil {
						t.Errorf("NewAgent() unexpected panic = %v", r)
					}
				}
			}()

			_ = NewAgent(&tt.config)
		})
	}
}
