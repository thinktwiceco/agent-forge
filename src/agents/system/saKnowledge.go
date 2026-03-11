package system

// CreateKnowledgeAgentTemplate creates the template for the knowledge agent.
//
// This agent acts as a dedicated memory layer backed by the knowledge graph plugin.
// It stores, retrieves, and organizes facts about the user and their context.
// The plugin injects the full graph schema and tool instructions at runtime.
func CreateKnowledgeAgentTemplate() *SystemAgentTemplate {
	template, err := LoadAgentTemplateFromMarkdown("knowledge", AgentNameSystemKnowledge, TraceKnowledge)
	if err != nil {
		panic(err)
	}
	return template
}
