package delegate

import (
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

const DELEGATE_TOOL = "delegate"

// Delegate represents a delegation tool with sub agents.
type Delegate struct {
	subAgents []*core.SubAgent
}

// NewDelegateTool creates a new DelegateTool with the given sub agents.
func NewDelegateTool(subAgents []*core.SubAgent) llms.Tool {
	delegate := &Delegate{subAgents: subAgents}

	return &core.Tool{
		Name:        DELEGATE_TOOL,
		Description: "Delegate a task to a sub agent",
		AdvanceDesc: `Advanced Details:
- Parameters:
  * subAgent (string, required): The exact name of the sub-agent to delegate to
  * message (string, required): The complete task description with all necessary context
- Behavior: 
  * Streams responses from the sub-agent back to the parent agent
  * Forwards all chunks including content, tool calls, and status updates
  * Accumulates and returns the full response when delegation completes
- Usage: 
  * Only delegate complex tasks that benefit from specialized analysis
  * Provide comprehensive context in the message - sub-agents don't inherit parent context
  * Sub-agent names must match exactly (case-sensitive)
- Integration: Automatically added to agents with sub-agents configured`,
		TroubleshootingInfo: `Troubleshooting:
- "sub agent not found" error: Verify the subAgent name matches exactly (check spelling and case)
- Empty responses: Ensure the message parameter contains sufficient context for the sub-agent
- Delegation loops: Avoid having sub-agents delegate back to parent agents
- Performance: Long-running delegations are normal for complex tasks
- Context isolation: Sub-agents don't see parent agent's history - include all relevant info in message`,
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
