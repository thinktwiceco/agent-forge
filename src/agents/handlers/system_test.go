package handlers

import (
	"context"
	"fmt"
	"testing"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// ===== Mock Implementations =====

// mockAgentOperations is a mock implementation of AgentOperations for testing.
type mockAgentOperations struct {
	loadDelegateToolCalled   bool
	ensureSystemPromptCalled bool
	initAgentContextCalled   bool
	callOrder                []string
}

func (m *mockAgentOperations) LoadDelegateTool() {
	m.loadDelegateToolCalled = true
	m.callOrder = append(m.callOrder, "LoadDelegateTool")
}

func (m *mockAgentOperations) EnsureSystemPrompt() {
	m.ensureSystemPromptCalled = true
	m.callOrder = append(m.callOrder, "EnsureSystemPrompt")
}

func (m *mockAgentOperations) InitAgentContext() {
	m.initAgentContextCalled = true
	m.callOrder = append(m.callOrder, "InitAgentContext")
}

func (m *mockAgentOperations) reset() {
	m.loadDelegateToolCalled = false
	m.ensureSystemPromptCalled = false
	m.initAgentContextCalled = false
	m.callOrder = nil
}

// mockSubAgent is a mock implementation of core.SubAgent for testing.
type mockSubAgent struct {
	name        string
	description string
}

func (m *mockSubAgent) Name() string                 { return m.name }
func (m *mockSubAgent) Description() string          { return m.description }
func (m *mockSubAgent) BasicDescription() string     { return m.description }
func (m *mockSubAgent) AdvanceDescription() string   { return "" }
func (m *mockSubAgent) Trace() string                { return "" }
func (m *mockSubAgent) SystemPrompt() string         { return "" }
func (m *mockSubAgent) Context() *core.AgentContext  { return nil }
func (m *mockSubAgent) ResponseCh() *core.ResponseCh { return nil }
func (m *mockSubAgent) Troubleshooting() string      { return "" }
func (m *mockSubAgent) DetailsAbout(item string) string {
	return fmt.Sprintf("Nothing to add about %s", item)
}
func (m *mockSubAgent) ChatStream(ctx context.Context, message string, chatId string) *core.ResponseCh {
	return nil
}

// mockHookRegistry is a mock implementation of HookRegistry for testing.
//
//nolint:unused // Reserved for future hook registry tests
type mockHookRegistry struct {
	registeredHooks map[core.Event]any
}

//nolint:unused // Reserved for future hook registry tests
func newMockHookRegistry() *mockHookRegistry {
	return &mockHookRegistry{
		registeredHooks: make(map[core.Event]any),
	}
}

//nolint:unused // Reserved for future hook registry tests
func (m *mockHookRegistry) on(event core.Event, hook any) {
	m.registeredHooks[event] = hook
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
	handlers := NewSystemHandlers()

	if handlers == nil {
		t.Fatal("NewSystemHandlers returned nil")
	}
}

func TestSystemHandlers_RegisterWith(t *testing.T) {
	handlers := NewSystemHandlers()

	// Track which handlers were called
	systemAgentCalled := false
	toolsCalled := false

	// Create mock adapter functions
	registerSystemAgent := func(handler func(AgentOperations, core.SubAgent) error) {
		systemAgentCalled = true
		// Verify we can call the handler
		mockAgent := &mockAgentOperations{}
		mockSub := &mockSubAgent{name: "test"}
		err := handler(mockAgent, mockSub)
		if err != nil {
			t.Errorf("Handler returned error: %v", err)
		}
	}

	registerTools := func(handler func(AgentOperations, []llms.Tool) error) {
		toolsCalled = true
		// Verify we can call the handler
		mockAgent := &mockAgentOperations{}
		tools := []llms.Tool{&mockTool{name: "test"}}
		err := handler(mockAgent, tools)
		if err != nil {
			t.Errorf("Handler returned error: %v", err)
		}
	}

	// Register handlers
	handlers.RegisterWith(registerSystemAgent, registerTools)

	// Verify both registration functions were called
	if !systemAgentCalled {
		t.Error("System agent registration function was not called")
	}
	if !toolsCalled {
		t.Error("Tools registration function was not called")
	}
}

func TestHandleSystemAgentAdded_Success(t *testing.T) {
	handlers := NewSystemHandlers()
	mockAgent := &mockAgentOperations{}
	mockSub := &mockSubAgent{name: "test-agent"}

	// Execute the handler directly
	err := handlers.handleSystemAgentAdded(mockAgent, mockSub)

	// Verify no error
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify all required methods were called
	if !mockAgent.loadDelegateToolCalled {
		t.Error("LoadDelegateTool was not called")
	}
	if !mockAgent.ensureSystemPromptCalled {
		t.Error("EnsureSystemPrompt was not called")
	}
	if !mockAgent.initAgentContextCalled {
		t.Error("InitAgentContext was not called")
	}

	// Verify call order
	expectedOrder := []string{"LoadDelegateTool", "EnsureSystemPrompt", "InitAgentContext"}
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

func TestHandleSystemAgentAdded_NilSubAgent(t *testing.T) {
	handlers := NewSystemHandlers()
	mockAgent := &mockAgentOperations{}

	// Execute the handler with nil subAgent
	err := handlers.handleSystemAgentAdded(mockAgent, nil)

	// Verify error is returned
	if err == nil {
		t.Error("Expected error for nil subAgent, got nil")
	}

	// Verify no methods were called after error
	if mockAgent.loadDelegateToolCalled {
		t.Error("LoadDelegateTool should not be called for nil subAgent")
	}
	if mockAgent.ensureSystemPromptCalled {
		t.Error("EnsureSystemPrompt should not be called for nil subAgent")
	}
	if mockAgent.initAgentContextCalled {
		t.Error("InitAgentContext should not be called for nil subAgent")
	}
}

func TestHandleToolsAdded_Success(t *testing.T) {
	handlers := NewSystemHandlers()
	mockAgent := &mockAgentOperations{}
	tools := []llms.Tool{
		&mockTool{name: "tool1"},
		&mockTool{name: "tool2"},
	}

	// Execute the handler directly
	err := handlers.handleToolsAdded(mockAgent, tools)

	// Verify no error
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify required methods were called
	if !mockAgent.ensureSystemPromptCalled {
		t.Error("EnsureSystemPrompt was not called")
	}
	if !mockAgent.initAgentContextCalled {
		t.Error("InitAgentContext was not called")
	}

	// Verify LoadDelegateTool was NOT called (not needed for tools)
	if mockAgent.loadDelegateToolCalled {
		t.Error("LoadDelegateTool should not be called for tools")
	}

	// Verify call order
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
	handlers := NewSystemHandlers()
	mockAgent := &mockAgentOperations{}

	// Execute the handler with nil tools
	err := handlers.handleToolsAdded(mockAgent, nil)

	// Verify error is returned
	if err == nil {
		t.Error("Expected error for nil tools, got nil")
	}

	// Verify no methods were called after error
	if mockAgent.ensureSystemPromptCalled {
		t.Error("EnsureSystemPrompt should not be called for nil tools")
	}
	if mockAgent.initAgentContextCalled {
		t.Error("InitAgentContext should not be called for nil tools")
	}
}

func TestHandleToolsAdded_EmptyTools(t *testing.T) {
	handlers := NewSystemHandlers()
	mockAgent := &mockAgentOperations{}
	tools := []llms.Tool{}

	// Execute the handler with empty tools slice
	err := handlers.handleToolsAdded(mockAgent, tools)

	// Verify no error (empty slice is valid, just not nil)
	if err != nil {
		t.Errorf("Unexpected error for empty tools: %v", err)
	}

	// Verify methods were called (empty slice is still valid)
	if !mockAgent.ensureSystemPromptCalled {
		t.Error("EnsureSystemPrompt should be called even for empty tools")
	}
	if !mockAgent.initAgentContextCalled {
		t.Error("InitAgentContext should be called even for empty tools")
	}
}

func TestSystemHandlers_Integration(t *testing.T) {
	// This test verifies that handlers execute correctly through the registration flow
	handlers := NewSystemHandlers()
	mockAgent := &mockAgentOperations{}

	// Track handler calls
	var systemAgentHandler func(AgentOperations, core.SubAgent) error
	var toolsHandler func(AgentOperations, []llms.Tool) error

	// Create mock adapter functions that capture the handlers
	registerSystemAgent := func(handler func(AgentOperations, core.SubAgent) error) {
		systemAgentHandler = handler
	}

	registerTools := func(handler func(AgentOperations, []llms.Tool) error) {
		toolsHandler = handler
	}

	// Register handlers
	handlers.RegisterWith(registerSystemAgent, registerTools)

	// Test system agent handler
	if systemAgentHandler == nil {
		t.Fatal("System agent handler was not registered")
	}

	mockSub := &mockSubAgent{name: "integration-test"}
	err := systemAgentHandler(mockAgent, mockSub)
	if err != nil {
		t.Errorf("System agent handler failed: %v", err)
	}
	if !mockAgent.loadDelegateToolCalled {
		t.Error("System agent handler did not call LoadDelegateTool")
	}

	// Reset mock
	mockAgent.reset()

	// Test tools handler
	if toolsHandler == nil {
		t.Fatal("Tools handler was not registered")
	}

	tools := []llms.Tool{&mockTool{name: "integration-tool"}}
	err = toolsHandler(mockAgent, tools)
	if err != nil {
		t.Errorf("Tools handler failed: %v", err)
	}
	if !mockAgent.ensureSystemPromptCalled {
		t.Error("Tools handler did not call EnsureSystemPrompt")
	}
}
