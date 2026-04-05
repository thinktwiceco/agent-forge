package expand

import (
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// Expand represents a tool that allows progressive discovery of tools.
type Expand struct{}

// NewExpandTool creates a tool that allows progressive discovery of tools.
//
// This tool enables agents to retrieve detailed information (AdvanceDescription and
// Troubleshooting) about tools that are available in their context.
//
// Progressive discovery pattern:
//   - BasicDescription: what the tool does and which actions are available
//   - AdvanceDescription: how to use the tool (general parameters and workflows)
//   - DetailsAbout(action): deep-dive on a specific action/operation/endpoint
//
// The tool expects the following items in agentContext:
//   - "tools": []llms.Tool - list of available tools
func NewExpandTool() llms.Tool {
	expand := &Expand{}

	return core.NewTool(core.ToolConfig{
		Name:        "expand",
		Description: "Get detailed information about a tool. Use this to discover advanced capabilities, troubleshooting information, or per-action details.",
		AdvanceDesc: `Advanced Details:
- Parameters:
  * subject_type (string, required): Must be "tool"
  * subject_name (string, required): The exact name of the tool
  * details_about (string, optional): Name of a specific action/operation/endpoint to get deep details about
  * troubleshoot (boolean, optional): Include troubleshooting information (default: false, ignored when details_about is set)
- Progressive discovery:
  * Omit details_about to get BasicDescription + AdvanceDescription (overview)
  * Set details_about="<action>" to get focused detail on that specific action/operation
  * Set troubleshoot=true to append troubleshooting guidance to the overview
- Integration: Can be added to any agent that needs discovery capabilities`,
		TroubleshootingInfo: `Troubleshooting:
- "Not found" errors: Verify subject_name matches exactly (case-sensitive)
- "Invalid subject_type": Must be exactly "tool"
- "Does not implement Discoverable": The tool doesn't support discovery
- Empty descriptions: The tool may not have advanced descriptions configured
- Tool not in context: Ensure the tool is added to the agent's tool list`,
		Parameters: []core.Parameter{
			{
				Name:        "subject_type",
				Type:        "string",
				Description: "Must be 'tool'",
				Required:    true,
			},
			{
				Name:        "subject_name",
				Type:        "string",
				Description: "The exact name of the tool to get information about",
				Required:    true,
			},
			{
				Name:        "details_about",
				Type:        "string",
				Description: "Name of a specific action, operation, or endpoint to get detailed information about (optional)",
				Required:    false,
			},
			{
				Name:        "troubleshoot",
				Type:        "boolean",
				Description: "Whether to include troubleshooting information in the overview (default: false, ignored when details_about is set)",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			subjectType := args["subject_type"].(string)
			subjectName := args["subject_name"].(string)

			detailsAbout, _ := args["details_about"].(string)

			troubleshoot := false
			if val, ok := args["troubleshoot"]; ok {
				troubleshoot = val.(bool)
			}

			return expand.expand(agentContext, subjectType, subjectName, troubleshoot, detailsAbout)
		},
	})
}
