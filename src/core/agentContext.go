package core

import (
	"github.com/thinktwice/agentForge/src/llms"
)

// AgentContext holds static agent context information that is built once
// at agent instantiation. This avoids rebuilding the context on every tool execution.
type AgentContext struct {
	// AgentName is the name of the agent
	AgentName string
	// Trace is optional trace information (e.g., "thinking", "response")
	Trace string
	// Tools is the list of tools available to the agent
	Tools []llms.Tool
	// SubAgents is the list of sub-agents available for delegation
	SubAgents []*SubAgent
}

// BuildContext converts the AgentContext struct to a map[string]any and merges
// in the session-specific responseCh parameter.
//
// Parameters:
//   - responseCh: The response channel for the current chat session
//
// Returns:
//   - map[string]any: The complete agent context map ready for tool execution
func (ac *AgentContext) BuildContext(responseCh *ResponseCh) map[string]any {
	context := make(map[string]any)
	context["agentName"] = ac.AgentName
	context["trace"] = ac.Trace
	context["responseCh"] = responseCh
	context["tools"] = ac.Tools
	context["subAgents"] = ac.SubAgents
	return context
}
