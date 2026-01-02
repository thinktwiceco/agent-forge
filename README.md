<div align="center">
  <img src="assets/agent_forge_logo.png" alt="Agent Forge Logo" width="400"/>
</div>

# ThinkTwice Agent 🤖

A powerful Go framework for building intelligent agents with LLM integration, tool execution, and multi-agent collaboration.

## Features

- 🚀 **Simple Agent Creation** - Create AI agents with just a few lines of code
- 🔧 **Extensible Tool System** - Build custom tools with automatic validation and execution
- 👥 **Multi-Agent Teams** - Orchestrate teams of specialized agents working together
- 🔄 **Streaming Responses** - Real-time streaming of agent responses and tool execution
- 💾 **Conversation Persistence** - Built-in support for conversation history storage
- 🎯 **Progressive Discovery** - Agents can discover tool and sub-agent capabilities at runtime
- 🔌 **Multiple LLM Providers** - Support for OpenAI, DeepSeek, TogetherAI, and any OpenAI-compatible API

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
        Description:  "A helpful AI assistant",
        SystemPrompt: "You are a helpful and intelligent AI assistant.",
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

## Basic Agent Usage

### Creating an Agent

Create an agent using the `AgentConfig` struct:

```go
agent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine:         llm,                    // Required: LLM engine
    AgentName:         "my-agent",             // Required: Agent name
    Description:       "Basic description",    // Optional: Short description
    SystemPrompt:      "You are...",           // Optional: Custom system prompt
    Tools:             []llms.Tool{},          // Optional: Available tools
    MaxToolIterations: 10,                     // Optional: Max tool execution loops (default: 10)
    MainAgent:         true,                   // Optional: Is this the main agent?
    Persistence:       "json",                 // Optional: Enable conversation history
    CanExpand:         true,                   // Optional: Enable expand tool (default: false)
    Reasoning:         false,                 // Optional: Enable reasoning sub-agent
    SubAgents:         []*core.SubAgent{},      // Optional: Custom sub-agents
})
```

### Streaming Responses

All agent responses are streamed in real-time:

```go
responseCh := agent.ChatStream("What is the capital of France?")

// Use Start() to get ExtendedChunkResponse with agent metadata
for chunk := range responseCh.Start() {
    switch chunk.Type {
    case llms.TypeContent:
        // Content being streamed
        // chunk is ExtendedChunkResponse with AgentName and Trace
        fmt.Printf("[%s] %s", chunk.AgentName, chunk.Content)
        
    case llms.TypeCompletion:
        // Final completion with token usage
        fmt.Printf("\nTokens: %d total\n", chunk.TotalTokens)
        
    case llms.TypeToolExecuting:
        // Tool is being executed
        fmt.Printf("Executing: %s\n", chunk.ToolExecuting.Name)
        
    case llms.TypeToolResult:
        // Tool execution completed
        for _, result := range chunk.ToolResults {
            fmt.Printf("Result: %s\n", result.Result)
        }
    }
    
    if chunk.Status == llms.StatusError {
        fmt.Printf("Error: %s\n", chunk.Content)
    }
}
```

### LLM Engine Setup

#### TogetherAI

```go
llm, err := llms.GetTogetherAILLM(ctx, llms.Llama3170BInstructTurbo)
// Requires: TOGETHERAI_API_KEY environment variable
```

#### DeepSeek

```go
llm, err := llms.GetDeepSeekLLM(ctx, "deepseek-chat")
// Requires: DEEPSEEK_API_KEY environment variable
```

#### Custom OpenAI-Compatible API

```go
import "github.com/thinktwice/agentForge/src/llms"

// For any OpenAI-compatible API, use the internal constructor
// (Note: This requires accessing the internal newOpenAILLM function)
```

## Creating Tools

Tools extend agent capabilities using a universal tool system where all tools receive agent context:

### Creating Tools

All tools are created using the `core.Tool` struct and receive both agent context and arguments:

```go
import "github.com/thinktwice/agentForge/src/core"

calculatorTool := &core.Tool{
    Name:        "calculate",
    Description: "Performs mathematical calculations",
    AdvanceDesc: `Advanced Details:
- Supports: +, -, *, /
- Returns numeric result`,
    TroubleshootingInfo: `Troubleshooting:
- Division by zero returns error
- Use proper syntax: "2 + 2"`,
    Parameters: []core.Parameter{
        {
            Name:        "expression",
            Type:        "string",
            Description: "Mathematical expression to evaluate",
            Required:    true,
        },
    },
    Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
        // Tools that don't need context can ignore it
        expression := args["expression"].(string)
        
        // Perform calculation
        result := calculate(expression)
        
        return core.NewSuccessResponse(result)
    },
}
```

### Using Agent Context

