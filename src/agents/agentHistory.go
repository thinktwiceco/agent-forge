package agents

import (
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/persistence"
)

// ==============================
// ===== History Management =====
// ==============================

// createHistory creates a new History instance for a request.
// This eliminates shared state and concurrency issues.
func (a *Agent) createHistory(chatId string) *History {
	history := &History{}

	// Set up persistence if configured using the factory
	if a.config.Persistence != "" {
		history.persistence = persistence.NewPersistence(a.Name(), a.config.Persistence)
		if history.persistence != nil {
			agentforge.Debug("Initialized %s persistence for agent '%s'", a.config.Persistence, a.Name())
		}
	}

	// Load existing conversation history if chatId is provided
	history.get(chatId)
	agentforge.Debug("Created new History instance for chatId '%s' with %d existing messages", chatId, len(history.History()))

	return history
}

// injectSystemPrompt adds the system prompt to the history.
func (a *Agent) injectSystemPrompt(history *History) []*llms.UnifiedMessage {
	history.addSystemMessage(a.systemPrompt)
	return history.History()
}
