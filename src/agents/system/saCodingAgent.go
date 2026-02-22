package system

// CodingAgentTemplate defines the system agent template for coding-related operations.
//
// This agent handles coding tasks including reading and writing code files, analyzing codebases,
// understanding code structure, and discovering tool/agent capabilities. It specializes in
// code-related file operations and progressive discovery of available tools and agents.

// CreateCodingAgentTemplate creates the template for coding operations agent.
func CreateCodingAgentTemplate() *SystemAgentTemplate {
	template, err := LoadAgentTemplateFromMarkdown("coding", AgentNameSystemCoding, TraceCoding)
	if err != nil {
		panic(err)
	}
	return template
}
