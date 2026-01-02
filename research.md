# Reasoning Agent Usage and Prompt Injection Analysis

## When the Reasoning Agent is Used

### 1. Initialization
The reasoning agent is automatically added as a sub-agent when the main agent is configured with `Reasoning: true` in the `AgentConfig`:

```536:551:src/agents/agent.go
	if a.config.Reasoning {
		// Create reasoning agent from template
		// Check if a specific engine is configured for this sub agent
		var engineForReasoning llms.LLMEngine
		if a.config.ExtraEngines != nil {
			if engine, ok := a.config.ExtraEngines["system-reasoning"]; ok && engine != nil {
				engineForReasoning = engine
			} else {
				engineForReasoning = a.config.LLMEngine
			}
		} else {
			engineForReasoning = a.config.LLMEngine
		}
		raAsSubAgent := ReasoningAgent(engineForReasoning)
		systemAgents = append(systemAgents, raAsSubAgent)
	}
```

### 2. Delegation Trigger
The main agent delegates to the reasoning agent using the `delegate` tool when:
- The problem requires breaking down into multiple logical steps
- The problem requires systematic analysis and logical reasoning
- The problem is complex and would benefit from structured decomposition

The delegation happens via the delegate tool:

```42:42:src/tools/delegate/actDelegate.go
	delegateResponseCh := assignedSubAgent.ChatStream(message)
```

### 3. Main Agent's Decision Logic
The main agent receives instructions in its system prompt about when to use sub-agents:

```412:441:src/agents/agent.go
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
```

## Prompts Injected into the Main Agent

### 1. Sub-Agent List Prompt
The main agent's system prompt includes a list of available sub-agents with their basic descriptions:

```448:466:src/agents/agent.go
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
```

### 2. Reasoning Agent's Basic Description
The reasoning agent's basic description (injected into main agent's prompt) is:

```61:88:src/agents/saReasoning.go
	template.AddDescription(
		// Incipit
		`Use reasoning agent to break down and analyze COMPLEX problems into steps.
Use the reasoning agent when:
- The problem requires breaking down into multiple logical steps
- The problem requires systematic analysis and logical reasoning
- The problem is complex and would benefit from structured decomposition

DO NOT use the reasoning agent for:
- Simple informational questions
- Questions about your own capabilities or sub agents (you already have this information)
- Straightforward tasks that don't require step-by-step analysis
- Direct calculation requests (delegate those after you know HOW to solve them)

Once the reasoning agent has broken down the problem into steps, execute each step one by one.

[HOW TO USE THE REASONING AGENT]
Ask the reasoning agent "How do I ...?" or "What are the steps to ...?" for COMPLEX problems only.`,
		// Examples
		[]string{
			`❌ Wrong: Can you tell me what is the surface of a trapezoid with the following sides: 10, 20, 10, 20 and height 10?`,
			`✅ Correct: How do I calculate the surface of a trapezoid?`,
			`❌ Wrong: How many sub agents do I have? (You already know this from your system prompt)`,
			`❌ Wrong: When should I use the reasoning agent? (This is simple information in your prompt)`,
			`✅ Correct: How do I design a scalable microservices architecture for an e-commerce platform?`,
			`✅ Correct: What are the steps to implement a machine learning pipeline for fraud detection?`,
		},
	)
```

## Prompts Injected into the Reasoning Agent

### Reasoning Agent's System Prompt
When the reasoning agent is invoked, it uses its own system prompt built from the template:

```20:58:src/agents/saReasoning.go
	template.AddSystemPrompt(`
You are given with a question or a task. You need to analyze the question
and break it down into a series of logical steps. Start asking yourself
- What is the problem that we are trying to solve?
- Is there anything that it's not clear and should be clarified?
- What are the key steps to solve the problem?`,
		// Steps
		[]string{
			"Ask yourself what is the question or task that the user is asking for?",
			"What are the main problems that I should consider?",
			"What are the key steps to solve the problem?",
		},
		// Output format
		`
You will express your reasoning though the problem in the form of
🔎<reasoning for step 1>
🔎<reasoning for step 2>
🔎<reasoning for step 3>
Once you have expressed your reasoning you will enumerate the steps to solve the problem.`,
		// Examples
		[]string{`
'user': How do I plan a trip to Tokyo?

'assistant':
🔎 The user is asking for a step by step guide to plan a trip to Tokyo.
🔎 Is the user thinking about a business trip or a personal trip? Does the user want to travel by plane?
🔎 Let's assume that the user is thinking about a personal trip and wants to travel by plane. The user wants to visit Tokyo for 3 days.
Here are the steps to plan a trip to Tokyo:
1. Assess how many days the user wants to visit Tokyo
2. Assess the user's budget for the trip
3. Assess if the user is going on a business trip or a personal trip
4. ...`,
		},
		// Critical rules
		[]string{
			`If the request is not a "how to" question, reject it and say that you are a reasoning agent and you break down the problem into steps.`,
			`You should only provide the reasoning of each step and the steps to solve the problem. Do not provide the solution to the problem.`,
		},
	)
```

### Context Isolation
**Important**: When the reasoning agent is called via delegation, it receives **only the message** passed to it. It does **not** inherit the parent agent's conversation history or context. This is explicitly mentioned in the delegate tool's troubleshooting:

```33:38:src/tools/delegate/tool.go
		TroubleshootingInfo: `Troubleshooting:
- "sub agent not found" error: Verify the subAgent name matches exactly (check spelling and case)
- Empty responses: Ensure the message parameter contains sufficient context for the sub-agent
- Delegation loops: Avoid having sub-agents delegate back to parent agents
- Performance: Long-running delegations are normal for complex tasks
- Context isolation: Sub-agents don't see parent agent's history - include all relevant info in message`,
```

## Summary

1. **When Used**: The reasoning agent is used when the main agent determines a problem requires systematic analysis and step-by-step decomposition.

2. **Main Agent Prompts**:
   - Delegation workflow instructions
   - List of available sub-agents with basic descriptions
   - When to use vs. not use sub-agents

3. **Reasoning Agent Prompts**:
   - Its own system prompt with reasoning methodology
   - Instructions to break down problems into steps
   - Format requirements (🔎 markers for reasoning)
   - Critical rules (reject non-"how to" questions, don't provide solutions)

4. **Context**: The reasoning agent operates in isolation - it only receives the delegated message, not the parent's conversation history.

