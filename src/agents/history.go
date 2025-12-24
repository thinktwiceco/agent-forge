package agents

import (
	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/persistence"
)

type History struct {
	history          []llms.UnifiedMessage
	hasSystemMessage bool
	persistence      persistence.Persistence
}

func (h *History) History() []llms.UnifiedMessage {
	return h.history
}

func (h *History) addUserMessage(message string) {
	h.history = append(h.history, llms.UserMessage(message))
}

func (h *History) addSystemMessage(message string) {
	// System message should be the first message in the history
	if !h.hasSystemMessage {
		h.history = append([]llms.UnifiedMessage{llms.SystemMessage(message)}, h.history...)
		h.hasSystemMessage = true
	}
}

func (h *History) addAssistantMessage(message string, promptTokens, completionTokens, totalTokens int) {
	h.history = append(h.history, llms.AssistantMessage(message, promptTokens, completionTokens, totalTokens))
}

func (h *History) addAssistantMessageWithToolCalls(content string, toolCalls []llms.ToolCall, promptTokens, completionTokens, totalTokens int) {
	h.history = append(h.history, llms.AssistantMessageWithToolCalls(content, toolCalls, promptTokens, completionTokens, totalTokens))
}

func (h *History) addToolMessage(toolCallID, result string) {
	h.history = append(h.history, llms.ToolMessage(toolCallID, result))
}

func (h *History) save() {
	if h.persistence != nil {
		h.persistence.SaveHystory(h.history)
	} else {
		agentforge.Warn("No persistence layer configured, history will not be saved")
	}
}

func (h *History) get() {
	var limit = 0
	var offset = 0
	if h.persistence != nil {
		h.history = h.persistence.GetHystory(limit, offset)
	} else {
		agentforge.Warn("No persistence layer configured, history will not be returned")
	}
}
