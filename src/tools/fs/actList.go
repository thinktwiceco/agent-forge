package fs

import (
	"fmt"
	"os"
	"time"
)

// listFiles lists all files and directories in the specified directory.
// The path is validated to ensure it stays within the root directory.
// Returns detailed information about each entry in the directory.
func (fs *Fs) listFiles(path string) (string, error) {
	validatedPath, err := fs.validatePath(path)
	if err != nil {
		return "", err
	}

	// Check if path exists and is a directory
	dirInfo, err := os.Stat(validatedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("directory not found: %s", path)
		}
		return "", fmt.Errorf("failed to get directory info for '%s': %w", path, err)
	}

	if !dirInfo.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", path)
	}

	// Read directory entries
	entries, err := os.ReadDir(validatedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory '%s': %w", path, err)
	}

	// Build file entries list
	fileEntries := make([]fileEntry, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			// Skip entries we can't get info for
			continue
		}

		entryType := "FILE"
		if entry.IsDir() {
			entryType = "DIR "
		}

		fileEntries = append(fileEntries, fileEntry{
			Name:     entry.Name(),
			Type:     entryType,
			Size:     info.Size(),
			Modified: info.ModTime().Format(time.RFC3339),
			IsDir:    entry.IsDir(),
		})
	}

	// Build response
	response := &listFilesResponse{
		Directory:    validatedPath,
		RelativePath: path,
		AbsolutePath: validatedPath,
		Entries:      fileEntries,
		Count:        len(fileEntries),
	}

	return response.String(), nil
}
