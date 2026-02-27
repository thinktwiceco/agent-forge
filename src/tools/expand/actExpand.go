package expand

import (
	"fmt"
	"strings"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// expand retrieves detailed information about a tool or sub-agent.
func (e *Expand) expand(agentContext map[string]any, subjectType string, subjectName string, troubleshoot bool, detailsAbout string) llms.ToolReturn {
	ctx, err := core.RehydrateContext(agentContext)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("Error rehydrating context: %v", err))
	}

	tools := ctx.Tools
	subAgents := ctx.SubAgents

	if errResp := validateSubjectTypeAndReturnError(subjectType); errResp != nil {
		return errResp
	}

	var discoverable agentforge.Discoverable
	var found bool

	// Search based on subject type
	if subjectType == "tool" {
		for _, tool := range tools {
			if tool != nil && tool.GetName() == subjectName {
				if disc, ok := tool.(agentforge.Discoverable); ok {
					discoverable = disc
					found = true
					break
				}
			}
		}
		if !found {
			return core.NewErrorResponse(fmt.Sprintf(
				"Tool '%s' not found in context. Available tools can be seen in your system prompt or by listing your tools.",
				subjectName,
			))
		}
	}

	if subjectType == "agent" {
		for _, subAgent := range subAgents {
			if subAgent != nil && subAgent.Name() == subjectName {
				discoverable = subAgent
				found = true
				break
			}
		}
		if !found {
			return core.NewErrorResponse(fmt.Sprintf(
				"Agent '%s' not found in context. Available agents can be seen in your system prompt or by listing your agents.",
				subjectName,
			))
		}
	}

	// If details_about is set, return per-item details only
	if detailsAbout != "" {
		var response strings.Builder
		fmt.Fprintf(&response, "=== %s: %s — details about: %s ===\n\n", strings.ToUpper(subjectType), subjectName, detailsAbout)
		response.WriteString(discoverable.DetailsAbout(detailsAbout))
		response.WriteString("\n")
		return core.NewSuccessResponse(response.String())
	}

	// Build the full response
	var response strings.Builder
	fmt.Fprintf(&response, "=== %s: %s ===\n\n", strings.ToUpper(subjectType), subjectName)

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

	return core.NewSuccessResponse(response.String())
}
