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

### Creating Agents

There are three ways to instantiate an agent:

| Method | Package | Returns |
|--------|---------|---------|
| `agents.NewAgent(config)` | agents | `*Agent` (panics on invalid config) |
| `agents.NewBuilder(llm, name).Build()` | agents | `(*Agent, error)` |
| `builder.NewAgentBuilder(name, persistence)` + `.Build()` | builder | `(*Agent, error)` |
| `builder.NewAgentBuilderFromConfig(path)` + `.Build()` | builder | `(*Agent, error)` |
| `builder.NewAgentBuilderFromConfigStruct(cfg)` + `.Build()` | builder | `(*Agent, error)` |

**Required fields for direct instantiation:** `LLMEngine`, `AgentName`.

See the [Agent Builder Guide](docs/AGENT_BUILDER.md) for comprehensive documentation.

**Using agents.Builder (fluent API):**

```go
agent, err := agents.NewBuilder(llm, "my-agent").
    WithDescription("Agent description").
    WithSystemPrompt("You are...").
    WithTools(tool1, tool2).
    AsMainAgent().
    WithPersistence("json").
    WithContextWindow(128000).        // Enables history truncation
    WithTracer(telemetry.NewLogTracer()). // Optional: Observability
    Build()
if err != nil {
    log.Fatal(err)
}

// Add system agents after creation
reasoningAgent := agents.ReasoningAgent(llm)
agent.AddSystemAgent(reasoningAgent)
```

**Using builder from YAML config:**
```go
import "github.com/thinktwiceco/agent-forge/src/builder"

agentBuilder, err := builder.NewAgentBuilderFromConfig("config.yaml")
if err != nil {
    log.Fatal(err)
}
agent, err := agentBuilder.Build()
```

**Using builder from code (programmatic):**
```go
b := builder.NewAgentBuilder("My Agent", "json")
b.SetSystemPrompt("You are a helpful assistant.")
b.SetModel("deepseek::deepseek-chat")
b.SetWorkingDir("/path/to/working/dir")
b.AddTools(builder.Tool{Name: builder.FILE_SYSTEM_TOOL}, builder.Tool{Name: builder.GIT_TOOL})
b.AddSubagent(builder.REASONING_AGENT, "deepseek::deepseek-reasoner")
b.AddPlugin("logger")
agent, err := b.Build()
```
See [Plugins](src/plugins/README.md) for available plugins.

**Using AgentConfig (direct, advanced):**

```go
agent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine:         llm,              // Required: LLM engine
    AgentName:         "my-agent",        // Required: Agent name
    Description:       "Agent description", // Optional
    SystemPrompt:      "You are...",      // Optional
    Tools:             []llms.Tool{},     // Optional: Available tools
    MainAgent:         true,              // Optional: Is this the main agent?
    Persistence:       "json",            // Optional: Enable conversation history
    Tone:              "keep-it-short",  // Optional: Response tone
    Trace:             "response",        // Optional: Trace identifier
    SubAgents:         []core.SubAgent{}, // Optional: Sub-agents for delegation
    MaxContextTokens:   128000,          // Optional: Context window size
    TruncationStrategy: strategy,        // Optional: Custom truncation strategy
    Tracer:            tracer,           // Optional: Telemetry tracer
})
```

### Creating Tools

Tools extend agent capabilities:

```go
import "github.com/thinktwiceco/agent-forge/src/core"

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

### How to Add Tools

**AgentConfig (direct):**
```go
agent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine: llm,
    AgentName: "my-agent",
    Tools:     []llms.Tool{fsTool, gitTool, customTool},
})
```

**agents.Builder (fluent API):**
```go
agent, _ := agents.NewBuilder(llm, "my-agent").
    WithTools(tool1, tool2, tool3).
    Build()
