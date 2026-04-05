package prompts

import (
	"strings"
	"testing"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

func TestBuilderBuild_NoToolsOmitsToolsSection(t *testing.T) {
	builder := NewBuilder(Config{
		SystemPrompt: "Base instructions",
	})

	got := builder.Build()

	if !strings.Contains(got, "Base instructions") {
		t.Fatalf("Build() = %q, want base system prompt included", got)
	}
	if strings.Contains(got, "[TOOLS]") {
		t.Fatalf("Build() = %q, want no tools section when no tools are configured", got)
	}
}

func TestBuilderBuild_WithToolsAddsGenericHintOnly(t *testing.T) {
	builder := NewBuilder(Config{
		SystemPrompt: "Base instructions",
		Tools: []llms.Tool{
			&promptTestTool{name: "alpha_tool", description: "Alpha tool description"},
			&promptTestTool{name: "beta_tool", description: "Beta tool description"},
		},
	})

	got := builder.Build()

	if count := strings.Count(got, "[TOOLS]"); count != 1 {
		t.Fatalf("Build() tools section count = %d, want 1 in %q", count, got)
	}
	if !strings.Contains(got, "Use native tool calls when needed.") {
		t.Fatalf("Build() = %q, want generic tools guidance", got)
	}
	if strings.Contains(got, "alpha_tool") || strings.Contains(got, "beta_tool") {
		t.Fatalf("Build() = %q, want no per-tool names in system prompt", got)
	}
	if strings.Contains(got, "Alpha tool description") || strings.Contains(got, "Beta tool description") {
		t.Fatalf("Build() = %q, want no per-tool descriptions in system prompt", got)
	}
}

func TestBuilderBuild_MainAgentKeepsLeanNormalizedPrompt(t *testing.T) {
	builder := NewBuilder(Config{
		SystemPrompt: "Base instructions",
		MainAgent:    true,
		Tools: []llms.Tool{
			&promptTestTool{name: "alpha_tool", description: "Alpha tool description"},
		},
	})

	got := builder.Build()

	if !strings.Contains(got, "[ROLE]") {
		t.Fatalf("Build() = %q, want main-agent role guidance", got)
	}
	if !strings.Contains(got, "Use spawn_subagent only for focused work") {
		t.Fatalf("Build() = %q, want lean spawn_subagent guidance", got)
	}
	if strings.Contains(got, "\t") {
		t.Fatalf("Build() = %q, want tabs normalized out of the prompt", got)
	}
	if strings.Contains(got, "\n    [ROLE]") {
		t.Fatalf("Build() = %q, want normalized indentation for main-agent prompt", got)
	}
}

type promptTestTool struct {
	name        string
	description string
}

func (t *promptTestTool) GetName() string {
	return t.name
}

func (t *promptTestTool) GetFunctionDefinition() llms.FunctionDefinition {
	return llms.FunctionDefinition{
		Name:        t.name,
		Description: t.description,
	}
}

func (t *promptTestTool) Call(map[string]any, map[string]any) llms.ToolReturn {
	return core.NewSuccessResponse("ok")
}

var _ llms.Tool = (*promptTestTool)(nil)
