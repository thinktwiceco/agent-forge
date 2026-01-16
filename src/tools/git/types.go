package git

import "fmt"

// gitInitResponse represents a git init operation result.
type gitInitResponse struct {
	Operation  string
	WorkingDir string
	Branch     string
	Success    bool
	Output     string
}

// String formats the git init response as a string.
func (r *gitInitResponse) String() string {
	result := fmt.Sprintf(`Git Operation: Init
Working Directory: %s
Status: %s
`, r.WorkingDir, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Branch != "" {
		result += fmt.Sprintf("Default Branch: %s\n", r.Branch)
	}

	if r.Output != "" {
		result += fmt.Sprintf("\nOutput:\n%s\n", r.Output)
	}

	return result
}

// gitStatusResponse represents a git status operation result.
type gitStatusResponse struct {
	Operation   string
	WorkingDir  string
	Status      string
	Porcelain   string
	Branch      string
	HasChanges  bool
	StagedFiles []string
	Untracked   []string
	Modified    []string
}

// String formats the git status response as a string.
func (r *gitStatusResponse) String() string {
	result := fmt.Sprintf(`Git Operation: Status
Working Directory: %s
Branch: %s
`, r.WorkingDir, r.Branch)

	if r.HasChanges {
		result += "\nStatus:\n"
		result += r.Status
		if len(r.StagedFiles) > 0 {
			result += "\n\nStaged files:\n"
			for _, file := range r.StagedFiles {
				result += fmt.Sprintf("  %s\n", file)
			}
		}
		if len(r.Modified) > 0 {
			result += "\nModified files:\n"
			for _, file := range r.Modified {
				result += fmt.Sprintf("  %s\n", file)
			}
		}
		if len(r.Untracked) > 0 {
			result += "\nUntracked files:\n"
			for _, file := range r.Untracked {
				result += fmt.Sprintf("  %s\n", file)
			}
		}
	} else {
		result += "\nWorking tree clean - no changes to commit.\n"
	}

	return result
}

// gitAddResponse represents a git add operation result.
type gitAddResponse struct {
	Operation string
	Path      string
	Files     []string
	Status    string
}

// String formats the git add response as a string.
func (r *gitAddResponse) String() string {
	result := fmt.Sprintf(`Git Operation: Add
Path: %s
`, r.Path)

	if len(r.Files) > 0 {
		result += "\nStaged files:\n"
		for _, file := range r.Files {
			result += fmt.Sprintf("  %s\n", file)
		}
	} else {
		result += "\nNo files were staged.\n"
	}

	if r.Status != "" {
		result += fmt.Sprintf("\nStatus:\n%s\n", r.Status)
	}

	return result
}

// gitCommitResponse represents a git commit operation result.
type gitCommitResponse struct {
	Operation string
	Hash      string
	Message   string
	Branch    string
	Files     []string
	Status    string
}

// String formats the git commit response as a string.
func (r *gitCommitResponse) String() string {
	result := fmt.Sprintf(`Git Operation: Commit
Commit Hash: %s
Branch: %s
Message: %s
`, r.Hash, r.Branch, r.Message)

	if len(r.Files) > 0 {
		result += "\nCommitted files:\n"
		for _, file := range r.Files {
			result += fmt.Sprintf("  %s\n", file)
		}
	}

	if r.Status != "" {
		result += fmt.Sprintf("\nStatus:\n%s\n", r.Status)
	}

	return result
}

// gitPushResponse represents a git push operation result.
type gitPushResponse struct {
	Operation string
	Remote    string
	Branch    string
	Output    string
	Success   bool
}

// String formats the git push response as a string.
func (r *gitPushResponse) String() string {
	result := fmt.Sprintf(`Git Operation: Push
Remote: %s
Branch: %s
Status: %s
`, r.Remote, r.Branch, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Output != "" {
		result += fmt.Sprintf("\nOutput:\n%s\n", r.Output)
	}

	return result
}

// gitPullResponse represents a git pull operation result.
type gitPullResponse struct {
	Operation string
	Remote    string
	Branch    string
	Output    string
	Success   bool
}

// String formats the git pull response as a string.
func (r *gitPullResponse) String() string {
	result := fmt.Sprintf(`Git Operation: Pull
Remote: %s
Branch: %s
Status: %s
`, r.Remote, r.Branch, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Output != "" {
		result += fmt.Sprintf("\nOutput:\n%s\n", r.Output)
	}

	return result
}

// gitBranchResponse represents a git branch operation result.
type gitBranchResponse struct {
	Operation string
	Action    string // "list" or "create"
	Branch    string
	Branches  []branchInfo
	Success   bool
	Message   string
}

// branchInfo represents information about a single branch.
type branchInfo struct {
	Name      string
	IsCurrent bool
	Hash      string
	Message   string
}

// String formats the git branch response as a string.
func (r *gitBranchResponse) String() string {
	if r.Action == "create" {
		result := fmt.Sprintf(`Git Operation: Branch (Create)
Branch: %s
Status: %s
`, r.Branch, map[bool]string{true: "Created", false: "Failed"}[r.Success])

		if r.Message != "" {
			result += fmt.Sprintf("Message: %s\n", r.Message)
		}

		return result
	}

	result := fmt.Sprintf(`Git Operation: Branch (List)
Total branches: %d

`, len(r.Branches))

	for _, branch := range r.Branches {
		current := ""
		if branch.IsCurrent {
			current = "* "
		}
		result += fmt.Sprintf("%s%s", current, branch.Name)
		if branch.Hash != "" {
			result += fmt.Sprintf(" (%s)", branch.Hash)
		}
		if branch.Message != "" {
			result += fmt.Sprintf(" - %s", branch.Message)
		}
		result += "\n"
	}

	return result
}

// gitCheckoutResponse represents a git checkout operation result.
type gitCheckoutResponse struct {
	Operation string
	Branch    string
	Output    string
	Success   bool
}

// String formats the git checkout response as a string.
func (r *gitCheckoutResponse) String() string {
	result := fmt.Sprintf(`Git Operation: Checkout
Branch: %s
Status: %s
`, r.Branch, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Output != "" {
		result += fmt.Sprintf("\nOutput:\n%s\n", r.Output)
	}

	return result
}

// gitLogResponse represents a git log operation result.
type gitLogResponse struct {
	Operation string
	Limit     int
	Commits   []commitInfo
}

// commitInfo represents information about a single commit.
type commitInfo struct {
	Hash    string
	Message string
	Author  string
	Date    string
}

// String formats the git log response as a string.
func (r *gitLogResponse) String() string {
	result := fmt.Sprintf(`Git Operation: Log
Limit: %d
Total commits shown: %d

`, r.Limit, len(r.Commits))

	if len(r.Commits) == 0 {
		result += "No commits found.\n"
	} else {
		for _, commit := range r.Commits {
			result += fmt.Sprintf("%s  %s\n", commit.Hash, commit.Message)
			if commit.Author != "" {
				result += fmt.Sprintf("    Author: %s", commit.Author)
				if commit.Date != "" {
					result += fmt.Sprintf("  Date: %s", commit.Date)
				}
				result += "\n"
			}
			result += "\n"
		}
	}

	return result
}

// gitDiffResponse represents a git diff operation result.
type gitDiffResponse struct {
	Operation  string
	Path       string
	Diff       string
	HasChanges bool
}

// String formats the git diff response as a string.
func (r *gitDiffResponse) String() string {
	result := fmt.Sprintf(`Git Operation: Diff
Path: %s
`, r.Path)

	if r.HasChanges {
		result += "\nChanges:\n"
		result += r.Diff
	} else {
		result += "\nNo changes found.\n"
	}

	return result
}
