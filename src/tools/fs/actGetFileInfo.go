package fs

import (
	"fmt"
	"os"

	"time"
)

// getFileInfo retrieves detailed information about a file or directory.
// The path is validated to ensure it stays within the root directory.
// Returns information including permissions, ownership, size, and timestamps.
func (fs *Fs) getFileInfo(path string) (string, error) {
	validatedPath, err := fs.validatePath(path)
	if err != nil {
		return "", err
	}

	// Get file info
	fileInfo, err := os.Stat(validatedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", path)
		}
		return "", fmt.Errorf("failed to get file info for '%s': %w", path, err)
	}

	// Get ownership information (UID/GID)
	// Get ownership information (UID/GID)
	uid, gid := getOwnerInfo(fileInfo)

	// Build detailed response
	response := &fileInfoResponse{
		RelativePath: path,
		AbsolutePath: validatedPath,
		Size:         fileInfo.Size(),
		Permissions:  fileInfo.Mode().String(),
		Mode:         fmt.Sprintf("%04o", fileInfo.Mode().Perm()),
		IsDirectory:  fileInfo.IsDir(),
		Modified:     fileInfo.ModTime().Format(time.RFC3339),
		UID:          uid,
		GID:          gid,
	}

	return response.String(), nil
}
