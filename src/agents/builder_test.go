package agents

import (
	"testing"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/mocks"
)

func TestNewBuilder(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	b := NewBuilder(mockLLM, "test-agent")

	if b == nil {
		t.Fatal("NewBuilder returned nil")
		return
	}
	if b.config.LLMEngine != mockLLM {
		t.Error("LLMEngine not set correctly")
	}
	if b.config.AgentName != "test-agent" {
		t.Errorf("AgentName = %q, want %q", b.config.AgentName, "test-agent")
	}
	if !b.config.CanExpand {
		t.Error("CanExpand should default to true")
	}
	if b.config.MaxToolIterations != 0 {
		t.Errorf("MaxToolIterations = %d, want 0", b.config.MaxToolIterations)
	}

	agent, err := b.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if agent.config.MaxToolIterations != defaultMaxToolIterations {
		t.Errorf("MaxToolIterations = %d, want %d after build", agent.config.MaxToolIterations, defaultMaxToolIterations)
	}
}

func TestBuilder_FluentChain(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	mockTool := &mockToolForBuilder{name: "test_tool"}

	agent, err := NewBuilder(mockLLM, "assistant").
		WithDescription("Helpful assistant").
		WithSystemPrompt("You are helpful").
		WithTools(mockTool).
		WithContextWindow(128000).
		WithMaxToolIterations(5).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}
	if agent.Name() != "assistant" {
		t.Errorf("agent.Name() = %q, want %q", agent.Name(), "assistant")
	}
}

func TestBuilder_AsMainAgent(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	agent, err := NewBuilder(mockLLM, "main").
		AsMainAgent().
		WithTone(ToneKeepItShort).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !agent.config.MainAgent {
		t.Error("MainAgent should be true")
	}
	if agent.config.Tone != ToneKeepItShort {
		t.Errorf("Tone = %q, want %q", agent.config.Tone, ToneKeepItShort)
	}
}

func TestBuilder_Validation_NoLLM(t *testing.T) {
	_, err := NewBuilder(nil, "agent").Build()
	if err == nil {
		t.Fatal("Build() should fail when LLM is nil")
	}
}

func TestBuilder_Validation_NoName(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	_, err := NewBuilder(mockLLM, "").Build()
	if err == nil {
		t.Fatal("Build() should fail when name is empty")
	}
}

func TestBuilder_BuildConfig(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	cfg, err := NewBuilder(mockLLM, "config-agent").
		WithSystemPrompt("test").
		BuildConfig()

	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("BuildConfig() returned nil config")
		return
	}
	if cfg.AgentName != "config-agent" {
		t.Errorf("AgentName = %q, want %q", cfg.AgentName, "config-agent")
	}
	if cfg.SystemPrompt != "test" {
		t.Errorf("SystemPrompt = %q, want %q", cfg.SystemPrompt, "test")
	}
}

func TestBuilder_WithContextWindow_SetsDefaults(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	agent, err := NewBuilder(mockLLM, "context-agent").
		WithContextWindow(128000).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if agent.config.MaxContextTokens != 128000 {
		t.Errorf("MaxContextTokens = %d, want 128000", agent.config.MaxContextTokens)
	}
	if agent.config.ReservedOutputTokens != 4000 {
		t.Errorf("ReservedOutputTokens = %d, want 4000", agent.config.ReservedOutputTokens)
	}
	if agent.config.MinRecentMessages != 10 {
		t.Errorf("MinRecentMessages = %d, want 10", agent.config.MinRecentMessages)
	}
}

// mockToolForBuilder implements llms.Tool for builder tests.
type mockToolForBuilder struct {
	name string
}

func (m *mockToolForBuilder) GetName() string { return m.name }
func (m *mockToolForBuilder) GetFunctionDefinition() llms.FunctionDefinition {
	return llms.FunctionDefinition{Name: m.name}
}
func (m *mockToolForBuilder) Call(ctx map[string]any, args map[string]any) llms.ToolReturn {
	return core.NewSuccessResponse("ok")
}

// Ensure mockToolForBuilder implements llms.Tool at compile time.
var _ llms.Tool = (*mockToolForBuilder)(nil)

func ExampleNewBuilder() {
	mockLLM := mocks.NewMockLLMEngine()
	agent, err := NewBuilder(mockLLM, "assistant").
		WithDescription("Helpful assistant").
		WithSystemPrompt("You are helpful").
		WithContextWindow(128000).
		Build()
	if err != nil {
		panic(err)
	}
	_ = agent
}
