package mocks

import (
	"github.com/thinktwiceco/agent-forge/src/history"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// MockHistoryManager is a mock implementation of history.Manager for testing.
type MockHistoryManager struct {
	AddUserMessageFunc                   func(msg string)
	AddSystemMessageFunc                 func(msg string)
	AddAssistantMessageFunc              func(msg string, tokens history.TokenUsage)
	AddAssistantMessageWithToolCallsFunc func(content string, reasoningContent string, toolCalls []llms.ToolCall, tokens history.TokenUsage)
	AddToolMessageFunc                   func(toolCallID, result string, ephemeral bool)
	MessagesFunc                         func() []*llms.UnifiedMessage
	SetMessagesFunc                      func(messages []*llms.UnifiedMessage)
	ChatIdFunc                           func() string
	SaveFunc                             func() string
	LoadFunc                             func(chatId string) error
}

// Ensure MockHistoryManager implements history.Manager
var _ history.Manager = (*MockHistoryManager)(nil)

func (m *MockHistoryManager) AddUserMessage(msg string) {
	if m.AddUserMessageFunc != nil {
		m.AddUserMessageFunc(msg)
	}
}

func (m *MockHistoryManager) AddSystemMessage(msg string) {
	if m.AddSystemMessageFunc != nil {
		m.AddSystemMessageFunc(msg)
	}
}

func (m *MockHistoryManager) AddAssistantMessage(msg string, tokens history.TokenUsage) {
	if m.AddAssistantMessageFunc != nil {
		m.AddAssistantMessageFunc(msg, tokens)
	}
}

func (m *MockHistoryManager) AddAssistantMessageWithToolCalls(content string, reasoningContent string, toolCalls []llms.ToolCall, tokens history.TokenUsage) {
	if m.AddAssistantMessageWithToolCallsFunc != nil {
		m.AddAssistantMessageWithToolCallsFunc(content, reasoningContent, toolCalls, tokens)
	}
}

func (m *MockHistoryManager) AddToolMessage(toolCallID, result string, ephemeral bool) {
	if m.AddToolMessageFunc != nil {
		m.AddToolMessageFunc(toolCallID, result, ephemeral)
	}
}

func (m *MockHistoryManager) Messages() []*llms.UnifiedMessage {
	if m.MessagesFunc != nil {
		return m.MessagesFunc()
	}
	return []*llms.UnifiedMessage{}
}

func (m *MockHistoryManager) SetMessages(messages []*llms.UnifiedMessage) {
	if m.SetMessagesFunc != nil {
		m.SetMessagesFunc(messages)
	}
}

func (m *MockHistoryManager) ChatId() string {
	if m.ChatIdFunc != nil {
		return m.ChatIdFunc()
	}
	return ""
}

func (m *MockHistoryManager) Save() string {
	if m.SaveFunc != nil {
		return m.SaveFunc()
	}
	return ""
}

func (m *MockHistoryManager) Load(chatId string) error {
	if m.LoadFunc != nil {
		return m.LoadFunc(chatId)
	}
	return nil
}
