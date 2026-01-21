package delegate

import (
	"encoding/json"
	"fmt"

	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// delegate executes the delegation to a sub-agent.
func (d *Delegate) delegate(ctx map[string]any, subAgentName string, message string) llms.ToolReturn {
	// Extract parent response channel from context
	agentContext, err := core.RehydrateContext(ctx)

	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("error rehydrating context: %v", err))
	}

	parentResponseCh := agentContext.ResponseCh

	// Find the sub agent
	var assignedSubAgent core.SubAgent
	for _, subAgent := range d.subAgents {
		if subAgent != nil {
			subAgentVal := *subAgent
			if subAgentVal.Name() == subAgentName {
				assignedSubAgent = subAgentVal
				break
			}
		}
	}

	if assignedSubAgent == nil {
		return core.NewErrorResponse(fmt.Sprintf("sub agent '%s' not found", subAgentName))
	}

	// Get parent agent name from context
	parentAgentName := agentContext.AgentName
	agentforge.Info("[%s] ➡️ [%s] ➡️ \n%s", parentAgentName, subAgentName, message)

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
				// Use TrySend to safely send to parent channel (may be closed)
				parentResponseCh.TrySend(chunkBytes)
			}
		}
	}

	// Store the subagent message content in the agent context
	// The context map will be synced back to the struct by executeTool() after execution
	agentContext.SetLastSubagentMessage(fullResponse)
	ctx["lastSubagentMessage"] = fullResponse

	// Return the accumulated result
	if delegationError != nil {
		return core.NewFailureResponse(delegationError.Error(), fullResponse)
	}

	return core.NewSuccessEphemeralResponse(fullResponse)
}
