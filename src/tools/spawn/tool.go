package spawn

import (
	"fmt"
	"strings"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// AsyncSubagentSpawner starts an ephemeral subagent in the background and returns a spawn ID immediately.
// The subagent result is delivered to the parent conversation via the agent turn queue.
type AsyncSubagentSpawner func(parentChatID, prompt string, tools []llms.Tool) (spawnID string, err error)

// NewSpawnSubagentTool returns the spawn_subagent built-in tool.
//
// The tool accepts:
//   - prompt  (string, required): the task the subagent should execute
//   - tools   ([]string, required): names of parent-agent tools to give the subagent
//
// It resolves tool names against the parent agent's tool list (passed via the
// agentContext map), starts an ephemeral subagent asynchronously, and returns
// immediately with a spawn_id. The subagent result arrives as a follow-up turn.
func NewSpawnSubagentTool(spawner AsyncSubagentSpawner) llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name:        "spawn_subagent",
		Description: "Spawn an ephemeral subagent to handle a task. The subagent runs the given prompt with a specified subset of tools; its result arrives as a follow-up message in this conversation.",
		AdvanceDesc: `Advanced Details:
- prompt  (string, required): full task description the subagent will receive as its user message
- tools   (array of strings, required): tool names from the parent agent's tool list to grant the subagent
- The subagent always has the todo_handler, meta, and expand tools in addition to those listed
- Execution is asynchronous — the tool returns immediately with spawn_id only (not the subagent answer)
- The subagent result arrives later as a follow-up turn (task_type: subagent_result) in this conversation
- Requires an active conversation chatId in agent context`,
		TroubleshootingInfo: `Troubleshooting:
- If a tool name in 'tools' is not found in the parent agent's tool list it is silently skipped
- If all listed tool names are invalid the subagent will still run but with no custom tools
- Ensure 'prompt' clearly describes the task; the subagent has no prior conversation context
- spawn_subagent requires a non-empty chatId (active conversation)`,
		Parameters: []core.Parameter{
			{
				Name:        "prompt",
				Type:        "string",
				Description: "The task the subagent should perform",
				Required:    true,
			},
			{
				Name:        "tools",
				Type:        "array",
				Description: "List of tool names (strings) from the parent agent's tool list to give the subagent",
				Required:    true,
				Items: map[string]any{
					"type": "string",
				},
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			prompt, _ := args["prompt"].(string)

			parentChatID, _ := agentContext["chatId"].(string)
			if parentChatID == "" {
				return core.NewErrorResponse("spawn_subagent requires an active conversation chatId")
			}

			rawTools, _ := args["tools"].([]any)
			requestedNames := make(map[string]struct{}, len(rawTools))
			for _, v := range rawTools {
				if name, ok := v.(string); ok && name != "" {
					requestedNames[name] = struct{}{}
				}
			}

			var selectedTools []llms.Tool
			if parentTools, ok := agentContext["tools"].([]llms.Tool); ok {
				for _, t := range parentTools {
					if _, wanted := requestedNames[t.GetName()]; wanted {
						selectedTools = append(selectedTools, t)
					}
				}
			}

			if len(requestedNames) > 0 && len(selectedTools) < len(requestedNames) {
				resolved := make(map[string]struct{}, len(selectedTools))
				for _, t := range selectedTools {
					resolved[t.GetName()] = struct{}{}
				}
				var missing []string
				for name := range requestedNames {
					if _, ok := resolved[name]; !ok {
						missing = append(missing, name)
					}
				}
				if len(missing) > 0 {
					agentforge.Debug("spawn_subagent: requested tools not found in parent: %s", strings.Join(missing, ", "))
				}
			}

			spawnID, err := spawner(parentChatID, prompt, selectedTools)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("subagent spawn failed: %v", err))
			}

			return core.NewSuccessResponse(fmt.Sprintf(
				"Subagent spawned (spawn_id: %s). Result will arrive as a follow-up message in this conversation.",
				spawnID,
			))
		},
	})
}
