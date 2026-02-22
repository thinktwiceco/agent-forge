package prompts

import (
	"fmt"
	"strings"
)

// Ensure Builder implements agents.PromptBuilder interface at compile time
// This is a forward declaration - the actual interface check is done
// in the agents package to avoid circular dependencies

// Tone constants for prompt building.
const (
	ToneKeepItShort = "keep-it-short"
	ToneSystemAgent = "system-agent"
)

// Builder constructs system prompts.
type Builder struct {
	config Config
}

// NewBuilder creates a new prompt Builder.
func NewBuilder(config Config) *Builder {
	return &Builder{config: config}
}

// UpdateConfig updates the builder's configuration (call when tools or sub-agents change).
func (b *Builder) UpdateConfig(cfg interface{}) {
	if config, ok := cfg.(Config); ok {
		b.config = config
	}
}

// Build returns the complete system prompt.
func (b *Builder) Build() string {
	prompt := b.addSystemPrompt()
	prompt = b.buildSubAgentsSystemPrompt(prompt)
	prompt = b.buildToolsDescriptions(prompt)
	return normalizePromptIndentation(prompt)
}

func getTonePrompt(tone string) string {
	var name string
	switch tone {
	case ToneKeepItShort:
		name = "tone-keep-it-short"
	case ToneSystemAgent:
		name = "tone-system-agent"
	default:
		return ""
	}
	s, err := LoadMainPrompt(name)
	if err != nil {
		panic(err)
	}
	return s + "\n"
}

func (b *Builder) addSystemPrompt() string {
	prompt := b.config.SystemPrompt

	if prompt == "" {
		defaultPrompt, err := LoadMainPrompt("default")
		if err != nil {
			panic(err)
		}
		prompt = defaultPrompt
	}

	if b.config.MainAgent {
		mainAgent, err := LoadMainPrompt("main-agent")
		if err != nil {
			panic(err)
		}
		prompt += "\n\n" + mainAgent
		if tonePrompt := getTonePrompt(b.config.Tone); tonePrompt != "" {
			prompt += "\n" + tonePrompt
		}
	} else {
		if tonePrompt := getTonePrompt(b.config.Tone); tonePrompt != "" {
			prompt += "\n" + tonePrompt
		}
	}

	return prompt
}

func (b *Builder) buildSubAgentsSystemPrompt(prompt string) string {
	if len(b.config.SubAgents) == 0 {
		return prompt
	}

	header, err := LoadMainPrompt("sub-agents-header")
	if err != nil {
		panic(err)
	}
	saPrompt := "\n" + header + "\n"
	for _, sa := range b.config.SubAgents {
		saPrompt += fmt.Sprintf("📌 %s: %s\n\n", sa.Name(), sa.BasicDescription())
	}
	prompt += saPrompt
	return prompt
}

func (b *Builder) buildToolsDescriptions(prompt string) string {
	if len(b.config.Tools) == 0 {
		return prompt
	}

	header, err := LoadMainPrompt("tools-header")
	if err != nil {
		panic(err)
	}
	toolsPrompt := "\n" + header + "\n"
	for _, tool := range b.config.Tools {
		toolsPrompt += fmt.Sprintf("📌 %s: %s\n\n", tool.GetName(), tool.GetFunctionDefinition().Description)
	}
	prompt += toolsPrompt
	return prompt
}

// NormalizePromptIndentation normalizes the indentation of a prompt string.
func normalizePromptIndentation(prompt string) string {
	if prompt == "" {
		return prompt
	}

	lines := strings.Split(prompt, "\n")
	if len(lines) == 0 {
		return prompt
	}

	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.ReplaceAll(line, "\t", "    ")
		line = strings.TrimRight(line, " ")
		normalized = append(normalized, line)
	}

	minIndent := -1
	hasIndentedLines := false
	for _, line := range normalized {
		if line == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent > 0 {
			hasIndentedLines = true
		}
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent == 0 && hasIndentedLines {
		for i := range normalized {
			if normalized[i] != "" {
				normalized[i] = strings.TrimLeft(normalized[i], " ")
			}
		}
		return strings.Join(normalized, "\n")
	}

	result := make([]string, 0, len(normalized))
	for _, line := range normalized {
		if line == "" {
			result = append(result, "")
			continue
		}
		currentIndent := len(line) - len(strings.TrimLeft(line, " "))
		if currentIndent >= minIndent && len(line) >= minIndent {
			line = line[minIndent:]
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
