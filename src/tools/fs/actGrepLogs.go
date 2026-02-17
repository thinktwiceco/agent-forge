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
func (fs *Fs) grepLogs(pattern string, flags []string) (string, error) {
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
					MatchCount:   0,
					Output:       "",
					Status:       "No matches found",
				}
				return response.String(), nil
			}
			return "", fmt.Errorf("ripgrep failed with exit code %d: %s", exitErr.ExitCode(), stderr.String())
		}
		return "", fmt.Errorf("failed to execute ripgrep: %w", err)
	}

	output := stdout.String()
	matchCount := 0
	if output != "" {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			if line != "" {
				matchCount++
			}
		}
	}

	response := &ripgrepResponse{
		Pattern:      pattern,
		RelativePath: logFilePath,
		AbsolutePath: absPath,
		Flags:        flags,
		MatchCount:   matchCount,
		Output:       output,
		Status:       fmt.Sprintf("Found %d matches", matchCount),
	}

	return response.String(), nil
}
