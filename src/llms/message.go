package llms

import "encoding/json"

/// Common message interface

// ContentPartType identifies the kind of content within a multimodal message.
type ContentPartType string

const (
	ContentPartTypeText     ContentPartType = "text"
	ContentPartTypeImageURL ContentPartType = "image_url"
)

// ContentPart is a single element in a multimodal user message.
// Use Type to discriminate: Text is populated for text parts,
// ImageURL is populated for image parts (data URI or https URL).
type ContentPart struct {
	Type     ContentPartType
	Text     string
	ImageURL string
}

type MessageRole string

func (r MessageRole) String() string {
	return string(r)
}

const (
	MessageRoleSystem    MessageRole = "system"
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleTool      MessageRole = "tool"
)

type UnifiedMessage struct {
	role             MessageRole
	content          string
	contentParts     []ContentPart // Non-nil when message carries multimodal content
	reasoningContent string        // For assistant messages - reasoning content from thinking models (e.g. DeepSeek Reasoner)
	toolCallID       string        // For tool messages - the ID of the tool call this responds to
	toolCalls        []ToolCall    // For assistant messages - tool calls made by the assistant
	promptTokens     int           // Input tokens consumed
	completionTokens int           // Output tokens generated
	totalTokens      int           // Total tokens used
	ephemeral        bool          // Whether the message is ephemeral. If ephemeral is not useful for the chat history, it should be set to true.
}

func (m *UnifiedMessage) Role() MessageRole {
	return m.role
}

func (m *UnifiedMessage) Content() string {
	return m.content
}

// ContentParts returns the multimodal content parts when present.
// Returns nil for plain text messages.
func (m *UnifiedMessage) ContentParts() []ContentPart {
	return m.contentParts
}

func (m *UnifiedMessage) ToolCallID() string {
	return m.toolCallID
}

func (m *UnifiedMessage) ReasoningContent() string {
	return m.reasoningContent
}

func (m *UnifiedMessage) ToolCalls() []ToolCall {
	return m.toolCalls
}

func (m *UnifiedMessage) PromptTokens() int {
	return m.promptTokens
}

func (m *UnifiedMessage) CompletionTokens() int {
	return m.completionTokens
}

func (m *UnifiedMessage) TotalTokens() int {
	return m.totalTokens
}

func (m *UnifiedMessage) Ephemeral() bool {
	return m.ephemeral
}

func (m *UnifiedMessage) SetContent(content string) {
	m.content = content
}

func UserMessage(content string) *UnifiedMessage {
	return &UnifiedMessage{
		role:    MessageRoleUser,
		content: content,
	}
}

// UserMessageWithImages creates a multimodal user message containing text and one or more
// images. Each imageURL may be a data URI (e.g. "data:image/png;base64,...") or an
// https URL pointing to a publicly accessible image.
func UserMessageWithImages(text string, imageURLs ...string) *UnifiedMessage {
	parts := make([]ContentPart, 0, 1+len(imageURLs))
	if text != "" {
		parts = append(parts, ContentPart{Type: ContentPartTypeText, Text: text})
	}
	for _, url := range imageURLs {
		parts = append(parts, ContentPart{Type: ContentPartTypeImageURL, ImageURL: url})
	}
	return &UnifiedMessage{
		role:         MessageRoleUser,
		content:      text,
		contentParts: parts,
	}
}

func AssistantMessage(content string, promptTokens, completionTokens, totalTokens int) *UnifiedMessage {
	return &UnifiedMessage{
		role:             MessageRoleAssistant,
		content:          content,
		promptTokens:     promptTokens,
		completionTokens: completionTokens,
		totalTokens:      totalTokens,
	}
}

func SystemMessage(content string) *UnifiedMessage {
	return &UnifiedMessage{
		role:    MessageRoleSystem,
		content: content,
	}
}

func ToolMessage(toolCallID, content string, ephemeral bool) *UnifiedMessage {
	return &UnifiedMessage{
		role:       MessageRoleTool,
		content:    content,
		toolCallID: toolCallID,
		ephemeral:  ephemeral,
	}
}

func AssistantMessageWithToolCalls(content string, reasoningContent string, toolCalls []ToolCall, promptTokens, completionTokens, totalTokens int) *UnifiedMessage {
	return &UnifiedMessage{
		role:             MessageRoleAssistant,
		content:          content,
		reasoningContent: reasoningContent,
		toolCalls:        toolCalls,
		promptTokens:     promptTokens,
		completionTokens: completionTokens,
		totalTokens:      totalTokens,
	}
}

// MarshalJSON implements custom JSON marshaling for UnifiedMessage
func (m UnifiedMessage) MarshalJSON() ([]byte, error) {
	type Alias struct {
		Role             MessageRole `json:"role"`
		Content          string      `json:"content"`
		ReasoningContent string      `json:"reasoningContent,omitempty"`
		ToolCallID       string      `json:"toolCallId,omitempty"`
		ToolCalls        []ToolCall  `json:"toolCalls,omitempty"`
		PromptTokens     int         `json:"promptTokens,omitempty"`
		CompletionTokens int         `json:"completionTokens,omitempty"`
		TotalTokens      int         `json:"totalTokens,omitempty"`
	}
	return json.Marshal(Alias{
		Role:             m.role,
		Content:          m.content,
		ReasoningContent: m.reasoningContent,
		ToolCallID:       m.toolCallID,
		ToolCalls:        m.toolCalls,
		PromptTokens:     m.promptTokens,
		CompletionTokens: m.completionTokens,
		TotalTokens:      m.totalTokens,
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for UnifiedMessage
func (m *UnifiedMessage) UnmarshalJSON(data []byte) error {
	type Alias struct {
		Role             MessageRole `json:"role"`
		Content          string      `json:"content"`
		ReasoningContent string      `json:"reasoningContent,omitempty"`
		ToolCallID       string      `json:"toolCallId,omitempty"`
		ToolCalls        []ToolCall  `json:"toolCalls,omitempty"`
		PromptTokens     int         `json:"promptTokens,omitempty"`
		CompletionTokens int         `json:"completionTokens,omitempty"`
		TotalTokens      int         `json:"totalTokens,omitempty"`
	}
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	m.role = alias.Role
	m.content = alias.Content
	m.reasoningContent = alias.ReasoningContent
	m.toolCallID = alias.ToolCallID
	m.toolCalls = alias.ToolCalls
	m.promptTokens = alias.PromptTokens
	m.completionTokens = alias.CompletionTokens
	m.totalTokens = alias.TotalTokens
	return nil
}
