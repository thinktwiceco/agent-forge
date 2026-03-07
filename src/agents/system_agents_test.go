package agents

import (
	"os"
	"testing"

	"github.com/thinktwiceco/agent-forge/src/agents/system"
	"github.com/thinktwiceco/agent-forge/src/mocks"
)

func TestOsAgent_Constructor(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	tmpDir, err := os.MkdirTemp("", "os-agent-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	agent := OsAgent(mockLLM, tmpDir)

	if agent == nil {
		t.Fatal("OsAgent returned nil")
	}

	if agent.Name() != AgentNameSystemOS {
		t.Errorf("Expected name %s, got %s", AgentNameSystemOS, agent.Name())
	}

	// SubAgent interface methods
	if agent.BasicDescription() == "" {
		t.Error("BasicDescription should not be empty")
	}
	if agent.AdvanceDescription() == "" {
		t.Error("AdvanceDescription should not be empty")
	}
}

func TestGitAgent_Constructor(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	tmpDir, err := os.MkdirTemp("", "git-agent-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	agent := GitAgent(mockLLM, tmpDir)

	if agent == nil {
		t.Fatal("GitAgent returned nil")
	}

	if agent.Name() != AgentNameSystemGit {
		t.Errorf("Expected name %s, got %s", AgentNameSystemGit, agent.Name())
	}

	if agent.BasicDescription() == "" {
		t.Error("BasicDescription should not be empty")
	}
}

func TestWebAgent_Constructor(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	tmpDir, err := os.MkdirTemp("", "web-agent-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	agent := WebAgent(mockLLM, tmpDir)

	if agent == nil {
		t.Fatal("WebAgent returned nil")
	}

	if agent.Name() != AgentNameSystemWeb {
		t.Errorf("Expected name %s, got %s", AgentNameSystemWeb, agent.Name())
	}

	if agent.BasicDescription() == "" {
		t.Error("BasicDescription should not be empty")
	}
}

func TestCodingAgent_Constructor(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	tmpDir, err := os.MkdirTemp("", "coding-agent-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	agent := CodingAgent(mockLLM, tmpDir)

	if agent == nil {
		t.Fatal("CodingAgent returned nil")
	}

	if agent.Name() != AgentNameSystemCoding {
		t.Errorf("Expected name %s, got %s", AgentNameSystemCoding, agent.Name())
	}

	if agent.BasicDescription() == "" {
		t.Error("BasicDescription should not be empty")
	}
}

func TestReasoningAgent_Constructor(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	agent := ReasoningAgent(mockLLM)

	if agent == nil {
		t.Fatal("ReasoningAgent returned nil")
	}

	if agent.Name() != AgentNameSystemReasoning {
		t.Errorf("Expected name %s, got %s", AgentNameSystemReasoning, agent.Name())
	}

	if agent.BasicDescription() == "" {
		t.Error("BasicDescription should not be empty")
	}
}

func TestSystemAgentTemplate_Methods(t *testing.T) {
	tmpl, err := system.NewSystemAgentTemplate("test", "trace")
	if err != nil {
		t.Fatal(err)
	}

	tmpl.AddSystemPrompt("incipit", []string{"step1"}, "output", []string{"example"}, []string{"crit"})
	if tmpl.SystemPrompt() == "" {
		t.Error("SystemPrompt should be built")
	}

	tmpl.AddDescription("usage", []string{"ex"})
	if tmpl.Description() == "" {
		t.Error("Description should be built")
	}

	tmpl.AddAdvanceDescription("advanced")
	if tmpl.AdvanceDescription() != "advanced" {
		t.Error("AdvanceDescription mismatch")
	}

	tmpl.AddTroubleshooting("trouble")
	if tmpl.Troubleshooting() != "trouble" {
		t.Error("Troubleshooting mismatch")
	}

	_, err = system.NewSystemAgentTemplate("", "trace")
	if err == nil {
		t.Error("Expected error for empty name")
	}
}
