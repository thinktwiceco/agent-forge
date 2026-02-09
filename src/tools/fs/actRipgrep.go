package fs

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ripgrep performs a search using ripgrep in the specified path.
// The path is validated to ensure it stays within the root directory.
// Pattern is required, and additional ripgrep flags can be provided.
func (fs *Fs) ripgrep(path, pattern string, flags []string) (string, error) {
	validatedPath, err := fs.validatePath(path)
	if err != nil {
		return "", err
	}

	// Check if ripgrep is available
	if _, err := exec.LookPath("rg"); err != nil {
		return "", fmt.Errorf("ripgrep (rg) is not installed or not in PATH")
	}

	// Build ripgrep command
	args := []string{}

	// Add user-provided flags if any
	if len(flags) > 0 {
		args = append(args, flags...)
	}

	// Add pattern
	args = append(args, pattern)

	// Add path to search in
	args = append(args, validatedPath)

	// Execute ripgrep
	cmd := exec.Command("rg", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// ripgrep returns exit code 1 when no matches are found, which is not an error
	// Exit code 2 or higher indicates an actual error
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				// No matches found
				response := &ripgrepResponse{
					Pattern:      pattern,
					RelativePath: path,
					AbsolutePath: validatedPath,
					Flags:        flags,
					MatchCount:   0,
					Output:       "",
					Status:       "No matches found",
				}
				return response.String(), nil
			}
			// Other exit codes indicate errors
			return "", fmt.Errorf("ripgrep failed with exit code %d: %s", exitErr.ExitCode(), stderr.String())
		}
		return "", fmt.Errorf("failed to execute ripgrep: %w", err)
	}

	// Parse output to count matches
	output := stdout.String()
	matchCount := 0
	if output != "" {
		// Count lines that contain matches (not headers or context)
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			if line != "" {
				matchCount++
			}
		}
	}

	// Build detailed response
	response := &ripgrepResponse{
		Pattern:      pattern,
		RelativePath: path,
		AbsolutePath: validatedPath,
		Flags:        flags,
		MatchCount:   matchCount,
		Output:       output,
		Status:       fmt.Sprintf("Found %d matches", matchCount),
	}

	return response.String(), nil
}
