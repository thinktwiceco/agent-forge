package git

import (
	"fmt"
	"path/filepath"
	"strings"
)

// clone executes git clone and returns clone result.
func (git *Git) clone(url, directory string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("repository URL is required for clone operation")
	}

	var args []string
	args = append(args, "clone", url)

	// If directory is specified, add it to the clone command
	if directory != "" {
		// Validate that the directory path doesn't escape the root
		_, err := git.validatePath(directory)
		if err != nil {
			return "", fmt.Errorf("invalid directory: %w", err)
		}
		args = append(args, directory)
	}

	// Execute git clone
	output, stderr, err := git.executeGitCommand(args...)
	success := err == nil

	// Combine output and stderr
	fullOutput := strings.TrimSpace(output)
	if stderr != "" {
		if fullOutput != "" {
			fullOutput += "\n" + strings.TrimSpace(stderr)
		} else {
			fullOutput = strings.TrimSpace(stderr)
		}
	}

	// Determine the cloned directory name
	clonedDir := directory
	if clonedDir == "" {
		// Extract repo name from URL (e.g., "repo.git" or "repo" from "https://github.com/user/repo.git")
		urlParts := strings.Split(strings.TrimSuffix(url, "/"), "/")
		if len(urlParts) > 0 {
			repoName := urlParts[len(urlParts)-1]
			clonedDir = strings.TrimSuffix(repoName, ".git")
		}
	}

	// Get absolute path of the cloned directory
	absRoot, _ := filepath.Abs(git.dir)
	clonedPath := filepath.Join(absRoot, clonedDir)

	response := &gitCloneResponse{
		Operation: "Clone",
		URL:       url,
		Directory: clonedDir,
		FullPath:  clonedPath,
		Output:    fullOutput,
		Success:   success,
	}

	if !success {
		return response.String(), fmt.Errorf("git clone failed: %w", err)
	}

	return response.String(), nil
}