```

**builder.AgentBuilder (code):**
```go
b := builder.NewAgentBuilder("My Agent", "json")
b.SetWorkingDir("/path/to/working/dir")
b.AddTools(
    builder.Tool{Name: builder.FILE_SYSTEM_TOOL},
    builder.Tool{Name: builder.GIT_TOOL},
    builder.Tool{Name: builder.WEB_BROWSER_TOOL},
    builder.Tool{Name: builder.POSTGRES_TOOL, PostgresURL: "...", Mode: "read", AllowedTables: []string{"users"}},
)
agent, _ := b.Build()
```

**YAML config:**
```yaml
agent:
  working_dir: "/path"
  tools:
    - name: fs
    - name: web
    - name: git
    - name: postgres
      postgresURL: "postgresql://..."
      mode: "read"
      allowedTables: ["users"]
```

**After creation:**
```go
agent.AddTools([]llms.Tool{customTool})
```

Built-in tool names: `fs`, `web`, `vector`, `git`, `postgres`, `api`.

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

### Paths & Working Directory

Every tool and plugin path is relative to the agent's `working_dir`. Set `working_dir` in agent config (YAML or builder) or via `b.SetWorkingDir()`.

```
working_dir/
├── data/
│   └── conversations/   ← Persistence (json): conversation history
├── repos/               ← Git tool (cloned repos)
├── web/                 ← Web tool (saved content, browser data)
├── vault/               ← Vault plugin (encrypted secrets)
└── procedures/          ← Procedures plugin (procedure manifests)
```

| Component | Path | Notes |
|-----------|------|-------|
| fs tool | `working_dir` | Root for file operations |
| git tool | `working_dir/repos` | Repos are cloned here |
| web tool | `working_dir/web` | Saved content, browser data |
| vault plugin | `working_dir/vault` | Encrypted secrets |
| procedures plugin | `working_dir/procedures` | Procedure manifests |
| persistence (json) | `working_dir/data/conversations/{agentName}` | Conversation history |

When `working_dir` is not set, persistence uses `data/conversations/{agentName}` relative to the process CWD.

## System Agents

System agents are pre-defined specialized agents that can be added to your main agent. They provide common functionality like reasoning analysis, OS operations, Git operations, coding assistance, web automation, and vector database operations.

### Pre-built Subagents

| Subagent | Constructor | Args | Purpose |
|----------|-------------|------|---------|
| **reasoning** | `agents.ReasoningAgent(llm)` | `llm` | Analyzes questions, finds ambiguities, guides responses |
| **os** | `agents.OsAgent(llm, root)` | `llm`, `root string` | File system and OS tasks within restricted root |
| **git** | `agents.GitAgent(llm, workingDir)` | `llm`, `workingDir string` | Git operations (status, diff, add, commit, etc.) |
| **coding** | `agents.CodingAgent(llm, root)` | `llm`, `root string` | Code generation, analysis, refactoring |
| **web** | `agents.WebAgent(llm, workingDir)` | `llm`, `workingDir string` | Web navigation, automation, content extraction |
| **vector** | `agents.VectorAgent(llm, vectorDB, embeddingGenerator)` | `llm`, `vectorDB`, `embeddingGenerator` | Semantic search and document indexing |

### How to Add Subagents

**Programmatic (agents package):**
```go
agent := agents.NewAgent(&agents.AgentConfig{...})
// or: agent, _ := agents.NewBuilder(llm, "main").Build()

agent.AddSystemAgent(agents.ReasoningAgent(llm))
agent.AddSystemAgent(agents.OsAgent(llm, "/path/to/root"))
agent.AddSystemAgent(agents.GitAgent(llm, "/path/to/repos"))
agent.AddSystemAgent(agents.WebAgent(llm, "/path/to/working/dir"))
agent.AddSystemAgent(agents.VectorAgent(llm, vectorDB, embeddingGenerator))
```

**Via builder (code):**
```go
b := builder.NewAgentBuilder("My Agent", "json")
b.SetWorkingDir("/path/to/working/dir")
b.AddSubagent(builder.REASONING_AGENT, "deepseek::deepseek-reasoner")
b.AddSubagent(builder.OS_AGENT, "deepseek::deepseek-chat")
b.AddSubagent(builder.GIT_AGENT, "deepseek::deepseek-chat")
b.AddSubagent(builder.WEB_AGENT, "deepseek::deepseek-chat")
b.AddSubagent(builder.VECTOR_DB_AGENT, "togetherai::model")  // requires vectorDB + embeddingGenerator
agent, _ := b.Build()
```

**Via YAML config:**
```yaml
agent:
  working_dir: "/path/to/working/dir"
  subagents:
    reasoning: "deepseek::deepseek-reasoner"
    web: "deepseek::deepseek-chat"
    git: "deepseek::deepseek-chat"