Tools can access agent context for advanced functionality. The agent context is automatically built once at agent initialization and includes:

- `agentName` - The name of the agent executing the tool
- `trace` - Trace information for the agent
- `model` - The LLM model being used
- `tools` - List of available tools
- `subAgents` - List of available sub-agents
- `responseCh` - The response channel for streaming responses

```go
loggerTool := &core.Tool{
    Name:        "log_message",
    Description: "Logs a message with agent context",
    AdvanceDesc: "Logs messages with agent name and trace information",
    TroubleshootingInfo: "Check file permissions if logging fails",
    Parameters: []core.Parameter{
        {Name: "message", Type: "string", Required: true},
        {Name: "level", Type: "string", Required: false},
    },
    Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
        // Access agent context
        agentName := agentContext["agentName"].(string)
        message := args["message"].(string)
        
        // Log with context
        log.Printf("[%s] %s", agentName, message)
        
        return core.NewSuccessResponse("Logged successfully")
    },
}
```

### Parameter Types

Supported parameter types with automatic validation:

```go
[]core.Parameter{
    {Name: "text", Type: "string", Required: true},
    {Name: "count", Type: "number", Required: true},
    {Name: "enabled", Type: "boolean", Required: false},
    {Name: "config", Type: "object", Required: false},
    {Name: "items", Type: "array", Required: false},
}
```

### Custom Validation

Add custom validators for complex validation logic:

```go
{
    Name:        "email",
    Type:        "string",
    Description: "User email address",
    Required:    true,
    Validator: func(value any) error {
        email := value.(string)
        if !strings.Contains(email, "@") {
            return fmt.Errorf("invalid email format")
        }
        return nil
    },
}
```

### Tool Response Types

```go
// Success response
return core.NewSuccessResponse("Operation completed successfully")

// Error response (no data)
return core.NewErrorResponse("Invalid parameter: count must be positive")

// Failure response (with partial data)
return core.NewFailureResponse("Timeout occurred", "Partial result: 42")
```

### Adding Tools to Agents

```go
agent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine: llm,
    AgentName: "tool-user",
    Tools: []llms.Tool{
        calculatorTool,
        weatherTool,
        databaseTool,
    },
})

// Or add/modify tools after creation
agent.SetTools([]llms.Tool{newTool1, newTool2})
existingTools := agent.GetTools()
```

## Creating Teams of Agents

Multi-agent systems allow specialization and delegation:

### Basic Team Structure

```go
import "github.com/thinktwice/agentForge/src/core"

// Create specialized sub-agents
reasoningAgent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine:   llm,
    AgentName:   "reasoning-agent",
    Description: "Breaks down complex problems into logical steps",
    SystemPrompt: `You are a reasoning agent that excels at analytical thinking.
Break down complex problems systematically.`,
})

dataAgent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine:   llm,
    AgentName:   "data-agent",
    Description: "Analyzes and processes data",
    SystemPrompt: "You are a data analysis expert.",
    Tools: []llms.Tool{
        databaseTool,
        analyticsTool,
    },
})

// Convert agents to sub-agents for delegation
reasoningSubAgent := reasoningAgent.AgentAsSubAgent()
dataSubAgent := dataAgent.AgentAsSubAgent()

// Create main agent that coordinates the team
mainAgent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine:   llm,
    AgentName:   "coordinator",
    Description: "Main coordinator agent",
    SystemPrompt: "You coordinate a team of specialized agents.",
    MainAgent:   true,
    SubAgents: []*core.SubAgent{
        reasoningSubAgent,
        dataSubAgent,
    },
})

// The delegate tool is automatically added when sub-agents are present
// Sub-agents are available through the "delegate" tool
```

### Built-in Team Features

#### Reasoning Mode

Enable automatic reasoning capabilities:

```go
agent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine:   llm,
    AgentName:   "smart-agent",
    Reasoning:   true,  // Automatically adds a reasoning sub-agent
    MainAgent:   true,
})
```

#### Delegation Tool

When agents have sub-agents, they automatically get a `delegate` tool:

```go
// The main agent can delegate to sub-agents
// This happens automatically based on the system prompt
```

The delegation is transparent:

```go
mainAgent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine: llm,
    AgentName: "main",
    Reasoning: true,
    MainAgent: true,
})

// User asks a complex question
responseCh := mainAgent.ChatStream("Analyze the implications of quantum computing on cryptography")

// The agent may automatically delegate to the reasoning agent
for chunk := range responseCh.Start() {
    // Chunks include AgentName and Trace to identify the source
    // ExtendedChunkResponse provides full context
    if chunk.Content != "" {
        fmt.Printf("[%s] %s", chunk.AgentName, chunk.Content)
    }
}
```

### Custom Sub-Agent Configuration

