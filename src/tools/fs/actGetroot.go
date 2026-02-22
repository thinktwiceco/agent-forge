package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// getRoot returns the absolute path of the sandbox root directory.
// This is the path that the agent can safely operate within.
// Returns detailed information about the root directory.
func (fs *Fs) getRoot() (string, error) {
	absRoot, err := filepath.Abs(fs.dir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path of root directory: %w", err)
	}

	// Get root directory info
	rootInfo, err := os.Stat(absRoot)
	if err != nil {
		return "", fmt.Errorf("failed to get root directory info: %w", err)
	}

	// Build detailed response
	response := &rootDirectoryResponse{
		AbsolutePath: absRoot,
		RelativePath: fs.dir,
		IsDirectory:  rootInfo.IsDir(),
		Permissions:  rootInfo.Mode().String(),
	}

	return response.String(), nil
}
