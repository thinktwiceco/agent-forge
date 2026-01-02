package expand

import (
	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/llms"
)

// findTool searches for a tool by name in the agent context.
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