```

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

## Built-in Tools

Agent Forge includes several built-in tools that can be easily integrated with your agents. These tools provide common functionality for file system operations, Git operations, database access, and API interactions.

### File System Tool

The file system tool provides safe file operations within a restricted root directory.

```go
import "github.com/thinktwiceco/agent-forge/src/tools/fs"

fsTool := fs.NewFsTool("/path/to/root")
```

**Available operations:** Read, write, list, delete files and directories.

### Git Tool

The Git tool enables Git operations within a repository.

```go
import "github.com/thinktwiceco/agent-forge/src/tools/git"

gitTool := git.NewGitTool("/path/to/repo")
```

**Available operations:** Status, diff, log, add, commit, push, branch, reset, checkout, clone.

### Postgres Tool

The Postgres tool provides secure database operations with table whitelisting.

```go
import "github.com/thinktwiceco/agent-forge/src/tools/postgres"

pgTool := postgres.NewPostgresTool(
    "postgresql://user:pass@localhost:5432/db",
    "read",  // or "write" for INSERT/UPDATE/DELETE
    []string{"users", "products"},  // allowed tables
    []string{"public"},              // allowed schemas
)
```

**Modes:** `read` (SELECT only), `write` (full DML access).

### API Tool

The API tool enables agents to make HTTP API calls to configured endpoints with authentication and validation support.

```go
import "github.com/thinktwiceco/agent-forge/src/tools/api"

// Define endpoints
endpoints := []api.Endpoint{
    {
        Name:          "get_user",
        URL:           "https://api.example.com/users/{user_id}",
        Method:        "GET",
        Description:   "Get user by ID",
        URLParameters: `- user_id: string - The user ID`,
    },
    {
        Name:        "create_post",
        URL:         "https://api.example.com/posts",
        Method:      "POST",
        Description: "Create a new post",
        Payload:     `- title: string - Post title
- content: string - Post content`,
    },
}

// Optional: Create authentication hook
authHook := func(url string, headers map[string]string, body string) (map[string]string, error) {
    headers["Authorization"] = "Bearer " + os.Getenv("API_TOKEN")
    return headers, nil
}

// Create API tool
apiTool := api.NewApiTool("my_api", endpoints, authHook)
```

**Features:**
- Dynamic endpoint discovery - Agent automatically sees all available endpoints
- URL parameter substitution (e.g., `/users/{user_id}`)
- Query parameter support (e.g., `?limit=10&offset=0`)
- Request body support for POST/PUT/PATCH
- Authentication hooks for adding auth headers without exposing secrets
- Per-endpoint validation for parameter safety
- YAML configuration support

**YAML Configuration:**

```yaml
agent:
  tools:
    - name: "api"
      endpoints:
        - name: "get_user"
          url: "https://api.example.com/users/{user_id}"
          method: "GET"
          description: "Get user by ID"
          urlParameters: |
            - user_id: string - The user ID
          validator: "validate_positive_id"  # Optional
      onApiCallHook: "add_auth_token"  # Optional
```

**Parameter Validation:**

Register validators to ensure parameters are safe before making API calls:

```go
// Register validators
api.RegisterValidator("validate_positive_id", 
    api.ValidatePositiveIntParam("user_id"))

api.RegisterValidator("validate_body_size",
    api.ValidateBodyMaxSize(10000)) // 10KB limit

// Register authentication hooks
api.RegisterHook("add_auth_token", func(url string, headers map[string]string, body string) (map[string]string, error) {
    headers["Authorization"] = "Bearer " + getAPIToken()
    return headers, nil
})
```

**Built-in validators:**
- `ValidatePositiveIntParam` - Ensures parameters are positive integers
- `ValidateRequiredParams` - Ensures parameters are present and non-empty  
- `ValidateBodyMaxSize` - Limits request body size

**Examples:**
- [API Tool Documentation](src/tools/api/README.md) - Comprehensive guide
- [Pokemon API Example](examples/README.md) - Working integration with PokeAPI

### Web Browser Tool

The web browser tool provides web automation capabilities using a headless browser.

```go
import "github.com/thinktwiceco/agent-forge/src/tools/web"

