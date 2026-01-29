package mocks

import (
	"encoding/json"
	"time"

	"github.com/thinktwiceco/agent-forge/src/llms"
)

// MockLLMEngine is a mock implementation of llms.LLMEngine
type MockLLMEngine struct {
	ModelName     string
	ProviderName  string
	Responses     []string // Queue of text responses to return
	RecordedCalls [][]*llms.UnifiedMessage
	ResponseDelay time.Duration
	ToolCalls     []llms.ToolCall
}

func NewMockLLMEngine() *MockLLMEngine {
	return &MockLLMEngine{
		ModelName:     "mock-model",
		ProviderName:  "mock-provider",
		Responses:     []string{},
		RecordedCalls: [][]*llms.UnifiedMessage{},
	}
}

func (m *MockLLMEngine) ChatStream(messages []*llms.UnifiedMessage, tools []llms.Tool) *llms.ResponseCh {
	m.RecordedCalls = append(m.RecordedCalls, messages)
	responseCh := llms.NewResponseCh()

	go func() {
		defer responseCh.Close()

		if m.ResponseDelay > 0 {
			time.Sleep(m.ResponseDelay)
		}

		// Send tool calls if any
		if len(m.ToolCalls) > 0 {
			toolChunk := llms.ChunkResponse{
				Status:    llms.StatusToolCall,
				Type:      llms.TypeToolCall,
				ToolCalls: m.ToolCalls,
			}
			jsonBytes, _ := json.Marshal(toolChunk)
			responseCh.Response <- jsonBytes
			return // Stop after sending tools for this turn
		}

		// Send text response
		var content string
		if len(m.Responses) > 0 {
			content = m.Responses[0]
			m.Responses = m.Responses[1:]
		} else {
			content = "Mock response"
		}

		// Stream content in chunks
		chunkSize := 5
		for i := 0; i < len(content); i += chunkSize {
			end := i + chunkSize
			if end > len(content) {
				end = len(content)
			}
			chunk := content[i:end]

			respChunk := llms.ChunkResponse{
				Content:     chunk,
				Delta:       chunk,
				FullContent: content[:end], // Approximation
				Status:      llms.StatusStreaming,
				Type:        llms.TypeContent,
			}
			jsonBytes, _ := json.Marshal(respChunk)
			responseCh.Response <- jsonBytes
		}

		// Send completion chunk
		finalChunk := llms.ChunkResponse{
			Content:     "",
			FullContent: content,
			Status:      llms.StatusCompleted,
			Type:        llms.TypeCompletion,
		}
		jsonBytes, _ := json.Marshal(finalChunk)
		responseCh.Response <- jsonBytes
	}()

	return responseCh
}

func (m *MockLLMEngine) Model() string {
	return m.ModelName
}

func (m *MockLLMEngine) Provider() string {
	return m.ProviderName
}
