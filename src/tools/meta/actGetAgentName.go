package meta

import (
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// getAgentName retrieves the agent's name from the agent context.
func (m *Meta) getAgentName(agentContext map[string]any) llms.ToolReturn {
	agentName, ok := agentContext["agentName"].(string)
	if !ok || agentName == "" {
		return core.NewSuccessResponse("Agent name not available")
	}
	return core.NewSuccessResponse(agentName)
}
