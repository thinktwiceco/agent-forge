package agents

// ReasoningAgentTemplate defines the system agent template for reasoning and problem decomposition.
//
// This agent analyzes questions and breaks them down into logical steps.
// It focuses on "how to" questions and rejects direct solution requests.
var ReasoningAgentTemplate = createReasoningAgentTemplate()

func createReasoningAgentTemplate() *SystemAgentTemplate {
	template, err := NewSystemAgentTemplate("system-reasoning", "reasoning")
	if err != nil {
		panic(err)
	}

	// Build system prompt with structured components
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

	// Build description with structured components
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

	// Add advanced description
	template.AddAdvanceDescription(`
Advanced Details:
- Purpose: Breaks down complex problems into logical, actionable steps
- Reasoning Style: Uses 🔎 markers to show thought process before providing steps
- Input Requirements: Works best with "How do I...?" or "What are the steps...?" questions
- Output Format: First provides reasoning (🔎), then enumerates concrete steps
- Capabilities:
  * Identifies missing information and makes reasonable assumptions
  * Clarifies ambiguities in problem statements
  * Structures unstructured problems into logical sequences
  * Focuses on methodology rather than direct solutions
- Limitations:
  * Does NOT execute the steps - only provides the breakdown
  * Does NOT provide final solutions - only the path to reach them
  * Rejects simple questions that don't require decomposition
  * Best suited for multi-step, complex analysis tasks
- Integration: Automatically available as a sub-agent when reasoning mode is enabled`)

	// Add troubleshooting information
	template.AddTroubleshooting(`
Troubleshooting:
- "Rejection response": If the agent rejects your query, ensure you're asking a "how to" question for a complex task
- "Too simple response": The problem might not be complex enough - try asking directly without delegation
- "Missing context": The agent makes assumptions when information is unclear - provide more specific requirements if needed
- "No solution provided": This is expected behavior - the reasoning agent only provides steps, not solutions
- Common mistakes:
  * Asking for simple information already in your context
  * Requesting direct calculations instead of calculation methodology
  * Using it for straightforward tasks that don't need decomposition
- Best practices:
  * Frame questions as "How do I..." for complex problems
  * Provide context about constraints and requirements
  * Use the output steps to guide your subsequent actions
  * Don't expect final answers - expect roadmaps`)

	return template
}
