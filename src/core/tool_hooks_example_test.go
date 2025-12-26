package core_test

import (
	"strings"
	"testing"

	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// mockHooks implements the Hooks interface for testing
type mockHooks struct {
	safePaths    []string
	safeCommands []string
}

func (m *mockHooks) IsSafePath(path string) bool {
	for _, safe := range m.safePaths {
		if strings.HasPrefix(path, safe) {
			return true
		}
	}
	return false
}

func (m *mockHooks) IsSafeCommand(cmd string) bool {
	for _, safe := range m.safeCommands {
		if cmd == safe {
			return true
		}
	}
	return false
}

func TestTool_SetHooks(t *testing.T) {
	// Create a tool
	tool := core.NewTool(
		"test-tool",
		"A test tool",
		"Advanced test tool",
		"Troubleshooting info",
		[]core.Parameter{},
		func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			return core.NewSuccessResponse("success")
		},
	)

	// Cast to *Tool to access GetHooks
	concreteTool, ok := tool.(*core.Tool)
	if !ok {
		t.Fatal("Expected tool to be *core.Tool")
	}

	// Initially hooks should be nil
	if concreteTool.GetHooks() != nil {
		t.Error("Expected initial hooks to be nil")
	}

	// Set hooks
	hooks := &mockHooks{
		safePaths:    []string{"/tmp", "/home/user"},
		safeCommands: []string{"ls", "pwd"},
	}
	concreteTool.SetHooks(hooks)

	// Verify hooks are set
	if concreteTool.GetHooks() == nil {
		t.Fatal("Expected hooks to be set")
	}

	// Test hook methods
	if !concreteTool.GetHooks().IsSafePath("/tmp/test.txt") {
		t.Error("Expected /tmp/test.txt to be safe")
	}

	if concreteTool.GetHooks().IsSafePath("/etc/passwd") {
		t.Error("Expected /etc/passwd to NOT be safe")
	}

	if !concreteTool.GetHooks().IsSafeCommand("ls") {
		t.Error("Expected 'ls' to be safe")
	}

	if concreteTool.GetHooks().IsSafeCommand("rm -rf /") {
		t.Error("Expected 'rm -rf /' to NOT be safe")
	}
}

func TestTool_HooksNilByDefault(t *testing.T) {
	tool := core.NewTool(
		"test-tool",
		"A test tool",
		"Advanced test tool",
		"Troubleshooting info",
		[]core.Parameter{},
		func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			return core.NewSuccessResponse("success")
		},
	)

	concreteTool := tool.(*core.Tool)

	if concreteTool.GetHooks() != nil {
		t.Error("Hooks should be nil by default")
	}
}
