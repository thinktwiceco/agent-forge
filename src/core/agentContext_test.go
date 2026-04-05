package core

import (
	"testing"

	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/mocks"
)

func TestAgentContext_BuildContext(t *testing.T) {
	mockTool := mocks.NewMockTool("mock-tool")
	ac := &AgentContext{
		AgentName: "test-agent",
		Trace:     "test-trace",
		Model:     "test-model",
		Tools:     []llms.Tool{mockTool},
	}

	responseCh := NewResponseCh("test-agent", "test-trace", "chat-123", nil)
	ctx := ac.BuildContext(responseCh)

	if ctx["agentName"] != "test-agent" {
		t.Errorf("Expected agentName to be 'test-agent', got %v", ctx["agentName"])
	}
	if ctx["responseCh"] != responseCh {
		t.Error("Expected responseCh to be set in context")
	}

	// Verify session storage initialization
	if ctx["sessionStorage"] == nil {
		t.Error("Expected sessionStorage to be initialized")
	}
}

func TestRehydrateContext(t *testing.T) {
	responseCh := NewResponseCh("test-agent", "test-trace", "chat-123", nil)
	mockTool := mocks.NewMockTool("mock-tool")

	validMap := map[string]any{
		"agentName":      "test-agent",
		"trace":          "test-trace",
		"model":          "test-model",
		"tools":          []llms.Tool{mockTool},
		"responseCh":     responseCh,
		"sessionStorage": make(map[string]any),
		"pluginFields":   make(map[string]any),
	}

	ac, err := RehydrateContext(validMap)
	if err != nil {
		t.Fatalf("RehydrateContext failed: %v", err)
	}

	if ac.AgentName != "test-agent" {
		t.Errorf("Expected AgentName 'test-agent', got '%s'", ac.AgentName)
	}
	if len(ac.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(ac.Tools))
	}
}

func TestRehydrateContext_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		wantErr bool
	}{
		{
			name: "invalid agentName type",
			input: map[string]any{
				"agentName": 123,
			},
			wantErr: true,
		},
		{
			name: "invalid tools type",
			input: map[string]any{
				"tools": "not-a-slice",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RehydrateContext(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("RehydrateContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgentContext_SyncFromMap(t *testing.T) {
	ac := &AgentContext{
		SessionStorage: make(map[string]any),
		PluginFields:   make(map[string]any),
	}

	ctxMap := map[string]any{
		"pluginFields": map[string]any{
			"customKey": "customValue",
		},
	}

	// Update session storage directly (shared by reference)
	sessionMap := ac.SessionStorage
	sessionMap["key"] = "value"

	err := ac.SyncFromMap(ctxMap)
	if err != nil {
		t.Fatalf("SyncFromMap failed: %v", err)
	}

	if ac.PluginFields["customKey"] != "customValue" {
		t.Errorf("Expected PluginFields to be updated")
	}
	if ac.SessionStorage["key"] != "value" {
		t.Errorf("Expected SessionStorage to persist")
	}
}
