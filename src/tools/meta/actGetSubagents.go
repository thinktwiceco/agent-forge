package meta

import (
	"encoding/json"
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// getSubagents retrieves a list of available subagents with their names and descriptions.
func (m *Meta) getSubagents(agentContext map[string]any) llms.ToolReturn {
	subAgentsRaw, ok := agentContext["subAgents"]
	if !ok {
		return core.NewEphemeralResponse("[]")
	}

	// Handle []core.SubAgent (slice of interfaces, not pointers)
	subAgents, ok := subAgentsRaw.([]core.SubAgent)
	if !ok {
		// Try to handle []interface{} case
		subAgentsSlice, ok := subAgentsRaw.([]interface{})
		if !ok {
			return core.NewEphemeralResponse("[]")
		}

		subAgents = make([]core.SubAgent, 0, len(subAgentsSlice))
		for _, saRaw := range subAgentsSlice {
			if sa, ok := saRaw.(core.SubAgent); ok {
				subAgents = append(subAgents, sa)
			}
		}
	}

	if len(subAgents) == 0 {
		return core.NewEphemeralResponse("[]")
	}

	subAgentList := make([]map[string]string, 0, len(subAgents))
	for _, subAgent := range subAgents {
		if subAgent == nil {
			continue
		}
		// subAgent is already an interface value, no need to dereference
		subAgentInfo := map[string]string{
			"name":             subAgent.Name(),
			"basicDescription": subAgent.BasicDescription(),
		}
		subAgentList = append(subAgentList, subAgentInfo)
	}

	subAgentsJSON, err := json.Marshal(subAgentList)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to serialize subagents: %v", err))
	}

	return core.NewEphemeralResponse(string(subAgentsJSON))
}
