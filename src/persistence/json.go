package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/llms"
)

// JSONPersistence implements the Persistence interface using JSON file storage
// with directory-based organization for multiple conversations
type JSONPersistence struct {
	baseDir string
}

// NewJSONPersistence creates a new JSONPersistence instance with the specified base directory
// Conversations will be stored as {baseDir}/{chatId}.json
func NewJSONPersistence(baseDir string) *JSONPersistence {
	return &JSONPersistence{
		baseDir: baseDir,
	}
}

// SaveHistory saves the conversation history to a JSON file
// If chatId is empty, a new UUID is generated
// Returns the chatId (newly generated or the one provided)
func (jp *JSONPersistence) SaveHistory(chatId string, history []*llms.UnifiedMessage) string {
	// Generate chatId if not provided
	if chatId == "" {
		chatId = uuid.New().String()
		agentforge.Debug("Generated new chatId: %s", chatId)
	}

	// Construct file path
	filePath := filepath.Join(jp.baseDir, chatId+".json")

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		agentforge.Error("Failed to marshal history to JSON: %v", err)
		return chatId
	}

	// Ensure directory exists
	if err := os.MkdirAll(jp.baseDir, 0755); err != nil {
		agentforge.Error("Failed to create directory for history file: %v", err)
		return chatId
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		agentforge.Error("Failed to write history to file: %v", err)
		return chatId
	}

	agentforge.Debug("Successfully saved history to %s", filePath)
	return chatId
}

// GetHistory retrieves the conversation history from the JSON file for a specific chatId
// If limit == 0 and offset == 0, returns all messages
// Otherwise applies standard pagination (offset = start index, limit = page size)
func (jp *JSONPersistence) GetHistory(chatId string, limit, offset int) []*llms.UnifiedMessage {
	// Empty chatId means new conversation - return empty history
	if chatId == "" {
		agentforge.Debug("Empty chatId provided, returning empty history for new conversation")
		return []*llms.UnifiedMessage{}
	}

	// Construct file path
	filePath := filepath.Join(jp.baseDir, chatId+".json")

	// Read file
	var messages []*llms.UnifiedMessage

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			agentforge.Debug("History file does not exist for chatId %s: %s", chatId, filePath)
			return []*llms.UnifiedMessage{}
		}
		agentforge.Error("Failed to read history file: %v", err)
		return messages
	}

	// Unmarshal from JSON
	if err := json.Unmarshal(data, &messages); err != nil {
		agentforge.Error("Failed to unmarshal history from JSON: %v", err)
		return messages
	}

	// Apply pagination
	if limit == 0 && offset == 0 {
		// Return all messages
		agentforge.Debug("Retrieved %d messages for chatId %s", len(messages), chatId)
		return messages
	}

	// Validate pagination parameters
	if offset < 0 {
		agentforge.Warn("Invalid offset %d, using 0", offset)
		offset = 0
	}

	if limit < 0 {
		agentforge.Warn("Invalid limit %d, returning empty result", limit)
		return messages
	}

	// Apply offset
	if offset >= len(messages) {
		agentforge.Debug("Offset %d is beyond message count %d, returning empty result", offset, len(messages))
		return messages
	}

	start := offset
	end := start + limit

	// Ensure we don't go beyond array bounds
	if end > len(messages) || limit == 0 {
		end = len(messages)
	}

	agentforge.Debug("Retrieved %d messages from history (chatId: %s, offset: %d, limit: %d)", end-start, chatId, offset, limit)
	return messages[start:end]
}
