package spawn

import (
	"context"
	"fmt"
	"strings"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// SubagentFactory creates a subagent and runs a prompt against it, returning the final response.
// The factory is injected at agent-init time (in initSystemTools) to avoid a circular import
// between this package and src/agents.
type SubagentFactory func(ctx context.Context, prompt string, tools []llms.Tool) (string, error)

// NewSpawnSubagentTool returns the spawn_subagent built-in tool.
//
// The tool accepts:
//   - prompt  (string, required): the task the subagent should execute
//   - tools   ([]string, required): names of parent-agent tools to give the subagent
//
// It resolves tool names against the parent agent's tool list (passed via the
// agentContext map), builds an ephemeral subagent through factory, waits for it
// to complete, and returns the final response as a tool result.
func NewSpawnSubagentTool(factory SubagentFactory) llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name:        "spawn_subagent",
		Description: "Spawn an ephemeral subagent to handle a task. The subagent runs the given prompt with a specified subset of tools and returns its final answer.",
		AdvanceDesc: `Advanced Details:
- prompt  (string, required): full task description the subagent will receive as its user message
- tools   (array of strings, required): tool names from the parent agent's tool list to grant the subagent
- The subagent always has the todo_handler, meta, and expand tools in addition to those listed
- Execution is synchronous — the tool blocks until the subagent finishes
- Returns the subagent's final text response`,
		TroubleshootingInfo: `Troubleshooting:
- If a tool name in 'tools' is not found in the parent agent's tool list it is silently skipped
- If all listed tool names are invalid the subagent will still run but with no custom tools
- Ensure 'prompt' clearly describes the task; the subagent has no prior conversation context`,
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

			// Collect requested tool names
			rawTools, _ := args["tools"].([]any)
			requestedNames := make(map[string]struct{}, len(rawTools))
			for _, v := range rawTools {
				if name, ok := v.(string); ok && name != "" {
					requestedNames[name] = struct{}{}
				}
			}

			// Resolve tool instances from the parent agent's context
			var selectedTools []llms.Tool
			if parentTools, ok := agentContext["tools"].([]llms.Tool); ok {
				for _, t := range parentTools {
					if _, wanted := requestedNames[t.GetName()]; wanted {
						selectedTools = append(selectedTools, t)
					}
				}
			}

			// Report any names that could not be resolved (informational only)
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
					// Non-fatal: log and continue
					_ = fmt.Sprintf("spawn_subagent: requested tools not found in parent: %s", strings.Join(missing, ", "))
				}
			}

			result, err := factory(context.Background(), prompt, selectedTools)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("subagent execution failed: %v", err))
			}

			return core.NewSuccessResponse(result)
		},
	})
}
