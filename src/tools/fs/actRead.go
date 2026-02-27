package fs

import (
	"fmt"
	"os"
	"time"
)

// readFile reads the content of a file and returns detailed information about the operation.
// The path is validated to ensure it stays within the root directory.
func (fs *Fs) readFile(path string) (string, error) {
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

	// Validate file size before reading (reuse fileInfo to avoid redundant stat)
	if err := fs.validateFileSize(fileInfo); err != nil {
		return "", err
	}

	// Restrict to text documents only (no images, PDFs, etc.)
	if err := fs.validateTextFile(path); err != nil {
		return "", err
	}

	content, err := os.ReadFile(validatedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file '%s': %w", path, err)
	}

	// Build detailed response
	response := &fileOperationResponse{
		Operation:    "Read",
		RelativePath: path,
		AbsolutePath: validatedPath,
		Size:         fileInfo.Size(),
		Modified:     fileInfo.ModTime().Format(time.RFC3339),
		Content:      string(content),
	}

	return response.String(), nil
}
