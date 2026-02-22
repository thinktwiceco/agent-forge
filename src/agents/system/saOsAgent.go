package system

// OsAgentTemplate defines the system agent template for OS-related operations.
//
// This agent handles file system operations, executing executables, and other OS-level tasks.
// It specializes in reading files, writing files, executing commands, and managing file system resources.

// CreateOsAgentTemplate creates the template for OS operations agent.
func CreateOsAgentTemplate() *SystemAgentTemplate {
	template, err := LoadAgentTemplateFromMarkdown("os", AgentNameSystemOS, TraceOS)
	if err != nil {
		panic(err)
	}
	return template
}
