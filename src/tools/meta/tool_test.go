package meta

import (
	"testing"

	"github.com/thinktwiceco/agent-forge/src/llms"
)

func TestNewMetaTool(t *testing.T) {
	tool := NewMetaTool()

	if tool.GetName() != "meta" {
		t.Errorf("Expected tool name 'meta', got '%s'", tool.GetName())
	}

	def := tool.GetFunctionDefinition()
	if def.Name != "meta" {
		t.Errorf("Expected function name 'meta', got '%s'", def.Name)
	}
}

func TestMetaTool_HandleMethod(t *testing.T) {
	tool := NewMetaTool()
	context := map[string]any{
		"agentName": "test-agent",
		"model":     "test-model",
		"tools": []llms.Tool{
			NewMetaTool(),
		},
	}

	tests := []struct {
		name        string
		method      string
		wantSuccess bool
		wantContent string // Partial match
	}{
		{
			name:        "get_current_model",
			method:      "get_current_model",
			wantSuccess: true,
			wantContent: "test-model",
		},
		{
			name:        "get_agent_name",
			method:      "get_agent_name",
			wantSuccess: true,
			wantContent: "test-agent",
		},
		{
			name:        "get_tools",
			method:      "get_tools",
			wantSuccess: true,
			wantContent: "meta",
		},
		{
			name:        "get_subagents",
			method:      "get_subagents",
			wantSuccess: true,
			wantContent: "[]", // No subagents in context
		},
		{
			name:        "unknown_method",
			method:      "invalid_method",
			wantSuccess: false,
			wantContent: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]any{
				"method": tt.method,
			}
			result := tool.Call(context, args)

			if result.Success() != tt.wantSuccess {
				t.Errorf("Expected success %v, got %v", tt.wantSuccess, result.Success())
			}

			if !tt.wantSuccess {
				if result.Error() == "" {
					t.Error("Expected error message, got empty string")
				}
				if !contains(result.Error(), tt.wantContent) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.wantContent, result.Error())
				}
			} else if tt.wantContent != "" {
				content := result.Data()
				if !contains(content, tt.wantContent) {
					t.Errorf("Expected content to contain '%s', got '%s'", tt.wantContent, content)
				}
			}
		})
	}
}

// Helper to check if string contains substring (simplified for test)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && len(substr) > 0 && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	// Simple brute force for test dependency avoidance
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
