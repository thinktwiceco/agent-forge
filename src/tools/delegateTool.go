package tools

import (
	"encoding/json"
	"fmt"
	"reflect"

	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/llms"
)

// IResponseChStarter provides a Start method that returns a channel
type IResponseChStarter interface {
	Start() interface{} // Returns a receive-only channel that can be ranged over
}

// IParentResponseCh represents the parent agent's response channel
// that can be used to send updates during tool execution
type IParentResponseCh interface {
	GetResponseChan() chan<- []byte
	GetErrorChan() chan<- error
}

type SubAgent interface {
	ChatStream(message string) IResponseChStarter
	Name() string
}

// NewDelegateTool creates a new DelegateTool with the given sub agents.
func NewDelegateTool(subAgents []SubAgent) llms.Tool {
	return NewContextAwareTool(
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
		[]Parameter{
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
			var parentResponseCh IParentResponseCh
			if responseCh, ok := agentContext["responseCh"].(IParentResponseCh); ok {
				parentResponseCh = responseCh
			}

			// Find the sub agent
			var assignedSubAgent SubAgent
			for _, subAgent := range subAgents {
				if subAgent.Name() == subAgentName {
					assignedSubAgent = subAgent
					break
				}
			}

			if assignedSubAgent == nil {
				return NewErrorResponse(fmt.Sprintf("sub agent '%s' not found", subAgentName))
			}

			// Send delegation start notification if parent response channel is available
			if parentResponseCh != nil {
				startChunk := llms.ChunkResponse{
					Status:  llms.StatusStreaming,
					Type:    llms.TypeContent,
					Content: fmt.Sprintf("\n [🛠️ Delegating to %s...]\n", subAgentName),
				}
				if startBytes, err := json.Marshal(startChunk); err == nil {
					parentResponseCh.GetResponseChan() <- startBytes
				}
			}

			// Get parent agent name from context
			parentAgentName, ok := agentContext["agentName"].(string)
			if !ok {
				return NewErrorResponse("agentName must be a string")
			}

			agentforge.Info("%s ➡️ %s ➡️ %s", parentAgentName, subAgentName, message)

			// Execute delegation by calling sub agent's ChatStream
			delegateResponseChStarter := assignedSubAgent.ChatStream(message)

			// Accumulate the full response
			var fullResponse string
			var delegationError error

			// Get the channel from Start() - it returns interface{} so we use reflection to iterate
			chunkChannel := delegateResponseChStarter.Start()
			chValue := reflect.ValueOf(chunkChannel)

			// Use reflection to receive from the channel
			for {
				chunk, ok := chValue.Recv()
				if !ok {
					// Channel closed
					break
				}

				// Get the chunk as an interface{}
				chunkInterface := chunk.Interface()

				// Try to extract Content and Status using reflection
				chunkVal := reflect.ValueOf(chunkInterface)
				if chunkVal.Kind() == reflect.Struct {
					// Try to get Content field
					contentField := chunkVal.FieldByName("Content")
					if contentField.IsValid() && contentField.Kind() == reflect.String {
						content := contentField.String()
						if content != "" {
							fullResponse += content
						}
					}

					// Try to get Status field
					statusField := chunkVal.FieldByName("Status")
					if statusField.IsValid() && statusField.Kind() == reflect.String {
						status := statusField.String()
						if status == llms.StatusError {
							// Get error content
							if contentField.IsValid() && contentField.Kind() == reflect.String {
								delegationError = fmt.Errorf("delegation error: %s", contentField.String())
							} else {
								delegationError = fmt.Errorf("delegation error occurred")
							}
						}
					}
				}

				// Forward chunk to parent if available
				if parentResponseCh != nil {
					if chunkBytes, err := json.Marshal(chunkInterface); err == nil {
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
				return NewFailureResponse(delegationError.Error(), fullResponse)
			}

			return NewSuccessResponse(fullResponse)
		},
	)
}
