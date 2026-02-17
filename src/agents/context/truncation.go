package context

import (
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// TruncationStrategy defines how message history should be truncated
// when it exceeds the available context window.
//
// Different strategies can be implemented to handle truncation
// in various ways (sliding window, summarization, importance-based, etc.)
type TruncationStrategy interface {
	// Truncate reduces message history to fit within maxTokens budget.
	// The implementation should preserve message coherence and critical context.
	//
	// Parameters:
	//   - messages: The full message history to potentially truncate
	//   - maxTokens: Maximum allowed tokens for the returned history
	//
	// Returns:
	//   - Truncated message slice that fits within maxTokens budget
	Truncate(messages []*llms.UnifiedMessage, maxTokens int) []*llms.UnifiedMessage
}
