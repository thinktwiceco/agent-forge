package agents

import (
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// ==============================
// ===== History Truncation =====
// ==============================

// truncateHistory applies intelligent truncation to conversation history
// when it exceeds the configured context window limit.
//
// This method delegates to the configured TruncationStrategy,
// allowing different truncation approaches without modifying the Agent.
func (a *Agent) truncateHistory(messages []*llms.UnifiedMessage) []*llms.UnifiedMessage {
	// Skip truncation if not configured
	if a.config.TruncationStrategy == nil || a.maxContextTokens <= 0 {
		return messages
	}

	// Calculate available token budget (reserve space for completion)
	maxAllowed := a.maxContextTokens - a.reservedOutputTokens

	// Delegate to configured strategy
	return a.config.TruncationStrategy.Truncate(messages, maxAllowed)
}
