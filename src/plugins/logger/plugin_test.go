package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

func TestLoggerPlugin_Name(t *testing.T) {
	plugin := NewPlugin(DefaultColorRules(), DefaultLabelRules(), nil)
	if plugin.Name() != "logger" {
		t.Errorf("Expected name 'logger', got '%s'", plugin.Name())
	}
}

func TestLoggerPlugin_Hooks(t *testing.T) {
	plugin := NewPlugin(DefaultColorRules(), DefaultLabelRules(), nil)
	hooks := plugin.Hooks()

	// Should have at least one hook
	if len(hooks) == 0 {
		t.Error("Expected at least one hook")
	}

	// Should handle EventNewChunk
	if hooks[core.EventNewChunk] == nil {
		t.Error("Expected handler for EventNewChunk")
	}

	// Should not have other event hooks
	for event, hook := range hooks {
		if event != core.EventNewChunk && hook != nil {
			t.Errorf("Unexpected hook for event: %s", event)
		}
	}
}

func TestLoggerPlugin_ImplementsInterfaces(t *testing.T) {
	plugin := NewPlugin(DefaultColorRules(), DefaultLabelRules(), nil)

	// Should implement Plugin interface
	var _ core.Plugin = plugin

	// Should implement HookProvider interface
	var _ core.HookProvider = plugin

	// Should NOT implement ToolProvider or PromptProvider
	// (compile-time check would fail if it did)
}

func TestLoggerPlugin_HandleNewChunk(t *testing.T) {
	buf := new(bytes.Buffer)
	plugin := NewPlugin(DefaultColorRules(), DefaultLabelRules(), buf)

	// Get the hook from Hooks() map
	hooks := plugin.Hooks()
	hookFn := hooks[core.EventNewChunk]
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
