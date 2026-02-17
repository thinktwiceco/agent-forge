package context

import (
	"testing"

	"github.com/thinktwiceco/agent-forge/src/llms"
)

// ===============================
// ===== NoTruncationStrategy Tests
// ===============================

func TestNoTruncationStrategy_ReturnsUnchanged(t *testing.T) {
	strategy := NewNoTruncation()

	messages := []*llms.UnifiedMessage{
		llms.SystemMessage("System"),
		llms.UserMessage("Hello"),
		llms.AssistantMessage("Hi", 0, 0, 0),
		llms.UserMessage("How are you?"),
	}

	result := strategy.Truncate(messages, 100)

	if len(result) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(result))
	}

	for i, msg := range messages {
		if result[i] != msg {
			t.Errorf("Message at index %d was modified", i)
		}
	}
}

func TestNoTruncationStrategy_EmptyMessages(t *testing.T) {
	strategy := NewNoTruncation()
	messages := []*llms.UnifiedMessage{}

	result := strategy.Truncate(messages, 100)

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d messages", len(result))
	}
}

// ================================
// ===== SlidingWindowStrategy Tests
// ================================

func TestSlidingWindowStrategy_NilTokenCounter(t *testing.T) {
	strategy := NewSlidingWindow(2, nil)

	messages := []*llms.UnifiedMessage{
		llms.SystemMessage("System"),
		llms.UserMessage("Hello"),
		llms.AssistantMessage("Hi", 0, 0, 0),
	}

	result := strategy.Truncate(messages, 100)

	// Should return unchanged when token counter is nil
	if len(result) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(result))
	}
}

func TestSlidingWindowStrategy_UnderLimit(t *testing.T) {
	tokenCounter := llms.NewTokenCounter("approximate")
	strategy := NewSlidingWindow(3, tokenCounter)

	messages := []*llms.UnifiedMessage{
		llms.SystemMessage("You are helpful"),
		llms.UserMessage("Hello"),
		llms.AssistantMessage("Hi there", 0, 0, 0),
	}

	result := strategy.Truncate(messages, 10000)

	if len(result) != len(messages) {
		t.Errorf("Expected no truncation for small history, got %d messages instead of %d", len(result), len(messages))
	}
}

func TestSlidingWindowStrategy_ExceedsLimit(t *testing.T) {
	tokenCounter := llms.NewTokenCounter("approximate")
	strategy := NewSlidingWindow(2, tokenCounter)

	// Create a long history that exceeds the limit
	messages := []*llms.UnifiedMessage{
		llms.SystemMessage("You are a helpful assistant with very detailed instructions about how to behave"),
		llms.UserMessage("Tell me about the complete history of computers starting from ancient times through modern era"),
		llms.AssistantMessage("Computers have a long and fascinating history spanning thousands of years...", 0, 0, 0),
		llms.UserMessage("What about the development of modern personal computers in detail?"),
		llms.AssistantMessage("Modern personal computers evolved through many stages and innovations...", 0, 0, 0),
		llms.UserMessage("Tell me about the future of computing and artificial intelligence"),
		llms.AssistantMessage("The future of computing includes quantum computers and advanced AI systems...", 0, 0, 0),
		llms.UserMessage("What's the latest development?"),
	}

	result := strategy.Truncate(messages, 50) // Very small limit to force truncation

	// Should have truncated some messages
	if len(result) >= len(messages) {
		t.Errorf("Expected truncation, but got same number of messages: %d", len(result))
	}

	// Should keep system message
	if len(result) == 0 || result[0].Role() != llms.MessageRoleSystem {
		t.Error("Expected system message to be preserved")
	}

	// Should keep at least minRecentMessages
	if len(result) < 2+1 { // +1 for system message
		t.Errorf("Expected at least 2 recent messages + system, got %d", len(result))
	}
}

func TestSlidingWindowStrategy_KeepsRecentMessages(t *testing.T) {
	tokenCounter := llms.NewTokenCounter("approximate")
	strategy := NewSlidingWindow(3, tokenCounter)

	messages := []*llms.UnifiedMessage{
		llms.SystemMessage("System message here"),
		llms.UserMessage("Message 1"),
		llms.AssistantMessage("Response 1", 0, 0, 0),
		llms.UserMessage("Message 2"),
		llms.AssistantMessage("Response 2", 0, 0, 0),
		llms.UserMessage("Message 3"),
		llms.AssistantMessage("Response 3", 0, 0, 0),
		llms.UserMessage("Message 4 - most recent"),
	}

	result := strategy.Truncate(messages, 150)

	// Should have system message
	if len(result) == 0 || result[0].Role() != llms.MessageRoleSystem {
		t.Error("System message should be preserved")
	}

	// Last message should be the most recent user message
	if len(result) > 0 {
		lastMsg := result[len(result)-1]
		if lastMsg.Content() != "Message 4 - most recent" {
			t.Errorf("Expected most recent message to be preserved, got: %s", lastMsg.Content())
		}
	}
}

