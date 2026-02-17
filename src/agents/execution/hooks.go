package execution

import (
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// HooksRunner provides hook callbacks for the Executor.
// The Agent populates these with closures that capture the Agent instance,
// allowing the Executor to trigger hooks without importing the agents package.
type HooksRunner struct {
	OnContextBuild                 func(agentContext *core.AgentContext) []error
	OnBeforeToolExecution          func(toolCall *llms.ToolCall) []error
	OnToolExecution                func(toolResult *llms.ToolResult) []error
	OnNewAssistantMessage          func(message string, promptTokens, completionTokens, totalTokens int) []error
	OnNewAssistantMessageWithTools func(message string, toolCalls []llms.ToolCall, promptTokens, completionTokens, totalTokens int) []error
	// LogHookErrors is called to log any errors returned by hooks (e.g. agentforge.Debug)
	LogHookErrors func(errors []error)
}
