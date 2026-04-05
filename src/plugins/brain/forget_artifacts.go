package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validateConversationArtifactKey ensures convID is safe to use as a single path segment
// for filenames under workingDir (no traversal).
func validateConversationArtifactKey(convID string) error {
	convID = strings.TrimSpace(convID)
	if convID == "" {
		return fmt.Errorf("empty conversation id")
	}
	if convID != filepath.Base(convID) {
		return fmt.Errorf("invalid conversation id")
	}
	if strings.Contains(convID, "..") {
		return fmt.Errorf("invalid conversation id")
	}
	return nil
}

// RemoveConversationArtifacts deletes on-disk artifacts for a session:
//   - brain/persistence/<date>/<conv_id>.md under brainDir (any date folder)
//   - data/conversations/<agent>/<conv_id>.json under workingDir
//
// brainDir is typically filepath.Join(workingDir, "brain"). Missing roots or files are ignored.
// Returns paths successfully removed (best-effort; duplicates not deduplicated).
func RemoveConversationArtifacts(workingDir, brainDir, convID string) ([]string, error) {
	if err := validateConversationArtifactKey(convID); err != nil {
		return nil, err
	}
	if workingDir == "" || brainDir == "" {
		return nil, fmt.Errorf("working directory and brain directory are required")
	}

	var removed []string

	persRoot := filepath.Join(brainDir, "persistence")
	dateDirs, err := os.ReadDir(persRoot)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read persistence dir: %w", err)
	}
	if err == nil {
		for _, e := range dateDirs {
			if !e.IsDir() {
				continue
			}
			p := filepath.Join(persRoot, e.Name(), convID+".md")
			if err := os.Remove(p); err == nil {
				removed = append(removed, p)
			} else if !os.IsNotExist(err) {
				return removed, fmt.Errorf("remove distilled file %s: %w", p, err)
			}
		}
	}

	convRoot := filepath.Join(workingDir, "data", "conversations")
	agentDirs, err := os.ReadDir(convRoot)
	if err != nil && !os.IsNotExist(err) {
		return removed, fmt.Errorf("read conversations dir: %w", err)
	}
	if err == nil {
		for _, a := range agentDirs {
			if !a.IsDir() {
				continue
			}
			p := filepath.Join(convRoot, a.Name(), convID+".json")
			if err := os.Remove(p); err == nil {
				removed = append(removed, p)
			} else if !os.IsNotExist(err) {
				return removed, fmt.Errorf("remove conversation json %s: %w", p, err)
			}
		}
	}

	return removed, nil
}
