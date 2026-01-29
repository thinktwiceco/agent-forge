package meta

import (
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

	return &core.Tool{
		Name:        "meta",
		Description: "Get metadata about the agent including current model, agent name, available tools, and subagents.",
		AdvanceDesc: `Advanced Details:
- Methods:
  * get_current_model: Returns the name of the LLM model currently being used by the agent
  * get_agent_name: Returns the name of the agent
  * get_tools: Returns a JSON array of all available tools with their names and descriptions
  * get_subagents: Returns a JSON array of all available subagents with their names and basic descriptions
- Behavior:
  * All methods retrieve information from the agent's context
  * No external calls or side effects
  * Returns structured information for introspection
- Usage:
  * Use to understand agent configuration and capabilities
  * Useful for debugging and understanding available tools
  * Can be called to check agent identity and model information`,
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
