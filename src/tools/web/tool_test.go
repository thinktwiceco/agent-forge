package web

import (
	"os"
	"testing"
)

func TestValidateAction(t *testing.T) {
	tests := []struct {
		name    string
		action  any
		wantErr bool
	}{
		{"valid_open_session", "open_session", false},
		{"valid_navigate", "navigate", false},
		{"valid_click", "click", false},
		{"valid_get_content", "get_content", false},
		{"valid_save_content", "save_content", false},
		{"invalid_string", "destroy_internet", true},
		{"invalid_type", 123, true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAction(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSelector(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		wantErr  bool
	}{
		{"valid", ".class", false},
		{"valid_id", "#id", false},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSelector(tt.selector)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSelector() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewWebTool(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "web-tool-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tool := NewWebTool(tmpDir)
	if tool.GetName() != "web_browser" {
		t.Errorf("Expected tool name 'web_browser', got '%s'", tool.GetName())
	}

	// Verify parameters exist via FunctionDefinition
	def := tool.GetFunctionDefinition()
	if len(def.Parameters.Properties) == 0 {
		t.Error("Parameters should not be empty")
	}

	// Check action parameter definition
	if _, ok := def.Parameters.Properties["action"]; !ok {
		t.Error("action parameter not found definition")
	}

	// Check if action is required
	foundRequired := false
	for _, req := range def.Parameters.Required {
		if req == "action" {
			foundRequired = true
			break
		}
	}
	if !foundRequired {
		t.Error("action param should be required")
	}
}
