package git

import (
	"fmt"
	"path/filepath"
	"strings"
)

// add executes git add and returns information about staged files.
func (git *Git) add(path string) (string, error) {
	absRoot, err := filepath.Abs(git.root)
	if err != nil {
		return "", fmt.Errorf("invalid root directory: %w", err)
	}

	var args []string
	var targetPath string

	if path == "" {
		// Add all changes
		args = []string{"add", "."}
		targetPath = absRoot
	} else {
		// Validate and add specific path
		validatedPath, err := git.validatePath(path)
		if err != nil {
			return "", err
		}
		// Get relative path for display
		relPath, err := filepath.Rel(absRoot, validatedPath)
		if err != nil {
			relPath = path
		}
		args = []string{"add", relPath}
		targetPath = relPath
	}

	// Execute git add
	output, stderr, err := git.executeGitCommand(args...)
	if err != nil {
		return "", fmt.Errorf("failed to add files: %w (stderr: %s)", err, stderr)
	}

	// Get status to see what was staged
	statusOutput, _, err := git.executeGitCommand("status", "--porcelain")
	if err != nil {
		// Continue even if status fails
		statusOutput = ""
	}

	// Parse staged files from status
	var stagedFiles []string
	lines := strings.Split(strings.TrimSpace(statusOutput), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		status := line[:2]
		file := strings.TrimSpace(line[3:])
		// Staged files have non-space status in first column
		if status[0] != ' ' && status[0] != '?' {
			stagedFiles = append(stagedFiles, file)
		}
	}

	response := &gitAddResponse{
		Operation: "Add",
		Path:      targetPath,
		Files:     stagedFiles,
		Status:    output,
	}

	return response.String(), nil
}
