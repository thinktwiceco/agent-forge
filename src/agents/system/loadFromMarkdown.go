package system

import (
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/agents/prompts"
)

// LoadAgentTemplateFromMarkdown loads a system agent template from an embedded markdown file.
// The name corresponds to the filename without .md (e.g., "reasoning", "vector", "os").
func LoadAgentTemplateFromMarkdown(name, agentName, trace string) (*SystemAgentTemplate, error) {
	template, err := NewSystemAgentTemplate(agentName, trace)
	if err != nil {
		return nil, err
	}
	content, err := prompts.LoadSystemAgent(name)
	if err != nil {
		return nil, fmt.Errorf("load system agent %s: %w", name, err)
	}
	template.AddSystemPrompt(
		content.Incipit,
		content.Steps,
		content.Output,
		content.Examples,
		content.Critical,
	)
	template.AddDescription(content.DescriptionIncipit, content.DescriptionExamples)
	template.AddAdvanceDescription(content.AdvanceDescription)
	template.AddTroubleshooting(content.Troubleshooting)
	return template, nil
}
