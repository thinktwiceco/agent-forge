package agents

import (
	"fmt"
	"strings"
)

// ==============================
// ===== System Prompt Management
// ==============================

func (a *Agent) addSystemPrompt() {
	a.systemPrompt = a.config.SystemPrompt

	if a.systemPrompt == "" {
		a.systemPrompt = `You are an helpful assistant`
	}

	if a.config.MainAgent {
		a.systemPrompt += `
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
		if a.config.Tone == "keep-it-short" {
			a.systemPrompt += `
RESPONSE TONE:
- Remain concise and to the point
- Answer only what is asked, nothing more
- Keep responses brief while maintaining accuracy
- Avoid unnecessary elaboration or verbose explanations
- Stay focused on the specific question or task at hand
`
		}
	}
}

func (a *Agent) buildSubAgentsSystemPrompt() {
	if len(a.subAgents) == 0 || a.subAgents == nil {
		return
	}

	var saPrompt = `
=== AVAILABLE SUB-AGENTS ===
Specialized tools for complex problems. Use "delegate" tool to access them.

[SUB AGENTS]:
	`

	for _, sa := range a.subAgents {
		// Use BasicDescription() to ensure only basic info is injected into system prompt
		saPrompt += fmt.Sprintf("📌 %s: %s\n\n", (*sa).Name(), (*sa).BasicDescription())
	}

	a.systemPrompt += saPrompt
}

func (a *Agent) buildToolsDescrptions() {
	if len(a.tools) == 0 || a.tools == nil {
		return
	}

	var toolsPrompt = `
=== AVAILABLE TOOLS ===
Tools available to the agent. Use "tool" tool to access them.

[TOOLS]:
	`

	for _, tool := range a.tools {
		toolsPrompt += fmt.Sprintf("📌 %s: %s\n\n", tool.GetName(), tool.GetFunctionDefinition().Description)
	}

	a.systemPrompt += toolsPrompt
}

// normalizePromptIndentation normalizes the indentation of a prompt string.
// It converts tabs to spaces, removes trailing whitespace, and normalizes indentation
// by detecting the minimum indentation level and preserving relative indentation.
// It also handles cases where indentation is inconsistent by normalizing to the most common base level.
func normalizePromptIndentation(prompt string) string {
	if prompt == "" {
		return prompt
	}

	lines := strings.Split(prompt, "\n")
	if len(lines) == 0 {
		return prompt
	}

	// First pass: convert tabs to spaces and trim trailing whitespace
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		// Convert tabs to spaces (assuming 4 spaces per tab)
		line = strings.ReplaceAll(line, "\t", "    ")
		// Remove trailing whitespace
		line = strings.TrimRight(line, " ")
		normalized = append(normalized, line)
	}

	// Find minimum indentation (excluding empty lines)
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

	// If minimum indent is 0 but we have indented lines, those are likely
	// accidental indentation from source code formatting - remove all indentation
	if minIndent == 0 && hasIndentedLines {
		for i := range normalized {
			if normalized[i] != "" {
				normalized[i] = strings.TrimLeft(normalized[i], " ")
			}
		}
		return strings.Join(normalized, "\n")
	}

	// Normalize all lines by removing the minimum indent (preserves relative indentation)
	result := make([]string, 0, len(normalized))
	for _, line := range normalized {
		if line == "" {
			result = append(result, "")
			continue
		}
		// Remove the minimum indent from each line
		currentIndent := len(line) - len(strings.TrimLeft(line, " "))
		if currentIndent >= minIndent && len(line) >= minIndent {
			line = line[minIndent:]
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

func (a *Agent) ensureSystemPrompt() {
	a.addSystemPrompt()
	a.buildSubAgentsSystemPrompt()
	a.buildToolsDescrptions()
	// Normalize indentation before finalizing
	a.systemPrompt = normalizePromptIndentation(a.systemPrompt)
}
