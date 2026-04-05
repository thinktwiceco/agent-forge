package context

import (
	"testing"

	agentcore "github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

func TestNewManager(t *testing.T) {
	config := Config{
		AgentName: "test-agent",
		Trace:     "test-trace",
		Model:     "gpt-4",
		Tools:     []llms.Tool{},
	}

	manager := NewManager(config)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	ctx := manager.Context()
	if ctx == nil {
		t.Fatal("Context() returned nil")
		return
	}

	if ctx.AgentName != "test-agent" {
		t.Errorf("Expected AgentName to be 'test-agent', got '%s'", ctx.AgentName)
	}

	if ctx.Trace != "test-trace" {
		t.Errorf("Expected Trace to be 'test-trace', got '%s'", ctx.Trace)
	}

	if ctx.Model != "gpt-4" {
		t.Errorf("Expected Model to be 'gpt-4', got '%s'", ctx.Model)
	}
}

func TestManager_BuildContext(t *testing.T) {
	config := Config{
		AgentName: "test-agent",
		Trace:     "test-trace",
		Model:     "gpt-4",
	}

	manager := NewManager(config)
	responseCh := agentcore.NewResponseCh("test-agent", "test-trace", "", nil)

	contextMap := manager.BuildContext(responseCh)

	if contextMap == nil {
		t.Fatal("BuildContext returned nil")
	}

	if contextMap["agentName"] != "test-agent" {
		t.Errorf("Expected agentName to be 'test-agent', got '%v'", contextMap["agentName"])
	}

	if contextMap["trace"] != "test-trace" {
		t.Errorf("Expected trace to be 'test-trace', got '%v'", contextMap["trace"])
	}

	if contextMap["model"] != "gpt-4" {
		t.Errorf("Expected model to be 'gpt-4', got '%v'", contextMap["model"])
	}

	if contextMap["responseCh"] != responseCh {
		t.Error("Expected responseCh to be set in context map")
	}

	// Verify SessionStorage is initialized
	if _, ok := contextMap["sessionStorage"]; !ok {
		t.Error("Expected sessionStorage to be present in context map")
	}

	// Verify PluginFields is initialized
	if _, ok := contextMap["pluginFields"]; !ok {
		t.Error("Expected pluginFields to be present in context map")
	}
}

func TestManager_SyncFromMap(t *testing.T) {
	config := Config{
		AgentName: "test-agent",
	}

	manager := NewManager(config)
	responseCh := agentcore.NewResponseCh("test-agent", "", "", nil)

	// Build initial context
	contextMap := manager.BuildContext(responseCh)

	pluginFields := contextMap["pluginFields"].(map[string]any)
	pluginFields["customField"] = "customValue"

	// Sync changes back
	err := manager.SyncFromMap(contextMap)
	if err != nil {
		t.Fatalf("SyncFromMap failed: %v", err)
	}

	ctx := manager.Context()
	if ctx.PluginFields["customField"] != "customValue" {
		t.Errorf("Expected PluginFields['customField'] to be 'customValue', got '%v'", ctx.PluginFields["customField"])
	}
}

func TestManager_TruncateHistory_NoStrategy(t *testing.T) {
	config := Config{
		AgentName: "test-agent",
	}

	manager := NewManager(config)

	messages := []*llms.UnifiedMessage{
		llms.SystemMessage("system"),
		llms.UserMessage("user1"),
		llms.AssistantMessage("assistant1", 0, 0, 0),
	}

	// Without strategy, should return messages unchanged
	result := manager.TruncateHistory(messages)

	if len(result) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(result))
	}
}

func TestManager_TruncateHistory_WithStrategy(t *testing.T) {
	tokenCounter := llms.NewTokenCounter("approximate")
	strategy := NewSlidingWindow(2, tokenCounter)

	config := Config{
		AgentName:          "test-agent",
		TokenCounter:       tokenCounter,
		TruncationStrategy: strategy,
		MaxContextTokens:   100,
		ReservedTokens:     20,
	}

	manager := NewManager(config)

	messages := []*llms.UnifiedMessage{
		llms.SystemMessage("system"),
		llms.UserMessage("user1"),
		llms.AssistantMessage("assistant1", 0, 0, 0),
	}

	// Should delegate to strategy
	result := manager.TruncateHistory(messages)

	// Result should be non-nil (strategy will handle truncation logic)
	if result == nil {
		t.Error("Expected non-nil result from TruncateHistory")
	}
}

func TestManager_UpdateConfig(t *testing.T) {
	config := Config{
		AgentName: "test-agent",
		Tools:     []llms.Tool{},
	}

	manager := NewManager(config)

	// Update config
	newConfig := Config{
		AgentName: "updated-agent",
		Model:     "gpt-4-turbo",
	}

	manager.UpdateConfig(newConfig)

	ctx := manager.Context()
	if ctx.AgentName != "updated-agent" {
		t.Errorf("Expected AgentName to be 'updated-agent', got '%s'", ctx.AgentName)
	}

	if ctx.Model != "gpt-4-turbo" {
		t.Errorf("Expected Model to be 'gpt-4-turbo', got '%s'", ctx.Model)
	}
}

func TestManager_UpdateTools(t *testing.T) {
	config := Config{
		AgentName: "test-agent",
		Tools:     []llms.Tool{},
	}

	manager := NewManager(config)

	// Create a mock tool
	mockTool := &mockTool{name: "test-tool"}
	tools := []llms.Tool{mockTool}

	manager.UpdateTools(tools)

	ctx := manager.Context()
	if len(ctx.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(ctx.Tools))
	}

	if ctx.Tools[0].GetName() != "test-tool" {
		t.Errorf("Expected tool name to be 'test-tool', got '%s'", ctx.Tools[0].GetName())
	}
}

func TestManager_PreservesSessionStorage(t *testing.T) {
	config := Config{
		AgentName: "test-agent",
	}

	manager := NewManager(config)
	ctx := manager.Context()

	// Set some session storage
	ctx.SessionStorage["key1"] = "value1"

	// Update config (which rebuilds context)
	newConfig := Config{
		AgentName: "test-agent",
		Model:     "gpt-4",
	}
	manager.UpdateConfig(newConfig)

	// Session storage should be preserved
	newCtx := manager.Context()
	if newCtx.SessionStorage["key1"] != "value1" {
		t.Error("SessionStorage was not preserved after UpdateConfig")
	}
}

func TestManager_PreservesPluginFields(t *testing.T) {
	config := Config{
		AgentName: "test-agent",
	}

	manager := NewManager(config)
	ctx := manager.Context()

	// Set some plugin fields
	ctx.PluginFields["plugin1"] = "data1"

	// Update config (which rebuilds context)
	newConfig := Config{
		AgentName: "test-agent",
		Model:     "gpt-4",
	}
	manager.UpdateConfig(newConfig)

	// Plugin fields should be preserved
	newCtx := manager.Context()
	if newCtx.PluginFields["plugin1"] != "data1" {
		t.Error("PluginFields were not preserved after UpdateConfig")
	}
}

// Mock implementations for testing

type mockTool struct {
	name string
}

func (m *mockTool) GetName() string {
	return m.name
}

func (m *mockTool) GetFunctionDefinition() llms.FunctionDefinition {
	return llms.FunctionDefinition{
		Name:        m.name,
		Description: "mock tool",
	}
}

func (m *mockTool) Call(contextMap map[string]any, arguments map[string]any) llms.ToolReturn {
	return agentcore.NewSuccessResponse("success")
}
