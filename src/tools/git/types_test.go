package git

import (
	"strings"
	"testing"
)

func TestGitTypes_String(t *testing.T) {
	// Init
	initResp := &gitInitResponse{
		Operation:  "init",
		WorkingDir: "/tmp",
		Branch:     "main",
		Success:    true,
		Output:     "Initialized",
	}
	if !strings.Contains(initResp.String(), "Git Operation: Init") {
		t.Error("Init String() mismatch")
	}

	// Status
	statusResp := &gitStatusResponse{
		Operation:   "status",
		WorkingDir:  "/tmp",
		Branch:      "main",
		HasChanges:  true,
		Status:      "M file.txt",
		StagedFiles: []string{"file.txt"},
		Modified:    []string{"mod.txt"},
		Untracked:   []string{"new.txt"},
	}
	statusStr := statusResp.String()
	if !strings.Contains(statusStr, "Git Operation: Status") {
		t.Error("Status String() mismatch")
	}
	if !strings.Contains(statusStr, "Staged files") {
		t.Error("Status String() missing staged")
	}

	// Status Clean
	statusClean := &gitStatusResponse{
		HasChanges: false,
	}
	if !strings.Contains(statusClean.String(), "clean") {
		t.Error("Status String() clean mismatch")
	}

	// Add
	addResp := &gitAddResponse{
		Operation: "add",
		Path:      ".",
		Files:     []string{"file1", "file2"},
		Status:    "ok",
	}
	addStr := addResp.String()
	if !strings.Contains(addStr, "Git Operation: Add") {
		t.Error("Add String() mismatch")
	}
	if !strings.Contains(addStr, "file1") {
		t.Error("Add String() missing file")
	}

	// Add Empty
	addEmpty := &gitAddResponse{}
	if !strings.Contains(addEmpty.String(), "No files were staged") {
		t.Error("Add String() empty mismatch")
	}

	// Commit
	commitResp := &gitCommitResponse{
		Operation: "commit",
		Hash:      "abc1234",
		Message:   "feat: test",
		Branch:    "main",
		Files:     []string{"file1"},
		Status:    "ok",
	}
	commitStr := commitResp.String()
	if !strings.Contains(commitStr, "Git Operation: Commit") {
		t.Error("Commit String() mismatch")
	}
	if !strings.Contains(commitStr, "abc1234") {
		t.Error("Commit String() missing hash")
	}

	// Push
	pushResp := &gitPushResponse{
		Operation: "push",
		Remote:    "origin",
		Branch:    "main",
		Success:   true,
		Output:    "pushed",
	}
	if !strings.Contains(pushResp.String(), "Git Operation: Push") {
		t.Error("Push String() mismatch")
	}

	// Pull
	pullResp := &gitPullResponse{
		Operation: "pull",
		Remote:    "origin",
		Branch:    "main",
		Success:   false,
		Output:    "failed",
	}
	if !strings.Contains(pullResp.String(), "Git Operation: Pull") {
		t.Error("Pull String() mismatch")
	}

	// Branch Create
	branchCreate := &gitBranchResponse{
		Action:  "create",
		Branch:  "new-feature",
		Success: true,
		Message: "created",
	}
	if !strings.Contains(branchCreate.String(), "Git Operation: Branch (Create)") {
		t.Error("Branch Create String() mismatch")
	}

	// Branch List
	branchList := &gitBranchResponse{
		Action: "list",
		Branches: []branchInfo{
			{Name: "main", IsCurrent: true},
			{Name: "dev", IsCurrent: false, Hash: "123", Message: "msg"},
		},
	}
	branchListStr := branchList.String()
	if !strings.Contains(branchListStr, "Git Operation: Branch (List)") {
		t.Error("Branch List String() mismatch")
	}
	if !strings.Contains(branchListStr, "* main") {
		t.Error("Branch List String() missing current")
	}

	// Checkout
	checkoutResp := &gitCheckoutResponse{
		Operation: "checkout",
		Branch:    "main",
		Success:   true,
		Output:    "switched",
	}
	if !strings.Contains(checkoutResp.String(), "Git Operation: Checkout") {
		t.Error("Checkout String() mismatch")
	}

	// Log
	logResp := &gitLogResponse{
		Operation: "log",
		Limit:     10,
		Commits: []commitInfo{
			{Hash: "123", Message: "init", Author: "Me", Date: "Now"},
		},
	}
	logStr := logResp.String()
	if !strings.Contains(logStr, "Git Operation: Log") {
		t.Error("Log String() mismatch")
	}
	if !strings.Contains(logStr, "init") {
		t.Error("Log String() missing message")
	}

	// Log Empty
	logEmpty := &gitLogResponse{Commits: []commitInfo{}}
	if !strings.Contains(logEmpty.String(), "No commits found") {
		t.Error("Log String() empty mismatch")
	}

	// Diff
	diffResp := &gitDiffResponse{
		Operation:  "diff",
		Path:       ".",
		Diff:       "+ line",
		HasChanges: true,
	}
	if !strings.Contains(diffResp.String(), "Git Operation: Diff") {
		t.Error("Diff String() mismatch")
	}

	// Diff Empty
	diffEmpty := &gitDiffResponse{HasChanges: false}
	if !strings.Contains(diffEmpty.String(), "No changes found") {
		t.Error("Diff String() empty mismatch")
	}
}
