# How-To: Add System Agents

System agents are pre-configured agents with specific capabilities.

## Create Template

```go
// src/agents/system/myagent.go
package system

func CreateMyAgentTemplate() *Template {
    return &Template{
        Name:               "system-myagent",
        Trace:              "myagent",
        SystemPrompt:       "You are a specialized agent for...",
        Tone:               agents.ToneSystemAgent,
        ToolConstructor:    func(rootDir string) llms.Tool { 
            return mytool.NewMyTool(rootDir) 
        },
    }
}
```

## Add Constructor

```go
// src/agents/systemAgentConstructors.go
package agents

func MyAgent(llmEngine llms.LLMEngine, rootDir string) core.SubAgent {
    template := system.CreateMyAgentTemplate()
    agent := template.BuildAgent(llmEngine, rootDir)
    return agent.AgentAsSubAgent()
}
```

## Use System Agent

```go
myAgent := agents.MyAgent(llm, "/path/to/root")
mainAgent.AddSystemAgent(myAgent)
```

## Existing System Agents

- `ReasoningAgent(llm)` - Analyzes questions before responding
- `CodingAgent(llm, root)` - Code generation and analysis
- `OsAgent(llm, root)` - File system operations
- `GitAgent(llm, root)` - Git operations
- `WebAgent(llm, workingDir)` - Web automation
