package fs

import (
	"fmt"
	"os"
	"time"
)

// deleteFile deletes a file.
// The path is validated to ensure it stays within the root directory.
// Returns detailed information about the deletion operation.
func (fs *Fs) deleteFile(path string) (string, error) {
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
	response := &fileOperationResponse{
		Operation:    "Delete",
		RelativePath: path,
		AbsolutePath: validatedPath,
		Size:         fileInfo.Size(),
		LastModified: fileInfo.ModTime().Format(time.RFC3339),
		Status:       "Successfully deleted",
	}

	return response.String(), nil
}
