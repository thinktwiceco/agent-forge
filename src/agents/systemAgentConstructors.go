package agents

import (
	"github.com/thinktwiceco/agent-forge/src/agents/system"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/tools/expand"
	"github.com/thinktwiceco/agent-forge/src/tools/fs"
	"github.com/thinktwiceco/agent-forge/src/tools/git"
	"github.com/thinktwiceco/agent-forge/src/tools/vector"
	"github.com/thinktwiceco/agent-forge/src/tools/web"
)

// systemConfigToAgentConfig converts a system.Config to an AgentConfig
func systemConfigToAgentConfig(cfg system.Config) AgentConfig {
	return AgentConfig{
		LLMEngine:          cfg.LLMEngine,
		AgentName:          cfg.AgentName,
		Trace:              cfg.Trace,
		SystemPrompt:       cfg.SystemPrompt,
		Description:        cfg.Description,
		AdvanceDescription: cfg.AdvanceDescription,
		Troubleshooting:    cfg.Troubleshooting,
		MainAgent:          cfg.MainAgent,
		Tone:               cfg.Tone,
		Tools:              nil, // Tools are added by specific agent constructors
	}
}

// CodingAgent creates a coding operations agent with file system and expand tools.
func CodingAgent(llmEngine llms.LLMEngine, root string) core.SubAgent {
	template := system.CreateCodingAgentTemplate()
	config := systemConfigToAgentConfig(template.ToConfig(llmEngine))

	// Add fs and expand tools
	fsTool := fs.NewFsTool(root)
	expandTool := expand.NewExpandTool()
	config.Tools = []llms.Tool{fsTool, expandTool}

	agent := NewAgent(&config)
	return agent.AgentAsSubAgent()
}

// GitAgent creates a git operations agent with git tools.
func GitAgent(llmEngine llms.LLMEngine, root string) core.SubAgent {
	template, err := system.CreateGitAgentTemplate()
	if err != nil {
		panic(err)
	}
	config := systemConfigToAgentConfig(template.ToConfig(llmEngine))

	// Add git tool
	gitTool := git.NewGitTool(root)
	config.Tools = []llms.Tool{gitTool}

	agent := NewAgent(&config)
	return agent.AgentAsSubAgent()
}

// OsAgent creates an OS operations agent with file system tools.
func OsAgent(llmEngine llms.LLMEngine, root string) core.SubAgent {
	template := system.CreateOsAgentTemplate()
	config := systemConfigToAgentConfig(template.ToConfig(llmEngine))

	// Add fs tool
	fsTool := fs.NewFsTool(root)
	config.Tools = []llms.Tool{fsTool}

	agent := NewAgent(&config)
	return agent.AgentAsSubAgent()
}

// ReasoningAgent creates a reasoning agent.
func ReasoningAgent(llmEngine llms.LLMEngine) core.SubAgent {
	template := system.CreateReasoningAgentTemplate()
	config := systemConfigToAgentConfig(template.ToConfig(llmEngine))

	agent := NewAgent(&config)
	return agent.AgentAsSubAgent()
}

// VectorAgent creates a vector database operations agent.
func VectorAgent(llmEngine llms.LLMEngine, vectorDB core.VectorDB, embeddingGenerator core.EmbeddingGenerator) core.SubAgent {
	template := system.CreateVectorAgentTemplate()
	config := systemConfigToAgentConfig(template.ToConfig(llmEngine))

	// Add vector tool
	vectorTool := vector.NewVectorTool(vectorDB, embeddingGenerator)
	config.Tools = []llms.Tool{vectorTool}

	agent := NewAgent(&config)
	return agent.AgentAsSubAgent()
}

// WebAgent creates a web operations agent.
func WebAgent(llmEngine llms.LLMEngine, workingDir string) core.SubAgent {
	template := system.CreateWebAgentTemplate()
	config := systemConfigToAgentConfig(template.ToConfig(llmEngine))

	// Add web tool
	webTool := web.NewWebTool(workingDir)
	config.Tools = []llms.Tool{webTool}

	agent := NewAgent(&config)
	return agent.AgentAsSubAgent()
}
