package history

import (
	"testing"

	"github.com/thinktwiceco/agent-forge/src/llms"
)

// mockPersistence implements persistence.Persistence for testing.
type mockPersistence struct {
	saveHistoryFunc func(chatId string, history []*llms.UnifiedMessage) string
	getHistoryFunc  func(chatId string, limit, offset int) []*llms.UnifiedMessage
}

func (m *mockPersistence) SaveHistory(chatId string, history []*llms.UnifiedMessage) string {
	if m.saveHistoryFunc != nil {
		return m.saveHistoryFunc(chatId, history)
	}
	return chatId
}

func (m *mockPersistence) GetHistory(chatId string, limit, offset int) []*llms.UnifiedMessage {
	if m.getHistoryFunc != nil {
		return m.getHistoryFunc(chatId, limit, offset)
	}
	return nil
}

func TestConversationHistory_AddMessages(t *testing.T) {
	h := NewConversationHistory()
	tokens := TokenUsage{PromptTokens: 0, CompletionTokens: 0, TotalTokens: 0}

	h.AddUserMessage("user msg")
	if len(h.Messages()) != 1 {
		t.Errorf("Expected 1 message, got %d", len(h.Messages()))
	}
	if h.Messages()[0].Role() != llms.MessageRoleUser {
		t.Errorf("Expected role user, got %s", h.Messages()[0].Role())
	}

	h.AddAssistantMessage("assistant msg", tokens)
	if len(h.Messages()) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(h.Messages()))
	}

	h.AddSystemMessage("system msg")
	if len(h.Messages()) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(h.Messages()))
	}
	if h.Messages()[0].Role() != llms.MessageRoleSystem {
		t.Errorf("Expected first message to be system, got %s", h.Messages()[0].Role())
	}

	h.AddSystemMessage("new system msg")
	if len(h.Messages()) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(h.Messages()))
	}
	if h.Messages()[0].Content() != "system msg" {
		t.Errorf("System message changed")
	}
}

func TestConversationHistory_ToolMessages(t *testing.T) {
	h := NewConversationHistory()
	tokens := TokenUsage{}

	h.AddAssistantMessageWithToolCalls("calling tool", []llms.ToolCall{{ID: "1"}}, tokens)
	if len(h.Messages()) != 1 {
		t.Error("Failed to add tool call message")
	}

	msg := h.Messages()[0]
	if len(msg.ToolCalls()) != 1 {
		t.Error("Tool calls not set")
	}

	h.AddToolMessage("1", "result", true)
	if len(h.Messages()) != 2 {
		t.Error("Failed to add tool result")
	}
	if !h.Messages()[1].Ephemeral() {
		t.Error("Tool message should be ephemeral")
	}
}

func TestConversationHistory_Save(t *testing.T) {
	mockP := &mockPersistence{
		saveHistoryFunc: func(chatId string, history []*llms.UnifiedMessage) string {
			return "new-id"
		},
	}

	h := NewConversationHistory(
		WithPersistence(mockP),
		WithChatId("old-id"),
	)
	h.SetMessages([]*llms.UnifiedMessage{llms.ToolMessage("1", "res", true)})

	newId := h.Save()
	if newId != "new-id" {
		t.Errorf("Expected new-id, got %s", newId)
	}

	if h.Messages()[0].Content() != "[Tool Call Executed]" {
		t.Errorf("Ephemeral message content not redacted: %s", h.Messages()[0].Content())
	}
}

func TestConversationHistory_Load(t *testing.T) {
	// Case 1: Loading history WITH system message
	mockP1 := &mockPersistence{
		getHistoryFunc: func(chatId string, limit, offset int) []*llms.UnifiedMessage {
			return []*llms.UnifiedMessage{
				llms.SystemMessage("system msg"),
				llms.UserMessage("user msg"),
			}
		},
	}

	h1 := NewConversationHistory(WithPersistence(mockP1))
	if err := h1.Load("test-id"); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(h1.Messages()) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(h1.Messages()))
	}
	if h1.Messages()[0].Role() != llms.MessageRoleSystem {
		t.Errorf("Expected first message to be system, got %s", h1.Messages()[0].Role())
	}

	// Case 2: Loading history WITHOUT system message
	mockP2 := &mockPersistence{
		getHistoryFunc: func(chatId string, limit, offset int) []*llms.UnifiedMessage {
			return []*llms.UnifiedMessage{
				llms.UserMessage("user msg"),
			}
		},
	}

	h2 := NewConversationHistory(WithPersistence(mockP2))
	if err := h2.Load("test-id-2"); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(h2.Messages()) != 1 {
		t.Errorf("Expected 1 message, got %d", len(h2.Messages()))
	}

	// Case 3: Loading empty history
	mockP3 := &mockPersistence{
		getHistoryFunc: func(chatId string, limit, offset int) []*llms.UnifiedMessage {
			return []*llms.UnifiedMessage{}
		},
	}

	h3 := NewConversationHistory(WithPersistence(mockP3))
	if err := h3.Load("test-id-3"); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(h3.Messages()) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(h3.Messages()))
	}
}

func TestConversationHistory_SetMessages(t *testing.T) {
	h := NewConversationHistory()
	h.AddUserMessage("original")
	h.AddAssistantMessage("reply", TokenUsage{})

	newMsgs := []*llms.UnifiedMessage{
		llms.SystemMessage("new system"),
		llms.UserMessage("new user"),
	}
	h.SetMessages(newMsgs)

	msgs := h.Messages()
	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages after SetMessages, got %d", len(msgs))
	}
	if msgs[0].Role() != llms.MessageRoleSystem {
		t.Errorf("Expected first message to be system, got %s", msgs[0].Role())
	}
	if msgs[1].Content() != "new user" {
		t.Errorf("Expected content 'new user', got %s", msgs[1].Content())
	}
}
