package expand

import (
	"fmt"
	"strings"

	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// expand retrieves detailed information about a tool or sub-agent.
func (e *Expand) expand(agentContext map[string]any, subjectType string, subjectName string, troubleshoot bool) llms.ToolReturn {
	// Validate subject_type

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
			if tool.GetName() == subjectName {
				discoverable = tool.(agentforge.Discoverable)
				found = true
				break
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
			if (*subAgent).Name() == subjectName {
				discoverable = *subAgent
				found = true
				break
			}
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

	return core.NewSuccessResponse(response.String())
}
