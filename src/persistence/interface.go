package persistence

import "github.com/thinktwice/agentForge/src/llms"

// Persistence interface defines methods for saving and retrieving conversation history
type Persistence interface {
	SaveHystory(history []llms.UnifiedMessage)
	GetHystory(limit, offset int) []llms.UnifiedMessage
}
