package tools

import (
	"strings"
	"testing"

	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// mockDiscoverableTool creates a mock tool for testing
func newMockDiscoverableTool() llms.Tool {
	return core.NewTool(
		"mock-tool",
		"A mock tool for testing",
		"Advanced mock tool with detailed capabilities",
		"Troubleshooting: Check parameters carefully",
		[]core.Parameter{},
		func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			return core.NewSuccessResponse("mock result")
		},
	)
}

// mockDiscoverableAgent is a mock agent that implements SubAgent and Discoverable
type mockDiscoverableAgent struct {
	name                string
	basicDesc           string
	advanceDesc         string
	troubleshootingInfo string
}

func (m *mockDiscoverableAgent) Name() string {
	return m.name
}

func (m *mockDiscoverableAgent) ChatStream(message string) *core.ResponseCh {
	return nil // Not needed for this test
}

func (m *mockDiscoverableAgent) BasicDescription() string {
	return m.basicDesc
}

func (m *mockDiscoverableAgent) AdvanceDescription() string {
	return m.advanceDesc
}

func (m *mockDiscoverableAgent) Troubleshooting() string {
	return m.troubleshootingInfo
}

func TestExpandTool_ExpandTool(t *testing.T) {
	expandTool := NewExpandTool()

	// Verify tool name
	if expandTool.GetName() != "expand" {
		t.Errorf("Expected tool name 'expand', got '%s'", expandTool.GetName())
	}

	// Verify it implements Discoverable
	if discoverable, ok := expandTool.(agentforge.Discoverable); ok {
		basicDesc := discoverable.BasicDescription()
		if !strings.Contains(basicDesc, "detailed information") {
			t.Errorf("Basic description should mention detailed information")
		}
	} else {
		t.Error("Expand tool should implement Discoverable interface")
	}
}

func TestExpandTool_ExpandToolInfo(t *testing.T) {
	expandTool := NewExpandTool()

	// Create mock tools
	mockTool := newMockDiscoverableTool()
	fooTool := NewFooTool()

	// Create agent context with tools
	agentContext := map[string]any{
		"tools": []llms.Tool{mockTool, fooTool},
	}

	// Test expanding the mock tool without troubleshooting
	args := map[string]any{
		"subject_type": "tool",
		"subject_name": "mock-tool",
		"troubleshoot": false,
	}

	result := expandTool.Call(agentContext, args)

	if !result.Success() {
		t.Errorf("Expected success, got error: %s", result.Error())
	}

	data := result.Data()
	t.Logf("Expand result:\n%s", data)

	// Verify the response contains expected sections
	if !strings.Contains(data, "TOOL: mock-tool") {
		t.Error("Response should contain tool header")
	}
	if !strings.Contains(data, "Basic Description:") {
		t.Error("Response should contain basic description section")
	}
	if !strings.Contains(data, "A mock tool for testing") {
		t.Error("Response should contain the basic description text")
	}
	if !strings.Contains(data, "Advanced Description:") {
		t.Error("Response should contain advanced description section")
	}
	if !strings.Contains(data, "Advanced mock tool with detailed capabilities") {
		t.Error("Response should contain the advanced description text")
	}
	if strings.Contains(data, "Troubleshooting:") {
		t.Error("Response should NOT contain troubleshooting section when troubleshoot=false")
	}
}

func TestExpandTool_ExpandToolWithTroubleshooting(t *testing.T) {
	expandTool := NewExpandTool()

	// Create mock tool
	mockTool := newMockDiscoverableTool()

	// Create agent context with tools
	agentContext := map[string]any{
		"tools": []llms.Tool{mockTool},
	}

	// Test expanding with troubleshooting
	args := map[string]any{
		"subject_type": "tool",
		"subject_name": "mock-tool",
		"troubleshoot": true,
	}

	result := expandTool.Call(agentContext, args)

	if !result.Success() {
		t.Errorf("Expected success, got error: %s", result.Error())
	}

	data := result.Data()

	// Verify troubleshooting section is present
	if !strings.Contains(data, "Troubleshooting:") {
		t.Error("Response should contain troubleshooting section when troubleshoot=true")
	}
	if !strings.Contains(data, "Check parameters carefully") {
		t.Error("Response should contain the troubleshooting text")
	}
}

