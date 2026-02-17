package mocks

import (
	"github.com/thinktwiceco/agent-forge/src/agents"
)

// MockPromptBuilder is a mock implementation of agents.PromptBuilder for testing.
type MockPromptBuilder struct {
	BuildFunc        func() string
	UpdateConfigFunc func(config interface{})
}

// Ensure MockPromptBuilder implements agents.PromptBuilder
var _ agents.PromptBuilder = (*MockPromptBuilder)(nil)

func (m *MockPromptBuilder) Build() string {
	if m.BuildFunc != nil {
		return m.BuildFunc()
	}
	return "mock system prompt"
}

func (m *MockPromptBuilder) UpdateConfig(config interface{}) {
	if m.UpdateConfigFunc != nil {
		m.UpdateConfigFunc(config)
	}
}
