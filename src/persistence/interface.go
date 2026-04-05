package persistence

import "github.com/thinktwiceco/agent-forge/src/llms"

// Persistence interface defines methods for saving and retrieving conversation history
type Persistence interface {
	SaveHistory(chatId string, history []*llms.UnifiedMessage) (string, error)
	GetHistory(chatId string, limit, offset int) []*llms.UnifiedMessage
}

// ConversationMetadata interface defines methods for managing conversation metadata (e.g. title)
type ConversationMetadata interface {
	GetTitle(chatId string) string
	SetTitle(chatId string, title string) error
	DeleteTitle(chatId string) error
}