Use different LLM engines for different sub-agents:

```go
fastLLM, _ := llms.GetTogetherAILLM(ctx, llms.Llama323BInstructTurbo)
powerfulLLM, _ := llms.GetTogetherAILLM(ctx, llms.Llama3170BInstructTurbo)

agent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine: powerfulLLM,
    AgentName: "main",
    Reasoning: true,
    MainAgent: true,
    ExtraEngines: map[string]llms.LLMEngine{
        "system-reasoning": fastLLM,  // Use faster model for reasoning
    },
})
```

### Progressive Discovery of Agents

Agents can discover information about other agents at runtime using the `expand` tool:

```go
import "github.com/thinktwice/agentForge/src/tools/expand"

agent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine: llm,
    AgentName: "explorer",
    CanExpand: true,  // Enables the expand tool automatically
    // Or manually add it:
    // Tools: []llms.Tool{
    //     expand.NewExpandTool(),
    // },
})

// The agent can now query detailed information about tools and sub-agents
// Example: expand(subject_type="agent", subject_name="reasoning-agent", troubleshoot=true)
```

The Expand tool provides three levels of information:
- **BasicDescription**: Short one-line description (always visible in system prompt)
- **AdvanceDescription**: Detailed capabilities and usage patterns
- **Troubleshooting**: Common issues and debugging tips

## Advanced Features

### Hook System

The agent framework provides a comprehensive hook system for lifecycle events, allowing you to inject custom logic at key points in the agent's execution.

#### Available Hooks

- `contextBuild` - Triggered when agent context is being built
- `beforeToolExecution` - Triggered before a tool is executed
- `toolExecution` - Triggered after a tool execution completes
- `newUserMessage` - Triggered when a new user message is received
- `addSystemAgent` - Triggered before a system agent is added
- `addedSystemAgent` - Triggered after a system agent is added

#### Using Hooks

Hooks are registered using the `on()` method on the agent:

```go
agent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine: llm,
    AgentName: "monitored-agent",
    // ... other config
})

// Register a hook for tool execution
agent.on(agents.EventBeforeToolExecution, func(a *agents.Agent, toolCall *llms.ToolCall) error {
    fmt.Printf("About to execute tool: %s\n", toolCall.Name)
    return nil // Return error to abort (errors are logged but don't abort by default)
})

// Register a hook for new user messages
agent.on(agents.EventNewUserMessage, func(a *agents.Agent, message string) error {
    fmt.Printf("New message received: %s\n", message)
    return nil
})
```

#### System Hooks

The framework automatically registers system hooks for essential functionality:
- History management on new user messages
- System prompt injection
- Delegate tool reloading when sub-agents are added

These hooks ensure the agent functions correctly without requiring manual hook registration.

### Conversation Persistence

Store and retrieve conversation history:

```go
agent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine:   llm,
    AgentName:   "persistent-agent",
    Persistence: "json",  // Stores history as JSON files
})

// History is automatically saved and loaded
// Each agent gets its own history file based on AgentName
```

### Agent Context System

The agent context system provides structured access to agent information. The context is built once at agent initialization and reused for all tool calls, improving performance.

The `core.AgentContext` struct contains:
- `AgentName` - The agent's name
- `Trace` - Trace identifier
- `Model` - LLM model identifier
- `Tools` - Available tools
- `SubAgents` - Available sub-agents

The context is automatically merged with session-specific data (like `responseCh`) when tools are executed. Tools receive the complete context map with all agent information.

```go
// The agent context is automatically built and provided to tools
customTool := &core.Tool{
    Name:        "query_db",
    Description: "Queries the database",
    AdvanceDesc: "Advanced database query capabilities",
    TroubleshootingInfo: "Check database connection if queries fail",
    Parameters: []core.Parameter{
        {Name: "query", Type: "string", Required: true},
    },
    Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
        // Access agent context
        agentName := agentContext["agentName"].(string)
        model := agentContext["model"].(string)
        
        // Access response channel for streaming
        responseCh := agentContext["responseCh"].(*core.ResponseCh)
        
        // Use context in tool execution
        query := args["query"].(string)
        result := executeQuery(query)
        
        return core.NewSuccessResponse(result)
    },
}
```

### Discoverable Interface

All tools created with `core.Tool` automatically implement the Discoverable interface:

```go
// Tools automatically provide:
// - BasicDescription() - returns the Description field
// - AdvanceDescription() - returns the AdvanceDesc field
// - Troubleshooting() - returns the TroubleshootingInfo field

// For custom tool implementations, implement the interface:
type MyCustomTool struct {
    // ... your fields
}

func (t *MyCustomTool) BasicDescription() string {
    return "Short one-line description"
}

func (t *MyCustomTool) AdvanceDescription() string {
    return "Detailed information about capabilities"
}

func (t *MyCustomTool) Troubleshooting() string {
    return "Common issues and solutions"
}
```

