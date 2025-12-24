package tools

import (
	"fmt"
	"strings"

	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/llms"
)

// NewExpandTool creates a tool that allows progressive discovery of tools and agents.
//
// This tool enables agents to retrieve detailed information (AdvanceDescription and
// Troubleshooting) about tools and sub-agents that are available in their context.
//
// The tool expects the following items in agentContext:
//   - "tools": []llms.Tool - list of available tools
//   - "subAgents": []SubAgent - list of available sub-agents
func NewExpandTool() llms.Tool {
	return NewContextAwareTool(
		"expand",
		"Get detailed information about a tool or sub-agent. Use this to discover advanced capabilities and troubleshooting information.",
		`Advanced Details:
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
		`Troubleshooting:
- "Not found" errors: Verify subject_name matches exactly (case-sensitive)
- "Invalid subject_type": Must be exactly "tool" or "agent"
- "Does not implement Discoverable": The tool/agent doesn't support discovery
- Empty descriptions: The tool/agent may not have advanced descriptions configured
- Tool not in context: Ensure the tool is added to the agent's tool list`,
		[]Parameter{
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
		func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			subjectType := args["subject_type"].(string)
			subjectName := args["subject_name"].(string)

			// Get troubleshoot flag (default to false if not provided)
			troubleshoot := false
			if val, ok := args["troubleshoot"]; ok {
				troubleshoot = val.(bool)
			}

			// Validate subject_type
			if subjectType != "tool" && subjectType != "agent" {
				return NewErrorResponse(fmt.Sprintf(
					"Invalid subject_type '%s'. Must be either 'tool' or 'agent'",
					subjectType,
				))
			}

			var discoverable agentforge.Discoverable
			var found bool

			// Search based on subject type
			if subjectType == "tool" {
				discoverable, found = findTool(agentContext, subjectName)
				if !found {
					return NewErrorResponse(fmt.Sprintf(
						"Tool '%s' not found in context. Available tools can be seen in your system prompt or by listing your tools.",
						subjectName,
					))
				}
			} else { // subjectType == "agent"
				discoverable, found = findAgent(agentContext, subjectName)
				if !found {
					return NewErrorResponse(fmt.Sprintf(
						"Agent '%s' not found in context. Available agents are listed in your system prompt under [SUB AGENTS].",
						subjectName,
					))
				}
			}

			// Build the response
			var response strings.Builder
			response.WriteString(fmt.Sprintf("=== %s: %s ===\n\n", strings.ToUpper(subjectType), subjectName))

			// Basic description
			response.WriteString("📄 Basic Description:\n")
			response.WriteString(discoverable.BasicDescription())
			response.WriteString("\n\n")

			// Advanced description
			response.WriteString("📚 Advanced Description:\n")
			advDesc := discoverable.AdvanceDescription()
			if advDesc == "" {
				response.WriteString("(No advanced description available)")
			} else {
				response.WriteString(advDesc)
			}
			response.WriteString("\n")

			// Troubleshooting (if requested)
			if troubleshoot {
				response.WriteString("\n🔧 Troubleshooting:\n")
				troubleshootInfo := discoverable.Troubleshooting()
				if troubleshootInfo == "" {
					response.WriteString("(No troubleshooting information available)")
				} else {
					response.WriteString(troubleshootInfo)
				}
				response.WriteString("\n")
			}

			return NewSuccessResponse(response.String())
		},
	)
}

// findTool searches for a tool by name in the agent context
func findTool(agentContext map[string]any, toolName string) (agentforge.Discoverable, bool) {
	toolsInterface, ok := agentContext["tools"]
	if !ok {
		return nil, false
	}

	tools, ok := toolsInterface.([]llms.Tool)
	if !ok {
		return nil, false
	}

	for _, tool := range tools {
		if tool.GetName() == toolName {
			// Try to cast to Discoverable
			if discoverable, ok := tool.(agentforge.Discoverable); ok {
				return discoverable, true
			}
			return nil, false
		}
	}

	return nil, false
}

// SubAgentDiscoverable wraps a SubAgent to provide Discoverable interface access
type SubAgentDiscoverable interface {
	SubAgent
	agentforge.Discoverable
}

// findAgent searches for a sub-agent by name in the agent context
func findAgent(agentContext map[string]any, agentName string) (agentforge.Discoverable, bool) {
	subAgentsInterface, ok := agentContext["subAgents"]
	if !ok {
		return nil, false
	}

	// The subAgents in context could be various types, we need to handle them
	// They should be SubAgent interfaces that also implement Discoverable
	switch subAgents := subAgentsInterface.(type) {
	case []SubAgentDiscoverable:
		for _, agent := range subAgents {
			if agent.Name() == agentName {
				return agent, true
			}
		}
	case []SubAgent:
		for _, agent := range subAgents {
			if agent.Name() == agentName {
				// Try to cast to Discoverable
				if discoverable, ok := agent.(agentforge.Discoverable); ok {
					return discoverable, true
				}
			}
		}
	}

	return nil, false
}
