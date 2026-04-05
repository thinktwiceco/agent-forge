package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const maxRecallTitleRunes = 120
const maxRecallDescriptionRunes = 200

// effectiveRecallTime returns the last_access timestamp from node metadata.
// Returns an error if the field is absent or not a valid RFC3339 string.
func effectiveRecallTime(n *Node) (time.Time, error) {
	if n == nil || n.Metadata == nil {
		return time.Time{}, fmt.Errorf("node has no metadata")
	}
	s, ok := n.Metadata["last_access"].(string)
	if !ok || s == "" {
		return time.Time{}, fmt.Errorf("metadata missing last_access")
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("metadata last_access is not valid RFC3339: %w", err)
	}
	return t, nil
}

// recallTitle returns the title column, then metadata fallbacks, then a short line from content.
func recallTitle(n *Node) string {
	if n == nil {
		return ""
	}
	if strings.TrimSpace(n.Title) != "" {
		return strings.TrimSpace(n.Title)
	}
	if n.Metadata != nil {
		if t, ok := n.Metadata["title"].(string); ok && strings.TrimSpace(t) != "" {
			return strings.TrimSpace(t)
		}
		if id, ok := n.Metadata["conv_id"].(string); ok && strings.TrimSpace(id) != "" && n.Content == id {
			return strings.TrimSpace(id)
		}
	}
	t := strings.TrimSpace(firstLine(n.Content))
	if t != "" {
		return truncateRunes(t, maxRecallTitleRunes)
	}
	if n.Metadata != nil {
		if id, ok := n.Metadata["conv_id"].(string); ok && id != "" {
			return id
		}
	}
	return truncateRunes(n.Content, maxRecallTitleRunes)
}

// recallDescription returns the description column, then metadata, else a truncated summary of content.
func recallDescription(n *Node) string {
	if n == nil {
		return ""
	}
	if strings.TrimSpace(n.Description) != "" {
		return truncateRunes(strings.TrimSpace(n.Description), maxRecallDescriptionRunes)
	}
	if n.Metadata != nil {
		if d, ok := n.Metadata["description"].(string); ok && strings.TrimSpace(d) != "" {
			return strings.TrimSpace(d)
		}
	}
	body := strings.TrimSpace(n.Content)
	if body == "" {
		return ""
	}
	return truncateRunes(body, maxRecallDescriptionRunes)
}

// recallTopics extracts the topics array from node metadata.
func recallTopics(n *Node) []string {
	if n == nil || n.Metadata == nil {
		return nil
	}
	raw, ok := n.Metadata["topics"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func firstLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max]) + "..."
	}
	return s
}

// deriveTitleAndDescriptionFromSummary sets listing fields when dreaming writes content.
func deriveTitleAndDescriptionFromSummary(summary string) (title, description string) {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return "", ""
	}
	lines := strings.Split(strings.ReplaceAll(summary, "\r\n", "\n"), "\n")
	var nonEmpty []string
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" {
			nonEmpty = append(nonEmpty, t)
		}
	}
	if len(nonEmpty) == 0 {
		return "", ""
	}
	title = truncateRunes(nonEmpty[0], maxRecallTitleRunes)
	if len(nonEmpty) > 1 {
		desc := strings.TrimSpace(strings.Join(nonEmpty[1:], " "))
		description = truncateRunes(desc, maxRecallDescriptionRunes)
	} else {
		description = truncateRunes(nonEmpty[0], maxRecallDescriptionRunes)
	}
	return title, description
}

// formatRetrieveConversation builds ephemeral text (reason + summary) for a conversation
// node. The summary is read from the persistence markdown file (stm_path in metadata)
// rather than the content column, which is cleared after dreaming.
func (p *BrainPlugin) formatRetrieveConversation(node *Node) string {
	if node == nil {
		return ""
	}
	reason := strings.TrimSpace(node.DistillationReason)
	if node.Metadata != nil {
		if r, ok := node.Metadata["distillation_reason"].(string); ok && reason == "" {
			reason = strings.TrimSpace(r)
		}
	}
	body := p.readConversationNodeFile(node)
	if reason == "" {
		return body
	}
	return "Distillation reason:\n" + reason + "\n\nSummary:\n" + body
}

// readConversationNodeFile reads the persistence markdown file for a conversation node
// using stm_path from metadata. Returns an empty string if the path is missing or unreadable.
func (p *BrainPlugin) readConversationNodeFile(node *Node) string {
	if node == nil || node.Metadata == nil || p.workingDir == "" {
		return ""
	}
	stmPath, _ := node.Metadata["stm_path"].(string)
	if strings.TrimSpace(stmPath) == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(p.workingDir, stmPath))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func startOfDayLocal(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func endOfDayLocal(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 23, 59, 59, 999999999, t.Location())
}

func normalizeTopicName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	return strings.Join(strings.Fields(s), " ")
}

func normalizeTopicNames(topics []string) []string {
	seen := make(map[string]bool, len(topics))
	out := make([]string, 0, len(topics))
	for _, topic := range topics {
		norm := normalizeTopicName(topic)
		if norm == "" || seen[norm] {
			continue
		}
		seen[norm] = true
		out = append(out, norm)
	}
	sort.Strings(out)
	return out
}
