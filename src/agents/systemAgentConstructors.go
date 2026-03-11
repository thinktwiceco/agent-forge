package agents

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/thinktwiceco/agent-forge/src/agents/system"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/tools/api"
	"github.com/thinktwiceco/agent-forge/src/tools/expand"
	"github.com/thinktwiceco/agent-forge/src/tools/fs"
	"github.com/thinktwiceco/agent-forge/src/tools/git"
	imagetool "github.com/thinktwiceco/agent-forge/src/tools/image"
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
func GitAgent(llmEngine llms.LLMEngine, workingDir string) core.SubAgent {
	template, err := system.CreateGitAgentTemplate()
	if err != nil {
		panic(err)
	}
	config := systemConfigToAgentConfig(template.ToConfig(llmEngine))

	// Add git tool
	gitTool := git.NewGitTool(filepath.Join(workingDir, "repos"))
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

// VisionAgent creates a vision agent that loads images and answers visual questions.
// The llmEngine must be backed by a vision-capable model (e.g. gpt-4o).
func VisionAgent(llmEngine llms.LLMEngine, workingDir string) core.SubAgent {
	template := system.CreateVisionAgentTemplate()
	config := systemConfigToAgentConfig(template.ToConfig(llmEngine))

	imgTool := imagetool.NewImageTool(workingDir)
	config.Tools = []llms.Tool{imgTool}

	agent := NewAgent(&config)
	return agent.AgentAsSubAgent()
}

// WebAgent creates a web operations agent.
// extraTools are appended after the built-in web tool (e.g. the API tool).
// If an API tool is present, the agent description is enriched with the
// available service names so the main agent can discover them without a live call.
func WebAgent(llmEngine llms.LLMEngine, workingDir string, extraTools ...llms.Tool) core.SubAgent {
	template := system.CreateWebAgentTemplate()
	config := systemConfigToAgentConfig(template.ToConfig(llmEngine))

	webTool := web.NewWebTool(filepath.Join(workingDir, "web"))
	config.Tools = append([]llms.Tool{webTool}, extraTools...)

	// Handshake: enrich the description with API service names so the main
	// agent's system prompt includes them without needing a live delegation.
	var apiServices []string
	for _, t := range extraTools {
		if sp, ok := t.(api.ServiceProvider); ok {
			apiServices = append(apiServices, sp.ServiceNames()...)
		}
	}
	if len(apiServices) > 0 {
		config.Description = strings.TrimRight(config.Description, "\n") +
			fmt.Sprintf("\nAPI services available: %s. Use action=show_apis to list endpoints.", strings.Join(apiServices, ", "))
		config.AdvanceDescription = strings.TrimRight(config.AdvanceDescription, "\n") +
			fmt.Sprintf("\n- API services: %s (use show_apis → show_api → call endpoint)", strings.Join(apiServices, ", "))
	}

	agent := NewAgent(&config)
	return agent.AgentAsSubAgent()
}