webTool := web.NewWebTool("/path/to/working/dir")
```

**Available actions:** Navigate, click, fill forms, extract content, take screenshots, execute JavaScript.

### Vector Database Tool

The vector database tool enables semantic search over indexed documents.

```go
import "github.com/thinktwiceco/agent-forge/src/tools/vector"

vectorTool := vector.NewVectorTool(vectorDB, embeddingGenerator)
```

**Available actions:** Index documents, semantic search, list documents, delete documents.

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

Enable conversation history storage to maintain context across sessions. The system supports multiple concurrent conversations per agent.

#### Go SDK

```go
agent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine:   llm,
    AgentName:   "persistent-agent",
    Persistence: "json",  // Stores history as JSON files in data/conversations/
})

// Start a new conversation (pass empty string as chatId)
// Returns the responses and the generated chatId
responseCh := agent.ChatStream("Hello!", "")
chatId := responseCh.ChatId() // Save this for later!

// Continue an existing conversation
// Pass the chatId to resume context
responseCh = agent.ChatStream("Continue...", chatId)
```

#### REST API

**Start New Conversation:**
```bash
POST /api/server/{agentName}/chat
{
    "message": "Hello!"
}
# Response includes "chatId" in every chunk
```

**Resume Conversation:**
```bash
POST /api/server/{agentName}/chat?conversationId={uuid}
{
    "message": "Continue..."
}
```

### Hook System & Observability

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

### Telemetry & Observability

Built-in telemetry system for debugging and monitoring:

```go
import "github.com/thinktwiceco/agent-forge/src/telemetry"

// Log-based tracer (structured logs)
agent, err := agents.NewBuilder(llm, "assistant").
    WithTracer(telemetry.NewLogTracer()).
    Build()

// Metrics tracer (OpenTelemetry/Prometheus)
agent, err := agents.NewBuilder(llm, "assistant").
    WithTracer(telemetry.NewMetricsTracer(meter)).
    Build()
```

**Events tracked:**
- Tool execution duration and success/failure
- Token usage per LLM call
- History truncation events
- Agent lifecycle timing

See [Telemetry Documentation](src/telemetry/README.md) for integration details.

## Plugins

See [Plugins README](src/plugins/README.md) for details. Plugins extend agent functionality by providing tools, hooks, and system prompt enhancements. The plugin system uses **interface segregation** - plugins only implement the interfaces they need:

```go
// Minimal plugin (just a name)
type Plugin interface {
    Name() string
}

// Optional interfaces
type HookProvider interface {
    Plugin
    Hooks() map[Event]AgentHookFn
}

type ToolProvider interface {
    Plugin
    Tools() []llms.Tool
}

