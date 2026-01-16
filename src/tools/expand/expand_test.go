package expand

import (
	"strings"
	"testing"

	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/tools"
)

// Helper functions

// setupExpandTool returns a new expand tool instance for testing
func setupExpandTool() llms.Tool {
	return NewExpandTool()
}

// createMockTool creates a mock discoverable tool for testing
func createMockTool() llms.Tool {
	return &core.Tool{
		Name:                "mock-tool",
		Description:         "A mock tool for testing",
		AdvanceDesc:         "Advanced mock tool with detailed capabilities",
		TroubleshootingInfo: "Troubleshooting: Check parameters carefully",
		Parameters:          []core.Parameter{},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			return core.NewSuccessResponse("mock result")
		},
	}
}

// createMockAgent creates a mock agent with given properties
func createMockAgent(name, basicDesc, advanceDesc, troubleshootingInfo string) *mockDiscoverableAgent {
	return &mockDiscoverableAgent{
		name:                name,
		basicDesc:           basicDesc,
		advanceDesc:         advanceDesc,
		troubleshootingInfo: troubleshootingInfo,
	}
}

// createAgentContextWithTools creates an agent context map with tools
func createAgentContextWithTools(tools []llms.Tool) map[string]any {
	return map[string]any{
		"tools": tools,
	}
}

// createAgentContextWithSubAgents creates an agent context map with sub-agents
func createAgentContextWithSubAgents(subAgents []*core.SubAgent) map[string]any {
	return map[string]any{
		"subAgents": subAgents,
	}
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

// Tests

func TestExpandTool_ExpandTool(t *testing.T) {
	expandTool := setupExpandTool()

	t.Run("tool name", func(t *testing.T) {
		if expandTool.GetName() != "expand" {
			t.Errorf("Expected tool name 'expand', got '%s'", expandTool.GetName())
		}
	})

	t.Run("implements Discoverable", func(t *testing.T) {
		discoverable, ok := expandTool.(agentforge.Discoverable)
		if !ok {
			t.Fatal("Expand tool should implement Discoverable interface")
		}

		basicDesc := discoverable.BasicDescription()
		if !strings.Contains(basicDesc, "detailed information") {
			t.Errorf("Basic description should mention detailed information")
		}
	})
}

func TestExpandTool_ExpandToolInfo(t *testing.T) {
	tests := []struct {
		name                string
		troubleshoot        bool
		wantTroubleshooting bool
		wantTroubleshootText string
	}{
		{
			name:                "without troubleshooting",
			troubleshoot:        false,
			wantTroubleshooting: false,
		},
		{
			name:                "with troubleshooting",
			troubleshoot:        true,
			wantTroubleshooting: true,
			wantTroubleshootText: "Check parameters carefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expandTool := setupExpandTool()
			mockTool := createMockTool()
			fooTool := tools.NewFooTool()

			agentContext := createAgentContextWithTools([]llms.Tool{mockTool, fooTool})

			args := map[string]any{
				"subject_type": "tool",
				"subject_name": "mock-tool",
				"troubleshoot": tt.troubleshoot,
			}

			result := expandTool.Call(agentContext, args)

			if !result.Success() {
				t.Fatalf("Expected success, got error: %s", result.Error())
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

			// Verify troubleshooting section presence
			hasTroubleshooting := strings.Contains(data, "Troubleshooting:")
			if tt.wantTroubleshooting {
				if !hasTroubleshooting {
					t.Error("Response should contain troubleshooting section when troubleshoot=true")
				}
				if tt.wantTroubleshootText != "" && !strings.Contains(data, tt.wantTroubleshootText) {
					t.Errorf("Response should contain the troubleshooting text '%s'", tt.wantTroubleshootText)
				}
			} else {
				if hasTroubleshooting {
					t.Error("Response should NOT contain troubleshooting section when troubleshoot=false")
				}
			}
		})
	}
}

func TestExpandTool_ExpandAgent(t *testing.T) {
	expandTool := setupExpandTool()

	mockAgent := createMockAgent(
		"test-agent",
		"A test agent",
		"Advanced test agent with special capabilities",
		"Check agent configuration",
	)

	// Convert to pointer to interface, similar to AgentAsSubAgent()
	var sa core.SubAgent = mockAgent
	agentContext := createAgentContextWithSubAgents([]*core.SubAgent{&sa})

	args := map[string]any{
		"subject_type": "agent",
		"subject_name": "test-agent",
		"troubleshoot": true,
	}

	result := expandTool.Call(agentContext, args)

	if !result.Success() {
		t.Fatalf("Expected success, got error: %s", result.Error())
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

func TestExpandTool_NotFound(t *testing.T) {
	tests := []struct {
		name         string
		subjectType  string
		subjectName  string
		agentContext map[string]any
	}{
		{
			name:        "tool not found",
			subjectType: "tool",
			subjectName: "nonexistent-tool",
			agentContext: createAgentContextWithTools([]llms.Tool{}),
		},
		{
			name:        "agent not found",
			subjectType: "agent",
			subjectName: "nonexistent-agent",
			agentContext: createAgentContextWithSubAgents([]*core.SubAgent{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expandTool := setupExpandTool()

			args := map[string]any{
				"subject_type": tt.subjectType,
				"subject_name": tt.subjectName,
			}

			result := expandTool.Call(tt.agentContext, args)

			if result.Success() {
				t.Error("Expected failure for non-existent subject")
			}

			if !strings.Contains(result.Error(), "not found") {
				t.Errorf("Error should mention 'not found', got: %s", result.Error())
			}
		})
	}
}

func TestExpandTool_InvalidSubjectType(t *testing.T) {
	expandTool := setupExpandTool()
	agentContext := map[string]any{}

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
	expandTool := setupExpandTool()
	mockTool := createMockTool()

	agentContext := createAgentContextWithTools([]llms.Tool{mockTool})

	// Test without troubleshoot parameter (should default to false)
	args := map[string]any{
		"subject_type": "tool",
		"subject_name": "mock-tool",
	}

	result := expandTool.Call(agentContext, args)

	if !result.Success() {
		t.Fatalf("Expected success, got error: %s", result.Error())
	}

	data := result.Data()

	// Verify troubleshooting section is NOT present
	if strings.Contains(data, "Troubleshooting:") {
		t.Error("Response should NOT contain troubleshooting section when troubleshoot is not specified")
	}
}


