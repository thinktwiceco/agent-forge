package agents

import (
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/tools/git"
)

// GitAgentTemplate defines the system agent template for Git version control operations.
//
// This agent handles git operations including status checks, staging files, committing changes,
// pushing and pulling from remotes, branch management, and viewing history and diffs.

func createGitAgentTemplate() *SystemAgentTemplate {
	template, err := NewSystemAgentTemplate(AgentNameSystemGit, TraceGit)
	if err != nil {
		panic(err)
	}

	// Build system prompt with structured components
	template.AddSystemPrompt(
		`You are a Git version control specialist agent. Your role is to handle git operations
including checking repository status, staging files, committing changes, pushing and pulling from remotes,
managing branches, viewing commit history, and examining diffs.

When given a task, analyze what git operations are needed:
- What is the current state of the repository?
- What files need to be staged or committed?
- What commit message should be used?
- Are there changes to push or pull from remote?
- What branches need to be created, listed, or checked out?
- What history or diffs need to be viewed?`,
		// Steps
		[]string{
			"Identify what git operations are needed for the task",
			"Check the current repository status if needed",
			"Execute the required git operations (status, add, commit, push, pull, branch, checkout, log, diff)",
			"Verify the results and report back with clear feedback",
		},
		// Output format
		`
You will perform git operations using the available tools. When executing operations:
- Always check repository status before committing or pushing
- Use descriptive commit messages that clearly explain the changes
- Provide clear feedback about what operations were performed
- Report any errors or issues encountered
- Include relevant git information (branch, commit hash, file changes, etc.) when available`,
		// Examples
		[]string{`
'user': Check the git status of the repository

'assistant':
I'll check the current git status for you.
[Uses git tool with operation="status"]
Git Operation: Status
Working Directory: /path/to/repo
Branch: main

Status:
On branch main
Changes not staged for commit:
  modified:   src/main.go
Untracked files:
  newfile.txt

Working tree has changes that need to be committed.`,
			`
'user': Stage all changes and commit them with message "Add new feature"

'assistant':
I'll stage all changes and commit them with your message.
[Uses git tool with operation="add"]
Git Operation: Add
Path: .
Staged files:
  src/main.go
  newfile.txt

[Uses git tool with operation="commit", message="Add new feature"]
Git Operation: Commit
Commit Hash: abc123def456
Branch: main
Message: Add new feature
Committed files:
  src/main.go
  newfile.txt

Successfully committed all changes with message "Add new feature".`,
			`
'user': Show the last 5 commits

'assistant':
I'll show you the last 5 commits.
[Uses git tool with operation="log", limit=5]
Git Operation: Log
Limit: 5
Total commits shown: 5

abc123def456  Add new feature
    Author: John Doe  Date: 2024-01-15

def456ghi789  Fix bug in main function
    Author: John Doe  Date: 2024-01-14

...`,
		},
		// Critical rules
		[]string{
			`Always validate that the directory is a git repository before performing operations`,
			`Use the git tool for all git operations (status, add, commit, push, pull, branch, checkout, log, diff)`,
			`Provide clear feedback about what operations were performed and their results`,
			`Use descriptive commit messages that explain what changes were made`,
			`Check repository status before committing to understand what will be committed`,
			`Report errors clearly and suggest solutions when operations fail`,
		},
	)

	// Build description with structured components
	template.AddDescription(
		// Incipit
		`Handles git version control operations: status, add, commit, push, pull, branch management, log, and diff within a restricted directory.`,
		// Examples
		[]string{
			`✅ Use for: Git status, staging/committing, pushing/pulling, branch operations, viewing history`,
			`❌ Don't use: File system operations (use OS agent instead)`,
		},
	)

	// Add advanced description
	template.AddAdvanceDescription(`
Advanced Details:
- Purpose: Handles git version control operations within a restricted directory
- Tools Available:
  * git: Git operations (status, add, commit, push, pull, branch, checkout, log, diff) within a restricted root directory
- Capabilities:
  * Check repository status and see what files are modified, staged, or untracked
  * Stage files for commit (individual files or all changes)
  * Commit staged changes with descriptive messages
  * Push commits to remote repositories (with optional remote and branch specification)
  * Pull changes from remote repositories (with optional remote and branch specification)
  * List all branches or create new branches
  * Switch between branches (checkout)
  * View commit history with customizable limits
  * View diffs to see what has changed (all changes or specific files)
  * All operations are sandboxed to a root directory for security
- Security:
  * All file paths are validated to ensure they stay within the root directory
  * Path traversal attempts (e.g., "../") are blocked
  * Operations are restricted to the configured root directory
  * Git repository must exist in the root directory
- Limitations:
  * Can only operate within the configured root directory
  * Cannot access git repositories outside the root directory
  * Git operations are synchronous
  * Requires git to be installed and accessible
- Integration: Automatically available as a sub-agent when git operations are needed`)

	// Add troubleshooting information
	template.AddTroubleshooting(`
Troubleshooting:
- "not a git repository": The directory is not a git repository - initialize with 'git init' first
- "path traversal detected": The file path attempts to escape the root directory - use relative paths only
- "missing required parameter: message": Commit message is required for commit operations
- "missing required parameter: branch": Branch name is required for checkout operations
- "git command failed": Git may not be installed or the repository may be in an invalid state
- Common mistakes:
  * Trying to commit without staging files first
  * Using absolute paths instead of relative paths
  * Trying to push/pull without a remote configured
  * Attempting operations on a non-existent branch
  * Forgetting to provide commit message
- Best practices:
  * Always check status before committing to see what will be committed
  * Use descriptive commit messages that explain the "what" and "why"
  * Verify branch exists before checking out
  * Check remote configuration before pushing/pulling
  * Review diffs before committing to ensure correct changes
  * Include relevant git information (branch, commit hash) in responses`)

	return template
}

// GitAgent creates a Git version control agent with git tools.
//
// Parameters:
//   - llmEngine: The LLM engine to use for this agent
//   - root: The root directory path that restricts all git operations
//
// Returns:
//   - *core.SubAgent: The Git agent as a sub-agent
func GitAgent(llmEngine llms.LLMEngine, root string) *core.SubAgent {
	gitTemplate := createGitAgentTemplate()
	gitConfig := gitTemplate.ToAgentConfig(llmEngine)

	// Add GitTool as the first tool
	gitTool := git.NewGitTool(root)
	gitConfig.Tools = []llms.Tool{gitTool}

	gitAgent := NewAgent(&gitConfig)
	return gitAgent.AgentAsSubAgent()
}
