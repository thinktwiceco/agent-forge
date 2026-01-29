package agents

import (
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/persistence"
)

// ==============================
// ===== History Management =====
// ==============================

func (a *Agent) ensureHistory(chatId string) {
	if a.history == nil {
		a.history = &History{}

		// Set up persistence if configured using the factory
		if a.config.Persistence != "" {
			a.history.persistence = persistence.NewPersistence(a.Name(), a.config.Persistence)
			if a.history.persistence != nil {
				agentforge.Debug("Initialized %s persistence for agent '%s'", a.config.Persistence, a.Name())
			}
		}
	}
	a.history.get(chatId)
}

// handleNewAssistantMessage handles new assistant messages.
// This method is currently unused but kept for potential future use.
//
//nolint:unused // Reserved for future use
func (a *Agent) handleNewAssistantMessage(message string) {
	a.ensureHistory("")
	a.history.addAssistantMessage(message, 0, 0, 0)
	a.history.save()
}

func (a *Agent) handleSystemPromptInjection() []*llms.UnifiedMessage {
	a.history.addSystemMessage(a.systemPrompt)
	return a.history.History()
}
