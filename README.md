<div align="center">
  <img src="assets/agent_forge_logo.png" alt="Agent Forge Logo" width="400"/>
</div>

A powerful Go framework for building intelligent agents with LLM integration, tool execution, and multi-agent collaboration.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Development](#development)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
  - [Creating Agents](#creating-agents)
  - [Creating Tools](#creating-tools)
  - [Multi-Agent Teams](#multi-agent-teams)
- [System Agents](#system-agents)
  - [Reasoning Agent](#reasoning-agent)
  - [OS Agent](#os-agent)
  - [Git Agent](#git-agent)
  - [Coding Agent](#coding-agent)
  - [Web Agent](#web-agent)
  - [Vector Agent](#vector-agent)
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
- 🌐 **Web Automation** - Navigate, interact with, and extract content from web pages using headless browser
- 📊 **Session Management** - Automatic browser session cleanup and resource management

## Installation

```bash
go get github.com/thinktwice/agentForge
```

## Development

### Running Tests

Run unit tests using the test script:

```bash
./scripts/test.sh --unit
```

This runs all unit tests with race detection and coverage reporting.

### Linting

Run the linter using the lint script:

```bash
./scripts/lint.sh
```

**Note:** You'll need to install `golangci-lint` first. See installation instructions below.

#### Installing golangci-lint

**Using the official installation script (recommended):**
```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin latest
```

Make sure `$(go env GOPATH)/bin` is in your PATH:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

**Using Go install:**
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

**Using package managers:**
- macOS: `brew install golangci-lint`
- Linux (snap): `sudo snap install golangci-lint`

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
    multiModelLLM, err := llms.NewOpenAILLMBuilder("togetherai").
        SetModel(llms.TOGETHERAI_Llama3170BInstructTurbo).
        SetCtx(ctx).
        Build()
    if err != nil {
        panic(err)
    }
    llm := multiModelLLM.MainModel()
    
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
    Tone:              "keep-it-short",  // Optional: Response tone
    Trace:             "response",        // Optional: Trace identifier
    CanExpand:         true,              // Optional: Enable tool/agent expansion
    SubAgents:         []*core.SubAgent{}, // Optional: Sub-agents for delegation
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

System agents are pre-defined specialized agents that can be added to your main agent. They provide common functionality like reasoning analysis, OS operations, Git operations, coding assistance, web automation, and vector database operations.

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

### Web Agent

The Web agent provides web navigation, automation, and content extraction capabilities using a headless browser.

```go
// Create and add Web agent with working directory
webAgent := agents.WebAgent(llm, "/path/to/working/dir")
agent.AddSystemAgent(webAgent)
```

**Use cases:**
- Web navigation and page interaction
- Form filling and clicking elements
- Screenshot capture
- Content extraction and saving
- JavaScript execution
- Browser history navigation

**Available actions:**
- `navigate`: Navigate to a URL (automatically adds https:// if scheme is missing)
- `click`: Click elements by CSS selector (with optional wait for visibility)
- `type`: Type text into input fields (with optional field clearing)
- `screenshot`: Capture page or element screenshots (saves to temp directory if path not provided)
- `get_content`: Extract page content as HTML, plain text, or title
- `save_content`: Save page content to file in `workingDir/web` directory (preferred when vector indexing is available)
- `wait`: Wait for elements to appear or pages to load (configurable timeout)
- `back`/`forward`: Navigate browser history
- `evaluate`: Execute JavaScript code and return results
- `close`: Close browser session and free resources

**Note:** 
- The Web agent maintains browser context across tool calls, preserving cookies, session, and history
- Browser sessions are automatically cleaned up after 5 minutes of inactivity
- When a vector/indexing system is available (check for "system-vector" in sub-agents), use `save_content` instead of `get_content` to enable the workflow: Save → Index → Semantic Search
- After saving content with `save_content`, the main agent can delegate to the vector agent to index it for semantic search capabilities

### Git Agent

The Git agent provides Git repository operations and version control capabilities.

```go
// Create and add Git agent with repository root
gitAgent := agents.GitAgent(llm, "/path/to/repo")
agent.AddSystemAgent(gitAgent)
```

**Use cases:**
- Git repository operations (add, commit, push, pull)
- Branch management
- Viewing git status and logs
- Managing version control

### Coding Agent

The coding agent specializes in code generation, analysis, and manipulation within a codebase.

```go
// Create and add coding agent with codebase root
codingAgent := agents.CodingAgent(codingLLM, "/path/to/codebase")
agent.AddSystemAgent(codingAgent)
```

**Use cases:**
- Code generation and modification
- Code analysis and review
- Refactoring assistance
- Codebase navigation

**Note:** The coding agent can use a different LLM engine optimized for code tasks.

### Vector Agent

The vector agent provides semantic search and document indexing capabilities using vector databases.

```go
// Create and add vector agent with vector DB and embedding generator
vectorAgent := agents.VectorAgent(llm, vectorDB, embeddingGenerator)
agent.AddSystemAgent(vectorAgent)
```

**Use cases:**
- Semantic document search
- Document indexing and storage
- Knowledge base queries
- Similarity searches

**Note:** Requires a vector database (e.g., Milvus) and an embedding generator (e.g., OpenAI embeddings) to be initialized separately.

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

Hooks are registered through plugins. Plugins can register hooks for various lifecycle events:

```go
// Hooks are registered via plugins, not directly on agents
// See the Plugins section for more information on creating custom plugins
```

Available hook events include:
- `EventAgentInitialization` - Before agent initialization
- `EventAgentInitialized` - After agent initialization
- `EventBeforeToolExecution` - Before tool execution
- `EventToolExecution` - After tool execution
- `EventNewUserMessage` - When a new user message is received
- `EventNewAssistantMessage` - When assistant generates a message
- `EventNewChunk` - For each streaming chunk
- And more (see `core.Events` for complete list)

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
    multiModelLLM, err := llms.NewOpenAILLMBuilder("togetherai").
        SetModel(llms.TOGETHERAI_Llama3170BInstructTurbo).
        SetCtx(ctx).
        Build()
    if err != nil {
        panic(err)
    }
    llm := multiModelLLM.MainModel()
    
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

- `AF_TOGETHERAI_API_KEY` - API key for TogetherAI
- `AF_DEEPSEEK_API_KEY` - API key for DeepSeek
- `AF_OPENAI_API_KEY` - API key for OpenAI
- `AF_LOG_LEVEL` - Logging level (DEBUG, INFO, WARN, ERROR). Default: INFO

Set via `.env` file or system environment variables. Environment variables take precedence over `.env` file values.

## Project Structure

```
src/
├── agents/          # Agent implementation
├── llms/            # LLM engine implementations
├── core/            # Core interfaces and implementations
├── tools/           # Tool implementations
├── persistence/     # Conversation persistence
├── integrations/    # External integrations (Milvus, embeddings)
└── plugins/         # Plugin system
    ├── logger/      # Logger plugin
    └── todo/        # Todo plugin
```

## Python SDK Usage

The project includes a Python SDK for interacting with the AgentForge server.

- **Documentation**: [Python SDK README](python-sdk/README.md)
- **Build Script**: Use `scripts/build-python-sdk.sh` to compile the required Go server binaries for the SDK.


## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

[Add your license here]
