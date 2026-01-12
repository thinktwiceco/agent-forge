package knowledge

import (
	"fmt"
	"os"

	agentforge "github.com/thinktwice/agentForge/src"
)

// readFile reads a file and returns its contents
// This is a helper function to keep file operations internal
func readFile(path string) ([]byte, error) {
	agentforge.Info("Loading document: %s", path)

	doc, err := os.ReadFile(path)
	if err != nil {
		agentforge.Error("Failed to read document %s: %v", path, err)
		return nil, err
	}

	return doc, nil
}

// readFileWithLimit reads a file with a size limit to prevent OOM
func readFileWithLimit(path string, maxSize int64) ([]byte, error) {
	agentforge.Info("Loading document: %s", path)

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.Size() > maxSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", fileInfo.Size(), maxSize)
	}

	doc, err := os.ReadFile(path)
	if err != nil {
		agentforge.Error("Failed to read document %s: %v", path, err)
		return nil, err
	}

	return doc, nil
}
