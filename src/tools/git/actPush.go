package git

import (
	"fmt"
	"strings"
)

// push executes git push and returns push result.
func (git *Git) push(remote, branch string) (string, error) {
	// Default to origin if remote not specified
	if remote == "" {
		remote = "origin"
	}

	var args []string
	args = append(args, "push")

	// Add remote and branch if specified
	if branch != "" {
		args = append(args, remote, branch)
	} else if remote != "origin" {
		// If remote is specified but not branch, just push to remote
		args = append(args, remote)
	}

	// Execute git push
	output, stderr, err := git.executeGitCommand(args...)
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

	response := &gitPushResponse{
		Operation: "Push",
		Remote:    remote,
		Branch:    branch,
		Output:    fullOutput,
		Success:   success,
	}

	if !success {
		return response.String(), fmt.Errorf("git push failed: %w", err)
	}

	return response.String(), nil
}
