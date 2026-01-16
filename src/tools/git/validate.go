package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// validateOperation ensures that the operation is one of the valid git operations.
func validateOperation(value any) error {
	operation, ok := value.(string)
	if !ok {
		return fmt.Errorf("operation must be a string")
	}
	validOperations := map[string]bool{
		"status":   true,
		"add":      true,
		"commit":   true,
		"push":     true,
		"pull":     true,
		"branch":   true,
		"checkout": true,
		"log":      true,
		"diff":     true,
	}
	if !validOperations[operation] {
		return fmt.Errorf("invalid operation: %s. Must be one of: status, add, commit, push, pull, branch, checkout, log, diff", operation)
	}
	return nil
}

// validatePath ensures that the given file path stays within the root directory.
// It returns the validated absolute path or an error if the path escapes the root.
func (git *Git) validatePath(filePath string) (string, error) {
	// Get absolute path of root
	absRoot, err := filepath.Abs(git.root)
	if err != nil {
		return "", fmt.Errorf("invalid root directory: %w", err)
	}

	// Clean and join root with the provided path
	joinedPath := filepath.Join(absRoot, filepath.Clean(filePath))

	// Get absolute path of the joined path
	absPath, err := filepath.Abs(joinedPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check if the resolved path is within root
	relPath, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("path validation failed: %w", err)
	}

	// If relative path starts with "..", it means we escaped the root directory
	if len(relPath) >= 2 && relPath[:2] == ".." {
		return "", fmt.Errorf("path traversal detected: path '%s' escapes root directory", filePath)
	}

	return absPath, nil
}

// validateGitRepo checks if the root directory is a git repository.
func (git *Git) validateGitRepo() error {
	absRoot, err := filepath.Abs(git.root)
	if err != nil {
		return fmt.Errorf("invalid root directory: %w", err)
	}

	gitDir := filepath.Join(absRoot, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s (no .git directory found)", absRoot)
	}

	return nil
}

// executeGitCommand executes a git command in the root directory and returns stdout and stderr.
func (git *Git) executeGitCommand(args ...string) (string, string, error) {
	absRoot, err := filepath.Abs(git.root)
	if err != nil {
		return "", "", fmt.Errorf("invalid root directory: %w", err)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = absRoot

	// Capture stdout and stderr separately
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if err != nil {
		return stdoutStr, stderrStr, fmt.Errorf("git command failed: %w", err)
	}

	return stdoutStr, stderrStr, nil
}
