package agents

import (
	"testing"

	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/mocks" // Using mocks package if available/needed
)

// We need a way to create a minimal agent without full config loading if possible,
// or just test methods on a struct if we can.
// Agent struct fields are private mostly, but public methods are what we test.
// NewAgent calls many initialization hooks.
// Let's rely on a minimal config.

func TestAgent_Getters(t *testing.T) {
	// Create a barebones agent manually if possible or via NewAgent
	cfg := &AgentConfig{
		AgentName:          "TestAgent",
		Description:        "Desc",
		AdvanceDescription: "AdvDesc",
		Trace:              "Trace",
		SystemPrompt:       "SysPrompt",
		Troubleshooting:    "Trouble",
		MainAgent:          false,
	}

	// Since NewAgent does a lot of heavy lifting (hooks, file checking potentially),
	// we might want to skip NewAgent if it requires too much env setup.
	// However, we can construct the struct directly since we are in the same package (whitebox testing).

	a := &Agent{
		config:       cfg,
		agentContext: nil, // set if needed
		tools:        []llms.Tool{},
	}

	// Test SubAgent interface methods
	if a.Name() != "TestAgent" {
		t.Errorf("Name() = %s, want TestAgent", a.Name())
	}
	if a.Description() != "Desc" {
		t.Errorf("Description() = %s, want Desc", a.Description())
	}
	if a.BasicDescription() != "Desc" {
		t.Errorf("BasicDescription() = %s, want Desc", a.BasicDescription())
	}
	if a.AdvanceDescription() != "AdvDesc" {
		t.Errorf("AdvanceDescription() = %s, want AdvDesc", a.AdvanceDescription())
	}
	if a.Trace() != "Trace" {
		t.Errorf("Trace() = %s, want Trace", a.Trace())
	}
	if a.SystemPrompt() != "SysPrompt" {
		t.Errorf("SystemPrompt() = %s, want SysPrompt", a.SystemPrompt())
	}
	if a.Troubleshooting() != "Trouble" {
		t.Errorf("Troubleshooting() = %s, want Trouble", a.Troubleshooting())
	}

	// Context and ResponseCh might be nil, verify it handles/returns them
	if a.Context() != nil {
		t.Error("Expected nil context")
	}
}

func TestAgent_ToolsAttributes(t *testing.T) {
	a := &Agent{
		tools:  nil,
		hooks:  newAgentHooks(), // needed for AddTools
		config: &AgentConfig{AgentName: "TestAgent", Trace: "test"},
	}

	// GetTools on nil
	if tools := a.GetTools(); len(tools) != 0 {
		t.Error("Expected 0 tools")
	}

	// AddTools
	// Mock tool?
	// We can use a nil interface slice or mock.
	// AddTools calls hooks, so we need hooks initialized.

	mockTool := &mocks.MockTool{}
	a.AddTools([]llms.Tool{mockTool})

	if len(a.GetTools()) != 1 {
		t.Errorf("Action AddTools failed, len=%d", len(a.GetTools()))
	}
}

func TestAgent_Initializers(t *testing.T) {
	// Simple test for ensureHooks, registerPlugins, etc if accessible
	// They are private.
	// We can test public behavior that relies on them.
	// Or leave it for integration tests.
}

func TestAgent_SubAgent(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	// Prepare correct arguments
	config := &AgentConfig{
		AgentName:   "Sub",
		Trace:       "sub",
		Description: "A sub agent",
		LLMEngine:   mockLLM,
	}
	// We need hooks for AddSystemAgent
	// Manually construct agent with hooks
	a := NewAgent(config)

	// Create another agent to act as sub-agent
	subConfig := &AgentConfig{
		AgentName:   "Child",
		Trace:       "child",
		Description: "Child agent",
		LLMEngine:   mockLLM, // Share mock LLM
	}
	sub := NewAgent(subConfig)

	// Convert to interface
	subInterface := sub.AgentAsSubAgent()

	// Add
	a.AddSystemAgent(subInterface)

	// Verify it's in the subagents map (private map, but maybe we can query it)
	// The agent struct has subAgents map[string]SubAgent
	// But it's private.
	// We can use GetSystemAgent methods if they exist, or rely on other public side effects.
	// Looking at agent.go, there is no GetSubAgents public method except as part of Expand/Meta.

	// However, we can check basic successful execution without panic.

	// Check AgentAsSubAgent properties
	if subInterface.Name() != "Child" {
		t.Error("SubAgent Name mismatch")
	}
}
