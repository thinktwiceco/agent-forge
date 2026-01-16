package git

import (
	"fmt"
	"strings"
)

// pull executes git pull and returns pull result.
func (git *Git) pull(remote, branch string) (string, error) {
	// Default to origin if remote not specified
	if remote == "" {
		remote = "origin"
	}

	var args []string
	args = append(args, "pull")

	// Add remote and branch if specified
	if branch != "" {
		args = append(args, remote, branch)
	} else if remote != "origin" {
		// If remote is specified but not branch, just pull from remote
		args = append(args, remote)
	}

	// Execute git pull
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

	response := &gitPullResponse{
		Operation: "Pull",
		Remote:    remote,
		Branch:    branch,
		Output:    fullOutput,
		Success:   success,
	}

	if !success {
		return response.String(), fmt.Errorf("git pull failed: %w", err)
	}

	return response.String(), nil
}
