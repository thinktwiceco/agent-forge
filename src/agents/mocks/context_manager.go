package mocks

import (
	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// MockContextManager is a mock implementation of agents.ContextManager for testing.
type MockContextManager struct {
	ContextFunc         func() *core.AgentContext
	BuildContextFunc    func(responseCh *core.ResponseCh) map[string]any
	SyncFromMapFunc     func(contextMap map[string]any) error
	TruncateHistoryFunc func(messages []*llms.UnifiedMessage) []*llms.UnifiedMessage
	UpdateConfigFunc    func(config interface{})
	UpdateToolsFunc     func(tools []llms.Tool)
	UpdateSubAgentsFunc func(agents []core.SubAgent)
}

// Ensure MockContextManager implements agents.ContextManager
var _ agents.ContextManager = (*MockContextManager)(nil)

func (m *MockContextManager) Context() *core.AgentContext {
	if m.ContextFunc != nil {
		return m.ContextFunc()
	}
	return &core.AgentContext{
		AgentName:      "mock-agent",
		Trace:          "mock-trace",
		Model:          "mock-model",
		Tools:          []llms.Tool{},
		SubAgents:      []core.SubAgent{},
		SessionStorage: make(map[string]any),
		PluginFields:   make(map[string]any),
	}
}

func (m *MockContextManager) BuildContext(responseCh *core.ResponseCh) map[string]any {
	if m.BuildContextFunc != nil {
		return m.BuildContextFunc(responseCh)
	}
	// Return a basic context map
	return map[string]any{
		"agentName":      "mock-agent",
		"trace":          "mock-trace",
		"model":          "mock-model",
		"responseCh":     responseCh,
		"tools":          []llms.Tool{},
		"subAgents":      []core.SubAgent{},
		"sessionStorage": make(map[string]any),
		"pluginFields":   make(map[string]any),
	}
}

func (m *MockContextManager) SyncFromMap(contextMap map[string]any) error {
	if m.SyncFromMapFunc != nil {
		return m.SyncFromMapFunc(contextMap)
	}
	return nil
}

func (m *MockContextManager) TruncateHistory(messages []*llms.UnifiedMessage) []*llms.UnifiedMessage {
	if m.TruncateHistoryFunc != nil {
		return m.TruncateHistoryFunc(messages)
	}
	// Default: return messages unchanged
	return messages
}

func (m *MockContextManager) UpdateConfig(config interface{}) {
	if m.UpdateConfigFunc != nil {
		m.UpdateConfigFunc(config)
	}
}

func (m *MockContextManager) UpdateTools(tools []llms.Tool) {
	if m.UpdateToolsFunc != nil {
		m.UpdateToolsFunc(tools)
	}
}

func (m *MockContextManager) UpdateSubAgents(agents []core.SubAgent) {
	if m.UpdateSubAgentsFunc != nil {
		m.UpdateSubAgentsFunc(agents)
	}
}
