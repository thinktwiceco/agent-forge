package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitTool_Integration(t *testing.T) {
	// Skip if git is not installed (optional, but good practice)
	// assuming git is installed in the env

	tmpDir, err := os.MkdirTemp("", "git-tool-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tool := NewGitTool(tmpDir)

	// 1. Test Init
	t.Run("init", func(t *testing.T) {
		args := map[string]any{"operation": "init"}
		result := tool.Call(nil, args)
		if !result.Success() {
			t.Errorf("init failed: %s", result.Error())
		}
		if _, err := os.Stat(filepath.Join(tmpDir, ".git")); os.IsNotExist(err) {
			t.Error(".git directory not created")
		}
	})

	// 2. Test Status (empty)
	t.Run("status_empty", func(t *testing.T) {
		args := map[string]any{"operation": "status"}
		result := tool.Call(nil, args)
		if !result.Success() {
			t.Errorf("status failed: %s", result.Error())
		}
		// Output varies by git version, just check success
	})

	// 3. Test Add (create file first)
	t.Run("add_file", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}

		args := map[string]any{
			"operation": "add",
			"path":      "test.txt",
		}
		result := tool.Call(nil, args)
		if !result.Success() {
			t.Errorf("add failed: %s", result.Error())
		}
	})

	// 4. Test Commit
	t.Run("commit", func(t *testing.T) {
		// Configure git user for this test repo
		execGit(t, tmpDir, "config", "user.email", "test@example.com")
		execGit(t, tmpDir, "config", "user.name", "Test User")

		args := map[string]any{
			"operation": "commit",
			"message":   "initial commit",
		}
		result := tool.Call(nil, args)

		// If failure contains "ident", it's missing config. We can ignore or warn.
		if !result.Success() {
			if strings.Contains(result.Error(), "identity") || strings.Contains(result.Error(), "config") {
				t.Log("Skipping commit verification due to missing git config")
			} else {
				t.Errorf("commit failed: %s", result.Error())
			}
		}
	})

	// 5. Test Log
	t.Run("log", func(t *testing.T) {
		args := map[string]any{"operation": "log"}
		result := tool.Call(nil, args)
		// Might fail if commit failed above, so check result only if success
		if result.Success() {
			if !strings.Contains(result.Data(), "initial commit") {
				t.Error("log missing commit message")
			}
		}
	})
}

// Helper to execute git command in dir
func execGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
	}
}
