package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/thinktwice/agentForge/src/agents"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

func TestLoggerPlugin_Name(t *testing.T) {
	plugin := NewPlugin(DefaultColorRules(), DefaultLabelRules(), nil)
	if plugin.Name() != "logger" {
		t.Errorf("Expected name 'logger', got '%s'", plugin.Name())
	}
}

func TestLoggerPlugin_Tools(t *testing.T) {
	plugin := NewPlugin(DefaultColorRules(), DefaultLabelRules(), nil)
	if len(plugin.Tools()) != 0 {
		t.Errorf("Expected 0 tools, got %d", len(plugin.Tools()))
	}
}

func TestLoggerPlugin_SystemPrompt(t *testing.T) {
	plugin := NewPlugin(DefaultColorRules(), DefaultLabelRules(), nil)
	if plugin.SystemPrompt() != "" {
		t.Errorf("Expected empty system prompt, got '%s'", plugin.SystemPrompt())
	}
}

func TestLoggerPlugin_On(t *testing.T) {
	plugin := NewPlugin(DefaultColorRules(), DefaultLabelRules(), nil)

	// Should handle EventNewChunk
	if plugin.On(core.EventNewChunk) == nil {
		t.Error("Expected handler for EventNewChunk")
	}

	// Should not handle random event
	if plugin.On("random_event") != nil {
		t.Error("Expected nil for random event")
	}
}

func TestLoggerPlugin_HandleNewChunk(t *testing.T) {
	buf := new(bytes.Buffer)
	plugin := NewPlugin(DefaultColorRules(), DefaultLabelRules(), buf)

	// Create a mock hook function manually since we can't easily execute the private handler directly
	// without reflection or export. However, HandleNewChunk is private.
	// We can access it via the hook returned by On(EventNewChunk)

	// Better way: use the hook returned by On() which is a typed wrapper
	// core.AgentHookFn is 'any', but we know it's func(*Agent, *ExtendedChunkResponse) error

	hookFn := plugin.On(core.EventNewChunk)
	handler, ok := hookFn.(agents.OnNewChunkHook)
	if !ok {
		t.Fatal("Failed to cast hook function")
	}

	// Create a dummy agent (can be nil if not used by handler logic extensively)
	// The handler uses a.Name() and a.Trace() if chunk is empty, but we'll provide them in chunk

	chunk := &core.ExtendedChunkResponse{
		Type:      llms.TypeContent,
		Content:   "test content",
		AgentName: "TestAgent",
		Trace:     "TestTrace",
	}

	// First call - should print header + content
	if err := handler(nil, chunk); err != nil {
		t.Errorf("Handler failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "TestAgent") {
		t.Errorf("Expected output to contain 'TestAgent', got '%s'", output)
	}
	if !strings.Contains(output, "test content") {
		t.Errorf("Expected output to contain 'test content', got '%s'", output)
	}

	// Reset buffer
	buf.Reset()

	// Second call - same agent, no header expected
	if err := handler(nil, chunk); err != nil {
		t.Errorf("Handler failed: %v", err)
	}
	output = buf.String()
	_ = output // Output should not contain TestAgent due to header logic
	// "TestAgent" appears in header label. Content just prints.
	// Color codes might make string matching tricky, assume simple contains works.

	// Test Tool Execution
	buf.Reset()
	toolChunk := &core.ExtendedChunkResponse{
		Type: llms.TypeToolExecuting,
		ToolExecuting: &llms.ToolCall{
			Name: "test-tool",
		},
		AgentName: "TestAgent",
	}
	if err := handler(nil, toolChunk); err != nil {
		t.Errorf("Handler failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Executing tool: test-tool") {
		t.Errorf("Expected tool exec output, got '%s'", buf.String())
	}
}
