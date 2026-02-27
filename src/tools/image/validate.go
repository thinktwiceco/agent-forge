package image

import (
	"fmt"
	"path/filepath"
	"strings"
)

var supportedMIMETypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
}

// validatePath ensures the given file path stays within the root directory.
// Returns the validated absolute path or an error if the path escapes root.
func (t *ImageTool) validatePath(filePath string) (string, error) {
	absRoot, err := filepath.Abs(t.dir)
	if err != nil {
		return "", fmt.Errorf("invalid root directory: %w", err)
	}

	joinedPath := filepath.Join(absRoot, filepath.Clean(filePath))

	absPath, err := filepath.Abs(joinedPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	relPath, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("path validation failed: %w", err)
	}

	if len(relPath) >= 2 && relPath[:2] == ".." {
		return "", fmt.Errorf("path traversal detected: path '%s' escapes root directory", filePath)
	}

	return absPath, nil
}

// mimeTypeForPath returns the MIME type for a given file path based on its extension.
// Returns an error if the extension is not a supported image format.
func mimeTypeForPath(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	mime, ok := supportedMIMETypes[ext]
	if !ok {
		return "", fmt.Errorf("unsupported image format '%s': must be one of jpg, jpeg, png, gif, webp", ext)
	}
	return mime, nil
}
