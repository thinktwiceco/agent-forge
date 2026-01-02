package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

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

const (
	// maxFileSize is the maximum file size (in bytes) that can be read.
	// Files larger than this will be rejected to prevent cluttering the agent's context.
	// Default: 1MB (1,048,576 bytes)
	maxFileSize = 1 << 20 // 1MB
)

// validateFileSize checks if a file's size is within the allowed limit.
// It accepts a file info object to check the size without requiring an additional stat call.
func (fs *Fs) validateFileSize(fileInfo os.FileInfo) error {
	if fileInfo.Size() > maxFileSize {
		return fmt.Errorf("file too large: %d bytes (maximum allowed: %d bytes). File reading is restricted to prevent cluttering the agent's context", fileInfo.Size(), maxFileSize)
	}

	return nil
}