func TestSlidingWindowStrategy_PreservesToolCallPairs(t *testing.T) {
	tokenCounter := llms.NewTokenCounter("approximate")
	strategy := NewSlidingWindow(2, tokenCounter)

	toolCalls := []llms.ToolCall{
		{ID: "call-1", Name: "search", Arguments: map[string]any{"query": "test"}},
	}

	messages := []*llms.UnifiedMessage{
		llms.SystemMessage("System"),
		llms.UserMessage("Old message that might get truncated"),
		llms.AssistantMessage("Old response", 0, 0, 0),
		llms.UserMessage("Search for something"),
		llms.AssistantMessageWithToolCalls("Searching", toolCalls, 0, 0, 0),
		llms.ToolMessage("call-1", "Found results", false),
		llms.AssistantMessage("Here are the results", 0, 0, 0),
	}

	result := strategy.Truncate(messages, 150)

	// Find assistant message with tool calls
	hasToolCall := false
	hasToolResult := false
	for i, msg := range result {
		if msg.Role() == llms.MessageRoleAssistant && len(msg.ToolCalls()) > 0 {
			hasToolCall = true
			// Check if corresponding tool result is present
			if i+1 < len(result) && result[i+1].Role() == llms.MessageRoleTool {
				hasToolResult = true
			}
		}
	}

	if hasToolCall && !hasToolResult {
		t.Error("Tool call present but corresponding tool result missing after truncation")
	}
}

func TestSlidingWindowStrategy_EmptyMessages(t *testing.T) {
	tokenCounter := llms.NewTokenCounter("approximate")
	strategy := NewSlidingWindow(2, tokenCounter)

	messages := []*llms.UnifiedMessage{}
	result := strategy.Truncate(messages, 100)

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d messages", len(result))
	}
}

func TestSlidingWindowStrategy_OnlySystemMessage(t *testing.T) {
	tokenCounter := llms.NewTokenCounter("approximate")
	strategy := NewSlidingWindow(2, tokenCounter)

	messages := []*llms.UnifiedMessage{
		llms.SystemMessage("System message"),
	}

	result := strategy.Truncate(messages, 100)

	if len(result) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result))
	}

	if result[0].Role() != llms.MessageRoleSystem {
		t.Error("Expected system message to be preserved")
	}
}

func TestSlidingWindowStrategy_NoSystemMessage(t *testing.T) {
	tokenCounter := llms.NewTokenCounter("approximate")
	strategy := NewSlidingWindow(2, tokenCounter)

	messages := []*llms.UnifiedMessage{
		llms.UserMessage("Message 1"),
		llms.AssistantMessage("Response 1", 0, 0, 0),
		llms.UserMessage("Message 2"),
		llms.AssistantMessage("Response 2", 0, 0, 0),
		llms.UserMessage("Message 3"),
	}

	result := strategy.Truncate(messages, 100)

	// Should keep recent messages even without system message
	if len(result) < 2 {
		t.Errorf("Expected at least 2 recent messages, got %d", len(result))
	}

	// Last message should be preserved
	if result[len(result)-1].Content() != "Message 3" {
		t.Error("Most recent message should be preserved")
	}
}

// ================================
// ===== Strategy Integration Tests
// ================================

func TestTruncationStrategy_Interface(t *testing.T) {
	// Verify both strategies implement the interface
	var _ TruncationStrategy = &NoTruncationStrategy{}
	var _ TruncationStrategy = &SlidingWindowStrategy{}
}

func TestTruncationStrategy_Swappable(t *testing.T) {
	// Test that strategies can be swapped at runtime
	tokenCounter := llms.NewTokenCounter("approximate")

	messages := []*llms.UnifiedMessage{
		llms.SystemMessage("System"),
		llms.UserMessage("Hello"),
		llms.AssistantMessage("Hi", 0, 0, 0),
	}

	// Use NoTruncation first
	var strategy TruncationStrategy = NewNoTruncation()
	result1 := strategy.Truncate(messages, 10)
	if len(result1) != len(messages) {
		t.Error("NoTruncation should not modify messages")
	}

	// Swap to SlidingWindow
	strategy = NewSlidingWindow(1, tokenCounter)
	result2 := strategy.Truncate(messages, 10)
	// Result may or may not be truncated depending on token count,
	// but the strategy swap should work without error
	if result2 == nil {
		t.Error("SlidingWindow should return non-nil result")
	}
}