type PromptProvider interface {
    Plugin
    SystemPrompt() string
}
```

### How Plugins Are Registered

1. **Self-registration**: Each plugin calls `registry.Register(name, factory)` in its `init()` function. The factory receives the agent's `workingDir` and returns a `core.Plugin` instance.
2. **Blank import**: [src/builder/allplugins.go](src/builder/allplugins.go) imports each plugin package with `_ "github.com/.../plugins/logger"`. This triggers `init()` so plugins register when the builder is loaded.
3. **Config-driven**: Agent config (YAML or builder) lists plugin names, e.g. `plugins: ["todo", "vault"]`. At build time, the builder fetches each plugin from the registry and instantiates it with the agent's working directory.
4. **Agent initialization**: During `EventAgentInitialization`, the agent iterates over plugins and registers their hooks, tools, and system prompts.

### How to Add a New Plugin

1. **Implement the plugin** in `src/plugins/<name>/`:
   - Implement `core.Plugin` (required) and optionally `HookProvider`, `ToolProvider`, `PromptProvider`.
   - Add an `init()` that calls `registry.Register("<name>", func(workingDir string) core.Plugin { return NewXxxPlugin(workingDir) })`.
2. **Register the import** in [src/builder/allplugins.go](src/builder/allplugins.go):
   - Add `_ "github.com/thinktwiceco/agent-forge/src/plugins/<name>"`.
3. **Enable in config**: Add the plugin name to `agent.plugins` in your YAML, e.g. `plugins: ["myplugin"]`.

**Best practice**: If the plugin operates inside a folder (e.g. `vault/`, `procedures/`), auto-create that folder at initialization time. All plugin paths are relative to the agent's `working_dir`.

### Available Plugins

| Plugin | Identifier | Description |
|--------|------------|-------------|
| [Logger](src/plugins/logger/README.md) | `"logger"` | Configurable output formatting for agent responses (colors, labels, trace formatting). Hooks into `EventNewChunk`. |
| [Todo](src/plugins/todo/README.md) | `"todo"` | Task management via `todo_handler` tool. Add, update, list, and clear todos. Hooks into `EventToolExecution` for update callbacks. |
| [Procedures](src/plugins/procedures/plugin.go) | `"procedures"` | Structured multi-phase procedures. Scans `working_dir/procedures/` for procedure manifests, provides a `procedure` tool to walk through phases. Uses `EventAgentInitialized` to load procedures and inject system prompt. |
| [Vault](src/plugins/vault/plugin.go) | `"vault"` | Encrypted secret storage in `working_dir/vault/`. Tools: `saveSecret`, `listSecrets`. Injects `resolveSecret` into SessionStorage. Auto-decrypts tool args prefixed with `resolveSecret` before execution. Requires `VAULT_MASTER_KEY` env var (base64-encoded 32 bytes). |

See the [Plugins README](src/plugins/README.md) for more information on creating custom plugins.

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/thinktwiceco/agent-forge/src/agents"
    "github.com/thinktwiceco/agent-forge/src/core"
    "github.com/thinktwiceco/agent-forge/src/llms"
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
    responseCh := agent.ChatStream("What is 15 multiplied by 23?", "")
    
    for chunk := range responseCh.Start() {
        if chunk.Content != "" {
            fmt.Print(chunk.Content)
        }
    }
}
```

## Environment Variables

- `AF_TOGETHERAI_API_KEY` - API key for TogetherAI (see [Builder README](src/builder/README.md#example-models) for available models including Kimi-K2.5, Llama, Qwen, GLM-4.7)
- `AF_DEEPSEEK_API_KEY` - API key for DeepSeek
- `AF_OPENAI_API_KEY` - API key for OpenAI
- `AF_LOG_LEVEL` - Logging level (DEBUG, INFO, WARN, ERROR). Default: INFO
- `VAULT_MASTER_KEY` - Base64-encoded 32-byte key for the Vault plugin (required when using `vault` plugin). Generate with `openssl rand -base64 32`.

Set via `.env` file or system environment variables. Environment variables take precedence over `.env` file values.

## Project Structure

```
src/
├── agents/          # Agent implementation
│   ├── context/     # Context management & truncation strategies
│   ├── execution/   # Chat loop & tool execution engine
│   ├── handlers/    # System event handlers
│   ├── mocks/       # Mock implementations for testing
│   ├── prompts/     # Prompt building
│   └── system/      # System agent templates
├── core/            # Core interfaces (SubAgent, Plugin, Tool)
├── history/         # Conversation history management
├── llms/            # LLM engine implementations
├── telemetry/       # Observability (tool exec, tokens, truncation)
├── tools/           # Tool implementations
├── persistence/     # Conversation persistence
├── integrations/    # External integrations (Milvus, embeddings)
└── plugins/         # Plugin system
    ├── logger/      # Logger plugin
    ├── todo/        # Todo plugin
    ├── procedures/  # Procedures plugin
    ├── vault/       # Vault plugin (encrypted secrets)
    └── registry/    # Plugin registry (self-registration)
```

**Key Improvements:**
- **Modular Design**: Separated concerns (execution, context, prompts, handlers)
- **Interface-Based**: All major components have interfaces for testing & flexibility
- **Extensible**: Strategy patterns for truncation, plugin system, telemetry
- **Testable**: Mock implementations available for all interfaces

See [File Structure Documentation](docs/FILE_STRUCTURE.md) for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

[Add your license here]
