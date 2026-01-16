package git

import (
	"fmt"
	"strings"
)

// checkout executes git checkout and returns checkout result.
func (git *Git) checkout(branch string) (string, error) {
	// Execute git checkout
	output, stderr, err := git.executeGitCommand("checkout", branch)
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

	response := &gitCheckoutResponse{
		Operation: "Checkout",
		Branch:    branch,
		Output:    fullOutput,
		Success:   success,
	}

	if !success {
		return response.String(), fmt.Errorf("git checkout failed: %w", err)
	}

	return response.String(), nil
}
