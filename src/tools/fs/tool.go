package fs

import (
	"fmt"

	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// Fs represents a file system tool with a restricted root directory.
type Fs struct {
	root string
}

// NewFsTool creates a file system tool that provides read, write, and delete operations.
// All file operations are restricted to the specified root directory for security.
//
// Parameters:
//   - root: The root directory path that restricts all file operations
func NewFsTool(root string) llms.Tool {
	fs := &Fs{root: root}

	return &core.Tool{
		Name:        "fs",
		Description: "Perform file system operations (read, write, delete, list) on files within a restricted directory.",
		AdvanceDesc: `Advanced Details:
- Parameters:
  * operation (string, required): The operation to perform - "read", "write", "delete", "list", "get_file_info", or "get_root"
  * path (string, required for read/write/delete/list/get_file_info): File or directory path relative to the root directory
  * content (string, optional): File content - required for "write" operation
- Behavior:
  * All file paths are validated to ensure they stay within the root directory
  * Path traversal attempts (e.g., "../") are blocked for security
  * Read operation returns file content as a string
  * Write operation creates the file if it doesn't exist, and creates parent directories if needed
  * Delete operation removes the specified file
  * List operation lists all files and directories in the specified directory
  * GetFileInfo operation returns detailed file information (permissions, ownership, size, timestamps)
  * GetRoot operation returns the sandbox root directory path
- Usage:
  * Use "read" to read file contents
  * Use "write" to create or update files (provide content parameter)
  * Use "delete" to remove files
  * Use "list" to list files and directories in a directory
  * Use "get_file_info" to retrieve detailed file information (permissions, ownership, size, timestamps)
  * Use "get_root" to retrieve the sandbox root directory path (no path parameter needed)
- Security: All operations are sandboxed to the root directory to prevent unauthorized access`,
		TroubleshootingInfo: `Troubleshooting:
- "path traversal detected": The provided path attempts to escape the root directory - use relative paths only
- "file not found": The file doesn't exist (for read/delete/get_file_info operations) - verify the path is correct
- "directory not found": The directory doesn't exist (for list operation) - verify the path is correct
- "path is not a directory": The path exists but is not a directory (for list operation)
- "missing required parameter: content": Content parameter is required for write operations
- "invalid operation": Operation must be exactly "read", "write", "delete", "list", "get_file_info", or "get_root"
- Permission errors: Ensure the process has read/write/delete permissions for the root directory
- "failed to create directory": Parent directory creation failed - check permissions
- "missing required parameter: path": Path parameter is required for read/write/delete/list/get_file_info operations (not needed for get_root)`,
		Parameters: []core.Parameter{
			{
				Name:        "operation",
				Type:        "string",
				Description: "The operation to perform: 'read', 'write', 'delete', 'list', 'get_file_info', or 'get_root'",
				Required:    true,
			},
			{
				Name:        "path",
				Type:        "string",
				Description: "File or directory path relative to the root directory (required for 'read', 'write', 'delete', 'list', 'get_file_info'; not needed for 'get_root')",
				Required:    false,
			},
			{
				Name:        "content",
				Type:        "string",
				Description: "File content - required for 'write' operation",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			operation := args["operation"].(string)

			// Validate operation
			if operation != "read" && operation != "write" && operation != "delete" && operation != "list" && operation != "get_file_info" && operation != "get_root" {
				return core.NewErrorResponse(fmt.Sprintf(
					"invalid operation '%s'. Must be 'read', 'write', 'delete', 'list', 'get_file_info', or 'get_root'",
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

			// This should never be reached, but included for completeness
			return core.NewErrorResponse(fmt.Sprintf("unhandled operation: %s", operation))
		},
	}
}
