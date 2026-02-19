package agents

import (
	"testing"

	agentctx "github.com/thinktwiceco/agent-forge/src/agents/context"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

func TestAgent_truncateHistory_Disabled(t *testing.T) {
	// Agent with truncation disabled (maxContextTokens = 0)
	agent := &Agent{
		config: &AgentConfig{
			TruncationStrategy: nil,
		},
		maxContextTokens: 0,
	}

	messages := []*llms.UnifiedMessage{
		llms.SystemMessage("System"),
		llms.UserMessage("Hello"),
		llms.AssistantMessage("Hi", 0, 0, 0),
	}

	result := agent.truncateHistory(messages)

	if len(result) != len(messages) {
		t.Errorf("Expected no truncation, got %d messages instead of %d", len(result), len(messages))
	}
}

func TestAgent_truncateHistory_UnderLimit(t *testing.T) {
	tokenCounter := llms.NewTokenCounter("approximate")
	agent := &Agent{
		config: &AgentConfig{
			TruncationStrategy: agentctx.NewSlidingWindow(5, tokenCounter),
		},
		maxContextTokens:     10000,
		reservedOutputTokens: 1000,
		minRecentMessages:    5,
		tokenCounter:         tokenCounter,
	}

	messages := []*llms.UnifiedMessage{
		llms.SystemMessage("You are helpful"),
		llms.UserMessage("Hello"),
		llms.AssistantMessage("Hi there", 0, 0, 0),
	}

	result := agent.truncateHistory(messages)

	if len(result) != len(messages) {
		t.Errorf("Expected no truncation for small history, got %d messages instead of %d", len(result), len(messages))
	}
}

func TestAgent_truncateHistory_ExceedsLimit(t *testing.T) {
	tokenCounter := llms.NewTokenCounter("approximate")
	agent := &Agent{
		config: &AgentConfig{
			TruncationStrategy: agentctx.NewSlidingWindow(2, tokenCounter),
		},
		maxContextTokens:     200, // Small limit to trigger truncation
		reservedOutputTokens: 50,
		minRecentMessages:    2,
		tokenCounter:         tokenCounter,
	}

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

	result := agent.truncateHistory(messages)

	// Should have truncated some messages
	if len(result) >= len(messages) {
		t.Errorf("Expected truncation, but got same number of messages: %d", len(result))
	}

	// Should keep system message
	if len(result) == 0 || result[0].Role() != llms.MessageRoleSystem {
		t.Error("Expected system message to be preserved")
	}

	// Should keep at least minRecentMessages
	if len(result) < agent.minRecentMessages+1 { // +1 for system message
		t.Errorf("Expected at least %d recent messages + system, got %d", agent.minRecentMessages, len(result))
	}
}

func TestAgent_truncateHistory_PreservesToolCallPairs(t *testing.T) {
	tokenCounter := llms.NewTokenCounter("approximate")
	agent := &Agent{
		config: &AgentConfig{
			TruncationStrategy: agentctx.NewSlidingWindow(2, tokenCounter),
		},
		maxContextTokens:     150,
		reservedOutputTokens: 20,
		minRecentMessages:    2,
		tokenCounter:         tokenCounter,
	}

	toolCalls := []llms.ToolCall{
		{ID: "call-1", Name: "search", Arguments: map[string]any{"query": "test"}},
	}

	messages := []*llms.UnifiedMessage{
		llms.SystemMessage("System"),
		llms.UserMessage("Old message that might get truncated"),
		llms.AssistantMessage("Old response", 0, 0, 0),
		llms.UserMessage("Search for something"),
		llms.AssistantMessageWithToolCalls("Searching", "", toolCalls, 0, 0, 0),
		llms.ToolMessage("call-1", "Found results", false),
		llms.AssistantMessage("Here are the results", 0, 0, 0),
	}

	result := agent.truncateHistory(messages)

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

func TestAgent_truncateHistory_WithNoTruncationStrategy(t *testing.T) {
	agent := &Agent{
		config: &AgentConfig{
			TruncationStrategy: agentctx.NewNoTruncation(),
		},
		maxContextTokens:     200,
		reservedOutputTokens: 50,
		minRecentMessages:    3,
		tokenCounter:         llms.NewTokenCounter("approximate"),
	}

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

	result := agent.truncateHistory(messages)

	// NoTruncationStrategy should return all messages unchanged
	if len(result) != len(messages) {
		t.Errorf("Expected no truncation with NoTruncationStrategy, got %d messages instead of %d", len(result), len(messages))
	}
}
