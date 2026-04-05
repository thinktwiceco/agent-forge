package handlers

import (
	"testing"

	"github.com/thinktwiceco/agent-forge/src/llms"
)

// ===== Mock Implementations =====

// mockAgentOperations is a mock implementation of AgentOperations for testing.
type mockAgentOperations struct {
	ensureSystemPromptCalled bool
	initAgentContextCalled   bool
	callOrder                []string
}

func (m *mockAgentOperations) EnsureSystemPrompt() {
	m.ensureSystemPromptCalled = true
	m.callOrder = append(m.callOrder, "EnsureSystemPrompt")
}

func (m *mockAgentOperations) InitAgentContext() {
	m.initAgentContextCalled = true
	m.callOrder = append(m.callOrder, "InitAgentContext")
}

// mockTool is a minimal mock implementation of llms.Tool for testing.
type mockTool struct {
	name string
}

func (m *mockTool) GetName() string { return m.name }
func (m *mockTool) Call(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	return &mockToolReturn{}
}
func (m *mockTool) GetFunctionDefinition() llms.FunctionDefinition {
	return llms.FunctionDefinition{
		Name:        m.name,
		Description: "mock tool",
		Parameters: llms.FunctionParameters{
			Type_:      "object",
			Properties: make(map[string]llms.FunctionObjectParameter),
			Required:   nil,
		},
	}
}

// mockToolReturn is a minimal mock implementation of llms.ToolReturn for testing.
type mockToolReturn struct{}

func (m *mockToolReturn) Success() bool   { return true }
func (m *mockToolReturn) Error() string   { return "" }
func (m *mockToolReturn) Data() string    { return "" }
func (m *mockToolReturn) Ephemeral() bool { return false }
func (m *mockToolReturn) Cleanup() func() { return nil }

// ===== Tests =====

func TestNewSystemHandlers(t *testing.T) {
	h := NewSystemHandlers()
	if h == nil {
		t.Fatal("NewSystemHandlers returned nil")
	}
}

func TestSystemHandlers_RegisterWith(t *testing.T) {
	h := NewSystemHandlers()
	toolsCalled := false

	registerTools := func(handler func(AgentOperations, []llms.Tool) error) {
		toolsCalled = true
		mockAgent := &mockAgentOperations{}
		tools := []llms.Tool{&mockTool{name: "test"}}
		err := handler(mockAgent, tools)
		if err != nil {
			t.Errorf("Handler returned error: %v", err)
		}
	}

	h.RegisterWith(registerTools)

	if !toolsCalled {
		t.Error("Tools registration function was not called")
	}
}

func TestHandleToolsAdded_Success(t *testing.T) {
	h := NewSystemHandlers()
	mockAgent := &mockAgentOperations{}
	tools := []llms.Tool{
		&mockTool{name: "tool1"},
		&mockTool{name: "tool2"},
	}

	err := h.handleToolsAdded(mockAgent, tools)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !mockAgent.ensureSystemPromptCalled {
		t.Error("EnsureSystemPrompt was not called")
	}
	if !mockAgent.initAgentContextCalled {
		t.Error("InitAgentContext was not called")
	}

	expectedOrder := []string{"EnsureSystemPrompt", "InitAgentContext"}
	if len(mockAgent.callOrder) != len(expectedOrder) {
		t.Errorf("Expected %d calls, got %d", len(expectedOrder), len(mockAgent.callOrder))
	}
	for i, expected := range expectedOrder {
		if i >= len(mockAgent.callOrder) || mockAgent.callOrder[i] != expected {
			t.Errorf("Expected call order[%d] to be %s, got %v",
				i, expected, mockAgent.callOrder)
		}
	}
}

func TestHandleToolsAdded_NilTools(t *testing.T) {
	h := NewSystemHandlers()
	mockAgent := &mockAgentOperations{}

	err := h.handleToolsAdded(mockAgent, nil)
	if err == nil {
		t.Error("Expected error for nil tools, got nil")
	}

	if mockAgent.ensureSystemPromptCalled {
		t.Error("EnsureSystemPrompt should not be called for nil tools")
	}
}

func TestHandleToolsAdded_EmptyTools(t *testing.T) {
	h := NewSystemHandlers()
	mockAgent := &mockAgentOperations{}
	tools := []llms.Tool{}

	err := h.handleToolsAdded(mockAgent, tools)
	if err != nil {
		t.Errorf("Unexpected error for empty tools: %v", err)
	}

	if !mockAgent.ensureSystemPromptCalled {
		t.Error("EnsureSystemPrompt should be called even for empty tools")
	}
}

func TestSystemHandlers_Integration(t *testing.T) {
	h := NewSystemHandlers()
	mockAgent := &mockAgentOperations{}

	var toolsHandler func(AgentOperations, []llms.Tool) error

	registerTools := func(handler func(AgentOperations, []llms.Tool) error) {
		toolsHandler = handler
	}

	h.RegisterWith(registerTools)

	if toolsHandler == nil {
		t.Fatal("Tools handler was not registered")
	}

	tools := []llms.Tool{&mockTool{name: "integration-tool"}}
	err := toolsHandler(mockAgent, tools)
	if err != nil {
		t.Errorf("Tools handler failed: %v", err)
	}
	if !mockAgent.ensureSystemPromptCalled {
		t.Error("Tools handler did not call EnsureSystemPrompt")
	}
}
