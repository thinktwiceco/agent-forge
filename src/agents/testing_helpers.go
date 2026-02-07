package agents

import "github.com/thinktwiceco/agent-forge/src/core"

// mockSubAgent is a mock implementation of the core.SubAgent interface for testing.
type mockSubAgent struct {
	nameFunc               func() string
	basicDescriptionFunc   func() string
	advanceDescriptionFunc func() string
	troubleshootingFunc    func() string
	chatStreamFunc         func(message string, chatId string) *core.ResponseCh
}

// newMockSubAgent creates a new mock subagent with the given functions.
func newMockSubAgent(name, basicDesc, advanceDesc string) *mockSubAgent {
	return &mockSubAgent{
		nameFunc: func() string {
			return name
		},
		basicDescriptionFunc: func() string {
			return basicDesc
		},
		advanceDescriptionFunc: func() string {
			return advanceDesc
		},
		troubleshootingFunc: func() string {
			return "No troubleshooting information available"
		},
	}
}

// Name returns the name of the mock subagent.
func (m *mockSubAgent) Name() string {
	if m.nameFunc != nil {
		return m.nameFunc()
	}
	return "mock-subagent"
}

// BasicDescription returns the basic description of the mock subagent.
func (m *mockSubAgent) BasicDescription() string {
	if m.basicDescriptionFunc != nil {
		return m.basicDescriptionFunc()
	}
	return "A mock subagent for testing"
}

// AdvanceDescription returns the full description of the mock subagent.
func (m *mockSubAgent) AdvanceDescription() string {
	if m.advanceDescriptionFunc != nil {
		return m.advanceDescriptionFunc()
	}
	return "A mock subagent with full description for testing"
}

// Troubleshooting returns troubleshooting information for the mock subagent.
func (m *mockSubAgent) Troubleshooting() string {
	if m.troubleshootingFunc != nil {
		return m.troubleshootingFunc()
	}
	return "No troubleshooting information available for mock subagent"
}

// ChatStream initiates a streaming chat interaction with the mock subagent.
func (m *mockSubAgent) ChatStream(message string, chatId string) *core.ResponseCh {
	if m.chatStreamFunc != nil {
		return m.chatStreamFunc(message, chatId)
	}
	// Return a simple response channel
	return core.NewResponseCh("mock-subagent", "test", "", nil)
}
