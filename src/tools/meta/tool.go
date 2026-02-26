package meta

import (
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// Meta represents a meta tool that provides information about the agent.
type Meta struct{}

// NewMetaTool creates a meta tool that provides information about the agent.
//
// Available methods:
//   - get_current_model: Returns the current LLM model name
//   - get_agent_name: Returns the agent's name
//   - get_tools: Returns a list of available tools with their names and descriptions
//   - get_subagents: Returns a list of available subagents with their names and descriptions
func NewMetaTool() llms.Tool {
	meta := &Meta{}

	detailsAbout := func(item string) string {
		switch item {
		case "get_current_model":
			return `get_current_model: Returns the name of the LLM model currently being used by the agent. No additional parameters required.`
		case "get_agent_name":
			return `get_agent_name: Returns the name of the agent. No additional parameters required.`
		case "get_tools":
			return `get_tools: Returns a JSON array of all available tools with their names and descriptions. No additional parameters required.`
		case "get_subagents":
			return `get_subagents: Returns a JSON array of all available subagents with their names and basic descriptions. No additional parameters required.`
		default:
			return fmt.Sprintf("Nothing to add about %s", item)
		}
	}

	return &core.Tool{
		Name:        "meta",
		Description: "Get metadata about the agent including current model, agent name, available tools, and subagents.",
		AdvanceDesc: `Advanced Details:
- Available methods: get_current_model, get_agent_name, get_tools, get_subagents
  Use expand tool with details_about="<method>" for full details on any method.
- Common parameters:
  * method (string, required): the method to call
- Behavior:
  * Reads from agent context only — no external calls or side effects`,
		DetailsAboutFunc: detailsAbout,
		TroubleshootingInfo: `Troubleshooting:
- If get_current_model returns empty: The model may not be configured in the agent context
- If get_tools returns empty array: No tools are currently available to the agent
- If get_subagents returns empty array: No subagents are currently configured for the agent
- Ensure the 'method' parameter is one of: get_current_model, get_agent_name, get_tools, get_subagents`,
		Parameters: []core.Parameter{
			{
				Name:        "method",
				Type:        "string",
				Description: "The meta method to call: 'get_current_model', 'get_agent_name', 'get_tools', or 'get_subagents'",
				Required:    true,
				Validator:   validateMethod,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			method := args["method"].(string)

			return meta.handleMethod(agentContext, method)
		},
	}
}
