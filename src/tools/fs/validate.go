package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// textExtensions is a whitelist of extensions for files that can be read as text.
// Binary files (images, PDFs, etc.) are excluded to avoid cluttering the agent's context.
var textExtensions = map[string]bool{
	".txt": true, ".md": true, ".json": true, ".yaml": true, ".yml": true,
	".xml": true, ".html": true, ".htm": true, ".css": true, ".js": true,
	".ts": true, ".tsx": true, ".jsx": true, ".go": true, ".py": true,
	".rb": true, ".rs": true, ".java": true, ".kt": true, ".c": true,
	".h": true, ".cpp": true, ".hpp": true, ".cs": true, ".php": true,
	".sh": true, ".bash": true, ".zsh": true, ".sql": true, ".csv": true,
	".toml": true, ".ini": true, ".cfg": true, ".conf": true, ".env": true,
	".log": true, ".lock": true, ".sum": true, ".mod": true,
}

// validatePath ensures that the given file path stays within the root directory.
// It returns the validated absolute path or an error if the path escapes the root.
func (fs *Fs) validatePath(filePath string) (string, error) {
	// Get absolute path of root
	absRoot, err := filepath.Abs(fs.dir)
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

// validateTextFile ensures the file has a text extension. Binary files (images, PDFs, etc.)
// are rejected to avoid cluttering the agent's context.
func (fs *Fs) validateTextFile(filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))
	if !textExtensions[ext] {
		base := filepath.Base(filePath)
		typeDesc := strings.TrimPrefix(ext, ".")
		if typeDesc == "" {
			typeDesc = "no extension"
		} else {
			typeDesc = "." + typeDesc
		}
		return fmt.Errorf("cannot read '%s': %s is not a text document. The fs read operation accepts only text files (.txt, .md, .json, .yaml, .go, .py, etc.). For images, use the image tool instead", base, typeDesc)
	}
	return nil
}
