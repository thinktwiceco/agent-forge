package git

import (
	"fmt"
	"strings"
)

// commit executes git commit and returns commit information.
func (git *Git) commit(message string) (string, error) {
	// Execute git commit
	output, stderr, err := git.executeGitCommand("commit", "-m", message)
	if err != nil {
		return "", fmt.Errorf("failed to commit: %w (stderr: %s)", err, stderr)
	}

	// Get commit hash
	hashOutput, _, err := git.executeGitCommand("rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}
	hash := strings.TrimSpace(hashOutput)

	// Get current branch
	branchOutput, _, err := git.executeGitCommand("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	branch := strings.TrimSpace(branchOutput)

	// Get list of committed files
	filesOutput, _, err := git.executeGitCommand("diff-tree", "--no-commit-id", "--name-only", "-r", hash)
	if err != nil {
		// Continue even if this fails
		filesOutput = ""
	}
	var files []string
	if filesOutput != "" {
		files = strings.Split(strings.TrimSpace(filesOutput), "\n")
		// Filter empty strings
		var filteredFiles []string
		for _, f := range files {
			if f != "" {
				filteredFiles = append(filteredFiles, f)
			}
		}
		files = filteredFiles
	}

	response := &gitCommitResponse{
		Operation: "Commit",
		Hash:      hash,
		Message:   message,
		Branch:    branch,
		Files:     files,
		Status:    output,
	}

	return response.String(), nil
}