func TestExpandTool_ExpandAgent(t *testing.T) {
	expandTool := NewExpandTool()

	// Create mock agent
	mockAgent := &mockDiscoverableAgent{
		name:                "test-agent",
		basicDesc:           "A test agent",
		advanceDesc:         "Advanced test agent with special capabilities",
		troubleshootingInfo: "Check agent configuration",
	}

	// Create agent context with sub-agents
	agentContext := map[string]any{
		"subAgents": []core.SubAgent{mockAgent},
	}

	// Test expanding the agent
	args := map[string]any{
		"subject_type": "agent",
		"subject_name": "test-agent",
		"troubleshoot": true,
	}

	result := expandTool.Call(agentContext, args)

	if !result.Success() {
		t.Errorf("Expected success, got error: %s", result.Error())
	}

	data := result.Data()
	t.Logf("Expand agent result:\n%s", data)

	// Verify the response contains expected content
	if !strings.Contains(data, "AGENT: test-agent") {
		t.Error("Response should contain agent header")
	}
	if !strings.Contains(data, "A test agent") {
		t.Error("Response should contain basic description")
	}
	if !strings.Contains(data, "Advanced test agent with special capabilities") {
		t.Error("Response should contain advanced description")
	}
	if !strings.Contains(data, "Check agent configuration") {
		t.Error("Response should contain troubleshooting info")
	}
}

func TestExpandTool_ToolNotFound(t *testing.T) {
	expandTool := NewExpandTool()

	// Create agent context with empty tools
	agentContext := map[string]any{
		"tools": []llms.Tool{},
	}

	// Test expanding a non-existent tool
	args := map[string]any{
		"subject_type": "tool",
		"subject_name": "nonexistent-tool",
	}

	result := expandTool.Call(agentContext, args)

	if result.Success() {
		t.Error("Expected failure for non-existent tool")
	}

	if !strings.Contains(result.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %s", result.Error())
	}
}

func TestExpandTool_AgentNotFound(t *testing.T) {
	expandTool := NewExpandTool()

	// Create agent context with empty sub-agents
	agentContext := map[string]any{
		"subAgents": []core.SubAgent{},
	}

	// Test expanding a non-existent agent
	args := map[string]any{
		"subject_type": "agent",
		"subject_name": "nonexistent-agent",
	}

	result := expandTool.Call(agentContext, args)

	if result.Success() {
		t.Error("Expected failure for non-existent agent")
	}

	if !strings.Contains(result.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %s", result.Error())
	}
}

func TestExpandTool_InvalidSubjectType(t *testing.T) {
	expandTool := NewExpandTool()

	agentContext := map[string]any{}

	// Test with invalid subject_type
	args := map[string]any{
		"subject_type": "invalid",
		"subject_name": "something",
	}

	result := expandTool.Call(agentContext, args)

	if result.Success() {
		t.Error("Expected failure for invalid subject_type")
	}

	if !strings.Contains(result.Error(), "Invalid subject_type") {
		t.Errorf("Error should mention invalid subject_type, got: %s", result.Error())
	}
}

func TestExpandTool_DefaultTroubleshootFalse(t *testing.T) {
	expandTool := NewExpandTool()

	mockTool := newMockDiscoverableTool()

	agentContext := map[string]any{
		"tools": []llms.Tool{mockTool},
	}

	// Test without troubleshoot parameter (should default to false)
	args := map[string]any{
		"subject_type": "tool",
		"subject_name": "mock-tool",
	}

	result := expandTool.Call(agentContext, args)

	if !result.Success() {
		t.Errorf("Expected success, got error: %s", result.Error())
	}

	data := result.Data()

	// Verify troubleshooting section is NOT present
	if strings.Contains(data, "Troubleshooting:") {
		t.Error("Response should NOT contain troubleshooting section when troubleshoot is not specified")
	}
}
