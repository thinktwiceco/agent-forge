package persistence

import "github.com/thinktwiceco/agent-forge/src/llms"

// Persistence interface defines methods for saving and retrieving conversation history
type Persistence interface {
	SaveHistory(chatId string, history []*llms.UnifiedMessage) string
	GetHistory(chatId string, limit, offset int) []*llms.UnifiedMessage
}
