package system

// WebAgentTemplate defines the system agent template for web navigation and content pulling operations.
//
// This agent handles web operations including navigating to URLs, clicking buttons, and pulling page content.

// CreateWebAgentTemplate creates the template for web operations agent.
func CreateWebAgentTemplate() *SystemAgentTemplate {
	template, err := LoadAgentTemplateFromMarkdown("web", AgentNameSystemWeb, TraceWeb)
	if err != nil {
		panic(err)
	}
	return template
}
