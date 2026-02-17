package agents

import (
	"context"

	"github.com/thinktwiceco/agent-forge/src/history"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// executeChatWithTools delegates to the executor. Kept for backward compatibility with tests.
// Syncs executor state when tools or agentContext were modified (e.g. in tests).
func (a *Agent) executeChatWithTools(ctx context.Context, hm history.Manager) error {
	a.executor.UpdateTools(a.tools)
	a.executor.UpdateAgentContext(a.agentContext)
	return a.executor.ExecuteChatWithTools(ctx, hm, a.responseCh)
}

// executeTool delegates to the executor. Kept for backward compatibility with tests.
// Syncs executor state when tools or agentContext were modified (e.g. in tests).
func (a *Agent) executeTool(toolCall llms.ToolCall) llms.ToolResult {
	a.executor.UpdateTools(a.tools)
	a.executor.UpdateAgentContext(a.agentContext)
	return a.executor.ExecuteTool(toolCall, a.responseCh)
}
