package delegate

import (
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/queue"
)

const DELEGATE_TOOL = "delegate"

// Delegate represents a delegation tool with sub agents.
// Uses map[string]Executable for lookup-by-name and execution-only dependency.
// inbox is the parent agent's message queue; when set, delegation is async and
// the sub-agent's response is enqueued with a reqId header for correlation.
type Delegate struct {
	agents map[string]core.Executable
	inbox  *queue.Queue // nil until agent.Drain is called
}

// NewDelegateTool creates a new DelegateTool with the given sub agents and inbox queue.
// inbox may be nil; if so, the tool returns an error when called (Drain must be started first).
func NewDelegateTool(subAgents []core.SubAgent, inbox *queue.Queue) llms.Tool {
	agents := make(map[string]core.Executable)
	for _, sa := range subAgents {
		if sa != nil {
			agents[sa.Name()] = sa
		}
	}
	delegate := &Delegate{agents: agents, inbox: inbox}

	return &core.Tool{
		Name:        DELEGATE_TOOL,
		Description: "Delegate a task asynchronously to a sub-agent and receive a reqId for correlation",
		AdvanceDesc: `Advanced Details:
- Parameters:
  * subAgent (string, required): The exact name of the sub-agent to delegate to
  * message (string, required): The complete task description with all necessary context
- Async behaviour:
  * The tool returns a reqId immediately — the sub-agent runs in the background
  * When the sub-agent finishes, its response is delivered to your message queue
  * The queued message will carry these headers:
      sender: <sub-agent name>
      reqId:  <the reqId returned by this tool>
      timestamp: <completion time>
  * Use the reqId to correlate the response to the original delegation request
  * Do NOT block or wait for the response — continue your current reasoning turn
- Usage:
  * Only delegate complex tasks that benefit from specialised analysis
  * Provide comprehensive context in the message — sub-agents do not inherit parent context
  * Sub-agent names must match exactly (case-sensitive)
- Integration: Automatically added to agents with sub-agents configured`,
		TroubleshootingInfo: `Troubleshooting:
- "no inbox queue" error: The parent agent's Drain loop must be started before delegating
- "sub agent not found" error: Verify the subAgent name matches exactly (check spelling and case)
- Missing response: The sub-agent response arrives via the message queue — ensure Drain is running
- reqId correlation: Match the reqId in the tool return to the reqId header of the incoming queue message
- Delegation loops: Avoid having sub-agents delegate back to parent agents
- Context isolation: Sub-agents do not see parent agent's history — include all relevant info in message`,
		Parameters: []core.Parameter{
			{
				Name:        "subAgent",
				Type:        "string",
				Description: "The name of the sub agent to delegate the task to",
				Required:    true,
			},
			{
				Name:        "message",
				Type:        "string",
				Description: "The request to delegate to the sub agent",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			subAgentName := args["subAgent"].(string)
			message := args["message"].(string)

			return delegate.delegate(agentContext, subAgentName, message)
		},
	}
}
