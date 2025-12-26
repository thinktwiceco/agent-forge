package tools

import (
	"encoding/json"
	"fmt"

	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// NewDelegateTool creates a new DelegateTool with the given sub agents.
func NewDelegateTool(subAgents []*core.SubAgent) llms.Tool {
	return core.NewTool(
		"delegate",
		"Delegate a task to a sub agent",
		`Advanced Details:
- Parameters:
  * subAgent (string, required): The exact name of the sub-agent to delegate to
  * message (string, required): The complete task description with all necessary context
- Behavior: 
  * Streams responses from the sub-agent back to the parent agent
  * Forwards all chunks including content, tool calls, and status updates
  * Accumulates and returns the full response when delegation completes
- Usage: 
  * Only delegate complex tasks that benefit from specialized analysis
  * Provide comprehensive context in the message - sub-agents don't inherit parent context
  * Sub-agent names must match exactly (case-sensitive)
- Integration: Automatically added to agents with sub-agents configured`,
		`Troubleshooting:
- "sub agent not found" error: Verify the subAgent name matches exactly (check spelling and case)
- Empty responses: Ensure the message parameter contains sufficient context for the sub-agent
- Delegation loops: Avoid having sub-agents delegate back to parent agents
- Performance: Long-running delegations are normal for complex tasks
- Context isolation: Sub-agents don't see parent agent's history - include all relevant info in message`,
		[]core.Parameter{
			{
				Name:        "subAgent",
				Type:        "string",
				Description: "The name of the sub agent to delegate the task to",
				Required:    true,
			},
			{
				Name:        "message",
				Type:        "string",
				Description: "The request to delegate to the sub agent",
				Required:    true,
			},
		},
		func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			// Extract validated parameters
			subAgentName := args["subAgent"].(string)
			message := args["message"].(string)

			// Extract parent response channel from context
			var parentResponseCh *core.ResponseCh
			if responseCh, ok := agentContext["responseCh"].(*core.ResponseCh); ok {
				parentResponseCh = responseCh
			}

			// Find the sub agent
			var assignedSubAgent core.SubAgent
			for _, subAgent := range subAgents {
				if (*subAgent).Name() == subAgentName {
					assignedSubAgent = *subAgent
					break
				}
			}

			if assignedSubAgent == nil {
				return core.NewErrorResponse(fmt.Sprintf("sub agent '%s' not found", subAgentName))
			}

			// Send delegation start notification if parent response channel is available
			if parentResponseCh != nil {
				startChunk := llms.ChunkResponse{
					Status:  llms.StatusStreaming,
					Type:    llms.TypeContent,
					Content: fmt.Sprintf("\n [🛠️ Delegating to %s...]\nQuestion: %s\n", subAgentName, message),
				}
				if startBytes, err := json.Marshal(startChunk); err == nil {
					parentResponseCh.GetResponseChan() <- startBytes
				}
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

			// Send delegation completion notification
			if parentResponseCh != nil {
				endChunk := llms.ChunkResponse{
					Status:  llms.StatusStreaming,
					Type:    llms.TypeContent,
					Content: fmt.Sprintf("\n[✅ Delegation to %s complete]\n", subAgentName),
				}
				if endBytes, err := json.Marshal(endChunk); err == nil {
					parentResponseCh.GetResponseChan() <- endBytes
				}
			}

			// Return the accumulated result
			if delegationError != nil {
				return core.NewFailureResponse(delegationError.Error(), fullResponse)
			}

			return core.NewSuccessResponse(fullResponse)
		},
	)
}
