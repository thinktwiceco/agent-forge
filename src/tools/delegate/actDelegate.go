package delegate

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// generateReqId produces a random 8-character hex string (4 random bytes).
func generateReqId() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// fallback: use a fixed placeholder (should never happen)
		return "00000000"
	}
	return hex.EncodeToString(b)
}

// delegate executes the delegation to a sub-agent asynchronously.
// It returns a reqId immediately; the sub-agent's response is enqueued into
// the parent agent's inbox when processing completes.
func (d *Delegate) delegate(ctx map[string]any, subAgentName string, message string) llms.ToolReturn {
	if d.inbox == nil {
		return core.NewErrorResponse("no inbox queue: start agent.Drain before delegating")
	}

	// Extract parent response channel from context
	agentContext, err := core.RehydrateContext(ctx)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("error rehydrating context: %v", err))
	}

	parentResponseCh := agentContext.ResponseCh

	// Find the sub agent by name
	assignedSubAgent, ok := d.agents[subAgentName]
	if !ok || assignedSubAgent == nil {
		return core.NewErrorResponse(fmt.Sprintf("sub agent '%s' not found", subAgentName))
	}

	parentAgentName := agentContext.AgentName
	reqId := generateReqId()

	agentforge.Info("[%s] ➡️ [%s] (async, reqId=%s) ➡️ \n%s", parentAgentName, subAgentName, reqId, message)

	inbox := d.inbox

	go func() {
		delegateResponseCh := assignedSubAgent.ChatStream(context.Background(), message, "")

		var fullResponse string
		var delegationError error

		for chunk := range delegateResponseCh.Start() {
			if chunk.Content != "" {
				fullResponse += chunk.Content
			}
			if chunk.Status == llms.StatusError {
				delegationError = fmt.Errorf("delegation error: %s", chunk.Content)
			}
			// Forward streaming chunks to parent SSE so the UI can show sub-agent progress
			if parentResponseCh != nil {
				if chunkBytes, err := json.Marshal(chunk); err == nil {
					parentResponseCh.TrySend(chunkBytes)
				}
			}
		}

		if delegationError != nil {
			fullResponse = fmt.Sprintf("error: %s\n%s", delegationError.Error(), fullResponse)
		}

		// Read the parent chatId now — by the time the sub-agent finishes the parent
		// ChatStream will have completed and saved, so GetChatId() returns the final ID.
		chatId := ""
		if parentResponseCh != nil {
			chatId = parentResponseCh.GetChatId()
		}

		agentforge.Info("[%s] ⬅️ [%s] response enqueued (reqId=%s, chatId=%s)", parentAgentName, subAgentName, reqId, chatId)

		inbox.Enqueue(fullResponse, chatId, map[string]string{
			"sender": subAgentName,
			"reqId":  reqId,
		})
	}()

	return core.NewSuccessResponse(fmt.Sprintf(
		"Delegation accepted.\nreqId: %s\nSub-agent '%s' is processing your request in the background. "+
			"The response will be delivered to your message queue with header 'reqId: %s'.",
		reqId, subAgentName, reqId,
	))
}
