package persistence

import (
	"os"
	"path/filepath"
	"strings"

	agentforge "github.com/thinktwiceco/agent-forge/src"
)

// JSONConversationMetadata implements ConversationMetadata using plain-text sidecar files.
// Each conversation title is stored at {baseDir}/{chatId}.title alongside the
// corresponding {chatId}.json history file.
type JSONConversationMetadata struct {
	baseDir string
}

// NewJSONConversationMetadata creates a new JSONConversationMetadata for the given base directory.
func NewJSONConversationMetadata(baseDir string) *JSONConversationMetadata {
	return &JSONConversationMetadata{baseDir: baseDir}
}

// GetTitle returns the stored title for the given chatId, or "" if none is set.
func (m *JSONConversationMetadata) GetTitle(chatId string) string {
	if chatId == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(m.baseDir, chatId+".title"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// SetTitle writes the title for the given chatId to a sidecar file.
func (m *JSONConversationMetadata) SetTitle(chatId string, title string) error {
	if err := os.MkdirAll(m.baseDir, 0755); err != nil {
		agentforge.Error("Failed to create directory for title file: %v", err)
		return err
	}
	path := filepath.Join(m.baseDir, chatId+".title")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(title)), 0644); err != nil {
		agentforge.Error("Failed to write title file for chatId %s: %v", chatId, err)
		return err
	}
	return nil
}

// DeleteTitle removes the title sidecar file for the given chatId.
// Returns nil if the file does not exist.
func (m *JSONConversationMetadata) DeleteTitle(chatId string) error {
	path := filepath.Join(m.baseDir, chatId+".title")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		agentforge.Error("Failed to delete title file for chatId %s: %v", chatId, err)
		return err
	}
	return nil
}
