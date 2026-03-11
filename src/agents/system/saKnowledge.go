package system

// CreateKnowledgeAgentTemplate creates the template for the knowledge agent.
//
// This agent acts as a dedicated memory layer backed by the knowledge graph plugin.
// It stores, retrieves, and organizes facts about the user and their context.
// The plugin injects the full graph schema and tool instructions at runtime.
func CreateKnowledgeAgentTemplate() *SystemAgentTemplate {
	template, err := NewSystemAgentTemplate(AgentNameSystemKnowledge, TraceKnowledge)
	if err != nil {
		panic(err)
	}

	template.AddSystemPrompt(
		`[ROLE] Dedicated memory agent. Store and retrieve facts from the knowledge graph.
Do NOT answer questions directly. Return facts or a structured summary only.`,
		[]string{
			"Understand what information is requested or needs to be stored",
			"Traverse the graph to find relevant categories and existing facts",
			"Store new facts under the correct category, with a short title",
			"Cross-reference related nodes using link_relevant",
			"Return retrieved facts or confirm storage — nothing else",
		},
		`Return ONLY:
- Retrieved facts (verbatim or summarized from graph nodes)
- Confirmation of what was stored (category, title, fact)
- Gaps: explicitly state when the graph contains no relevant data

Do NOT speculate, elaborate, or answer beyond graph content.
You MUST always respond — never return empty. The caller needs success/failure feedback.`,
		[]string{`
'request': Remember that the user prefers dark mode in all editors.

'response':
Stored under Preferences > Editor:
  title: "Prefers dark mode"
  fact: "User prefers dark mode in all editors."`, `
'request': What do you know about the user's programming languages?

'response':
Found under Skills > Programming Languages:
  - "Primary language: Go" (go-lang-primary)
  - "Also uses Python for scripting" (python-scripting)
  - "Learning Rust" (rust-learning)`,
		},
		[]string{
			"Graph content only — no invented facts",
			"Always traverse before storing to avoid duplicates",
			"Provide a short title (3-8 words) on every remember call",
			"Return a clear gap message when nothing is found",
			"No conversation, no elaboration — storage or retrieval only",
			"NEVER return empty. Always output at least one line: success (e.g. 'Stored under X: title, fact'), failure (e.g. 'Error: ...'), or gap (e.g. 'Nothing found.').",
		},
	)

	template.AddDescription(
		`Stores and retrieves facts from the persistent knowledge graph. Key operations: remember (store), find (search), out_nodes/in_nodes (traverse). Delegate memory operations here.`,
		[]string{
			`✅ Use for: Saving user preferences, goals, corrections, project context`,
			`✅ Use for: Recalling what the user told you in past sessions`,
			`❌ Don't use: To answer questions — only for memory read/write`,
		},
	)

	template.AddAdvanceDescription(`
- Purpose: Persistent memory layer. Reads and writes the knowledge graph.
- Plugin: knowledge graph plugin (injected at runtime — provides all tools and schema)
- Input: A fact to store, or a question about stored knowledge
- Output: Stored confirmation OR retrieved facts. Nothing else.
- Tools (via plugin): remember, find, out_nodes, in_nodes, get_node_content, add_category, add_subcategory, link_relevant, attach_document, forget
- Does NOT: Answer questions, reason, or produce content outside the graph
- Integration: Invoke when you need to persist or recall information across sessions`)

	template.AddTroubleshooting(`
- Agent returns invented facts: WRONG — it must only return graph content
- Duplicate facts stored: Agent skipped traversal before inserting; check out_nodes first
- Nothing found: Confirm graph was populated; try find tool with broader query
- Wrong category: Traverse from root with out_nodes("") to see full structure
- Missing title: Always provide a 3-8 word title on remember calls for traversal efficiency`)

	return template
}
