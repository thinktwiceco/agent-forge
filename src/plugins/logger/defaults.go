package logger

import "github.com/thinktwiceco/agent-forge/src/agents"

// DefaultColorRules returns default color rules matching the behavior from cmd/chat/main.go
func DefaultColorRules() []ColorRule {
	return []ColorRule{
		// Yellow for reasoning agents (agentName contains "reasoning" or trace contains "thinking"/"reasoning")
		{
			AgentNamePattern: agents.TraceReasoning,
			TracePattern:     "",
			Color:            "\033[33m", // ColorYellow
		},
		{
			AgentNamePattern: "",
			TracePattern:     agents.TraceThinking,
			Color:            "\033[33m", // ColorYellow
		},
		{
			AgentNamePattern: "",
			TracePattern:     agents.TraceReasoning,
			Color:            "\033[33m", // ColorYellow
		},
		// Dim yellow for sub-agents is handled in the logic (system agents)
		// Cyan for main agent (default - any non-system agent)
		// No specific rule needed as it's the default
	}
}

// DefaultLabelRules returns default label rules matching the behavior from cmd/chat/main.go
func DefaultLabelRules() []LabelRule {
	return []LabelRule{
		// 🧠 for reasoning agents
		{
			AgentNamePattern: agents.TraceReasoning,
			TracePattern:     "",
			Emoji:            "🧠",
			IsSubAgent:       false,
			Format:           "%s",
		},
		{
			AgentNamePattern: "",
			TracePattern:     agents.TraceThinking,
			Emoji:            "🧠",
			IsSubAgent:       false,
			Format:           "%s",
		},
		// → for sub-agents (system agents)
		{
			AgentNamePattern: "",
			TracePattern:     "",
			Emoji:            "→",
			IsSubAgent:       true,
			Format:           "%s",
		},
		// 💬 for main agent (default - any non-system agent)
		// No specific rule needed as it's the default
	}
}
