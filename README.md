A Go framework and application for building AI agents with LLM integration, tool execution, and optional ephemeral subagents via `spawn_subagent`.

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
- Optional **spawn_subagent** tool for synchronous ephemeral sub-tasks with a subset of tools
- Real-time streaming responses
- Multiple LLM providers (OpenAI, DeepSeek, TogetherAI, OpenRouter)
- Built-in tools (filesystem, git, web, postgres, API client, Instagram Graph, update script, Telegram dev helper, vector DB when configured)
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
├── core/         # Core interfaces (Tool, Plugin)
├── llms/         # LLM provider integrations
├── tools/        # Built-in tools (fs, git, web, postgres, api, instagram, update, telegram, …)
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
| Web | `web.NewWebTool(workingDir)` | Web automation with browser sessions; headless by default |
| Postgres | `postgres.NewPostgresTool(url, mode, tables, schemas)` | Database operations with whitelisting |
| API | `api.NewApiTool(name, endpoints, authHook)` | HTTP API calls with auth support |
| Vector | `vector.NewVectorTool(db, embeddings)` | Semantic search and indexing |

#### Ephemeral subagents (`spawn_subagent`)

There is no fixed roster of system subagents or a **delegate** tool. To run an isolated sub-task with only some tools, enable **spawn_subagent** and call it from the model:

```go
agent, err := agents.NewBuilder(llm, "main").
    WithTools(fsTool, webTool).
    WithSpawnSubagent().
    Build()
```

- **Parameters:** `prompt` (task for the child), `tools` (names from the parent’s tool list).
- The child agent always gets **meta** and **expand**; it also gets the **todo** plugin if that plugin is registered in the binary.
- The call is **synchronous**: the tool returns the subagent’s final text when done.

YAML (Localforge): set `agent.spawn_subagent: true` in `config.yaml`. See [docs/TOOLS.md](docs/TOOLS.md) for the full tool contract.

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

Plugins extend agents with tools, hooks, and system prompts. Available plugins: `logger`, `todo`, `vault`, `skills`, `brain`.
The `brain` and `skills` plugins are enabled by default; `skills` seeds a built-in `web-navigation` skill for browser tasks.

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
    responseCh := agent.ChatStream(ctx, "Hello! How can you help me?", "")
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
    
    // Create agent with tools
    agent, _ := agents.NewBuilder(llm, "MathAssistant").
        WithSystemPrompt("You are a helpful math assistant.").
        WithTools(calcTool, fs.NewFsTool("/tmp")).
        WithPersistence("json").
        AsMainAgent().
        Build()

    // Chat with streaming
    responseCh := agent.ChatStream(ctx, "What is 15 multiplied by 23?", "")
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

**Chat interface** — Real-time streaming, conversation history, and active tasks.

**Settings** — Agent identity, plugins, and API keys.

**Knowledge graph** — Brain DB (`topic` -> `conversation` long-term memory graph), filters, and visualization on `/knowledge`.

### Installation

#### Option A: Binary Release (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/thinktwiceco/agent-forge/main/scripts/install-release.sh | bash -s -- ./my-agent
cd my-agent
# Edit config.yaml (model, system_prompt, tools, plugins)
# Add API keys to .env
./start.sh
```

This creates:
- `bin/localforge` - The executable
- `config.yaml` - Agent configuration
- `.env` - API keys and secrets
- `data/` - Conversation history
- `skills/` - Local skill packages
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
AP_OPENROUTER_API_KEY=your-key
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
GET /api/config/providers    # Get push providers (Instagram, Telegram)
PUT /api/config/providers    # Update provider settings
POST /api/agent/reload       # Reload agent from config
```

**Other:**
- `GET /api/todos` - Get todos
- `POST /api/upload` - Upload files
- `GET /api/fs/list`, `GET /api/fs/read` - FS visualization
- `GET /api/knowledge/graph`, `GET /api/knowledge/stats`, `GET /api/knowledge/node/:id` - Knowledge graph
- `POST /api/webhooks/:provider` - Webhook receiver (Instagram, Telegram; set `WEBHOOK_SECRET_<PROVIDER>` for verification)
- `POST /api/webhooks/:provider/sync` - Webhook sync (SSE stream to caller)

