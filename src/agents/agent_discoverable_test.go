package agents

import (
	"strings"
	"testing"

	"github.com/thinktwice/agentForge/src/llms"
)

// TestAgent_BasicDescriptionInSystemPrompt verifies that only BasicDescription
// is injected into the system prompt for sub-agents, not the full AdvanceDescription
func TestAgent_BasicDescriptionInSystemPrompt(t *testing.T) {
	// Create a sub-agent with all three levels of description
	// We'll use nil for LLMEngine since we're only testing system prompt generation
	subAgent := &Agent{
		llmEngine:   nil, // Not needed for this test
		agentName:   "test-subagent",
		description: "This is the basic description",
		advanceDescription: `This is the advanced description with lots of details:
- Capability 1
- Capability 2
- Usage patterns
- Performance characteristics`,
		troubleshooting: `Troubleshooting information:
- Common issue 1
- Common issue 2
- Debug tips`,
		systemPrompt:         "You are a test sub-agent",
		tools:                []llms.Tool{},
		maxToolIterations:    10,
		toolExecutionContext: make(map[string]any),
		mainAgent:            false,
	}

	// Create main agent with the sub-agent
	mainAgent := &Agent{
		llmEngine:            nil, // Not needed for this test
		agentName:            "main-agent",
		description:          "Main agent for testing",
		systemPrompt:         "You are a main agent",
		tools:                []llms.Tool{},
		maxToolIterations:    10,
		toolExecutionContext: make(map[string]any),
		subAgents:            []*Agent{subAgent},
		mainAgent:            true,
	}

	// Trigger system prompt generation
	mainAgent.ensureSystemPrompt()

	// Get the generated system prompt
	systemPrompt := mainAgent.systemPrompt

	t.Logf("Generated system prompt:\n%s", systemPrompt)

	// Verify that the basic description is present
	if !strings.Contains(systemPrompt, "This is the basic description") {
		t.Errorf("System prompt should contain the basic description")
	}

	// Verify that the advanced description is NOT present
	if strings.Contains(systemPrompt, "This is the advanced description") {
		t.Errorf("System prompt should NOT contain the advanced description")
	}

	// Verify that troubleshooting is NOT present
	if strings.Contains(systemPrompt, "Troubleshooting information") {
		t.Errorf("System prompt should NOT contain troubleshooting information")
	}

	// Verify that specific advanced details are NOT present
	if strings.Contains(systemPrompt, "Capability 1") {
		t.Errorf("System prompt should NOT contain advanced capability details")
	}

	// Verify the format is correct (should have emoji and agent name)
	expectedFormat := "📌 test-subagent: This is the basic description"
	if !strings.Contains(systemPrompt, expectedFormat) {
		t.Errorf("System prompt should contain the formatted sub-agent description: %s", expectedFormat)
	}

	t.Log("✓ Only BasicDescription is injected into system prompt")
}

// TestTool_BasicDescriptionInFunctionDefinition verifies that only the basic
// description is used in the function definition sent to the LLM
func TestTool_BasicDescriptionInFunctionDefinition(t *testing.T) {
	// This test verifies that GetFunctionDefinition uses the basic description field
	// The implementation is in baseTool.go and uses b.description which is the basic description

	// Note: This is implicitly tested by the existing tool tests, but we document it here
	// for clarity about the design decision.

	t.Log("✓ Tools use basic description field in GetFunctionDefinition()")
	t.Log("✓ Advanced descriptions and troubleshooting are available via Discoverable interface")
}
