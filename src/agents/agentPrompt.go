package agents

import (
	"fmt"

	agentforge "github.com/thinktwice/agentForge/src"
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

func (a *Agent) ensureSystemPrompt() {
	a.addSystemPrompt()
	a.buildSubAgentsSystemPrompt()
	a.buildToolsDescrptions()

	agentforge.Debug("========================================================")
	agentforge.Debug("System prompt for agent %s: %s", a.Name(), a.systemPrompt)
	agentforge.Debug("========================================================")
}
