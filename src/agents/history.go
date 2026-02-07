package agents

import (
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/persistence"
)

type History struct {
	history          []*llms.UnifiedMessage
	hasSystemMessage bool
	persistence      persistence.Persistence
	chatId           string
}

func (h *History) History() []*llms.UnifiedMessage {
	return h.history
}

func (h *History) ChatId() string {
	return h.chatId
}

func (h *History) addUserMessage(message string) {
	h.history = append(h.history, llms.UserMessage(message))
}

func (h *History) addSystemMessage(message string) {
	// System message should be the first message in the history
	if !h.hasSystemMessage {
		h.history = append([]*llms.UnifiedMessage{llms.SystemMessage(message)}, h.history...)
		h.hasSystemMessage = true
	}
}

func (h *History) addAssistantMessage(message string, promptTokens, completionTokens, totalTokens int) {
	h.history = append(h.history, llms.AssistantMessage(message, promptTokens, completionTokens, totalTokens))
}

func (h *History) addAssistantMessageWithToolCalls(content string, toolCalls []llms.ToolCall, promptTokens, completionTokens, totalTokens int) {
	h.history = append(h.history, llms.AssistantMessageWithToolCalls(content, toolCalls, promptTokens, completionTokens, totalTokens))
}

func (h *History) addToolMessage(toolCallID, result string, ephemeral bool) {
	h.history = append(h.history, llms.ToolMessage(toolCallID, result, ephemeral))
}

func (h *History) save() string {
	if h.persistence != nil {
		// Cleanup the history to remove ephemeral messages
		for _, message := range h.history {
			if message.Ephemeral() {
				message.SetContent("[Tool Call Executed]")
			}
		}
		h.chatId = h.persistence.SaveHistory(h.chatId, h.history)
	}
	return h.chatId
}

// get loads conversation history from persistence for the specified chatId.
// It updates the hasSystemMessage flag based on whether the loaded history
// contains a system message, ensuring correct behavior when reusing History
// objects across multiple ChatStream calls.
func (h *History) get(chatId string) {
	h.chatId = chatId
	if h.persistence != nil {
		h.history = h.persistence.GetHistory(chatId, 0, 0)
		// Update hasSystemMessage flag based on loaded history
		// This is crucial when the History object is reused across calls
		h.hasSystemMessage = false
		if len(h.history) > 0 && h.history[0].Role() == llms.MessageRoleSystem {
			h.hasSystemMessage = true
		}
	}
}
