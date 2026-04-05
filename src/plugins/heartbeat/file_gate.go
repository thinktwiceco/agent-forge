package heartbeat

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// defaultHeartbeatMarkdown is written when HEARTBEAT.md is missing; the user may edit freely after that.
const defaultHeartbeatMarkdown = `# Heartbeat checklist

1. Look for outstanding tasks
`

const heartbeatFilename = "HEARTBEAT.md"

const defaultPrompt = "Read HEARTBEAT.md if it exists (workspace context). Follow it strictly.\n" +
	"Do not infer or repeat old tasks from prior chats.\n" +
	"If nothing needs attention, reply HEARTBEAT_OK."

var (
	reHeader = regexp.MustCompile(`^#+(\s|$)`)
	reBullet = regexp.MustCompile(`^[-*+]\s*(\[[\sXx]?\]\s*)?$`)
)

// isEffectivelyEmpty returns true when every non-blank line in content is a
// markdown ATX header, an empty bullet, or pure whitespace.
func isEffectivelyEmpty(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if reHeader.MatchString(t) {
			continue
		}
		if reBullet.MatchString(t) {
			continue
		}
		return false
	}
	return true
}

// ensureDefaultHeartbeatFile creates HEARTBEAT.md with default content if the file does not exist.
// An existing file is never modified. created is true only when a new file was written.
func ensureDefaultHeartbeatFile(workingDir string) (created bool, err error) {
	if workingDir == "" {
		return false, nil
	}
	path := filepath.Join(workingDir, heartbeatFilename)
	_, err = os.Stat(path)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := os.WriteFile(path, []byte(defaultHeartbeatMarkdown), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// resolvePrompt returns the prompt to inject and whether the heartbeat should
// be skipped. It reads HEARTBEAT.md from workingDir and applies the empty check.
//
// Rules:
//   - File missing → fire with default (or configured) prompt
//   - File exists but effectively empty → skip
//   - File exists with content → embed its contents (and brain/MEMORY.md if
//     present) directly in the prompt so the agent acts without needing tools
func resolvePrompt(cfg HeartbeatConfig, workingDir string) (prompt string, skip bool) {
	p := cfg.Prompt
	if p == "" {
		p = defaultPrompt
	}

	data, err := os.ReadFile(filepath.Join(workingDir, heartbeatFilename))
	if err != nil {
		// File missing: fire with the resolved prompt.
		return p, false
	}
	if isEffectivelyEmpty(string(data)) {
		return "", true
	}

	var sb strings.Builder
	sb.WriteString(p)
	sb.WriteString("\n\n[HEARTBEAT.md]\n")
	sb.WriteString(string(data))

	// Embed short-term memory so the agent can act on it without a tool call.
	if workingDir != "" {
		memData, merr := os.ReadFile(filepath.Join(workingDir, "brain", "MEMORY.md"))
		if merr == nil && len(bytes.TrimSpace(memData)) > 0 {
			sb.WriteString("\n\n[brain/MEMORY.md]\n")
			sb.WriteString(string(memData))
		}
	}

	return sb.String(), false
}
