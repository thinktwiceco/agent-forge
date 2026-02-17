package context

import (
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// =============================
// ===== NoTruncationStrategy =====
// =============================

// NoTruncationStrategy is a pass-through strategy that returns messages unchanged.
// Use when truncation is not needed or for testing.
type NoTruncationStrategy struct{}

// NewNoTruncation creates a new NoTruncationStrategy.
func NewNoTruncation() *NoTruncationStrategy {
	return &NoTruncationStrategy{}
}

// Truncate returns messages unchanged (no-op).
func (s *NoTruncationStrategy) Truncate(messages []*llms.UnifiedMessage, maxTokens int) []*llms.UnifiedMessage {
	return messages
}

// ===============================
// ===== SlidingWindowStrategy =====
// ===============================

// SlidingWindowStrategy keeps system message + most recent N messages
// that fit within the token budget.
//
// Algorithm:
// 1. Always preserve system message (if present)
// 2. Keep at least minRecentMessages from the end
// 3. Fill remaining budget with older messages
// 4. Ensure tool call/result pairs stay together
type SlidingWindowStrategy struct {
	minRecentMessages int
	tokenCounter      llms.TokenCounter
}

// NewSlidingWindow creates a new SlidingWindowStrategy.
//
// Parameters:
//   - minRecentMessages: Minimum number of recent messages to always keep
//   - tokenCounter: Counter for calculating message token counts
func NewSlidingWindow(minRecentMessages int, tokenCounter llms.TokenCounter) *SlidingWindowStrategy {
	return &SlidingWindowStrategy{
		minRecentMessages: minRecentMessages,
		tokenCounter:      tokenCounter,
	}
}

// Truncate applies sliding window truncation to keep recent messages.
func (s *SlidingWindowStrategy) Truncate(messages []*llms.UnifiedMessage, maxTokens int) []*llms.UnifiedMessage {
	if s.tokenCounter == nil {
		return messages
	}

	// Count total tokens in current history
	totalTokens := s.tokenCounter.CountMessagesTokens(messages)

	agentforge.Debug("History token count: %d/%d", totalTokens, maxTokens)

	// No truncation needed
	if totalTokens <= maxTokens {
		return messages
	}

	agentforge.Info("History exceeds token limit (%d > %d), applying sliding window truncation", totalTokens, maxTokens)

	return s.slidingWindowTruncate(messages, maxTokens, totalTokens)
}

// slidingWindowTruncate implements the core sliding window logic.
func (s *SlidingWindowStrategy) slidingWindowTruncate(messages []*llms.UnifiedMessage, maxTokens int, totalTokens int) []*llms.UnifiedMessage {
	if len(messages) == 0 {
		return messages
	}

	// Always preserve system message if present
	systemMsg := []*llms.UnifiedMessage{}
	startIdx := 0
	if len(messages) > 0 && messages[0].Role() == llms.MessageRoleSystem {
		systemMsg = []*llms.UnifiedMessage{messages[0]}
		startIdx = 1
	}

	systemTokens := s.tokenCounter.CountMessagesTokens(systemMsg)
	remainingBudget := maxTokens - systemTokens

	// Start from the end and work backwards, keeping recent messages
	recentMessages := []*llms.UnifiedMessage{}
	tokensUsed := 0
	minToKeep := s.minRecentMessages

	// First pass: ensure we keep at least minRecentMessages
	for i := len(messages) - 1; i >= startIdx && len(recentMessages) < minToKeep; i-- {
		msg := messages[i]
		msgTokens := s.tokenCounter.CountMessageTokens(msg)

		// Always add to maintain minimum
		recentMessages = append([]*llms.UnifiedMessage{msg}, recentMessages...)
		tokensUsed += msgTokens
	}

	// Second pass: keep adding older messages while under budget
	startFrom := len(messages) - len(recentMessages) - 1
	for i := startFrom; i >= startIdx; i-- {
		msg := messages[i]
		msgTokens := s.tokenCounter.CountMessageTokens(msg)

		if tokensUsed+msgTokens > remainingBudget {
			break
		}

		recentMessages = append([]*llms.UnifiedMessage{msg}, recentMessages...)
		tokensUsed += msgTokens
	}

	// Combine system message + recent messages
	truncated := append(systemMsg, recentMessages...)

	// Ensure tool call/result pairs are complete
	truncated = s.ensureToolCallPairs(truncated, messages)

	truncatedCount := len(messages) - len(truncated)
	truncatedTokens := totalTokens - s.tokenCounter.CountMessagesTokens(truncated)

	agentforge.Info("Truncation complete: removed %d messages (%d tokens), kept %d messages (%d tokens)",
		truncatedCount, truncatedTokens, len(truncated), s.tokenCounter.CountMessagesTokens(truncated))

	return truncated
}

// ensureToolCallPairs ensures assistant messages with tool calls
// have their corresponding tool result messages included.
func (s *SlidingWindowStrategy) ensureToolCallPairs(truncated, original []*llms.UnifiedMessage) []*llms.UnifiedMessage {
	// Build set of included message indices
	includedIndices := make(map[int]bool)
	for _, msg := range truncated {
		// Find index in original
		for i, origMsg := range original {
			if msg == origMsg {
				includedIndices[i] = true
				break
			}
		}
	}

	// Check each assistant message with tool calls
	additionalMsgs := []*llms.UnifiedMessage{}
	for _, msg := range truncated {
		if msg.Role() == llms.MessageRoleAssistant && len(msg.ToolCalls()) > 0 {
			// Find corresponding tool results in original
			origIdx := -1
			for j, origMsg := range original {
				if msg == origMsg {
					origIdx = j
					break
				}
			}

			if origIdx >= 0 {
				// Look for tool results after this message
				for j := origIdx + 1; j < len(original); j++ {
					if original[j].Role() == llms.MessageRoleTool {
						// Check if already included
						if !includedIndices[j] {
							additionalMsgs = append(additionalMsgs, original[j])
							includedIndices[j] = true
						}
					} else if original[j].Role() == llms.MessageRoleAssistant || original[j].Role() == llms.MessageRoleUser {
						// Stop at next assistant/user message
						break
					}
				}
			}
		}
	}

	// Insert additional tool messages after their corresponding assistant messages
	if len(additionalMsgs) > 0 {
		agentforge.Debug("Added %d tool result messages to maintain call/result pairs", len(additionalMsgs))
		truncated = append(truncated, additionalMsgs...)
	}

	return truncated
}
