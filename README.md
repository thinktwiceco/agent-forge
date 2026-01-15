<div align="center">
  <img src="assets/agent_forge_logo.png" alt="Agent Forge Logo" width="400"/>
</div>

A powerful Go framework for building intelligent agents with LLM integration, tool execution, and multi-agent collaboration.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
  - [Creating Agents](#creating-agents)
  - [Creating Tools](#creating-tools)
  - [Multi-Agent Teams](#multi-agent-teams)
- [System Agents](#system-agents)
  - [Reasoning Agent](#reasoning-agent)
  - [OS Agent](#os-agent)
- [Advanced Features](#advanced-features)
  - [Streaming Responses](#streaming-responses)
  - [Conversation Persistence](#conversation-persistence)
  - [Hook System](#hook-system)
- [Plugins](#plugins)
  - [Logger Plugin](src/plugins/logger/README.md)
  - [Todo Plugin](src/plugins/todo/README.md)
- [Complete Example](#complete-example)
- [Environment Variables](#environment-variables)
- [Project Structure](#project-structure)
- [Contributing](#contributing)
- [License](#license)

## Features

- 🚀 **Simple Agent Creation** - Create AI agents with just a few lines of code
- 🔧 **Extensible Tool System** - Build custom tools with automatic validation
- 👥 **Multi-Agent Teams** - Orchestrate teams of specialized agents
- 🔄 **Streaming Responses** - Real-time streaming of agent responses
- 💾 **Conversation Persistence** - Built-in conversation history storage
- 🔌 **Multiple LLM Providers** - Support for OpenAI, DeepSeek, TogetherAI, and OpenAI-compatible APIs

## Installation

```bash
go get github.com/thinktwice/agentForge
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/thinktwice/agentForge/src/agents"
    "github.com/thinktwice/agentForge/src/llms"
)

func main() {
    ctx := context.Background()
    
    // Create an LLM engine
    llm, err := llms.GetTogetherAILLM(ctx, llms.Llama3170BInstructTurbo)
    if err != nil {
        panic(err)
    }
    
    // Create an agent
    agent := agents.NewAgent(&agents.AgentConfig{
        LLMEngine:    llm,
        AgentName:    "Assistant",
        SystemPrompt: "You are a helpful AI assistant.",
        MainAgent:    true,
    })
    
    // Chat with the agent
    responseCh := agent.ChatStream("Hello! How can you help me?")
    
    // Process streaming response
    for chunk := range responseCh.Start() {
        if chunk.Content != "" {
            fmt.Print(chunk.Content)
        }
    }
}
```

## Core Concepts

### Creating Agents

Agents are created using `AgentConfig`:

```go
agent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine:         llm,              // Required: LLM engine
    AgentName:         "my-agent",        // Required: Agent name
    Description:       "Agent description", // Optional
    SystemPrompt:      "You are...",      // Optional
    Tools:             []llms.Tool{},     // Optional: Available tools
    MainAgent:         true,              // Optional: Is this the main agent?
    Persistence:       "json",            // Optional: Enable conversation history
    Plugins:           []core.Plugin{},   // Optional: Plugins
})

// Add system agents after creation
reasoningAgent := agents.ReasoningAgent(llm)
agent.AddSystemAgent(reasoningAgent)
```

### Creating Tools

Tools extend agent capabilities:

```go
import "github.com/thinktwice/agentForge/src/core"

calculatorTool := &core.Tool{
    Name:        "calculate",
    Description: "Performs mathematical calculations",
    Parameters: []core.Parameter{
        {
            Name:        "expression",
            Type:        "string",
            Description: "Mathematical expression to evaluate",
            Required:    true,
        },
    },
    Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
        expression := args["expression"].(string)
        result := evaluate(expression)
        return core.NewSuccessResponse(result)
    },
}

// Add tool to agent
agent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine: llm,
    AgentName: "calculator-agent",
    Tools:     []llms.Tool{calculatorTool},
})
```

### Multi-Agent Teams

Create specialized agents and coordinate them:

```go
// Create specialized custom sub-agents
customAgent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine:   llm,
    AgentName:   "custom-agent",
    Description: "Breaks down complex problems",
    SystemPrompt: "You are a specialized agent.",
})

// Convert to sub-agent
customSubAgent := customAgent.AgentAsSubAgent()

// Create main agent
mainAgent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine:   llm,
    AgentName:   "coordinator",
    MainAgent:   true,
    SubAgents:   []*core.SubAgent{customSubAgent},
})

// Add system agents (see System Agents section)
reasoningAgent := agents.ReasoningAgent(llm)
mainAgent.AddSystemAgent(reasoningAgent)

// The delegate tool is automatically added when sub-agents are present
```

## System Agents

System agents are pre-defined specialized agents that can be added to your main agent. They provide common functionality like reasoning analysis and OS operations.

### Reasoning Agent

The reasoning agent analyzes questions before the main agent responds, helping identify ambiguities, detect assumptions, and guide objective responses.

```go
// Create and add reasoning agent
reasoningAgent := agents.ReasoningAgent(llm)
agent.AddSystemAgent(reasoningAgent)
```

**Use cases:**
- Analyzing ambiguous questions
- Detecting missing information
- Providing guidance for objective responses
- Flagging assumptions before responding

### OS Agent

The OS agent handles file system operations and OS-related tasks within a restricted directory.

```go
// Create and add OS agent with root directory
osAgent := agents.OsAgent(llm, "/path/to/root")
agent.AddSystemAgent(osAgent)
```

**Use cases:**
- Reading and writing files
- File system operations
- Executing OS-level commands
- Managing file resources

**Note:** The OS agent operates within a restricted root directory for security. All file paths are validated to prevent directory traversal.

## Advanced Features

### Streaming Responses

All agent responses are streamed in real-time:

```go
responseCh := agent.ChatStream("What is the capital of France?")

for chunk := range responseCh.Start() {
    switch chunk.Type {
    case llms.TypeContent:
        fmt.Print(chunk.Content)
    case llms.TypeToolExecuting:
        fmt.Printf("Executing: %s\n", chunk.ToolExecuting.Name)
    case llms.TypeToolResult:
        fmt.Printf("Result: %s\n", chunk.ToolResults[0].Result)
    }
}
```

### Conversation Persistence

Enable conversation history storage:

```go
agent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine:   llm,
    AgentName:   "persistent-agent",
    Persistence: "json",  // Stores history as JSON files
})
```

### Hook System

Register hooks for lifecycle events:

```go
agent.on(agents.EventBeforeToolExecution, func(a *agents.Agent, toolCall *llms.ToolCall) error {
    fmt.Printf("About to execute tool: %s\n", toolCall.Name)
    return nil
})
```

## Plugins

Plugins extend agent functionality by providing tools, hooks, and system prompt enhancements.

- **[Logger Plugin](src/plugins/logger/README.md)** - Configurable output formatting for agent responses
- **[Todo Plugin](src/plugins/todo/README.md)** - Task management and todo list functionality

See the [Plugins README](src/plugins/README.md) for more information on creating custom plugins.

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/thinktwice/agentForge/src/agents"
    "github.com/thinktwice/agentForge/src/core"
    "github.com/thinktwice/agentForge/src/llms"
)

func main() {
    ctx := context.Background()
    
    // Initialize LLM
    llm, err := llms.GetTogetherAILLM(ctx, llms.Llama3170BInstructTurbo)
    if err != nil {
        panic(err)
    }
    
    // Create a calculator tool
    calcTool := &core.Tool{
        Name:        "calculate",
        Description: "Performs mathematical calculations",
        Parameters: []core.Parameter{
            {Name: "expression", Type: "string", Required: true},
        },
        Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
            expr := args["expression"].(string)
            // Your calculation logic here
            return core.NewSuccessResponse("Result: 42")
        },
    }
    
    // Create agent with tools
    agent := agents.NewAgent(&agents.AgentConfig{
        LLMEngine:   llm,
        AgentName:   "MathAssistant",
        SystemPrompt: "You are a helpful math assistant.",
        Tools:       []llms.Tool{calcTool},
        MainAgent:   true,
        Persistence: "json",
    })
    
    // Add reasoning system agent
    reasoningAgent := agents.ReasoningAgent(llm)
    agent.AddSystemAgent(reasoningAgent)
    
    // Chat with the agent
    responseCh := agent.ChatStream("What is 15 multiplied by 23?")
    
    for chunk := range responseCh.Start() {
        if chunk.Content != "" {
            fmt.Print(chunk.Content)
        }
    }
}
```

## Environment Variables

- `TOGETHERAI_API_KEY` - API key for TogetherAI
- `DEEPSEEK_API_KEY` - API key for DeepSeek
- `OPENAI_API_KEY` - API key for OpenAI

Set via `.env` file or system environment variables.

## Project Structure

```
src/
├── agents/          # Agent implementation
├── llms/            # LLM engine implementations
├── core/            # Core interfaces and implementations
├── tools/           # Tool implementations
├── persistence/     # Conversation persistence
└── plugins/         # Plugin system
    ├── logger/      # Logger plugin
    └── todo/        # Todo plugin
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

[Add your license here]
