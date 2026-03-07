package fs

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ripgrep performs a search using ripgrep in the specified path.
// The path is validated to ensure it stays within the root directory.
// Pattern is required, and additional ripgrep flags can be provided.
// offset skips that many match lines; headLimit caps the number returned (0 = no cap).
func (fs *Fs) ripgrep(path, pattern string, flags []string, offset, headLimit int) (string, error) {
	validatedPath, err := fs.validatePath(path)
	if err != nil {
		return "", err
	}

	// Check if ripgrep is available; fall back to grep transparently if not.
	if _, err := exec.LookPath("rg"); err != nil {
		return fs.grepFallback(path, validatedPath, pattern, flags, offset, headLimit)
	}

	// Build ripgrep command
	args := []string{}

	// Add user-provided flags if any
	if len(flags) > 0 {
		args = append(args, flags...)
	}

	if info, err := os.Stat(validatedPath); err == nil && info.IsDir() {
		args = append(args, "--glob=!.env", "--glob=!.env.*", "--glob=!*.env")
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
					TotalMatches: 0,
					MatchCount:   0,
					Offset:       offset,
					HeadLimit:    headLimit,
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

	// Collect non-empty match lines
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	totalMatches := len(lines)

	// Apply offset
	if offset > totalMatches {
		offset = totalMatches
	}
	lines = lines[offset:]

	// Apply head_limit (default 500 when caller omits it to avoid context overflow)
	if headLimit == 0 {
		headLimit = 500
	}
	if len(lines) > headLimit {
		lines = lines[:headLimit]
	}

	matchCount := len(lines)
	response := &ripgrepResponse{
		Pattern:      pattern,
		RelativePath: path,
		AbsolutePath: validatedPath,
		Flags:        flags,
		TotalMatches: totalMatches,
		MatchCount:   matchCount,
		Offset:       offset,
		HeadLimit:    headLimit,
		Output:       strings.Join(lines, "\n"),
		Status:       fmt.Sprintf("Showing %d of %d matches", matchCount, totalMatches),
	}

	return response.String(), nil
}
