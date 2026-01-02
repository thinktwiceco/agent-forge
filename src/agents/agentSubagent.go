package agents

import (
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/tools/delegate"
)

// ===== Sub Agent Management =====

// getSubAgentsAsInterfaces converts internal sub-agents to core.SubAgent interfaces
// for use in tool execution context (e.g., for the expand tool)
func (a *Agent) getSubAgentsAsInterfaces() []*core.SubAgent {
	var subAgentInterfaces []*core.SubAgent
	for _, sa := range a.subAgents {
		subAgentInterfaces = append(subAgentInterfaces, sa)
	}
	return subAgentInterfaces
}

// removeDelegateTool removes the delegate tool from the tools slice if it exists.
// Returns a new slice without the delegate tool.
func (a *Agent) removeDelegateTool(toolsList []llms.Tool) []llms.Tool {
	var filtered []llms.Tool
	for _, tool := range toolsList {
		if tool.GetName() != delegate.DELEGATE_TOOL {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}
