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
    agent := agents.NewAgent(agents.AgentConfig{
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
agent := agents.NewAgent(agents.AgentConfig{
    LLMEngine:         llm,                    // Required: LLM engine
    AgentName:         "my-agent",             // Required: Agent name
    Description:       "Basic description",    // Optional: Short description
    SystemPrompt:      "You are...",           // Optional: Custom system prompt
    Tools:             []llms.Tool{},          // Optional: Available tools
    MaxToolIterations: 10,                     // Optional: Max tool execution loops (default: 10)
    MainAgent:         true,                   // Optional: Is this the main agent?
    Persistence:       "json",                 // Optional: Enable conversation history
})
```

### Streaming Responses

All agent responses are streamed in real-time:

```go
responseCh := agent.ChatStream("What is the capital of France?")

for chunk := range responseCh.Start() {
    switch chunk.Type {
    case llms.TypeContent:
        // Content being streamed
        fmt.Print(chunk.Content)
        
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

Tools extend agent capabilities. There are two types of tools:

### 1. Simple Tools

Simple tools receive only the validated arguments:

```go
import "github.com/thinktwice/agentForge/src/tools"

calculatorTool := tools.NewSimpleTool(
    "calculate",                                    // Tool name
    "Performs mathematical calculations",           // Basic description
    `Advanced Details:                              // Advanced description
- Supports: +, -, *, /
- Returns numeric result`,
    `Troubleshooting:                               // Troubleshooting info
- Division by zero returns error
- Use proper syntax: "2 + 2"`,
    []tools.Parameter{                              // Parameters
        {
            Name:        "expression",
            Type:        "string",
            Description: "Mathematical expression to evaluate",
            Required:    true,
        },
    },
    func(args map[string]any) llms.ToolReturn {     // Handler function
        expression := args["expression"].(string)
        
        // Perform calculation
        result := calculate(expression)
        
        return tools.NewSuccessResponse(result)
    },
)
```

### 2. Context-Aware Tools

Context-aware tools receive both agent context and arguments:

```go
loggerTool := tools.NewContextAwareTool(
    "log_message",
    "Logs a message with agent context",
    "Logs messages with agent name and trace information",
    "Check file permissions if logging fails",
    []tools.Parameter{
        {Name: "message", Type: "string", Required: true},
        {Name: "level", Type: "string", Required: false},
    },
    func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
        // Access agent context
        agentName := agentContext["agentName"].(string)
        message := args["message"].(string)
        
        // Log with context
        log.Printf("[%s] %s", agentName, message)
        
        return tools.NewSuccessResponse("Logged successfully")
    },
)
```

### Parameter Types

Supported parameter types with automatic validation:

```go
[]tools.Parameter{
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
return tools.NewSuccessResponse("Operation completed successfully")

// Error response (no data)
return tools.NewErrorResponse("Invalid parameter: count must be positive")

// Failure response (with partial data)
return tools.NewFailureResponse("Timeout occurred", "Partial result: 42")
```

### Adding Tools to Agents

```go
agent := agents.NewAgent(agents.AgentConfig{
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
// Create specialized sub-agents
reasoningAgent := agents.NewAgent(agents.AgentConfig{
    LLMEngine:   llm,
    AgentName:   "reasoning-agent",
    Description: "Breaks down complex problems into logical steps",
    SystemPrompt: `You are a reasoning agent that excels at analytical thinking.
Break down complex problems systematically.`,
})

dataAgent := agents.NewAgent(agents.AgentConfig{
    LLMEngine:   llm,
    AgentName:   "data-agent",
    Description: "Analyzes and processes data",
    SystemPrompt: "You are a data analysis expert.",
    Tools: []llms.Tool{
        databaseTool,
        analyticsTool,
    },
})

// Create main agent that coordinates the team
mainAgent := agents.NewAgent(agents.AgentConfig{
    LLMEngine:   llm,
    AgentName:   "coordinator",
    Description: "Main coordinator agent",
    SystemPrompt: "You coordinate a team of specialized agents.",
    MainAgent:   true,
})

// Add sub-agents using the delegate tool
// Sub-agents are automatically available through the "delegate" tool
```

### Built-in Team Features

#### Reasoning Mode

Enable automatic reasoning capabilities:

```go
agent := agents.NewAgent(agents.AgentConfig{
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
mainAgent := agents.NewAgent(agents.AgentConfig{
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
    fmt.Printf("[%s] %s", chunk.AgentName, chunk.Content)
}
```

### Custom Sub-Agent Configuration

Use different LLM engines for different sub-agents:

```go
fastLLM, _ := llms.GetTogetherAILLM(ctx, llms.Llama323BInstructTurbo)
powerfulLLM, _ := llms.GetTogetherAILLM(ctx, llms.Llama3170BInstructTurbo)

agent := agents.NewAgent(agents.AgentConfig{
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
import "github.com/thinktwice/agentForge/src/tools"

agent := agents.NewAgent(agents.AgentConfig{
    LLMEngine: llm,
    AgentName: "explorer",
    Tools: []llms.Tool{
        tools.NewExpandTool(),  // Enables progressive discovery
    },
})

// The agent can now query detailed information about tools and sub-agents
// Example: expand(subject_type="agent", subject_name="reasoning-agent", troubleshoot=true)
```

The Expand tool provides three levels of information:
- **BasicDescription**: Short one-line description (always visible in system prompt)
- **AdvanceDescription**: Detailed capabilities and usage patterns
- **Troubleshooting**: Common issues and debugging tips

## Advanced Features

### Conversation Persistence

Store and retrieve conversation history:

```go
agent := agents.NewAgent(agents.AgentConfig{
    LLMEngine:   llm,
    AgentName:   "persistent-agent",
    Persistence: "json",  // Stores history as JSON files
})

// History is automatically saved and loaded
// Each agent gets its own history file based on AgentName
```

### Tool Execution Context

Pass custom context to all tools:

```go
agent := agents.NewAgent(agents.AgentConfig{
    LLMEngine: llm,
    AgentName: "contextual-agent",
    ToolExecutionContext: map[string]any{
        "database": dbConnection,
        "user_id":  userID,
        "config":   appConfig,
    },
})

// Tools can access this context
customTool := tools.NewContextAwareTool(
    "query_db",
    "Queries the database",
    "...",
    "...",
    []tools.Parameter{{Name: "query", Type: "string", Required: true}},
    func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
        db := agentContext["database"].(*sql.DB)
        userID := agentContext["user_id"].(string)
        // Use context in tool execution
        return tools.NewSuccessResponse(result)
    },
)
```

### Discoverable Interface

Make your tools and agents discoverable:

```go
// Tools created with NewSimpleTool or NewContextAwareTool automatically
// implement the Discoverable interface

// For custom implementations:
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
    "github.com/thinktwice/agentForge/src/llms"
    "github.com/thinktwice/agentForge/src/tools"
)

func main() {
    ctx := context.Background()
    
    // Initialize LLM
    llm, err := llms.GetTogetherAILLM(ctx, llms.Llama3170BInstructTurbo)
    if err != nil {
        panic(err)
    }
    
    // Create a calculator tool
    calcTool := tools.NewSimpleTool(
        "calculate",
        "Performs mathematical calculations",
        "Supports +, -, *, / operations",
        "Use proper syntax: '2 + 2'",
        []tools.Parameter{
            {Name: "expression", Type: "string", Required: true},
        },
        func(args map[string]any) llms.ToolReturn {
            expr := args["expression"].(string)
            result := evaluate(expr)
            return tools.NewSuccessResponse(fmt.Sprintf("Result: %s", result))
        },
    )
    
    // Create main agent with reasoning and tools
    mainAgent := agents.NewAgent(agents.AgentConfig{
        LLMEngine:   llm,
        AgentName:   "MathAssistant",
        Description: "An intelligent math assistant",
        SystemPrompt: `You are a helpful math assistant.
You can solve mathematical problems using your tools and reasoning capabilities.`,
        Tools:       []llms.Tool{calcTool, tools.NewExpandTool()},
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
│   ├── tools/           # Tool system
│   │   ├── baseTool.go  # Tool base implementation
│   │   ├── expandTool.go
│   │   └── delegateTool.go
│   ├── persistence/     # Conversation persistence
│   └── interfaces.go    # Core interfaces
└── examples/            # Example implementations
```

## API Reference

### Core Types

- `agents.Agent` - Main agent type
- `agents.AgentConfig` - Configuration for creating agents
- `llms.LLMEngine` - Interface for LLM providers
- `llms.Tool` - Interface for tools
- `llms.ChunkResponse` - Streaming response chunk

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

