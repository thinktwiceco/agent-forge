package agents

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/mocks"
)

// TestAgent_ChatStream_Unit tests the agent ChatStream with a mock LLM.
// This verifies the agent's state machine without making real API calls.
func TestAgent_ChatStream_Unit(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	mockLLM.Responses = []string{"Hello from mock!"}

	agent := NewAgent(&AgentConfig{
		LLMEngine: mockLLM,
		AgentName: "test-agent",
		Trace:     "test-trace",
	})

	responseCh := agent.ChatStream(context.Background(), "Hello", "")

	var receivedContent string
	var sawStreaming bool
	var sawCompleted bool

	for chunk := range responseCh.Start() {
		switch chunk.Status {
		case llms.StatusStreaming:
			receivedContent += chunk.Content
			sawStreaming = true
		case llms.StatusCompleted:
			sawCompleted = true
		}
	}

	if !sawStreaming {
		t.Error("Expected to see streaming chunks")
	}
	if !sawCompleted {
		t.Error("Expected to see completed status")
	}
	if !strings.Contains(receivedContent, "Hello from mock!") {
		t.Errorf("Expected content to contain 'Hello from mock!', got '%s'", receivedContent)
	}

	// Verify LLM was called with correct message
	if len(mockLLM.RecordedCalls) == 0 {
		t.Error("Expected LLM to be called")
	} else {
		lastCall := mockLLM.RecordedCalls[0]
		if len(lastCall) == 0 {
			t.Error("Expected messages in LLM call")
		} else {
			lastMsg := lastCall[len(lastCall)-1]
			if lastMsg.Content() != "Hello" {
				t.Errorf("Expected last message content 'Hello', got '%s'", lastMsg.Content())
			}
		}
	}
}

// TestAgentConfig_validate tests the validation of AgentConfig.
func TestAgentConfig_validate(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()

	tests := []struct {
		name    string
		config  AgentConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: AgentConfig{
				LLMEngine: mockLLM,
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
				LLMEngine: mockLLM,
				AgentName: "",
			},
			wantErr: true,
			errMsg:  "AgentName is required",
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
	mockLLM := mocks.NewMockLLMEngine()

	tests := []struct {
		name        string
		config      AgentConfig
		shouldPanic bool
		panicMsg    string
	}{
		{
			name: "valid config does not panic",
			config: AgentConfig{
				LLMEngine: mockLLM,
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
				LLMEngine: mockLLM,
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
