package git

import (
	"fmt"
	"path/filepath"
	"strings"
)

// status executes git status and returns formatted status information.
func (git *Git) status() (string, error) {
	absRoot, err := filepath.Abs(git.root)
	if err != nil {
		return "", fmt.Errorf("invalid root directory: %w", err)
	}

	// Get full status
	statusOutput, _, err := git.executeGitCommand("status")
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}

	// Get porcelain status for parsing
	porcelainOutput, _, err := git.executeGitCommand("status", "--porcelain")
	if err != nil {
		return "", fmt.Errorf("failed to get git status porcelain: %w", err)
	}

	// Get current branch
	branchOutput, _, err := git.executeGitCommand("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	branch := strings.TrimSpace(branchOutput)

	// Parse porcelain output
	var stagedFiles []string
	var modified []string
	var untracked []string

	lines := strings.Split(strings.TrimSpace(porcelainOutput), "\n")
	hasChanges := len(lines) > 0 && lines[0] != ""

	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		status := line[:2]
		file := strings.TrimSpace(line[3:])

		if status[0] != ' ' && status[0] != '?' {
			stagedFiles = append(stagedFiles, file)
		}
		if status[1] != ' ' && status[1] != '?' {
			modified = append(modified, file)
		}
		if status[0] == '?' && status[1] == '?' {
			untracked = append(untracked, file)
		}
	}

	response := &gitStatusResponse{
		Operation:   "Status",
		WorkingDir:  absRoot,
		Status:      statusOutput,
		Porcelain:   porcelainOutput,
		Branch:      branch,
		HasChanges:  hasChanges,
		StagedFiles: stagedFiles,
		Untracked:   untracked,
		Modified:    modified,
	}

	return response.String(), nil
}
