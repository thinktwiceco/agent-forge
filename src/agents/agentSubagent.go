package agents

import (
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/tools/delegate"
)

// ===== Sub Agent Management =====

// getSubAgentsAsInterfaces converts internal sub-agents to core.SubAgent interfaces
// for use in tool execution context (e.g., for the expand tool)
// This method is currently unused but kept for potential future use.
//
//nolint:unused // Reserved for future use
func (a *Agent) getSubAgentsAsInterfaces() []*core.SubAgent {
	subAgentInterfaces := make([]*core.SubAgent, 0, len(a.subAgents))
	subAgentInterfaces = append(subAgentInterfaces, a.subAgents...)
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
