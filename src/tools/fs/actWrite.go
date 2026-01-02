package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// writeFile writes content to a file, creating it if it doesn't exist.
// The path is validated to ensure it stays within the root directory.
// Returns detailed information about the file operation.
func (fs *Fs) writeFile(path string, content string) (string, error) {
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

	// Determine operation status
	status := "created"
	if fileExists {
		status = "updated"
		if !existingInfo.ModTime().Equal(fileInfo.ModTime()) {
			status = "updated (modified)"
		}
	}

	// Build detailed response
	response := &fileOperationResponse{
		Operation:    "Write",
		RelativePath: path,
		AbsolutePath: validatedPath,
		Size:         fileInfo.Size(),
		Modified:     fileInfo.ModTime().Format(time.RFC3339),
		Directory:    dir,
		Status:       status,
	}

	return response.String(), nil
}
