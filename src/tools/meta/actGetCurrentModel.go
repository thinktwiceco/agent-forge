package meta

import (
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// getCurrentModel retrieves the current LLM model name from the agent context.
func (m *Meta) getCurrentModel(agentContext map[string]any) llms.ToolReturn {
	model, ok := agentContext["model"].(string)
	if !ok || model == "" {
		return core.NewEphemeralResponse("Model information not available")
	}
	return core.NewEphemeralResponse(model)
}
