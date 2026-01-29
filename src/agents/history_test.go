package agents

import (
	"testing"

	"github.com/thinktwiceco/agent-forge/src/llms"
)

// MockPersistence
type MockPersistence struct {
	SaveHistoryFunc func(chatId string, history []*llms.UnifiedMessage) string
	GetHistoryFunc  func(chatId string, limit, offset int) []*llms.UnifiedMessage
}

func (m *MockPersistence) SaveHistory(chatId string, history []*llms.UnifiedMessage) string {
	if m.SaveHistoryFunc != nil {
		return m.SaveHistoryFunc(chatId, history)
	}
	return chatId
}

func (m *MockPersistence) GetHistory(chatId string, limit, offset int) []*llms.UnifiedMessage {
	if m.GetHistoryFunc != nil {
		return m.GetHistoryFunc(chatId, limit, offset)
	}
	return nil
}

func TestHistory_AddMessages(t *testing.T) {
	h := &History{
		history: []*llms.UnifiedMessage{},
	}

	h.addUserMessage("user msg")
	if len(h.History()) != 1 {
		t.Errorf("Expected 1 message, got %d", len(h.History()))
	}
	if h.History()[0].Role() != llms.MessageRoleUser {
		t.Errorf("Expected role user, got %s", h.History()[0].Role())
	}

	h.addAssistantMessage("assistant msg", 0, 0, 0)
	if len(h.History()) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(h.History()))
	}

	h.addSystemMessage("system msg")
	// System message should be prepend
	if len(h.History()) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(h.History()))
	}
	if h.History()[0].Role() != llms.MessageRoleSystem {
		t.Errorf("Expected first message to be system, got %s", h.History()[0].Role())
	}

	// Adding another system message should do nothing (if logic correct? code says if !h.hasSystemMessage)
	// But did addSystemMessage set a flag? Yes.
	h.addSystemMessage("new system msg")
	if len(h.History()) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(h.History()))
	}
	// Content shouldn't change
	if h.History()[0].Content() != "system msg" {
		t.Errorf("System message changed")
	}
}

func TestHistory_ToolMessages(t *testing.T) {
	h := &History{history: []*llms.UnifiedMessage{}}

	h.addAssistantMessageWithToolCalls("calling tool", []llms.ToolCall{{ID: "1"}}, 0, 0, 0)
	if len(h.History()) != 1 {
		t.Error("Failed to add tool call message")
	}

	msg := h.History()[0]
	if len(msg.ToolCalls()) != 1 {
		t.Error("Tool calls not set")
	}

	h.addToolMessage("1", "result", true)
	if len(h.History()) != 2 {
		t.Error("Failed to add tool result")
	}
	if !h.History()[1].Ephemeral() {
		t.Error("Tool message should be ephemeral")
	}
}

func TestHistory_Save(t *testing.T) {
	mockP := &MockPersistence{
		SaveHistoryFunc: func(chatId string, history []*llms.UnifiedMessage) string {
			return "new-id"
		},
	}

	h := &History{
		history:     []*llms.UnifiedMessage{llms.ToolMessage("1", "res", true)}, // One ephemeral message
		persistence: mockP,
		chatId:      "old-id",
	}

	newId := h.save()
	if newId != "new-id" {
		t.Errorf("Expected new-id, got %s", newId)
	}

	// Check if ephemeral content replaced
	if h.History()[0].Content() != "[Tool Call Executed]" {
		t.Errorf("Ephemeral message content not redacted: %s", h.History()[0].Content())
	}
}
