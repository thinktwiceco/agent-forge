package delegate

import (
	"encoding/json"
	"fmt"

	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// delegate executes the delegation to a sub-agent.
func (d *Delegate) delegate(agentContext map[string]any, subAgentName string, message string) llms.ToolReturn {
	// Extract parent response channel from context
	var parentResponseCh *core.ResponseCh
	if responseCh, ok := agentContext["responseCh"].(*core.ResponseCh); ok {
		parentResponseCh = responseCh
	}

	// Find the sub agent
	var assignedSubAgent core.SubAgent
	for _, subAgent := range d.subAgents {
		if (*subAgent).Name() == subAgentName {
			assignedSubAgent = *subAgent
			break
		}
	}

	if assignedSubAgent == nil {
		return core.NewErrorResponse(fmt.Sprintf("sub agent '%s' not found", subAgentName))
	}

	// Get parent agent name from context
	parentAgentName, ok := agentContext["agentName"].(string)
	if !ok {
		return core.NewErrorResponse("agentName must be a string")
	}

	agentforge.Info("%s ➡️ %s ➡️ %s", parentAgentName, subAgentName, message)

	// Execute delegation by calling sub agent's ChatStream
	delegateResponseCh := assignedSubAgent.ChatStream(message)

	// Accumulate the full response
	var fullResponse string
	var delegationError error

	// Process chunks from the sub-agent - no reflection needed!
	for chunk := range delegateResponseCh.Start() {
		// Accumulate content
		if chunk.Content != "" {
			fullResponse += chunk.Content
		}

		// Check for errors
		if chunk.Status == llms.StatusError {
			delegationError = fmt.Errorf("delegation error: %s", chunk.Content)
		}

		// Forward chunk to parent if available
		if parentResponseCh != nil {
			if chunkBytes, err := json.Marshal(chunk); err == nil {
				parentResponseCh.GetResponseChan() <- chunkBytes
			}
		}
	}

	// Return the accumulated result
	if delegationError != nil {
		return core.NewFailureResponse(delegationError.Error(), fullResponse)
	}

	return core.NewSuccessResponse(fullResponse)
}
