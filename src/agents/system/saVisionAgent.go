package system

// CreateVisionAgentTemplate creates the template for the vision agent.
// The vision agent loads images via the image tool and answers visual questions
// using the vision capability of the underlying LLM.
func CreateVisionAgentTemplate() *SystemAgentTemplate {
	template, err := LoadAgentTemplateFromMarkdown("vision", AgentNameSystemVision, TraceVision)
	if err != nil {
		panic(err)
	}
	return template
}
