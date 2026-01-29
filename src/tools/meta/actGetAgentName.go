package meta

import (
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// getAgentName retrieves the agent's name from the agent context.
func (m *Meta) getAgentName(agentContext map[string]any) llms.ToolReturn {
	agentName, ok := agentContext["agentName"].(string)
	if !ok || agentName == "" {
		return core.NewEphemeralResponse("Agent name not available")
	}
	return core.NewEphemeralResponse(agentName)
}
