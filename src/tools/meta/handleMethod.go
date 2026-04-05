package meta

import (
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// handleMethod routes the method call to the appropriate handler.
func (m *Meta) handleMethod(agentContext map[string]any, method string) llms.ToolReturn {
	switch method {
	case "get_current_model":
		return m.getCurrentModel(agentContext)
	case "get_agent_name":
		return m.getAgentName(agentContext)
	case "get_tools":
		return m.getTools(agentContext)
	default:
		return core.NewErrorResponse(fmt.Sprintf("unknown method: %s", method))
	}
}
