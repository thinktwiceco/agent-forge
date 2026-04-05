package history

import (
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/persistence"
)

// TokenUsage holds token counts for a completion.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Manager defines the interface for conversation history management.
// It can be implemented for different backends and used independently of the agents package.
type Manager interface {
	AddUserMessage(msg string)
	AddUserMessageWithImages(text string, imageURLs ...string)
	AddSystemMessage(msg string)
	AddAssistantMessage(msg string, tokens TokenUsage)
	AddAssistantMessageWithToolCalls(content string, reasoningContent string, toolCalls []llms.ToolCall, tokens TokenUsage)
	AddToolMessage(toolCallID, result string, ephemeral bool)
	Messages() []*llms.UnifiedMessage
	SetMessages(messages []*llms.UnifiedMessage)
	ChatId() string
	Save() (string, error)
	Load(chatId string) error
}

// ConversationHistory implements Manager with persistence support.
type ConversationHistory struct {
	messages         []*llms.UnifiedMessage
	hasSystemMessage bool
	persistence      persistence.Persistence
	chatId           string
}

// NewConversationHistory creates a new ConversationHistory with optional persistence.
func NewConversationHistory(opts ...Option) *ConversationHistory {
	ch := &ConversationHistory{
		messages: []*llms.UnifiedMessage{},
	}
	for _, opt := range opts {
		opt(ch)
	}
	return ch
}

// Option configures a ConversationHistory.
type Option func(*ConversationHistory)

// WithPersistence sets the persistence layer.
func WithPersistence(p persistence.Persistence) Option {
	return func(ch *ConversationHistory) {
		ch.persistence = p
	}
}

// WithChatId sets the initial chat ID.
func WithChatId(chatId string) Option {
	return func(ch *ConversationHistory) {
		ch.chatId = chatId
	}
}

// AddUserMessage adds a user message to the history.
func (ch *ConversationHistory) AddUserMessage(msg string) {
	ch.messages = append(ch.messages, llms.UserMessage(msg))
}

// AddUserMessageWithImages adds a multimodal user message with text and image URLs (data URIs or https URLs).
func (ch *ConversationHistory) AddUserMessageWithImages(text string, imageURLs ...string) {
	ch.messages = append(ch.messages, llms.UserMessageWithImages(text, imageURLs...))
}

// AddSystemMessage adds a system message. It is prepended as the first message if none exists.
func (ch *ConversationHistory) AddSystemMessage(msg string) {
	if !ch.hasSystemMessage {
		ch.messages = append([]*llms.UnifiedMessage{llms.SystemMessage(msg)}, ch.messages...)
		ch.hasSystemMessage = true
	}
}

// AddAssistantMessage adds an assistant message with token usage.
func (ch *ConversationHistory) AddAssistantMessage(msg string, tokens TokenUsage) {
	ch.messages = append(ch.messages, llms.AssistantMessage(msg, tokens.PromptTokens, tokens.CompletionTokens, tokens.TotalTokens))
}

// AddAssistantMessageWithToolCalls adds an assistant message with tool calls.
func (ch *ConversationHistory) AddAssistantMessageWithToolCalls(content string, reasoningContent string, toolCalls []llms.ToolCall, tokens TokenUsage) {
	ch.messages = append(ch.messages, llms.AssistantMessageWithToolCalls(content, reasoningContent, toolCalls, tokens.PromptTokens, tokens.CompletionTokens, tokens.TotalTokens))
}

// AddToolMessage adds a tool result message.
func (ch *ConversationHistory) AddToolMessage(toolCallID, result string, ephemeral bool) {
	ch.messages = append(ch.messages, llms.ToolMessage(toolCallID, result, ephemeral))
}

// Messages returns the conversation messages.
func (ch *ConversationHistory) Messages() []*llms.UnifiedMessage {
	return ch.messages
}

// SetMessages replaces the messages (e.g., for truncation).
func (ch *ConversationHistory) SetMessages(messages []*llms.UnifiedMessage) {
	ch.messages = messages
	ch.hasSystemMessage = false
	if len(ch.messages) > 0 && ch.messages[0].Role() == llms.MessageRoleSystem {
		ch.hasSystemMessage = true
	}
}

// ChatId returns the conversation ID.
func (ch *ConversationHistory) ChatId() string {
	return ch.chatId
}

// Save persists the history and returns the chat ID (possibly newly generated) and any error.
func (ch *ConversationHistory) Save() (string, error) {
	if ch.persistence != nil {
		for _, message := range ch.messages {
			if message.Ephemeral() {
				message.SetContent("[Tool Call Executed]")
			}
		}
		newChatId, err := ch.persistence.SaveHistory(ch.chatId, ch.messages)
		if err != nil {
			return ch.chatId, err
		}
		ch.chatId = newChatId
	}
	return ch.chatId, nil
}

// Load loads conversation history from persistence for the specified chatId.
func (ch *ConversationHistory) Load(chatId string) error {
	ch.chatId = chatId
	if ch.persistence != nil {
		ch.messages = ch.persistence.GetHistory(chatId, 0, 0)
		ch.hasSystemMessage = false
		if len(ch.messages) > 0 && ch.messages[0].Role() == llms.MessageRoleSystem {
			ch.hasSystemMessage = true
		}
	}
	return nil
}
