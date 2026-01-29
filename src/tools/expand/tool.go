package expand

import (
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// Expand represents a tool that allows progressive discovery of tools and agents.
type Expand struct{}

// NewExpandTool creates a tool that allows progressive discovery of tools and agents.
//
// This tool enables agents to retrieve detailed information (AdvanceDescription and
// Troubleshooting) about tools and sub-agents that are available in their context.
//
// The tool expects the following items in agentContext:
//   - "tools": []llms.Tool - list of available tools
//   - "subAgents": []core.SubAgent - list of available sub-agents
func NewExpandTool() llms.Tool {
	expand := &Expand{}

	return &core.Tool{
		Name:        "expand",
		Description: "Get detailed information about a tool or sub-agent. Use this to discover advanced capabilities and troubleshooting information.",
		AdvanceDesc: `Advanced Details:
- Parameters:
  * subject_type (string, required): Either "tool" or "agent"
  * subject_name (string, required): The exact name of the tool or agent
  * troubleshoot (boolean, optional): Include troubleshooting information (default: false)
- Behavior:
  * Retrieves AdvanceDescription for the specified tool or agent
  * Optionally includes Troubleshooting information
  * Returns formatted information as a string
- Usage:
  * Use when you need detailed information about a tool's capabilities
  * Use when you need to understand an agent's advanced features
  * Use when troubleshooting issues with tools or agents
- Integration: Can be added to any agent that needs discovery capabilities`,
		TroubleshootingInfo: `Troubleshooting:
- "Not found" errors: Verify subject_name matches exactly (case-sensitive)
- "Invalid subject_type": Must be exactly "tool" or "agent"
- "Does not implement Discoverable": The tool/agent doesn't support discovery
- Empty descriptions: The tool/agent may not have advanced descriptions configured
- Tool not in context: Ensure the tool is added to the agent's tool list`,
		Parameters: []core.Parameter{
			{
				Name:        "subject_type",
				Type:        "string",
				Description: "The type of subject to expand: 'tool' or 'agent'",
				Required:    true,
			},
			{
				Name:        "subject_name",
				Type:        "string",
				Description: "The exact name of the tool or agent to get information about",
				Required:    true,
			},
			{
				Name:        "troubleshoot",
				Type:        "boolean",
				Description: "Whether to include troubleshooting information (default: false)",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			subjectType := args["subject_type"].(string)
			subjectName := args["subject_name"].(string)

			// Get troubleshoot flag (default to false if not provided)
			troubleshoot := false
			if val, ok := args["troubleshoot"]; ok {
				troubleshoot = val.(bool)
			}

			return expand.expand(agentContext, subjectType, subjectName, troubleshoot)
		},
	}
}
