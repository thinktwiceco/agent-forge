package git

import (
	"fmt"
	"os"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// Git represents a git tool operating within a specific directory.
type Git struct {
	// dir is the directory this tool operates in (agent working_dir/repos).
	dir string
}

// NewGitTool creates a git tool that provides git operations.
// All git operations are restricted to dir.
//
// Parameters:
//   - dir: The directory this tool operates in (agent working_dir/repos).
func NewGitTool(dir string) llms.Tool {
	_ = os.MkdirAll(dir, 0755)
	git := &Git{dir: dir}

	detailsAbout := func(item string) string {
		switch item {
		case "init":
			return `init: Initialize a new git repository in the root directory. No additional parameters required.`
		case "status":
			return `status: Show the working tree status. No additional parameters required.`
		case "add":
			return `add: Stage files for commit.
- Optional: path (string) — file or directory path relative to root; omit to stage all changes`
		case "commit":
			return `commit: Commit staged changes.
- Required: message (string) — commit message`
		case "push":
			return `push: Push commits to a remote.
- Optional: remote (string, default "origin") — remote name
- Optional: branch (string) — branch name`
		case "pull":
			return `pull: Pull changes from a remote.
- Optional: remote (string, default "origin") — remote name
- Optional: branch (string) — branch name`
		case "branch":
			return `branch: List all branches or create a new branch.
- Optional: branch (string) — if provided, creates a new branch with this name`
		case "checkout":
			return `checkout: Switch to a branch.
- Required: branch (string) — branch name to switch to`
		case "log":
			return `log: View commit history.
- Optional: limit (number, default 10) — number of commits to show`
		case "diff":
			return `diff: View changes in the working tree.
- Optional: path (string) — file or directory path relative to root to diff`
		case "clone":
			return `clone: Clone a repository into a new directory.
- Required: url (string) — repository URL to clone
- Optional: directory (string) — target directory name; if omitted, uses repository name from URL`
		default:
			return fmt.Sprintf("Nothing to add about %s", item)
		}
	}

	return core.NewTool(core.ToolConfig{
		Name:        "git",
		Description: "Perform git operations (init, status, add, commit, push, pull, branch, checkout, log, diff, clone) within a restricted directory.",
		AdvanceDesc: `Advanced Details:
- Available operations: init, status, add, commit, push, pull, branch, checkout, log, diff, clone
  Use expand tool with details_about="<operation>" for full parameter details on any operation.
- Common parameters:
  * operation (string, required): the operation to perform
- Behavior:
  * All operations run within the root directory; path traversal ("../") is blocked
  * A git repository must already exist for all operations except "init" and "clone"
  * Commands delegate to the git CLI`,
		DetailsAboutFunc: detailsAbout,
		TroubleshootingInfo: `Troubleshooting:
- "not a git repository": The root directory is not a git repository - use "init" operation to initialize it first
- "path traversal detected": The provided path attempts to escape the root directory - use relative paths only
- "missing required parameter: message": Message parameter is required for commit operations
- "missing required parameter: branch": Branch parameter is required for checkout operations
- "missing required parameter: url": URL parameter is required for clone operations
- "invalid operation": Operation must be exactly "init", "status", "add", "commit", "push", "pull", "branch", "checkout", "log", "diff", or "clone"
- Git command errors: Check that git is installed and accessible, and that the repository is in a valid state
- Permission errors: Ensure the process has read/write permissions for the root directory
- Init on existing repository: Running "init" on an existing repository is safe and will not overwrite existing data`,
		Parameters: []core.Parameter{
			{
				Name:        "operation",
				Type:        "string",
				Description: "The operation to perform: 'init', 'status', 'add', 'commit', 'push', 'pull', 'branch', 'checkout', 'log', 'diff', or 'clone'",
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
			{
				Name:        "url",
				Type:        "string",
				Description: "Repository URL - required for 'clone' operation",
				Required:    false,
			},
			{
				Name:        "directory",
				Type:        "string",
				Description: "Target directory name - optional for 'clone' operation (defaults to repository name from URL)",
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

			// Handle clone operation (bypasses repository validation since it creates the repo)
			if operation == "clone" {
				url, ok := args["url"]
				if !ok {
					return core.NewErrorResponse("missing required parameter: url (required for clone operation)")
				}
				urlStr, ok := url.(string)
				if !ok {
					return core.NewErrorResponse("url parameter must be a string")
				}
				directory, _ := args["directory"].(string)
				result, err := git.clone(urlStr, directory)
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
				limit := 10
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
	})
}
