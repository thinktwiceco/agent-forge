package tools

import (
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// NewFooTool creates a new FooTool instance
func NewFooTool() llms.Tool {
	return core.NewTool(
		"foo",
		"A test tool that returns the echo argument. Use this to test only.",
		`Advanced Details:
- Parameters: 
  * echo (string, required): The text to echo back
- Behavior: Returns the input string unchanged
- Usage: Primarily for testing tool invocation and response handling
- Performance: Instant response with no side effects`,
		`Troubleshooting:
- If the tool fails, ensure the 'echo' parameter is provided as a string
- Empty strings are valid and will be returned as-is
- This tool has no external dependencies and should always succeed if called correctly`,
		[]core.Parameter{
			{Name: "echo", Type: "string", Required: true},
		},
		func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			return core.NewSuccessResponse(args["echo"].(string))
		},
	)
}
