package agents

import (
	"encoding/json"
	"testing"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/mocks"
	"github.com/thinktwiceco/agent-forge/src/tools/meta"
)

// TestAgent_MetaTool_GetSubagents verifies that the meta tool can correctly retrieve subagents
func TestAgent_MetaTool_GetSubagents(t *testing.T) {
	// Create mock LLM for main agent
	mainLLM := mocks.NewMockLLMEngine()

	// Create a mock subagent
	mockSubAgent := newMockSubAgent(
		"test-reasoning-agent",
		"A test reasoning agent",
		"A test reasoning agent with full description",
	)

	// Create agent with subagent
	agent := NewAgent(&AgentConfig{
		LLMEngine: mainLLM,
		AgentName: "test-agent",
		Trace:     "test",
		SubAgents: []core.SubAgent{mockSubAgent},
	})

	// Get the meta tool from the agent
	var metaTool core.Tool
	for _, tool := range agent.tools {
		if tool.GetName() == "meta" {
			if mt, ok := tool.(*core.Tool); ok {
				metaTool = *mt
			}
			break
		}
	}

	if metaTool.Name == "" {
		t.Fatal("Meta tool not found in agent tools")
	}

	// Build agent context
	agentContext := agent.agentContext.BuildContext(nil)

	// Call the meta tool with get_subagents method
	result := metaTool.Handler(agentContext, map[string]any{
		"method": "get_subagents",
	})

	// Verify the result
	if !result.Success() {
		t.Fatalf("Expected meta tool to succeed, got error: %s", result.Error())
	}

	// Parse the JSON result
	var subagents []map[string]string
	if err := json.Unmarshal([]byte(result.Data()), &subagents); err != nil {
		t.Fatalf("Failed to parse subagents JSON: %v", err)
	}

	// Verify we got the subagent
	if len(subagents) != 1 {
		t.Fatalf("Expected 1 subagent, got %d. Result: %s", len(subagents), result.Data())
	}

	if subagents[0]["name"] != "test-reasoning-agent" {
		t.Errorf("Expected subagent name 'test-reasoning-agent', got '%s'", subagents[0]["name"])
	}

	if subagents[0]["basicDescription"] != "A test reasoning agent" {
		t.Errorf("Expected subagent description 'A test reasoning agent', got '%s'", subagents[0]["basicDescription"])
	}
}

// TestMetaTool_GetSubagents_DirectCall tests the meta tool directly without an agent
func TestMetaTool_GetSubagents_DirectCall(t *testing.T) {
	// Create a mock subagent
	mockSubAgent := newMockSubAgent(
		"reasoning",
		"A reasoning agent",
		"A reasoning agent with full description",
	)

	// Create agent context with subagent
	agentContext := map[string]any{
		"subAgents": []core.SubAgent{mockSubAgent},
	}

	// Create meta tool
	metaTool := meta.NewMetaTool()

	// Call the tool
	result := metaTool.Call(agentContext, map[string]any{
		"method": "get_subagents",
	})

	// Verify the result
	if !result.Success() {
		t.Fatalf("Expected meta tool to succeed, got error: %s", result.Error())
	}

	// Parse the JSON result
	var subagents []map[string]string
	if err := json.Unmarshal([]byte(result.Data()), &subagents); err != nil {
		t.Fatalf("Failed to parse subagents JSON: %v", err)
	}

	// Verify we got the subagent
	if len(subagents) != 1 {
		t.Fatalf("Expected 1 subagent, got %d. Result: %s", len(subagents), result.Data())
	}

	if subagents[0]["name"] != "reasoning" {
		t.Errorf("Expected subagent name 'reasoning', got '%s'", subagents[0]["name"])
	}

	if subagents[0]["basicDescription"] != "A reasoning agent" {
		t.Errorf("Expected subagent description 'A reasoning agent', got '%s'", subagents[0]["basicDescription"])
	}
}