For Telegram local dev, enable the [telegram tool](docs/TOOLS.md#telegram-tool) in agent config (`tools: [{ name: telegram, port: "8080" }]`) or set `TELEGRAM_BOT_TOKEN` / tunnel manually. To start a **new** persisted chat thread from Telegram, send **`/new_conversation`**; see [Telegram webhook threads](docs/TOOLS.md#telegram-webhook-threads) in `docs/TOOLS.md`.

#### Directory Structure

```
working_dir/
├── data/
│   ├── conversations/      # Conversation history (JSON)
│   └── telegram_thread_map.json  # Optional; Telegram chat → active thread id (see docs/TOOLS.md)
├── repos/              # Git tool cloned repos
├── web/                # Web tool saved content
├── vault/              # Vault plugin encrypted secrets
└── skills/             # Skills plugin packages
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
| `brain` | bool | No | Set `false` to disable the default brain plugin | omit (brain on) |
| `brain_plugin` | object | No | Dreaming schedule (`dream`, `dreamTime`) when brain is on | — |
| `heartbeat` | object | No | Proactive ticks; only used if `heartbeat` is in `plugins` | — |
| `plugins` | array | No | List of plugin identifiers | `[]` |

**Model Format:** `provider::model-name`

**Supported Providers and Models:**

| Provider | Identifier | Environment Variable | Models |
|----------|-----------|---------------------|--------|
| OpenAI | `openai` | `AF_OPENAI_API_KEY` | `gpt-5`, `gpt-5.1`, `gpt-5.2` |
| DeepSeek | `deepseek` | `AF_DEEPSEEK_API_KEY` | `deepseek-chat`, `deepseek-reasoner` |
| TogetherAI | `togetherai` | `AF_TOGETHERAI_API_KEY` | `meta-llama/Llama-3.2-3B-Instruct-Turbo`, `meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo`, `Qwen/Qwen2.5-7B-Instruct-Turbo`, `Qwen/Qwen3-Coder-480B-A35B-Instruct-FP8`, `openai/gpt-oss-120b`, `zai-org/GLM-4.7`, `moonshotai/Kimi-K2.5` |
| OpenRouter | `openrouter` | `AP_OPENROUTER_API_KEY` | Any [OpenRouter](https://openrouter.ai/models) model id (e.g. `openai/gpt-4o`, `openai/gpt-4o-mini`); defaults in code use `openai/gpt-4o` |

#### Tools Configuration

`agent.working_dir` is required for `fs`, `git`, `web`, `update`, and `api` (relative paths resolve under it). See [src/builder/README.md](src/builder/README.md#tools-configuration) and [docs/TOOLS.md](docs/TOOLS.md).

| Tool | Identifier | Required YAML / env | Optional YAML |
|------|-----------|---------------------|---------------|
| File System | `fs` | `working_dir` | — |
| Git | `git` | `working_dir` | — |
| Web | `web` | `working_dir` | `headless` (bool) |
| Postgres | `postgres` | `postgresURL`, `mode`, `allowedTables` | `allowedSchemas` |
| API | `api` | `working_dir`, `config_folder` | — |
| Instagram | `instagram` | `INSTAGRAM_ACCESS_TOKEN` env | — |
| Update | `update` | `working_dir` | — |
| Telegram | `telegram` | — | `port` (default `8080`; ngrok tunnels this port) |
| Vector | `vector` | `vector-storage` section in YAML | — |

**Example:**

```yaml
agent:
  working_dir: "/path/to/agent-data"
  tools:
    - name: fs
    - name: web
    - name: postgres
      postgresURL: "postgresql://user:pass@host:5432/db"
      mode: "read"
      allowedTables: ["users", "products"]
      allowedSchemas: ["public"]
```

#### Ephemeral subtasks (`spawn_subagent`)

The old YAML `subagents` map and fixed system-agent roster are **removed**. For a short-lived child agent with a subset of tools, enable **spawn_subagent** in code (`agents.NewBuilder(...).WithSpawnSubagent().Build()`). See [docs/agents/how-to-system-agents.md](docs/agents/how-to-system-agents.md).

#### Plugins

| Plugin | Identifier | Description | Configuration |
|--------|-----------|-------------|---------------|
| Logger | `logger` | Formatted output with colors | None |
| Todo | `todo` | Task management | None |
| Vault | `vault` | Encrypted secret storage | Requires `VAULT_MASTER_KEY` env var |
| Skills | `skills` | Local `SKILL.md` packages with install/delete/list support | Auto-scans `skills/` directory |
| Scheduler | `scheduler` | Scheduled jobs | None |
| Heartbeat | `heartbeat` | Proactive timed agent turns | Optional `agent.heartbeat` YAML |
| Brain | *(default)* | Long-term memory graph; opt out with `brain: false` | Optional `agent.brain_plugin` for dreaming |

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
| `AP_OPENROUTER_API_KEY` | OpenRouter API key | If using OpenRouter (`openrouter::...`) | - |
| `AF_LOG_LEVEL` | Log level | No | `INFO` |
| `AF_LOG_FILE` | Log file path | No | - |
| `VAULT_MASTER_KEY` | Vault encryption key (base64-encoded 32 bytes) | If using vault plugin | - |

Generate `VAULT_MASTER_KEY` with: `openssl rand -base64 32`

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

[Add your license here]
