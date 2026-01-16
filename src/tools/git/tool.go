package git

import (
	"fmt"

	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// Git represents a git tool with a restricted root directory.
type Git struct {
	root string
}

// NewGitTool creates a git tool that provides git operations.
// All git operations are restricted to the specified root directory for security.
//
// Parameters:
//   - root: The root directory path that restricts all git operations
func NewGitTool(root string) llms.Tool {
	git := &Git{root: root}

	return &core.Tool{
		Name:        "git",
		Description: "Perform git operations (init, status, add, commit, push, pull, branch, checkout, log, diff) within a restricted directory.",
		AdvanceDesc: `Advanced Details:
- Parameters:
  * operation (string, required): The operation to perform - "init", "status", "add", "commit", "push", "pull", "branch", "checkout", "log", or "diff"
  * path (string, optional): File or directory path relative to the root directory (for add, diff operations)
  * message (string, optional): Commit message - required for "commit" operation
  * branch (string, optional): Branch name - required for "checkout" operation, optional for "branch", "push", "pull" operations
  * remote (string, optional): Remote name - optional for "push" and "pull" operations (defaults to "origin")
  * limit (number, optional): Number of commits to show - optional for "log" operation (defaults to 10)
- Behavior:
  * All operations are executed within the root directory
  * Git repository must exist in the root directory (except for "init" operation which creates it)
  * Path traversal attempts (e.g., "../") are blocked for security
  * Commands are executed using git CLI
- Usage:
  * Use "init" to initialize a new git repository in the root directory
  * Use "status" to check working tree status
  * Use "add" to stage files (provide path parameter, or omit to stage all changes)
  * Use "commit" to commit staged changes (provide message parameter)
  * Use "push" to push commits to remote (optionally provide remote and branch)
  * Use "pull" to pull changes from remote (optionally provide remote and branch)
  * Use "branch" to list branches or create a new branch (optionally provide branch name to create)
  * Use "checkout" to switch branches (provide branch parameter)
  * Use "log" to view commit history (optionally provide limit parameter)
  * Use "diff" to view changes (optionally provide path parameter)
- Security: All operations are sandboxed to the root directory to prevent unauthorized access`,
		TroubleshootingInfo: `Troubleshooting:
- "not a git repository": The root directory is not a git repository - use "init" operation to initialize it first
- "path traversal detected": The provided path attempts to escape the root directory - use relative paths only
- "missing required parameter: message": Message parameter is required for commit operations
- "missing required parameter: branch": Branch parameter is required for checkout operations
- "invalid operation": Operation must be exactly "init", "status", "add", "commit", "push", "pull", "branch", "checkout", "log", or "diff"
- Git command errors: Check that git is installed and accessible, and that the repository is in a valid state
- Permission errors: Ensure the process has read/write permissions for the root directory
- Init on existing repository: Running "init" on an existing repository is safe and will not overwrite existing data`,
		Parameters: []core.Parameter{
			{
				Name:        "operation",
				Type:        "string",
				Description: "The operation to perform: 'init', 'status', 'add', 'commit', 'push', 'pull', 'branch', 'checkout', 'log', or 'diff'",
				Required:    true,
				Validator:   validateOperation,
			},
			{
				Name:        "path",
				Type:        "string",
				Description: "File or directory path relative to the root directory (for 'add' and 'diff' operations)",
				Required:    false,
			},
			{
				Name:        "message",
				Type:        "string",
				Description: "Commit message - required for 'commit' operation",
				Required:    false,
			},
			{
				Name:        "branch",
				Type:        "string",
				Description: "Branch name - required for 'checkout' operation, optional for 'branch', 'push', 'pull' operations",
				Required:    false,
			},
			{
				Name:        "remote",
				Type:        "string",
				Description: "Remote name - optional for 'push' and 'pull' operations (defaults to 'origin')",
				Required:    false,
			},
			{
				Name:        "limit",
				Type:        "number",
				Description: "Number of commits to show - optional for 'log' operation (defaults to 10)",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			operation := args["operation"].(string)

			// Handle init operation (bypasses repository validation since it creates the repo)
			if operation == "init" {
				result, err := git.init()
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(result)
			}

			// Validate git repository exists for all other operations
			if err := git.validateGitRepo(); err != nil {
				return core.NewErrorResponse(err.Error())
			}

			// Handle status operation
			if operation == "status" {
				result, err := git.status()
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(result)
			}

			// Handle add operation
			if operation == "add" {
				path, _ := args["path"].(string)
				result, err := git.add(path)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(result)
			}

			// Handle commit operation
			if operation == "commit" {
				message, ok := args["message"]
				if !ok {
					return core.NewErrorResponse("missing required parameter: message (required for commit operation)")
				}
				messageStr, ok := message.(string)
				if !ok {
					return core.NewErrorResponse("message parameter must be a string")
				}
				result, err := git.commit(messageStr)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(result)
			}

			// Handle push operation
			if operation == "push" {
				remote, _ := args["remote"].(string)
				branch, _ := args["branch"].(string)
				result, err := git.push(remote, branch)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(result)
			}

			// Handle pull operation
			if operation == "pull" {
				remote, _ := args["remote"].(string)
				branch, _ := args["branch"].(string)
				result, err := git.pull(remote, branch)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(result)
			}

			// Handle branch operation
			if operation == "branch" {
				branch, _ := args["branch"].(string)
				result, err := git.branch(branch)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(result)
			}

			// Handle checkout operation
			if operation == "checkout" {
				branch, ok := args["branch"]
				if !ok {
					return core.NewErrorResponse("missing required parameter: branch (required for checkout operation)")
				}
				branchStr, ok := branch.(string)
				if !ok {
					return core.NewErrorResponse("branch parameter must be a string")
				}
				result, err := git.checkout(branchStr)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(result)
			}

			// Handle log operation
			if operation == "log" {
				var limit int = 10
				if limitVal, ok := args["limit"]; ok {
					switch v := limitVal.(type) {
					case float64:
						limit = int(v)
					case int:
						limit = v
					case int64:
						limit = int(v)
					}
				}
				result, err := git.log(limit)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(result)
			}

			// Handle diff operation
			if operation == "diff" {
				path, _ := args["path"].(string)
				result, err := git.diff(path)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(result)
			}

			// This should never be reached due to validation, but included for completeness
			return core.NewErrorResponse(fmt.Sprintf("unhandled operation: %s", operation))
		},
	}
}
