package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"

	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/llms"
)

// JSONPersistence implements the Persistence interface using JSON file storage
type JSONPersistence struct {
	filePath string
}

// NewJSONPersistence creates a new JSONPersistence instance with the specified file path
func NewJSONPersistence(filePath string) *JSONPersistence {
	return &JSONPersistence{
		filePath: filePath,
	}
}

// SaveHystory saves the conversation history to a JSON file
func (jp *JSONPersistence) SaveHystory(history []llms.UnifiedMessage) {
	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		agentforge.Error("Failed to marshal history to JSON: %v", err)
		return
	}

	// Ensure directory exists
	dir := filepath.Dir(jp.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		agentforge.Error("Failed to create directory for history file: %v", err)
		return
	}

	// Write to file
	if err := os.WriteFile(jp.filePath, data, 0644); err != nil {
		agentforge.Error("Failed to write history to file: %v", err)
		return
	}

	agentforge.Debug("Successfully saved history to %s", jp.filePath)
}

// GetHystory retrieves the conversation history from the JSON file
// If limit == 0 and offset == 0, returns all messages
// Otherwise applies standard pagination (offset = start index, limit = page size)
func (jp *JSONPersistence) GetHystory(limit, offset int) []llms.UnifiedMessage {
	// Read file
	data, err := os.ReadFile(jp.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			agentforge.Warn("History file does not exist: %s", jp.filePath)
			return []llms.UnifiedMessage{}
		}
		agentforge.Error("Failed to read history file: %v", err)
		return []llms.UnifiedMessage{}
	}

	// Unmarshal from JSON
	var messages []llms.UnifiedMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		agentforge.Error("Failed to unmarshal history from JSON: %v", err)
		return []llms.UnifiedMessage{}
	}

	// Apply pagination
	if limit == 0 && offset == 0 {
		// Return all messages
		return messages
	}

	// Validate pagination parameters
	if offset < 0 {
		agentforge.Warn("Invalid offset %d, using 0", offset)
		offset = 0
	}

	if limit < 0 {
		agentforge.Warn("Invalid limit %d, returning empty result", limit)
		return []llms.UnifiedMessage{}
	}

	// Apply offset
	if offset >= len(messages) {
		agentforge.Debug("Offset %d is beyond message count %d, returning empty result", offset, len(messages))
		return []llms.UnifiedMessage{}
	}

	start := offset
	end := start + limit

	// Ensure we don't go beyond array bounds
	if end > len(messages) || limit == 0 {
		end = len(messages)
	}

	agentforge.Debug("Retrieved %d messages from history (offset: %d, limit: %d)", end-start, offset, limit)
	return messages[start:end]
}
