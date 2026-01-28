package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateOperation(t *testing.T) {
	tests := []struct {
		name    string
		op      any
		wantErr bool
	}{
		{"valid_status", "status", false},
		{"valid_commit", "commit", false},
		{"valid_log", "log", false},
		{"invalid_op", "destroy", true},
		{"invalid_type", 123, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateOperation(tt.op); (err != nil) != tt.wantErr {
				t.Errorf("validateOperation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGit_ValidatePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	git := &Git{root: tmpDir}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid_file", "file.txt", false},
		{"valid_subdir", "subdir/file.txt", false},
		{"parent_traversal", "../escape.txt", true},
		{"deep_traversal", "subdir/../../escape.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := git.validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestGit_ValidateGitRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git-repo-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Case 1: Not a git repo
	git := &Git{root: tmpDir}
	if err := git.validateGitRepo(); err == nil {
		t.Error("Expected error for non-git repo, got nil")
	}

	// Case 2: Init git repo (simulate by creating .git dir)
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := git.validateGitRepo(); err != nil {
		t.Errorf("Expected nil error for valid git repo, got %v", err)
	}
}