## Complete Example: Multi-Agent System

```go
package main

import (
    "context"
    "fmt"
    "github.com/thinktwice/agentForge/src/agents"
    "github.com/thinktwice/agentForge/src/core"
    "github.com/thinktwice/agentForge/src/llms"
    "github.com/thinktwice/agentForge/src/tools/expand"
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
        AdvanceDesc: "Supports +, -, *, / operations",
        TroubleshootingInfo: "Use proper syntax: '2 + 2'",
        Parameters: []core.Parameter{
            {Name: "expression", Type: "string", Required: true},
        },
        Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
            expr := args["expression"].(string)
            result := evaluate(expr)
            return core.NewSuccessResponse(fmt.Sprintf("Result: %s", result))
        },
    }
    
    // Create main agent with reasoning and tools
    mainAgent := agents.NewAgent(&agents.AgentConfig{
        LLMEngine:   llm,
        AgentName:   "MathAssistant",
        Description: "An intelligent math assistant",
        SystemPrompt: `You are a helpful math assistant.
You can solve mathematical problems using your tools and reasoning capabilities.`,
        Tools:       []llms.Tool{calcTool, expand.NewExpandTool()},
        Reasoning:   true,  // Enable reasoning sub-agent
        MainAgent:   true,
        Persistence: "json",
    })
    
    // Chat with the agent
    responseCh := mainAgent.ChatStream("What is 15 multiplied by 23?")
    
    fmt.Println("=== Agent Response ===")
    for chunk := range responseCh.Start() {
        if chunk.Content != "" {
            fmt.Print(chunk.Content)
        }
        
        if chunk.Type == llms.TypeToolExecuting && chunk.ToolExecuting != nil {
            fmt.Printf("\n[Executing: %s]\n", chunk.ToolExecuting.Name)
        }
        
        if chunk.Type == llms.TypeToolResult && len(chunk.ToolResults) > 0 {
            fmt.Printf("\n[Tool Result: %s]\n", chunk.ToolResults[0].Result)
        }
    }
}

func evaluate(expr string) string {
    // Your calculation logic here
    return "345"
}
```

## Environment Variables

The framework uses the following environment variables:

- `TOGETHERAI_API_KEY` - API key for TogetherAI
- `DEEPSEEK_API_KEY` - API key for DeepSeek
- `OPENAI_API_KEY` - API key for OpenAI (if using OpenAI)

These can be set via:
1. `.env` file in your project directory
2. System environment variables

## Project Structure

```
github.com/thinktwice/agentForge/
├── src/
│   ├── agents/          # Agent implementation
│   │   ├── agent.go     # Main agent struct and methods
│   │   └── agentConfig.go
│   ├── llms/            # LLM engine implementations
│   │   ├── factory.go   # LLM factory functions
│   │   └── openai.go    # OpenAI-compatible client
│   ├── core/            # Core interfaces and implementations
│   │   ├── tool.go      # Universal tool implementation
│   │   ├── agentContext.go  # Agent context system
│   │   ├── response.go  # Response channel management
│   │   └── interfaces.go # Core interfaces (SubAgent, etc.)
│   ├── tools/           # Tool implementations
│   │   ├── delegate/     # Delegation tool
│   │   ├── expand/       # Expand tool for discovery
│   │   ├── meta/        # Meta tool for introspection
│   │   └── fs/          # File system tools
│   ├── persistence/     # Conversation persistence
│   └── interfaces.go    # Core interfaces
└── examples/            # Example implementations
```

## API Reference

### Core Types

- `agents.Agent` - Main agent type
- `agents.AgentConfig` - Configuration for creating agents
- `agents.AgentHooks` - Hook system for lifecycle events
- `core.Tool` - Universal tool implementation
- `core.AgentContext` - Agent context system
- `core.SubAgent` - Interface for sub-agents
- `core.ResponseCh` - Response channel for streaming
- `llms.LLMEngine` - Interface for LLM providers
- `llms.Tool` - Interface for tools
- `llms.ChunkResponse` - Base streaming response chunk
- `core.ExtendedChunkResponse` - Extended chunk with agent metadata

### Key Interfaces

```go
type Tool interface {
    GetName() string
    Call(agentContext map[string]any, args map[string]any) ToolReturn
    GetFunctionDefinition() FunctionDefinition
}

type ToolReturn interface {
    Success() bool
    Error() string
    Data() string
}

type Discoverable interface {
    BasicDescription() string
    AdvanceDescription() string
    Troubleshooting() string
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

[Add your license here]

## Support

For questions, issues, or feature requests, please open an issue on GitHub.

