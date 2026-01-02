package agents

import (
	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/persistence"
)

// ==============================
// ===== History Management =====
// ==============================

func (a *Agent) ensureHistory() {
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
	a.history.get()
}

func (a *Agent) handleNewAssistantMessage(message string) {
	a.ensureHistory()
	a.history.addAssistantMessage(message, 0, 0, 0)
	a.history.save()
}

func (a *Agent) handleSystemPromptInjection() []llms.UnifiedMessage {
	a.history.addSystemMessage(a.systemPrompt)
	a.history.save()
	return a.history.History()
}
