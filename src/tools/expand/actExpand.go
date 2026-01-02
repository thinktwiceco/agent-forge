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
	if errResp := validateSubjectTypeAndReturnError(subjectType); errResp != nil {
		return errResp
	}

	var discoverable agentforge.Discoverable
	var found bool

	// Search based on subject type
	if subjectType == "tool" {
		discoverable, found = findTool(agentContext, subjectName)
		if !found {
			return core.NewErrorResponse(fmt.Sprintf(
				"Tool '%s' not found in context. Available tools can be seen in your system prompt or by listing your tools.",
				subjectName,
			))
		}
	} else { // subjectType == "agent"
		discoverable, found = findAgent(agentContext, subjectName)
		if !found {
			return core.NewErrorResponse(fmt.Sprintf(
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

	return core.NewSuccessResponse(response.String())
}
