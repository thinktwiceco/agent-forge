<div align="center">
  <img src="assets/agent_forge_logo.png" alt="Agent Forge Logo" width="400"/>
</div>

A Go framework and application for building AI agents with LLM integration, tool execution, and multi-agent orchestration.

**Two ways to use agent-forge:**

1. **Go Library** - Import `github.com/thinktwiceco/agent-forge` into your Go projects to build custom AI agents
2. **localforge Application** - Pre-built server with web UI for local agent orchestration

## Table of Contents

- [Features](#features)
- [Development](#development)
- [localforge Library](#localforge-library)
  - [Installation](#installation)
  - [Library Structure](#library-structure)
  - [Agents, Subagents and Tools](#agents-subagents-and-tools)
  - [Adding Tools and Plugins](#adding-tools-and-plugins)
  - [Streamed Responses](#streamed-responses)
  - [The Builders](#the-builders)
  - [Examples](#examples)
- [localforge Application](#localforge-application)
  - [What is localforge?](#what-is-localforge)
  - [Installation](#installation-1)
  - [Configuration](#configuration)
  - [Using the Application](#using-the-application)
  - [YAML Configuration Reference](#yaml-configuration-reference)
- [Contributing](#contributing)
- [License](#license)

## Features

**Library:**
- Simple agent creation with fluent API
- Extensible tool system with custom tool support
- Multi-agent teams with delegation pattern
- Real-time streaming responses
- Multiple LLM providers (OpenAI, DeepSeek, TogetherAI)
- Built-in tools (filesystem, git, web, postgres, vector DB)
- Plugin system for extending functionality
- Conversation persistence and history management

**Application:**
- HTTP/SSE API server
- Web-based chat interface
- Pre-configured agent workflows
- Multi-agent orchestration
- Conversation management
- File uploads and knowledge integration

## Development

### Prerequisites

- Go 1.21 or later
- `golangci-lint` for linting (install: `brew install golangci-lint` or see [docs](https://golangci-lint.run/welcome/install/))

### Setup

```bash
git clone https://github.com/thinktwiceco/agent-forge
cd agent-forge
go mod download
```

### Testing

```bash
./scripts/test.sh --unit
```

### Linting

```bash
./scripts/lint.sh
```

### CI/CD

CI runs on all PRs and pushes to `main`. See [.github/workflows/ci.yml](.github/workflows/ci.yml) for details.

### Releases

Releases follow semantic versioning. See [docs/RELEASE_PIPELINE.md](docs/RELEASE_PIPELINE.md) for the complete release process.

## localforge Library

### Installation

```bash
go get github.com/thinktwiceco/agent-forge
```

### Library Structure

```
src/
├── agents/       # Agent creation and execution
├── builder/      # Config-driven agent builder
├── core/         # Core interfaces (Tool, Plugin, SubAgent)
├── llms/         # LLM provider integrations
├── tools/        # Built-in tools (fs, git, web, postgres, api, vector)
├── plugins/      # Plugin system and built-in plugins
├── history/      # Conversation history management
└── telemetry/    # Observability (tool exec, tokens, truncation)
```

See [docs/FILE_STRUCTURE.md](docs/FILE_STRUCTURE.md) for details.

### Agents, Subagents and Tools

#### Creating Agents

Three approaches:

| Method | Description |
|--------|-------------|
| `agents.NewAgent(config)` | Direct instantiation with `AgentConfig` |
| `agents.NewBuilder(llm, name)` | Fluent API builder |
| `builder.NewAgentBuilderFromConfig(path)` | YAML config-driven |

**Example (fluent API):**

```go
agent, err := agents.NewBuilder(llm, "my-agent").
    WithSystemPrompt("You are a helpful assistant.").
    WithTools(tool1, tool2).
    WithPersistence("json").
    Build()
```

**Example (YAML config):**

```go
import "github.com/thinktwiceco/agent-forge/src/builder"

agentBuilder, err := builder.NewAgentBuilderFromConfig("config.yaml")
agent, err := agentBuilder.Build()
```

See [docs/AGENT_BUILDER.md](docs/AGENT_BUILDER.md) for comprehensive documentation.

#### Built-in Tools

| Tool | Factory Function | Description |
|------|-----------------|-------------|
| File System | `fs.NewFsTool(root)` | File operations within root directory |
| Git | `git.NewGitTool(repoRoot)` | Git operations (add, commit, push, etc.) |
| Web | `web.NewWebTool(workingDir)` | Web automation with headless browser |
| Postgres | `postgres.NewPostgresTool(url, mode, tables, schemas)` | Database operations with whitelisting |
| API | `api.NewApiTool(name, endpoints, authHook)` | HTTP API calls with auth support |
| Vector | `vector.NewVectorTool(db, embeddings)` | Semantic search and indexing |

#### System Agents (Subagents)

| Subagent | Constructor | Description |
|----------|------------|-------------|
| Reasoning | `agents.ReasoningAgent(llm)` | Analyzes questions, finds ambiguities |
| OS | `agents.OsAgent(llm, root)` | File system and OS tasks |
| Git | `agents.GitAgent(llm, workingDir)` | Git repository operations |
| Coding | `agents.CodingAgent(llm, root)` | Code generation and analysis |
| Web | `agents.WebAgent(llm, workingDir)` | Web navigation and automation |
| Vision | `agents.VisionAgent(llm, workingDir)` | Loads images and answers visual questions (requires vision-capable model, e.g. gpt-4o) |

Add subagents with:

```go
agent.AddSystemAgent(agents.ReasoningAgent(llm))
agent.AddSystemAgent(agents.WebAgent(llm, workingDir))
```

### Adding Tools and Plugins

#### Custom Tools

```go
import "github.com/thinktwiceco/agent-forge/src/core"

tool := &core.Tool{
    Name:        "calculate",
    Description: "Performs calculations",
    Parameters: []core.Parameter{
        {Name: "expression", Type: "string", Required: true},
    },
    Handler: func(ctx map[string]any, args map[string]any) llms.ToolReturn {
        result := evaluate(args["expression"].(string))
        return core.NewSuccessResponse(result)
    },
}

agent.AddTools([]llms.Tool{tool})
```

#### Plugins

Plugins extend agents with tools, hooks, and system prompts. Available plugins: `logger`, `todo`, `vault`, `procedures`, `knowledge`.

```yaml
agent:
  plugins:
    - "logger"
    - "todo"
```

See [src/plugins/README.md](src/plugins/README.md) for creating custom plugins.

### Streamed Responses

All agent responses stream in real-time:

```go
responseCh := agent.ChatStream("What is the capital of France?", "")

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

### The Builders

Three builder patterns for different use cases:

| Builder | Use Case | Example |
|---------|----------|---------|
| `agents.NewBuilder()` | Fluent API in code | `agents.NewBuilder(llm, "name").WithTools(...).Build()` |
| `builder.NewAgentBuilder()` | Programmatic config | `b := builder.NewAgentBuilder("name", "json"); b.SetModel(...); b.Build()` |
| `builder.NewAgentBuilderFromConfig()` | YAML-driven config | `builder.NewAgentBuilderFromConfig("config.yaml")` |

See [docs/AGENT_BUILDER.md](docs/AGENT_BUILDER.md) for full builder documentation.

### Examples

#### Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/thinktwiceco/agent-forge/src/agents"
    "github.com/thinktwiceco/agent-forge/src/llms"
)

func main() {
    ctx := context.Background()
    
    // Create LLM engine
    multiModelLLM, _ := llms.NewOpenAILLMBuilder("togetherai").
        SetModel(llms.TOGETHERAI_Llama3170BInstructTurbo).
        SetCtx(ctx).
        Build()
    llm := multiModelLLM.MainModel()
    
    // Create agent
    agent := agents.NewAgent(&agents.AgentConfig{
        LLMEngine:    llm,
        AgentName:    "Assistant",
        SystemPrompt: "You are a helpful AI assistant.",
        MainAgent:    true,
    })
    
    // Chat with streaming
    responseCh := agent.ChatStream("Hello! How can you help me?", "")
    for chunk := range responseCh.Start() {
        if chunk.Content != "" {
            fmt.Print(chunk.Content)
        }
    }
}
```

#### Complete Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/thinktwiceco/agent-forge/src/agents"
    "github.com/thinktwiceco/agent-forge/src/core"
    "github.com/thinktwiceco/agent-forge/src/llms"
    "github.com/thinktwiceco/agent-forge/src/tools/fs"
)

func main() {
    ctx := context.Background()
    
    // Initialize LLM
    multiModelLLM, _ := llms.NewOpenAILLMBuilder("togetherai").
        SetModel(llms.TOGETHERAI_Llama3170BInstructTurbo).
        SetCtx(ctx).
        Build()
    llm := multiModelLLM.MainModel()
    
    // Create custom tool
    calcTool := &core.Tool{
        Name:        "calculate",
        Description: "Performs calculations",
        Parameters: []core.Parameter{
            {Name: "expression", Type: "string", Required: true},
        },
        Handler: func(ctx map[string]any, args map[string]any) llms.ToolReturn {
            return core.NewSuccessResponse("Result: 42")
        },
    }
    
    // Create agent with tools and subagents
    agent, _ := agents.NewBuilder(llm, "MathAssistant").
        WithSystemPrompt("You are a helpful math assistant.").
        WithTools(calcTool, fs.NewFsTool("/tmp")).
        WithPersistence("json").
        AsMainAgent().
        Build()
    
    agent.AddSystemAgent(agents.ReasoningAgent(llm))
    
    // Chat with streaming
    responseCh := agent.ChatStream("What is 15 multiplied by 23?", "")
    for chunk := range responseCh.Start() {
        if chunk.Content != "" {
            fmt.Print(chunk.Content)
        }
    }
}
```

## localforge Application

### What is localforge?

`localforge` is a standalone server application that provides:

- HTTP/SSE API server for agent interactions
- Web-based chat interface
- Pre-configured agent workflows
- Conversation persistence and history
- Multi-agent orchestration
- File uploads and knowledge integration

### Installation

#### Option A: Binary Release (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/thinktwiceco/agent-forge/main/scripts/install-release.sh | bash -s -- ./my-agent
cd my-agent
# Edit config.yaml (model, system_prompt, tools, subagents)
# Add API keys to .env
./start.sh
```

This creates:
- `bin/localforge` - The executable
- `config.yaml` - Agent configuration
- `.env` - API keys and secrets
- `data/` - Conversation history
- `procedures/` - Multi-phase workflows
- `start.sh` - Convenience launcher

#### Option B: From Source

```bash
git clone https://github.com/thinktwiceco/agent-forge
cd agent-forge/cmd/localforge
go build -o localforge src/*.go
./localforge -config config.yaml -port 8080
```

### Configuration

Create a `config.yaml` file:

```yaml
agent:
  name: "My Agent"
  model: "togetherai::moonshotai/Kimi-K2.5"
  system_prompt: |
    You are a helpful assistant.
  working_dir: "${AGENT_WORKING_DIR}"
  persistence: "json"
  tools:
    - name: fs
    - name: web
  subagents:
    reasoning: "deepseek::deepseek-reasoner"
  plugins:
    - "todo"
    - "vault"
```

Create a `.env` file:

```bash
AF_OPENAI_API_KEY=your-key
AF_DEEPSEEK_API_KEY=your-key
AF_TOGETHERAI_API_KEY=your-key
AGENT_WORKING_DIR=/path/to/working/dir
```

See [src/builder/README.md](src/builder/README.md) for full configuration options.

### Using the Application

#### Web UI

Navigate to `http://localhost:8080` after starting the server.

Features:
- Real-time chat with streaming responses
- Conversation history and management
- Settings page (`/settings`) - Agent config, providers, tools
- Knowledge page (`/knowledge`) - Knowledge graph visualization
- File uploads
- Todo list management

#### API Endpoints

**Chat:**
```bash
# Start new conversation
POST /api/chat
{"message": "Hello!"}

# Resume conversation
POST /api/chat?conversationId={uuid}
{"message": "Continue..."}

POST /api/chat/stop     # Stop in-progress chat
GET  /api/chat/push    # SSE push for real-time updates
```

**Conversations:**
```bash
GET    /api/conversations           # List conversations
GET    /api/conversations/:id       # Get conversation
DELETE /api/conversations/:id       # Delete conversation
PUT    /api/conversations/:id/title # Rename conversation
```

**Configuration:**
```bash
GET /api/config              # Get agent configuration
PUT /api/config              # Update agent config
PUT /api/config/tools/:name  # Update tool config
PUT /api/config/plugins      # Update plugins list
PUT /api/config/subagents    # Update subagents
GET /api/config/providers    # Get push providers (Instagram, Telegram)
PUT /api/config/providers    # Update provider settings
POST /api/agent/reload       # Reload agent from config
```

**Other:**
- `GET /api/todos` - Get todos
- `POST /api/upload` - Upload files
- `GET /api/fs/list`, `GET /api/fs/read` - FS visualization
- `GET /api/knowledge/graph`, `GET /api/knowledge/stats`, `GET /api/knowledge/node/:id` - Knowledge graph
- `POST /api/webhooks/:provider` - Webhook receiver (Instagram, Telegram)
- `POST /api/webhooks/:provider/sync` - Webhook sync

#### Directory Structure

```
working_dir/
├── data/
│   └── conversations/  # Conversation history (JSON)
├── repos/              # Git tool cloned repos
├── web/                # Web tool saved content
├── vault/              # Vault plugin encrypted secrets
└── procedures/         # Procedures plugin manifests
```

### YAML Configuration Reference

#### Agent Configuration

| Field | Type | Required | Description | Default |
|-------|------|----------|-------------|---------|
| `name` | string | Yes | Agent name | - |
| `model` | string | Yes | LLM model (`provider::model`) | - |
| `system_prompt` | string | No | System prompt | - |
| `working_dir` | string | No | Working directory for tools/plugins | - |
| `persistence` | string | No | Conversation persistence | `""` (none) |
| `tools` | array | No | List of tools to enable | `[]` |
| `subagents` | map | No | Map of subagent name to model | `{}` |
| `plugins` | array | No | List of plugin identifiers | `[]` |

**Model Format:** `provider::model-name`

**Supported Providers and Models:**

| Provider | Identifier | Environment Variable | Models |
|----------|-----------|---------------------|--------|
| OpenAI | `openai` | `AF_OPENAI_API_KEY` | `gpt-5`, `gpt-5.1`, `gpt-5.2` |
| DeepSeek | `deepseek` | `AF_DEEPSEEK_API_KEY` | `deepseek-chat`, `deepseek-reasoner` |
| TogetherAI | `togetherai` | `AF_TOGETHERAI_API_KEY` | `meta-llama/Llama-3.2-3B-Instruct-Turbo`, `meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo`, `Qwen/Qwen2.5-7B-Instruct-Turbo`, `Qwen/Qwen3-Coder-480B-A35B-Instruct-FP8`, `openai/gpt-oss-120b`, `zai-org/GLM-4.7`, `moonshotai/Kimi-K2.5` |

#### Tools Configuration

| Tool | Identifier | Required Parameters | Optional Parameters |
|------|-----------|-------------------|-------------------|
| File System | `fs` | `root` (string) | - |
| Git | `git` | `root` (string) | - |
| Web | `web` | `root` (string) | - |
| Postgres | `postgres` | `postgresURL` (string), `mode` (string), `allowedTables` (array) | `allowedSchemas` (array) |
| API | `api` | `endpoints` (array) | `onApiCallHook` (string) |
| Vector | `vector` | (requires vector-storage section) | - |

**Example:**

```yaml
tools:
  - name: fs
    root: "/path/to/sandbox"
  - name: postgres
    postgresURL: "postgresql://user:pass@host:5432/db"
    mode: "read"  # or "write"
    allowedTables: ["users", "products"]
    allowedSchemas: ["public"]
```

#### Subagents Configuration

| Subagent | Identifier | Requirements | Description |
|----------|-----------|--------------|-------------|
| Reasoning | `reasoning` | Model spec | Analyzes questions, finds ambiguities |
| OS | `os` | Model spec, working_dir | File system and OS operations |
| Git | `git` | Model spec, working_dir | Git repository operations |
| Web | `web` | Model spec, working_dir | Web navigation and automation |
| Coding | `coding` | Model spec, working_dir | Code generation and analysis |
| Vision | `vision` | Model spec (vision-capable, e.g. gpt-4o), working_dir | Loads images and answers visual questions |

**Example:**

```yaml
subagents:
  reasoning: "deepseek::deepseek-reasoner"
  web: "deepseek::deepseek-chat"
  git: "deepseek::deepseek-chat"
```

#### Plugins

| Plugin | Identifier | Description | Configuration |
|--------|-----------|-------------|---------------|
| Logger | `logger` | Formatted output with colors | None |
| Todo | `todo` | Task management | None |
| Vault | `vault` | Encrypted secret storage | Requires `VAULT_MASTER_KEY` env var |
| Procedures | `procedures` | Multi-phase workflows | Auto-scans `procedures/` directory |
| Knowledge | `knowledge` | Knowledge graph integration | None |

**Example:**

```yaml
plugins:
  - "logger"
  - "todo"
  - "vault"
```

#### Vector Storage Configuration

Required when using `vector` tool.

| Field | Type | Required | Description | Default |
|-------|------|----------|-------------|---------|
| `vector_db` | string | Yes | Database type | - |
| `embedding_model` | string | Yes | Embedding model | - |
| `sqlite.db_path` | string | If using SQLite | Path to SQLite DB file | - |
| `sqlite.vector_dim` | integer | No | Vector dimensions | `1536` |
| `milvus.host` | string | If using Milvus | Milvus server host | `localhost` |
| `milvus.port` | integer | If using Milvus | Milvus server port | `19530` |
| `milvus.vector_dim` | integer | No | Vector dimensions | `1536` |

**Example:**

```yaml
vector-storage:
  vector_db: "sqlite"
  embedding_model: "openai::text-embedding-3-small"
  sqlite:
    db_path: "./vector.db"
    vector_dim: 1536
```

#### Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `AF_OPENAI_API_KEY` | OpenAI API key | If using OpenAI | - |
| `AF_DEEPSEEK_API_KEY` | DeepSeek API key | If using DeepSeek | - |
| `AF_TOGETHERAI_API_KEY` | TogetherAI API key | If using TogetherAI | - |
| `AF_LOG_LEVEL` | Log level | No | `INFO` |
| `AF_LOG_FILE` | Log file path | No | - |
| `VAULT_MASTER_KEY` | Vault encryption key (base64-encoded 32 bytes) | If using vault plugin | - |

Generate `VAULT_MASTER_KEY` with: `openssl rand -base64 32`

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

[Add your license here]
