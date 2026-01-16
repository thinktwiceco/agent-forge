package git

import (
	"fmt"
	"strings"
)

// branch executes git branch operations (list or create) and returns branch information.
func (git *Git) branch(branchName string) (string, error) {
	if branchName == "" {
		// List branches
		return git.listBranches()
	}

	// Create new branch
	return git.createBranch(branchName)
}

// listBranches lists all branches.
func (git *Git) listBranches() (string, error) {
	// Get all branches
	branchesOutput, _, err := git.executeGitCommand("branch", "-v")
	if err != nil {
		return "", fmt.Errorf("failed to list branches: %w", err)
	}

	// Parse branch list
	var branches []branchInfo
	lines := strings.Split(strings.TrimSpace(branchesOutput), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse branch line format: "* branch-name hash commit-message" or "  branch-name hash commit-message"
		isCurrent := strings.HasPrefix(line, "*")
		line = strings.TrimSpace(line)
		if isCurrent {
			line = strings.TrimPrefix(line, "*")
			line = strings.TrimSpace(line)
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		branch := branchInfo{
			Name:      parts[0],
			IsCurrent: isCurrent,
		}

		if len(parts) > 1 {
			branch.Hash = parts[1]
		}
		if len(parts) > 2 {
			branch.Message = strings.Join(parts[2:], " ")
		}

		branches = append(branches, branch)
	}

	response := &gitBranchResponse{
		Operation: "Branch",
		Action:    "list",
		Branches:  branches,
	}

	return response.String(), nil
}

// createBranch creates a new branch.
func (git *Git) createBranch(branchName string) (string, error) {
	// Execute git branch create
	_, stderr, err := git.executeGitCommand("branch", branchName)
	success := err == nil

	var message string
	if success {
		message = fmt.Sprintf("Branch '%s' created successfully", branchName)
	} else {
		message = fmt.Sprintf("Failed to create branch: %s", stderr)
	}

	response := &gitBranchResponse{
		Operation: "Branch",
		Action:    "create",
		Branch:    branchName,
		Success:   success,
		Message:   message,
	}

	if !success {
		return response.String(), fmt.Errorf("failed to create branch '%s': %w", branchName, err)
	}

	return response.String(), nil
}
