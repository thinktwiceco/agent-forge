package heartbeat

import (
	"encoding/json"
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

const heartbeatManagerTool = "heartbeat_manager"

func newHeartbeatManagerTool(m *HeartbeatManager) llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name:        heartbeatManagerTool,
		Description: "Manage recurring instructions injected into every heartbeat prompt. Use to add, remove, or list named instructions.",
		AdvanceDesc: `Actions:
- add_instruction: Registers a named instruction block (## Title) in the heartbeat context.
  Required params: title (string), instruction (string).
- remove_instruction: Removes the instruction with the given title.
  Required params: title (string).
- list_instructions: Returns all registered instruction titles as a JSON array.
  No extra params required.`,
		TroubleshootingInfo: `Troubleshooting:
- add_instruction fails when title is empty.
- remove_instruction fails when the title does not exist — call list_instructions first to confirm.`,
		Parameters: []core.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "Action to perform: 'add_instruction', 'remove_instruction', or 'list_instructions'",
				Required:    true,
			},
			{
				Name:        "title",
				Type:        "string",
				Description: "Instruction title used as a ## heading. Required for add_instruction and remove_instruction.",
				Required:    false,
			},
			{
				Name:        "instruction",
				Type:        "string",
				Description: "Instruction body placed under the title heading. Required for add_instruction.",
				Required:    false,
			},
		},
		Handler: func(_ map[string]any, args map[string]any) llms.ToolReturn {
			action, ok := args["action"].(string)
			if !ok || action == "" {
				return core.NewErrorResponse("action parameter is required and must be a string")
			}

			switch action {
			case "add_instruction":
				title, _ := args["title"].(string)
				instruction, _ := args["instruction"].(string)
				if err := m.AddInstruction(title, instruction); err != nil {
					return core.NewErrorResponse(fmt.Sprintf("add_instruction failed: %v", err))
				}
				return core.NewSuccessResponse(fmt.Sprintf("instruction %q added", title))

			case "remove_instruction":
				title, _ := args["title"].(string)
				if err := m.RemoveInstruction(title); err != nil {
					return core.NewErrorResponse(fmt.Sprintf("remove_instruction failed: %v", err))
				}
				return core.NewSuccessResponse(fmt.Sprintf("instruction %q removed", title))

			case "list_instructions":
				titles := m.ListInstructions()
				b, err := json.Marshal(titles)
				if err != nil {
					return core.NewErrorResponse(fmt.Sprintf("list_instructions failed: %v", err))
				}
				return core.NewEphemeralResponse(string(b))

			default:
				return core.NewErrorResponse(fmt.Sprintf("unknown action %q — valid actions: add_instruction, remove_instruction, list_instructions", action))
			}
		},
	})
}
