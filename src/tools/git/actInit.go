package git

import (
	"fmt"
	"path/filepath"
	"strings"
)

// init executes git init and returns initialization result.
func (git *Git) init() (string, error) {
	absRoot, err := filepath.Abs(git.root)
	if err != nil {
		return "", fmt.Errorf("invalid root directory: %w", err)
	}

	// Execute git init
	output, stderr, err := git.executeGitCommand("init")
	success := err == nil

	// Combine output and stderr
	fullOutput := strings.TrimSpace(output)
	if stderr != "" {
		if fullOutput != "" {
			fullOutput += "\n" + strings.TrimSpace(stderr)
		} else {
			fullOutput = strings.TrimSpace(stderr)
		}
	}

	// Get default branch name if initialization was successful
	var branch string
	if success {
		// Try to get the default branch name
		// Note: After init, HEAD might not exist until first commit, so this might fail
		branchOutput, _, branchErr := git.executeGitCommand("rev-parse", "--abbrev-ref", "HEAD")
		if branchErr == nil {
			branch = strings.TrimSpace(branchOutput)
		} else {
			// Try alternative method
			branchOutput, _, branchErr := git.executeGitCommand("branch", "--show-current")
			if branchErr == nil {
				branch = strings.TrimSpace(branchOutput)
			} else {
				// Default to "main" if we can't determine (modern git default)
				branch = "main"
			}
		}
	}

	response := &gitInitResponse{
		Operation:  "Init",
		WorkingDir: absRoot,
		Branch:     branch,
		Success:    success,
		Output:     fullOutput,
	}

	if !success {
		return response.String(), fmt.Errorf("git init failed: %w", err)
	}

	return response.String(), nil
}
