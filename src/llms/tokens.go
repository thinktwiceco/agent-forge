package llms

import (
	"encoding/json"
	"strings"
	"unicode"
)

// TokenCounter estimates token counts for text and messages.
// This interface allows for different counting implementations
// (approximate, tiktoken, etc.) while maintaining a consistent API.
type TokenCounter interface {
	// CountTokens estimates the number of tokens in the given text
	CountTokens(text string) int

	// CountMessageTokens estimates tokens for a complete message
	// including role, content, and metadata overhead
	CountMessageTokens(message *UnifiedMessage) int

	// CountMessagesTokens estimates total tokens for a slice of messages
	CountMessagesTokens(messages []*UnifiedMessage) int
}

// ApproximateTokenCounter implements TokenCounter using heuristic estimation.
// Accuracy: ~80-90% compared to tiktoken. Fast and requires no external dependencies.
//
// Algorithm:
//   - For natural language: ~1.3 tokens per word
//   - For code/symbols: ~0.25 tokens per character
//   - Blended approach: average of both metrics
//   - Message overhead: ~4-8 tokens per message for role/formatting
type ApproximateTokenCounter struct{}

// CountTokens estimates tokens using a blended heuristic approach.
func (c *ApproximateTokenCounter) CountTokens(text string) int {
	if text == "" {
		return 0
	}

	// Count words (whitespace-separated)
	words := len(strings.Fields(text))

	// Count characters
	chars := len(text)

	// Count symbols (indicators of code/technical content)
	symbols := countSymbols(text)

	// Heuristic: Natural language ~1.3 tokens/word, code ~4 chars/token
	wordBasedEstimate := float64(words) * 1.3
	charBasedEstimate := float64(chars) / 4.0

	// If high symbol density, weight char-based estimate more
	symbolRatio := float64(symbols) / float64(chars)
	if symbolRatio > 0.15 {
		// Code-heavy content
		return int((charBasedEstimate*0.7 + wordBasedEstimate*0.3))
	}

	// Balanced content - average both approaches
	return int((wordBasedEstimate + charBasedEstimate) / 2.0)
}

// CountMessageTokens estimates tokens for a complete message.
// Includes overhead for role, formatting, and tool calls.
func (c *ApproximateTokenCounter) CountMessageTokens(message *UnifiedMessage) int {
	if message == nil {
		return 0
	}

	tokens := 0

	// Base overhead per message (role markers, formatting)
	tokens += 4

	// Content tokens
	tokens += c.CountTokens(message.Content())

	// Tool call overhead
	if len(message.ToolCalls()) > 0 {
		for _, toolCall := range message.ToolCalls() {
			// Tool call ID and name overhead
			tokens += 8
			tokens += c.CountTokens(toolCall.Name)

			// Serialize arguments to estimate their token count
			if len(toolCall.Arguments) > 0 {
				argsJSON, err := json.Marshal(toolCall.Arguments)
				if err == nil {
					tokens += c.CountTokens(string(argsJSON))
				}
			}
		}
	}

	// Tool message overhead (tool call ID)
	if message.Role() == MessageRoleTool {
		tokens += 4
	}

	return tokens
}

// CountMessagesTokens estimates total tokens for multiple messages.
func (c *ApproximateTokenCounter) CountMessagesTokens(messages []*UnifiedMessage) int {
	total := 0
	for _, msg := range messages {
		total += c.CountMessageTokens(msg)
	}
	return total
}

// countSymbols counts special characters that indicate code/technical content.
func countSymbols(text string) int {
	count := 0
	for _, r := range text {
		if !unicode.IsLetter(r) && !unicode.IsSpace(r) {
			count++
		}
	}
	return count
}

// NewTokenCounter creates a TokenCounter based on the specified method.
// Currently supported: "approximate" (default)
// Future: "tiktoken", "exact"
func NewTokenCounter(method string) TokenCounter {
	switch method {
	case "approximate", "":
		return &ApproximateTokenCounter{}
	default:
		// Default to approximate if unknown method
		return &ApproximateTokenCounter{}
	}
}
