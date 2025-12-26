package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

type Fs struct {
	root string
}

// validatePath ensures that the given file path stays within the root directory.
// It returns the validated absolute path or an error if the path escapes the root.
func (fs *Fs) validatePath(filePath string) (string, error) {
	// Get absolute path of root
	absRoot, err := filepath.Abs(fs.root)
	if err != nil {
		return "", fmt.Errorf("invalid root directory: %w", err)
	}

	// Clean and join root with the provided path
	joinedPath := filepath.Join(absRoot, filepath.Clean(filePath))

	// Get absolute path of the joined path
	absPath, err := filepath.Abs(joinedPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check if the resolved path is within root
	relPath, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("path validation failed: %w", err)
	}

	// If relative path starts with "..", it means we escaped the root directory
	if len(relPath) >= 2 && relPath[:2] == ".." {
		return "", fmt.Errorf("path traversal detected: path '%s' escapes root directory", filePath)
	}

	return absPath, nil
}

// ReadFile reads the content of a file and returns detailed information about the operation.
// The path is validated to ensure it stays within the root directory.
func (fs *Fs) ReadFile(path string) (string, error) {
	validatedPath, err := fs.validatePath(path)
	if err != nil {
		return "", err
	}

	// Get file info before reading
	fileInfo, err := os.Stat(validatedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", path)
		}
		return "", fmt.Errorf("failed to get file info for '%s': %w", path, err)
	}

	content, err := os.ReadFile(validatedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file '%s': %w", path, err)
	}

	// Build detailed response
	modTime := fileInfo.ModTime().Format(time.RFC3339)
	info := fmt.Sprintf(`File Operation: Read
Path (relative): %s
Path (absolute): %s
Size: %d bytes
Modified: %s
Content:
---
%s
---`, path, validatedPath, fileInfo.Size(), modTime, string(content))

	return info, nil
}

// WriteFile writes content to a file, creating it if it doesn't exist.
// The path is validated to ensure it stays within the root directory.
// Returns detailed information about the file operation.
func (fs *Fs) WriteFile(path string, content string) (string, error) {
	validatedPath, err := fs.validatePath(path)
	if err != nil {
		return "", err
	}

	// Check if file already exists
	fileExists := false
	var existingInfo os.FileInfo
	if info, err := os.Stat(validatedPath); err == nil {
		fileExists = true
		existingInfo = info
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(validatedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory for '%s': %w", path, err)
	}

	err = os.WriteFile(validatedPath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file '%s': %w", path, err)
	}

	// Get file info after writing
	fileInfo, err := os.Stat(validatedPath)
	if err != nil {
		return "", fmt.Errorf("failed to get file info after write: %w", err)
	}

	// Build detailed response
	modTime := fileInfo.ModTime().Format(time.RFC3339)
	operation := "created"
	if fileExists {
		operation = "updated"
		if !existingInfo.ModTime().Equal(fileInfo.ModTime()) {
			operation = "updated (modified)"
		}
	}

	info := fmt.Sprintf(`File Operation: Write (%s)
Path (relative): %s
Path (absolute): %s
Size: %d bytes
Modified: %s
Directory: %s`, operation, path, validatedPath, fileInfo.Size(), modTime, dir)

	return info, nil
}

// DeleteFile deletes a file.
// The path is validated to ensure it stays within the root directory.
// Returns detailed information about the deletion operation.
func (fs *Fs) DeleteFile(path string) (string, error) {
	validatedPath, err := fs.validatePath(path)
	if err != nil {
		return "", err
	}

	// Get file info before deletion
	fileInfo, err := os.Stat(validatedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", path)
		}
		return "", fmt.Errorf("failed to get file info for '%s': %w", path, err)
	}

	err = os.Remove(validatedPath)
	if err != nil {
		return "", fmt.Errorf("failed to delete file '%s': %w", path, err)
	}

	// Build detailed response
	modTime := fileInfo.ModTime().Format(time.RFC3339)
	info := fmt.Sprintf(`File Operation: Delete
Path (relative): %s
Path (absolute): %s
Size: %d bytes
Last Modified: %s
Status: Successfully deleted`, path, validatedPath, fileInfo.Size(), modTime)

	return info, nil
}

// NewFsTool creates a file system tool that provides read, write, and delete operations.
// All file operations are restricted to the specified root directory for security.
//
// Parameters:
//   - root: The root directory path that restricts all file operations
func NewFsTool(root string) llms.Tool {
	fs := &Fs{root: root}

	return core.NewTool(
		"fs",
		"Perform file system operations (read, write, delete) on files within a restricted directory.",
		`Advanced Details:
- Parameters:
  * operation (string, required): The operation to perform - "read", "write", or "delete"
  * path (string, required): File path relative to the root directory
  * content (string, optional): File content - required for "write" operation
- Behavior:
  * All file paths are validated to ensure they stay within the root directory
  * Path traversal attempts (e.g., "../") are blocked for security
  * Read operation returns file content as a string
  * Write operation creates the file if it doesn't exist, and creates parent directories if needed
  * Delete operation removes the specified file
- Usage:
  * Use "read" to read file contents
  * Use "write" to create or update files (provide content parameter)
  * Use "delete" to remove files
- Security: All operations are sandboxed to the root directory to prevent unauthorized access`,
		`Troubleshooting:
- "path traversal detected": The provided path attempts to escape the root directory - use relative paths only
- "file not found": The file doesn't exist (for read/delete operations) - verify the path is correct
- "missing required parameter: content": Content parameter is required for write operations
- "invalid operation": Operation must be exactly "read", "write", or "delete"
- Permission errors: Ensure the process has read/write/delete permissions for the root directory
- "failed to create directory": Parent directory creation failed - check permissions`,
		[]core.Parameter{
			{
				Name:        "operation",
				Type:        "string",
				Description: "The operation to perform: 'read', 'write', or 'delete'",
				Required:    true,
			},
			{
				Name:        "path",
				Type:        "string",
				Description: "File path relative to the root directory",
				Required:    true,
			},
			{
				Name:        "content",
				Type:        "string",
				Description: "File content - required for 'write' operation",
				Required:    false,
			},
		},
		func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			operation := args["operation"].(string)
			path := args["path"].(string)

			// Validate operation
			if operation != "read" && operation != "write" && operation != "delete" {
				return core.NewErrorResponse(fmt.Sprintf(
					"invalid operation '%s'. Must be 'read', 'write', or 'delete'",
					operation,
				))
			}

			// Handle read operation
			if operation == "read" {
				info, err := fs.ReadFile(path)
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
				info, err := fs.WriteFile(path, contentStr)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(info)
			}

			// Handle delete operation
			if operation == "delete" {
				info, err := fs.DeleteFile(path)
				if err != nil {
					return core.NewErrorResponse(err.Error())
				}
				return core.NewSuccessResponse(info)
			}

			// This should never be reached, but included for completeness
			return core.NewErrorResponse(fmt.Sprintf("unhandled operation: %s", operation))
		},
	)
}
