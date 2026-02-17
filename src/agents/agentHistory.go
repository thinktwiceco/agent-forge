package agents

import (
	"context"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/history"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/persistence"
	"github.com/thinktwiceco/agent-forge/src/telemetry"
)

// ==============================
// ===== History Management =====
// ==============================

// createHistoryFactory returns a factory that creates history.Manager instances.
func (a *Agent) createHistoryFactory() func(chatId string) history.Manager {
	return func(chatId string) history.Manager {
		opts := []history.Option{}
		if a.config.Persistence != "" {
			p := persistence.NewPersistence(a.Name(), a.config.Persistence)
			if p != nil {
				opts = append(opts, history.WithPersistence(p))
				agentforge.Debug("Initialized %s persistence for agent '%s'", a.config.Persistence, a.Name())
			}
		}
		hm := history.NewConversationHistory(opts...)
		if err := hm.Load(chatId); err != nil {
			agentforge.Debug("History Load returned error (ignored): %v", err)
		}
		if chatId == "" {
			agentforge.Debug("Created new History instance with no chatId - starting fresh conversation")
		} else {
			agentforge.Debug("Created History instance for chatId '%s' with %d existing messages", chatId, len(hm.Messages()))

			// Apply intelligent truncation if configured
			// Delegate to context manager for truncation
			if a.maxContextTokens > 0 {
				messagesBefore := hm.Messages()
				truncated := a.contextMgr.TruncateHistory(messagesBefore)
				hm.SetMessages(truncated)

				// Trace truncation if tracer configured and truncation actually occurred
				if a.config.Tracer != nil && len(truncated) < len(messagesBefore) {
					tokensBefore, tokensAfter := 0, 0
					if a.tokenCounter != nil {
						tokensBefore = a.tokenCounter.CountMessagesTokens(messagesBefore)
						tokensAfter = a.tokenCounter.CountMessagesTokens(truncated)
					}
					a.config.Tracer.TraceHistoryTruncation(context.Background(), telemetry.TruncationEvent{
						AgentName:      a.Name(),
						MessagesBefore: len(messagesBefore),
						MessagesAfter:  len(truncated),
						TokensBefore:   tokensBefore,
						TokensAfter:    tokensAfter,
						TokensRemoved:  tokensBefore - tokensAfter,
					})
				}
			}
		}
		return hm
	}
}

// createHistory creates a new history Manager for a request.
// This eliminates shared state and concurrency issues.
func (a *Agent) createHistory(chatId string) history.Manager {
	return a.historyFactory(chatId)
}

// injectSystemPrompt adds the system prompt to the history.
func (a *Agent) injectSystemPrompt(hm history.Manager) []*llms.UnifiedMessage {
	hm.AddSystemMessage(a.systemPrompt)
	return hm.Messages()
}
