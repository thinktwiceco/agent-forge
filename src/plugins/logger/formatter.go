package logger

import (
	"strings"

	"github.com/thinktwiceco/agent-forge/src/agents"
)

// ColorRule defines a rule for applying colors based on agent name and trace patterns
type ColorRule struct {
	AgentNamePattern string // Pattern to match agent names (case-insensitive substring)
	TracePattern     string // Pattern to match trace values (case-insensitive substring)
	Color            string // ANSI color code to apply
}

// LabelRule defines a rule for formatting labels based on agent name and trace patterns
type LabelRule struct {
	AgentNamePattern string // Pattern to match agent names (case-insensitive substring)
	TracePattern     string // Pattern to match trace values (case-insensitive substring)
	Emoji            string // Emoji to use for label
	IsSubAgent       bool   // Whether this rule applies to sub-agents
	Format           string // Format string for label (e.g., "%s %s - %s")
}

// matchesPattern performs case-insensitive substring matching
func matchesPattern(text, pattern string) bool {
	if pattern == "" {
		return true // Empty pattern matches everything
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(pattern))
}

// isSystemAgent checks if an agent name is a system-defined agent
func isSystemAgent(agentName string) bool {
	return strings.HasPrefix(agentName, agents.SystemAgentPrefix)
}

// getColorWithRules returns the color code based on agent name and trace using the provided rules
func getColorWithRules(agentName, trace string, rules []ColorRule) string {
	// First check if this is a reasoning agent (agentName contains "reasoning" or trace contains "thinking"/"reasoning")
	if matchesPattern(agentName, agents.TraceReasoning) ||
		matchesPattern(trace, agents.TraceThinking) ||
		matchesPattern(trace, agents.TraceReasoning) {
		// Find yellow color rule
		for _, rule := range rules {
			if rule.Color == "\033[33m" {
				return rule.Color
			}
		}
		return "\033[33m" // ColorYellow
	}

	// Check if this is a system agent (sub-agent)
	if isSystemAgent(agentName) {
		// Find dim yellow color rule
		for _, rule := range rules {
			if rule.Color == "\033[2m\033[33m" {
				return rule.Color
			}
		}
		return "\033[2m\033[33m" // ColorDim + ColorYellow
	}

	// Default to cyan for main agent (user-defined agents)
	for _, rule := range rules {
		if rule.Color == "\033[36m" {
			return rule.Color
		}
	}
	return "\033[36m" // ColorCyan
}

// formatLabelWithRules returns a formatted label based on agent name and trace using the provided rules
func formatLabelWithRules(agentName, trace string, rules []LabelRule) string {
	// Check if this is a system agent (sub-agent)
	isSubAgent := isSystemAgent(agentName) && !matchesPattern(agentName, agents.TraceReasoning)

	// Determine emoji based on agent type
	emoji := "💬" // Default for main agent
	if matchesPattern(agentName, agents.TraceReasoning) ||
		matchesPattern(trace, agents.TraceThinking) {
		emoji = "🧠"
	} else if isSubAgent {
		emoji = "→"
	}

	// Try to find matching rule for emoji
	for _, rule := range rules {
		if rule.IsSubAgent && !isSubAgent {
			continue
		}
		if !rule.IsSubAgent && isSubAgent {
			continue
		}
		if matchesPattern(agentName, rule.AgentNamePattern) && matchesPattern(trace, rule.TracePattern) {
			emoji = rule.Emoji
			break
		}
	}

	// Format based on agent type
	if isSubAgent {
		if trace != "" {
			return emoji + " " + trace
		}
		return emoji + " " + agentName
	}

	// Main agent format
	if trace != "" {
		return emoji + " " + agentName + " - " + trace
	}
	return emoji + " " + agentName
}
