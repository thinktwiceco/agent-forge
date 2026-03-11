package system

// CreateReasoningAgentTemplate creates the template for the reasoning agent.
//
// This agent acts as a critical thinking layer that analyzes user requests before
// the main agent responds. It helps identify ambiguities, detect assumptions, spot nuances,
// and guide the main agent toward objective, direct responses without condescension or excessive politeness.
func CreateReasoningAgentTemplate() *SystemAgentTemplate {
	template, err := LoadAgentTemplateFromMarkdown("reasoning", AgentNameSystemReasoning, TraceReasoning)
	if err != nil {
		panic(err)
	}
	return template
}
