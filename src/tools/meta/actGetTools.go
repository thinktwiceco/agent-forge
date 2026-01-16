package meta

import (
	"encoding/json"
	"fmt"

	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// getTools retrieves a list of available tools with their names and descriptions.
func (m *Meta) getTools(agentContext map[string]any) llms.ToolReturn {
	tools, ok := agentContext["tools"].([]llms.Tool)
	if !ok || len(tools) == 0 {
		return core.NewEphemeralResponse("[]")
	}

	toolList := make([]map[string]string, 0, len(tools))
	for _, tool := range tools {
		toolInfo := map[string]string{
			"name":        tool.GetName(),
			"description": tool.GetFunctionDefinition().Description,
		}
		toolList = append(toolList, toolInfo)
	}

	toolsJSON, err := json.Marshal(toolList)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to serialize tools: %v", err))
	}

	return core.NewEphemeralResponse(string(toolsJSON))
}
