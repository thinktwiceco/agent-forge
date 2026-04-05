package mocks

import (
	"context"

	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/agents/execution"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/history"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// MockExecutionEngine is a mock implementation of agents.ExecutionEngine for testing.
type MockExecutionEngine struct {
	ExecuteChatWithToolsFunc func(ctx context.Context, hm history.Manager, responseCh *core.ResponseCh) (execution.ExecuteResult, error)
	ExecuteToolFunc          func(toolCall llms.ToolCall, responseCh *core.ResponseCh) llms.ToolResult
	UpdateToolsFunc          func(tools []llms.Tool)
	UpdateAgentContextFunc   func(agentContext *core.AgentContext)
}

// Ensure MockExecutionEngine implements agents.ExecutionEngine
var _ agents.ExecutionEngine = (*MockExecutionEngine)(nil)

func (m *MockExecutionEngine) ExecuteChatWithTools(ctx context.Context, hm history.Manager, responseCh *core.ResponseCh) (execution.ExecuteResult, error) {
	if m.ExecuteChatWithToolsFunc != nil {
		return m.ExecuteChatWithToolsFunc(ctx, hm, responseCh)
	}
	return execution.ExecuteResult{}, nil
}

func (m *MockExecutionEngine) ExecuteTool(toolCall llms.ToolCall, responseCh *core.ResponseCh) llms.ToolResult {
	if m.ExecuteToolFunc != nil {
		return m.ExecuteToolFunc(toolCall, responseCh)
	}
	return llms.ToolResult{
		ToolCallID: toolCall.ID,
		ToolName:   toolCall.Name,
		Success:    true,
		Result:     "mock result",
	}
}

func (m *MockExecutionEngine) UpdateTools(tools []llms.Tool) {
	if m.UpdateToolsFunc != nil {
		m.UpdateToolsFunc(tools)
	}
}

func (m *MockExecutionEngine) UpdateAgentContext(agentContext *core.AgentContext) {
	if m.UpdateAgentContextFunc != nil {
		m.UpdateAgentContextFunc(agentContext)
	}
}
