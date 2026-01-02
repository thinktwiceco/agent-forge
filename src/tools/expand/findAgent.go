package expand

import (
	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/core"
)

// SubAgentDiscoverable wraps a core.SubAgent to provide Discoverable interface access.
type SubAgentDiscoverable interface {
	core.SubAgent
	agentforge.Discoverable
}

// findAgent searches for a sub-agent by name in the agent context.
func findAgent(agentContext map[string]any, agentName string) (agentforge.Discoverable, bool) {
	subAgentsInterface, ok := agentContext["subAgents"]
	if !ok {
		return nil, false
	}

	// The subAgents in context could be various types, we need to handle them
	// They should be core.SubAgent interfaces that also implement Discoverable
	switch subAgents := subAgentsInterface.(type) {
	case []SubAgentDiscoverable:
		for _, agent := range subAgents {
			if agent.Name() == agentName {
				return agent, true
			}
		}
	case []core.SubAgent:
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
