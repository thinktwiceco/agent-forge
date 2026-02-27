package system

// GitAgentTemplate defines the system agent template for Git version control operations.
//
// This agent handles git operations including status checks, staging files, committing changes,
// pushing and pulling from remotes, branch management, and viewing history and diffs.

// CreateGitAgentTemplate creates the template for git operations agent.
func CreateGitAgentTemplate() (*SystemAgentTemplate, error) {
	return LoadAgentTemplateFromMarkdown("git", AgentNameSystemGit, TraceGit)
}
