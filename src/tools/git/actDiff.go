package git

import (
	"fmt"
	"path/filepath"
	"strings"
)

// diff executes git diff and returns diff output.
func (git *Git) diff(path string) (string, error) {
	absRoot, err := filepath.Abs(git.dir)
	if err != nil {
		return "", fmt.Errorf("invalid root directory: %w", err)
	}

	var args []string
	var targetPath string

	if path == "" {
		// Diff all changes
		args = []string{"diff"}
		targetPath = absRoot
	} else {
		// Validate and diff specific path
		validatedPath, err := git.validatePath(path)
		if err != nil {
			return "", err
		}
		// Get relative path for display
		relPath, err := filepath.Rel(absRoot, validatedPath)
		if err != nil {
			relPath = path
		}
		args = []string{"diff", relPath}
		targetPath = relPath
	}

	// Execute git diff
	diffOutput, stderr, err := git.executeGitCommand(args...)
	if err != nil {
		// Git diff returns exit code 1 when there are differences, which is normal
		// Check if it's a real error by examining stderr
		if stderr != "" && !strings.Contains(err.Error(), "exit status 1") {
			return "", fmt.Errorf("failed to get git diff: %w (stderr: %s)", err, stderr)
		}
		// If exit status 1, treat as success (differences found)
		if strings.Contains(err.Error(), "exit status 1") {
			err = nil
		}
	}

	// Include stderr in output if present (git diff may output warnings to stderr)
	if stderr != "" && strings.TrimSpace(diffOutput) == "" {
		diffOutput = stderr
	} else if stderr != "" {
		diffOutput = diffOutput + "\n" + stderr
	}

	hasChanges := strings.TrimSpace(diffOutput) != ""

	response := &gitDiffResponse{
		Operation:  "Diff",
		Path:       targetPath,
		Diff:       diffOutput,
		HasChanges: hasChanges,
	}

	return response.String(), nil
}
