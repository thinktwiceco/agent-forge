package git

import (
	"fmt"
	"strings"
)

// log executes git log and returns commit history.
func (git *Git) log(limit int) (string, error) {
	// Execute git log with limit
	args := []string{"log", "--oneline", "-n", fmt.Sprintf("%d", limit)}
	logOutput, stderr, err := git.executeGitCommand(args...)
	if err != nil {
		// Check if error is due to no commits (common case)
		if strings.Contains(stderr, "does not have any commits") || strings.Contains(err.Error(), "does not have any commits") {
			// Return empty log response instead of error
			response := &gitLogResponse{
				Operation: "Log",
				Limit:     limit,
				Commits:   []commitInfo{},
			}
			return response.String(), nil
		}
		return "", fmt.Errorf("failed to get git log: %w", err)
	}

	// Get detailed log for author and date info
	detailedArgs := []string{"log", "-n", fmt.Sprintf("%d", limit), "--format=%H|%s|%an|%ad", "--date=short"}
	detailedOutput, _, err := git.executeGitCommand(detailedArgs...)
	if err != nil {
		// Fall back to simple log if detailed fails
		detailedOutput = ""
	}

	// Parse commits
	var commits []commitInfo
	lines := strings.Split(strings.TrimSpace(logOutput), "\n")
	detailedLines := strings.Split(strings.TrimSpace(detailedOutput), "\n")

	// Create a map of detailed info by hash
	detailedMap := make(map[string]commitInfo)
	for _, line := range detailedLines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 2 {
			info := commitInfo{
				Hash:    parts[0],
				Message: parts[1],
			}
			if len(parts) >= 3 {
				info.Author = parts[2]
			}
			if len(parts) >= 4 {
				info.Date = parts[3]
			}
			detailedMap[parts[0]] = info
		}
	}

	// Parse simple log lines
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Simple format: "hash message"
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		commit := commitInfo{
			Hash:    parts[0],
			Message: strings.Join(parts[1:], " "),
		}

		// Try to get detailed info
		if detailed, ok := detailedMap[commit.Hash]; ok {
			commit.Author = detailed.Author
			commit.Date = detailed.Date
		}

		commits = append(commits, commit)
	}

	response := &gitLogResponse{
		Operation: "Log",
		Limit:     limit,
		Commits:   commits,
	}

	return response.String(), nil
}
