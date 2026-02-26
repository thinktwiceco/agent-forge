package fs

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const grepLogsUnavailableMsg = "grep_logs is not available: logs are not streaming to a file. Set AF_LOG_FILE in your environment or .env to enable file logging and make grep_logs available."

// grepLogs searches for a pattern in the application log file.
// Only available when AF_LOG_FILE is set. Returns an error otherwise.
// offset skips that many match lines; headLimit caps the number returned (0 = no cap).
func (fs *Fs) grepLogs(pattern string, flags []string, offset, headLimit int) (string, error) {
	logFilePath := os.Getenv("AF_LOG_FILE")
	if logFilePath == "" {
		return "", fmt.Errorf("%s", grepLogsUnavailableMsg)
	}

	absPath, err := filepath.Abs(logFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve log file path %s: %w", logFilePath, err)
	}

	// Check if ripgrep is available
	if _, err := exec.LookPath("rg"); err != nil {
		return "", fmt.Errorf("ripgrep (rg) is not installed or not in PATH")
	}

	// Build ripgrep command - search in the single log file
	args := []string{}
	if len(flags) > 0 {
		args = append(args, flags...)
	}
	args = append(args, pattern, absPath)

	cmd := exec.Command("rg", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// ripgrep returns exit code 1 when no matches are found
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				response := &ripgrepResponse{
					Pattern:      pattern,
					RelativePath: logFilePath,
					AbsolutePath: absPath,
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

	// Apply head_limit
	if headLimit > 0 && len(lines) > headLimit {
		lines = lines[:headLimit]
	}

	matchCount := len(lines)
	response := &ripgrepResponse{
		Pattern:      pattern,
		RelativePath: logFilePath,
		AbsolutePath: absPath,
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
