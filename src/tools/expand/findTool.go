package expand

import (
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// findTool searches for a tool by name in the agent context.
// This function is currently unused but kept for potential future use.
//
//nolint:unused // Reserved for future use
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
