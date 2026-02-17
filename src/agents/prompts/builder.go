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
	switch tone {
	case ToneKeepItShort:
		return `RESPONSE TONE:
- Remain concise and to the point
- Answer only what is asked, nothing more
- Keep responses brief while maintaining accuracy
- Avoid unnecessary elaboration or verbose explanations
- Stay focused on the specific question or task at hand
`
	case ToneSystemAgent:
		return `RESPONSE TONE:
- You are a system agent. Execute operations silently.
- Do NOT provide commentary, explanations, or announcements.
- Do NOT use phrases like "I'll", "Let me", "I will", etc.
- Only return the requested results or data.
- No greetings, no explanations, just results.
- Perform your duty as a good system agent without unnecessary noise.
`
	default:
		return ""
	}
}

func (b *Builder) addSystemPrompt() string {
	prompt := b.config.SystemPrompt

	if prompt == "" {
		prompt = `You are an helpful assistant`
	}

	if b.config.MainAgent {
		prompt += `
[SYSTEM] You are the MAIN agent coordinating a team of specialized sub-agents.

SUB-AGENTS AS SPECIALIZED TOOLS:
Sub-agents are specialized tools for complex problems. Use the "delegate" tool ONLY when:
- The problem requires specialized expertise beyond your direct knowledge
- The task benefits from systematic analysis or reasoning
- You cannot answer directly from your context

DELEGATION WORKFLOW:
1. Understand the problem scope - identify what needs to be solved
2. Find the correct sub-agent - match the problem to the right specialization
3. Formalize the request - craft a clear, specific task description
4. Delegate via tool call - use the "delegate" tool with the chosen agent
5. Evaluate the response - assess if it fully addresses the problem
6. Iterate if needed - refine your request and delegate again if the response is incomplete

IMPORTANT - EXECUTION BEHAVIOR:
- Execute operations silently without revealing your internal decision-making process
- Do NOT announce that you are delegating to sub-agents or using tools
- Do NOT mention which sub-agent you are using or what actions you are taking
- Simply execute the requested operation and present the final result to the user
- Only mention sub-agents or capabilities if the user explicitly asks about them

RESPOND DIRECTLY (no tool calls) for:
- Greetings, casual conversation, simple Q&A
- Questions you can answer from your context
- Questions about your capabilities or sub-agents list
`
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

	saPrompt := `
=== AVAILABLE SUB-AGENTS ===
Specialized tools for complex problems. Use "delegate" tool to access them.

[SUB AGENTS]:
`
	for _, sa := range b.config.SubAgents {
		saPrompt += fmt.Sprintf("📌 %s: %s\n\n", sa.Name(), sa.BasicDescription())
	}

	prompt += "Use the 'expand' if you need to use one of the sub-agents."
	prompt += saPrompt
	return prompt
}

func (b *Builder) buildToolsDescriptions(prompt string) string {
	if len(b.config.Tools) == 0 {
		return prompt
	}

	toolsPrompt := `
=== AVAILABLE TOOLS ===
Tools available to the agent. Use "tool" tool to access them.

[TOOLS]:
`
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
