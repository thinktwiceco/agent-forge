package fs

import (
	"fmt"
	"os"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// toInt converts an interface{} value (number) to int, returning 0 on failure.
func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

// Fs represents a file system tool operating within a specific directory.
type Fs struct {
	// dir is the directory this tool is sandboxed to (agent working_dir).
	dir string
}

// NewFsTool creates a file system tool that provides read, write, and delete operations.
// All file operations are restricted to dir.
//
// Parameters:
//   - dir: The directory this tool operates in (agent working_dir).
func NewFsTool(dir string) llms.Tool {
	_ = os.MkdirAll(dir, 0755)
	fs := &Fs{dir: dir}

	detailsAbout := func(item string) string {
		switch item {
		case "read":
			return `read: Read the contents of a file.
- Required: path (string) — file path relative to the root directory
- Returns: file content as a string`
		case "write":
			return `write: Create or overwrite a file (parent directories are created automatically).
- Required: path (string) — file path relative to the root directory
- Required: content (string) — content to write to the file`
		case "delete":
			return `delete: Remove a file.
- Required: path (string) — file path relative to the root directory`
		case "list":
			return `list: List all files and directories inside a directory.
- Required: path (string) — directory path relative to the root directory`
		case "get_file_info":
			return `get_file_info: Return detailed file information (permissions, ownership, size, timestamps).
- Required: path (string) — file path relative to the root directory`
		case "get_root":
			return `get_root: Return the sandbox root directory path. No additional parameters required.`
		case "ripgrep":
			return `ripgrep: Search for a pattern in files using ripgrep (rg must be installed).
- Required: path (string) — file or directory path relative to root
- Required: pattern (string) — search pattern (regex supported)
- Optional: flags (array of strings) — additional rg flags, e.g. ["-i", "-n", "--color=never"]
- Optional: head_limit (integer, default 0) — max match lines to return; 0 = no limit
- Optional: offset (integer, default 0) — match lines to skip; use with head_limit for pagination`
		case "grep_logs":
			return `grep_logs: Search the application log file for a pattern (requires AF_LOG_FILE env var).
- Required: pattern (string) — search pattern
- Optional: flags (array of strings) — additional rg flags
- Optional: head_limit (integer, default 0) — max match lines to return
- Optional: offset (integer, default 0) — match lines to skip`
		default:
			return fmt.Sprintf("Nothing to add about %s", item)
		}
	}

	return &core.Tool{
		Name:        "fs",
		Description: "Perform file system operations (read, write, delete, list, ripgrep) on files within a restricted directory.",
		AdvanceDesc: `Advanced Details:
- Available operations: read, write, delete, list, get_file_info, get_root, ripgrep, grep_logs
  Use expand tool with details_about="<operation>" for full parameter details on any operation.
- Common parameters:
  * operation (string, required): the operation to perform
  * path (string): file or directory path relative to the root — required for most operations
- Behavior:
  * All paths are validated and sandboxed to the root directory; path traversal ("../") is blocked
  * write creates parent directories automatically
  * ripgrep/grep_logs results support pagination via head_limit and offset`,
		DetailsAboutFunc: detailsAbout,
		TroubleshootingInfo: `Troubleshooting:
- "path traversal detected": The provided path attempts to escape the root directory - use relative paths only
- "file not found": The file doesn't exist (for read/delete/get_file_info operations) - verify the path is correct
- "directory not found": The directory doesn't exist (for list operation) - verify the path is correct
- "path is not a directory": The path exists but is not a directory (for list operation)
- "missing required parameter: content": Content parameter is required for write operations
- "missing required parameter: pattern": Pattern parameter is required for ripgrep/grep_logs operations
- "ripgrep (rg) is not installed": ripgrep must be installed and in PATH for ripgrep operations
- "invalid operation": Operation must be exactly "read", "write", "delete", "list", "get_file_info", "get_root", "ripgrep", or "grep_logs"
- "grep_logs is not available": Set AF_LOG_FILE to enable file logging and grep_logs
- Permission errors: Ensure the process has read/write/delete permissions for the root directory
- "failed to create directory": Parent directory creation failed - check permissions
- "missing required parameter: path": Path parameter is required for read/write/delete/list/get_file_info/ripgrep operations (not needed for get_root, grep_logs)
- Large output: Use head_limit (e.g., 100) to cap results and offset to page through them`,
		Parameters: []core.Parameter{
			{
				Name:        "operation",
				Type:        "string",
				Description: "The operation to perform: 'read', 'write', 'delete', 'list', 'get_file_info', 'get_root', 'ripgrep', or 'grep_logs'",
				Required:    true,
			},
			{
				Name:        "path",
				Type:        "string",
				Description: "File or directory path relative to the root directory (required for 'read', 'write', 'delete', 'list', 'get_file_info', 'ripgrep'; not needed for 'get_root', 'grep_logs')",
				Required:    false,
			},
			{
				Name:        "content",
				Type:        "string",
				Description: "File content - required for 'write' operation",
				Required:    false,
			},
			{
				Name:        "pattern",
				Type:        "string",
				Description: "Search pattern - required for 'ripgrep' and 'grep_logs' operations",
				Required:    false,
			},
			{
				Name:        "flags",
				Type:        "array",
				Description: "Additional ripgrep flags - optional for 'ripgrep' and 'grep_logs' operations (e.g., [\"-i\", \"-n\", \"--color=never\"])",
				Required:    false,
			},
			{
				Name:        "head_limit",
				Type:        "integer",
				Description: "Maximum number of match lines to return for 'ripgrep' and 'grep_logs' operations; 0 = no limit (default 0). Use with 'offset' for pagination.",
				Required:    false,
			},
			{
				Name:        "offset",
				Type:        "integer",
				Description: "Number of match lines to skip before returning results for 'ripgrep' and 'grep_logs' operations (default 0). Use with 'head_limit' for pagination.",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			operation := args["operation"].(string)

			// Validate operation
			if operation != "read" && operation != "write" && operation != "delete" && operation != "list" && operation != "get_file_info" && operation != "get_root" && operation != "ripgrep" && operation != "grep_logs" {
				return core.NewErrorResponse(fmt.Sprintf(
					"invalid operation '%s'. Must be 'read', 'write', 'delete', 'list', 'get_file_info', 'get_root', 'ripgrep', or 'grep_logs'",
					operation,
				))
			}

			// Handle get_root operation (doesn't require path)
			if operation == "get_root" {
				info, err := fs.getRoot()
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(info)
			}

			// Handle grep_logs operation (doesn't require path, uses AF_LOG_FILE)
			if operation == "grep_logs" {
				pattern, ok := args["pattern"]
				if !ok {
					return core.NewErrorResponse("missing required parameter: pattern (required for grep_logs operation)")
				}
				patternStr, ok := pattern.(string)
				if !ok {
					return core.NewErrorResponse("pattern parameter must be a string")
				}

				var flags []string
				if flagsArg, ok := args["flags"]; ok {
					if flagsArray, ok := flagsArg.([]interface{}); ok {
						for _, flag := range flagsArray {
							if flagStr, ok := flag.(string); ok {
								flags = append(flags, flagStr)
							}
						}
					}
				}

				offset := 0
				if v, ok := args["offset"]; ok {
					offset = toInt(v)
				}
				headLimit := 0
				if v, ok := args["head_limit"]; ok {
					headLimit = toInt(v)
				}

				info, err := fs.grepLogs(patternStr, flags, offset, headLimit)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(info)
			}

			// For other operations, path is required
			path, ok := args["path"]
			if !ok {
				return core.NewErrorResponse("missing required parameter: path (required for read/write/delete/list operations)")
			}
			pathStr, ok := path.(string)
			if !ok {
				return core.NewErrorResponse("path parameter must be a string")
			}

			// Handle read operation
			if operation == "read" {
				info, err := fs.readFile(pathStr)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(info)
			}

			// Handle write operation
			if operation == "write" {
				content, ok := args["content"]
				if !ok {
					return core.NewErrorResponse("missing required parameter: content (required for write operation)")
				}
				contentStr, ok := content.(string)
				if !ok {
					return core.NewErrorResponse("content parameter must be a string")
				}
				info, err := fs.writeFile(pathStr, contentStr)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(info)
			}

			// Handle delete operation
			if operation == "delete" {
				info, err := fs.deleteFile(pathStr)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(info)
			}

			// Handle list operation
			if operation == "list" {
				info, err := fs.listFiles(pathStr)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(info)
			}

			// Handle get_file_info operation
			if operation == "get_file_info" {
				info, err := fs.getFileInfo(pathStr)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(info)
			}

			// Handle ripgrep operation
			if operation == "ripgrep" {
				pattern, ok := args["pattern"]
				if !ok {
					return core.NewErrorResponse("missing required parameter: pattern (required for ripgrep operation)")
				}
				patternStr, ok := pattern.(string)
				if !ok {
					return core.NewErrorResponse("pattern parameter must be a string")
				}

				// Extract flags if provided
				var flags []string
				if flagsArg, ok := args["flags"]; ok {
					if flagsArray, ok := flagsArg.([]interface{}); ok {
						for _, flag := range flagsArray {
							if flagStr, ok := flag.(string); ok {
								flags = append(flags, flagStr)
							}
						}
					}
				}

				offset := 0
				if v, ok := args["offset"]; ok {
					offset = toInt(v)
				}
				headLimit := 0
				if v, ok := args["head_limit"]; ok {
					headLimit = toInt(v)
				}

				info, err := fs.ripgrep(pathStr, patternStr, flags, offset, headLimit)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(info)
			}

			// This should never be reached, but included for completeness
			return core.NewErrorResponse(fmt.Sprintf("unhandled operation: %s", operation))
		},
	}
}
