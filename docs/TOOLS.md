# Built-in Tools Guide

Agent Forge includes several built-in tools that extend agent capabilities. This guide provides detailed documentation for each tool.

## Table of Contents

- [Overview](#overview)
- [File System Tool](#file-system-tool)
- [Git Tool](#git-tool)
- [Postgres Tool](#postgres-tool)
- [API Tool](#api-tool)
- [Web Browser Tool](#web-browser-tool)
- [Update Tool](#update-tool)
- [Image Tool](#image-tool)
- [Instagram Tool](#instagram-tool)
- [Telegram Tool](#telegram-tool)
- [Vector Database Tool](#vector-database-tool)
- [Creating Custom Tools](#creating-custom-tools)

## Overview

Tools are the primary way agents interact with external systems. Each tool:
- Implements the `llms.Tool` interface
- Provides parameter validation
- Returns structured responses
- Can be configured via code or YAML

**Documented in this guide:** filesystem (`fs`), git, postgres, API client, web browser, update script, image loader, Instagram Graph, Telegram dev helper, vector DB (when configured). The framework may also inject **meta**, **expand**, and **spawn_subagent** (async — result via follow-up turn); see [docs/agents/how-to-system-agents.md](agents/how-to-system-agents.md).

## File System Tool

The file system tool provides secure file operations within a restricted root directory.

### Basic Usage

```go
import "github.com/thinktwiceco/agent-forge/src/tools/fs"

// Create tool with root directory
fsTool := fs.NewFsTool("/path/to/root")

// Add to agent
agent := agents.NewAgent(&agents.AgentConfig{
    Tools: []llms.Tool{fsTool},
})
```

### Available Operations

- **read**: Read file contents
- **write**: Write or create files
- **list**: List directory contents
- **get_file_info**: Get detailed file/directory information
- **delete**: Delete files or directories
- **get_root**: Get root directory information
- **ripgrep**: Search for patterns in files (requires ripgrep)
- **grep_logs**: Search the application log file (requires `AF_LOG_FILE` to be set; see [Logger](LOGGER.md))

### Security

- All paths are validated against the root directory
- Directory traversal attempts are blocked
- Only operates within the specified root

### YAML Configuration

```yaml
tools:
  - name: "fs"
    root: "/path/to/root"
```

## Git Tool

The Git tool enables Git operations within a repository.

### Basic Usage

```go
import "github.com/thinktwiceco/agent-forge/src/tools/git"

// Create tool for a repository
gitTool := git.NewGitTool("/path/to/repo")

// Add to agent
agent := agents.NewAgent(&agents.AgentConfig{
    Tools: []llms.Tool{gitTool},
})
```

### Available Operations

- **init**: Initialize a new repository
- **status**: Show working tree status
- **diff**: Show changes between commits
- **log**: Show commit logs
- **add**: Add files to staging area
- **commit**: Record changes to repository
- **push**: Update remote repository
- **pull**: Fetch and integrate changes
- **branch**: List, create, or delete branches
- **checkout**: Switch branches or restore files
- **clone**: Clone a repository

### YAML Configuration

```yaml
tools:
  - name: "git"
    root: "/path/to/repo"
```

## Postgres Tool

The Postgres tool provides secure database access with table whitelisting.

### Basic Usage

```go
import "github.com/thinktwiceco/agent-forge/src/tools/postgres"

// Create tool with restrictions
pgTool := postgres.NewPostgresTool(
    "postgresql://user:pass@localhost:5432/db",
    "read",                          // Mode: "read" or "write"
    []string{"users", "products"},   // Allowed tables
    []string{"public"},              // Allowed schemas
)

// Add to agent
agent := agents.NewAgent(&agents.AgentConfig{
    Tools: []llms.Tool{pgTool},
})
```

### Modes

- **read**: Only SELECT queries allowed
- **write**: SELECT, INSERT, UPDATE, DELETE allowed

### Security Features

- Table whitelist prevents access to restricted tables
- Mode-based restrictions limit operations
- Dangerous operations (DROP, TRUNCATE, etc.) always blocked
- Query validation before execution

### YAML Configuration

```yaml
tools:
  - name: "postgres"
    postgresURL: "postgresql://user:pass@localhost:5432/db"
    mode: "read"
    allowedTables:
      - "users"
      - "products"
    allowedSchemas:
      - "public"
```

## API Tool

The API tool enables HTTP API calls to configured endpoints with authentication and validation.

### Basic Usage

```go
import "github.com/thinktwiceco/agent-forge/src/tools/api"

// Define endpoints
endpoints := []api.Endpoint{
    {
        Name:        "get_user",
        URL:         "https://api.example.com/users/{user_id}",
        Method:      "GET",
        Description: "Get user by ID",
        URLParameters: `- user_id: string - The user ID`,
        QueryParams:   `- include: string - Fields to include`,
    },
    {
        Name:        "create_post",
        URL:         "https://api.example.com/posts",
        Method:      "POST",
        Description: "Create a new post",
        Payload: `- title: string - Post title
- content: string - Post content`,
    },
}

// Create authentication hook (optional)
authHook := func(url string, headers map[string]string, body string) (map[string]string, error) {
    headers["Authorization"] = "Bearer " + os.Getenv("API_TOKEN")
    return headers, nil
}

// Create tool
apiTool := api.NewApiTool("my_api", endpoints, authHook)

// Add to agent
agent := agents.NewAgent(&agents.AgentConfig{
    Tools: []llms.Tool{apiTool},
})
```

### Features

- **Dynamic Endpoint Discovery**: Agent sees all available endpoints
- **URL Parameters**: Template-based URL parameter substitution
- **Query Parameters**: Easy query string building
- **Request Bodies**: JSON payload support
- **Authentication Hooks**: Add auth headers without exposing secrets
- **Validation**: Per-endpoint parameter validation
- **YAML Configuration**: Declarative endpoint configuration

### Parameter Types

1. **URL Parameters**: Path parameters like `/users/{user_id}`
2. **Query Parameters**: Query string like `?limit=10&offset=0`
3. **Body**: JSON request body for POST/PUT/PATCH

### Authentication Hooks

Register hooks for reusable authentication patterns:

```go
// Register hook
api.RegisterHook("bearer_auth", func(url string, headers map[string]string, body string) (map[string]string, error) {
    token := os.Getenv("API_TOKEN")
    if token == "" {
        return nil, fmt.Errorf("API_TOKEN not set")
    }
    headers["Authorization"] = "Bearer " + token
    return headers, nil
})

// Use in tool
apiTool := api.NewApiTool("my_api", endpoints, api.GetHook("bearer_auth"))
```

### Validation

Register validators to ensure parameters are safe:

```go
// Register validators
api.RegisterValidator("validate_positive_id", 
    api.ValidatePositiveIntParam("user_id", "post_id"))

api.RegisterValidator("validate_required", 
    api.ValidateRequiredParams("api_key", "user_id"))

api.RegisterValidator("validate_body_size",
    api.ValidateBodyMaxSize(50000)) // 50KB limit

// Assign to endpoint
endpoints := []api.Endpoint{
    {
        Name:     "get_user",
        URL:      "https://api.example.com/users/{user_id}",
        Method:   "GET",
        Validate: api.GetValidator("validate_positive_id"),
    },
}
```

### Built-in Validators

- **ValidatePositiveIntParam**: Ensures parameters are positive integers
- **ValidateRequiredParams**: Ensures parameters are present and non-empty
- **ValidateBodyMaxSize**: Limits request body size

### YAML Configuration

```yaml
tools:
  - name: "api"
    endpoints:
      - name: "get_user"
        url: "https://api.example.com/users/{user_id}"
        method: "GET"
        description: "Get user by ID"
        urlParameters: |
          - user_id: string - The user ID
        queryParams: |
          - include: string - Fields to include
        validator: "validate_positive_id"
      
      - name: "create_post"
        url: "https://api.example.com/posts"
        method: "POST"
        description: "Create a new post"
        payload: |
          - title: string - Post title
          - content: string - Post content
        validator: "validate_body_size"
    
    onApiCallHook: "bearer_auth"
```

### Examples

See:
- [API Tool README](../src/tools/api/README.md) - Comprehensive documentation
- [Pokemon API Example](../examples/README.md) - Working Pokemon API integration

## Web Browser Tool

The web browser tool provides web automation using a browser that runs headlessly by default. Set `WEB_TOOL_HEADLESS=false` to make new sessions visible by default.

### Basic Usage

```go
import "github.com/thinktwiceco/agent-forge/src/tools/web"

// Create tool with working directory
webTool := web.NewWebTool("/path/to/working/dir")

// Add to agent
agent := agents.NewAgent(&agents.AgentConfig{
    Tools: []llms.Tool{webTool},
})
```

### Available Actions

- **navigate**: Navigate to a URL
- **click**: Click an element by CSS selector
- **fill**: Fill a form field with a visible value
- **fill_secret**: Fill a form field with a secret value (value not logged)
- **get_content**: Get page content (HTML, text, title, or interactive_tree); on detected SPAs, text mode uses an accessibility snapshot
- **get_snapshot**: Get the page accessibility tree as JSON (semantic, SPA-friendly)
- **save_content**: Save content to file in working directory
- **fetch**: HTTP GET without Chrome (static HTML converted toward markdown, or raw text for non-HTML)
- **web_search**: Search the web (Brave if `AF_BRAVE_API_KEY`, else Tavily if `AF_TAVILY_API_KEY`; optional `WEB_SEARCH_CACHE_TTL_S`)
- **upload_file**: Upload a file via a file input element
- **refresh**: Refresh the current page
- **list_sessions**: List all active browser sessions
- **close_session**: Close a browser session

### YAML Configuration

```yaml
tools:
  - name: "web"
    root: "/path/to/working/dir"
```

## Update Tool

The update tool runs the root `update-release.sh` script for an agent installation.

### Basic Usage

```go
import "github.com/thinktwiceco/agent-forge/src/tools/update"

// Create tool for an agent working directory
updateTool := update.NewUpdateTool("/path/to/agent/root")

// Add to agent
agent := agents.NewAgent(&agents.AgentConfig{
    Tools: []llms.Tool{updateTool},
})
```

### Behavior

- Resolves exactly `./update-release.sh` in the configured agent root
- Refuses to run if the script is missing or not a regular file
- Executes the script with `bash` from the agent root
- Returns captured stdout and stderr so the caller can inspect the update result

### YAML Configuration

```yaml
tools:
  - name: "update"
```

## Image Tool

The image tool loads images from disk and returns them as base64 data URIs suitable for vision-capable LLMs.

### Basic Usage

```go
import "github.com/thinktwiceco/agent-forge/src/tools/image"

// Create tool sandboxed to a directory
imageTool := image.NewImageTool("/path/to/images")

// Add to agent
agent := agents.NewAgent(&agents.AgentConfig{
    Tools: []llms.Tool{imageTool},
})
```

### Available Operations

- **load**: Read an image file and return it as a base64 data URI (`data:<mime>;base64,<data>`)

### Supported Formats

- JPEG (`.jpg`, `.jpeg`)
- PNG (`.png`)
- GIF (`.gif`)
- WebP (`.webp`)

### Security

- All paths are validated and sandboxed to the root directory
- Path traversal (`../`) is blocked

### YAML Configuration

```yaml
tools:
  - name: "image"
    root: "/path/to/images"
```

## Instagram Tool

The Instagram tool provides access to the Instagram Graph API with a flat action-based interface.

### Basic Usage

```go
import "github.com/thinktwiceco/agent-forge/src/tools/instagram"

// Create tool with authorization headers
headers := map[string]string{
    "Authorization": "Bearer " + os.Getenv("INSTAGRAM_TOKEN"),
}
instaTool := instagram.NewInstagramTool(headers)

// Add to agent
agent := agents.NewAgent(&agents.AgentConfig{
    Tools: []llms.Tool{instaTool},
})
```

### Available Actions

| Action | Description | Key Parameters |
|---|---|---|
| `get_profile` | Get the authenticated user's profile | `fields` |
| `list_media` | List recent media posts | `fields`, `limit`, `after` |
| `get_media` | Get a single media post | `media_id`, `fields` |
| `create_media_container` | Create a media container for publishing | `image_url`, `video_url`, `media_type`, `caption`, `alt_text` |
| `publish_media` | Publish a created media container | `creation_id` |
| `get_account_insights` | Get account-level insights | `metric`, `period`, `since`, `until` |
| `get_media_insights` | Get insights for a specific post | `media_id`, `metric` |
| `get_comments` | List comments on a media post | `media_id`, `fields`, `limit`, `after` |
| `reply_to_comment` | Reply to a comment | `comment_id`, `message` |

## Vector Database Tool

The vector database tool enables semantic search over indexed documents.

### Basic Usage

```go
import "github.com/thinktwiceco/agent-forge/src/tools/vector"

// Requires vector database and embedding generator
vectorTool := vector.NewVectorTool(vectorDB, embeddingGenerator)

// Add to agent
agent := agents.NewAgent(&agents.AgentConfig{
    Tools: []llms.Tool{vectorTool},
})
```

### Available Actions

- **index**: Index text content for semantic search
- **indexFile**: Index a file from the filesystem for semantic search
- **search**: Search indexed documents
- **listDocuments**: List all indexed documents
- **delete**: Delete indexed documents

### YAML Configuration

```yaml
tools:
  - name: "vector"
```

Note: Vector database and embedding generator must be configured separately at the agent builder level.

## Creating Custom Tools

To create a custom tool, use `core.NewTool` with a `core.ToolConfig`:

```go
import (
    "fmt"

    "github.com/thinktwiceco/agent-forge/src/core"
    "github.com/thinktwiceco/agent-forge/src/llms"
)

func NewCustomTool() llms.Tool {
    return core.NewTool(core.ToolConfig{
        Name:        "custom_tool",
        Description: "Brief description of the tool",
        AdvanceDesc: `Detailed description with:
- Usage instructions
- Available operations
- Examples`,
        TroubleshootingInfo: `Common issues and solutions`,
        // Optional: return per-item details when the expand tool calls DetailsAbout(item)
        DetailsAboutFunc: func(item string) string {
            return fmt.Sprintf("Details about %s: ...", item)
        },
        Parameters: []core.Parameter{
            {
                Name:        "param1",
                Type:        "string",
                Description: "Parameter description",
                Required:    true,
                Validator:   validateParam1, // Optional
            },
        },
        Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
            // Extract parameters
            param1 := args["param1"].(string)
            
            // Execute tool logic
            result, err := doSomething(param1)
            if err != nil {
                return core.NewErrorResponse(err.Error())
            }
            
            // Return response
            return core.NewSuccessResponse(result)
        },
    })
}

func validateParam1(value any) error {
    str, ok := value.(string)
    if !ok {
        return fmt.Errorf("param1 must be a string")
    }
    if len(str) == 0 {
        return fmt.Errorf("param1 cannot be empty")
    }
    return nil
}
```

### Tool Response Types

- **NewSuccessResponse(data)**: Success with data
- **NewErrorResponse(error)**: Error response
- **NewFailureResponse(error, data)**: Error with partial data
- **NewEphemeralResponse(data)**: Success, not stored in history

### Hooks

`Hooks` is an optional interface for external validation injected by the agent framework (e.g. sandboxing):

```go
type Hooks interface {
    IsSafePath(path string) bool
    IsSafeCommand(cmd string) bool
}
```

Assign a `Hooks` implementation to the tool to enforce path or command restrictions at runtime:

```go
tool := NewCustomTool()
tool.SetHooks(myHooks)
```

The framework calls `GetHooks()` before invoking the handler; the tool itself must check the hooks when applicable.

## Telegram Tool

The Telegram tool wires a [@BotFather](https://t.me/botfather) bot to Localforge: validate the token, expose Localforge with ngrok, register `POST /api/webhooks/telegram` on Telegram with a **required** shared secret, optionally rotate that secret without spawning another ngrok, and probe readiness with `health_status`.

### Prerequisites

- Localforge running (default port 8080) on the machine ngrok will tunnel to
- ngrok installed and on `PATH` when using `start_ngrok` ([download](https://ngrok.com/download))
- A bot token from [@BotFather](https://t.me/botfather)

### Configuration

```yaml
tools:
  - name: telegram
    port: "8080"   # local server port ngrok will tunnel (default: 8080)
```

Set **`WEBHOOK_SECRET_TELEGRAM`** in `.env` before using `start_ngrok` or `set_webhook` (or pass `webhook_secret`). Localforge **rejects** Telegram webhooks unless this secret is configured and Telegram sends matching `X-Telegram-Bot-Api-Secret-Token`.

### Telegram webhook threads

When **`TELEGRAM_BOT_TOKEN`** is set, Localforge registers a webhook provider that forwards Telegram messages to the agent. Conversation history uses the same JSON persistence as the UI (`agent.persistence: json` and `agent.working_dir`).

- **Default thread:** one Telegram chat maps to a stable conversation id `webhook-telegram-{chat_id}` (one JSON file under `working_dir/data/conversations/{agent_name}/`).
- **New thread:** send **`/new_conversation`** (optionally `/new_conversation@YourBot`). Localforge acknowledges, assigns a **new** conversation id, and persists the mapping in **`working_dir/data/telegram_thread_map.json`** (or `data/telegram_thread_map.json` if `working_dir` is unset). Further messages from that chat use the new file until you send **`/new_conversation`** again.
- The command must be the **only** text on the line (no extra words).

### Actions

| Action | Description | Required params | Optional params |
|---|---|---|---|
| `register_token` | Validate a bot token via `getMe` and set `TELEGRAM_BOT_TOKEN` in the process env | `token` | — |
| `start_ngrok` | Start ngrok, register the public HTTPS URL as the Telegram webhook with `secret_token` | — | `port`, `webhook_secret` (if `WEBHOOK_SECRET_TELEGRAM` unset) |
| `set_webhook` | Call Telegram `setWebhook` only (no new ngrok). Use to **apply or rotate** the secret, or re-register after adding `WEBHOOK_SECRET_TELEGRAM` | — | `webhook_secret`, `webhook_public_url` |
| `health_status` | Probe ngrok’s local API and report `TELEGRAM_BOT_TOKEN` / `WEBHOOK_SECRET_TELEGRAM` presence (no secrets printed) | — | — |

### Typical workflow

1. If Telegram might not be running yet, the agent should ask whether you want it set up; it can call `health_status` first to see ngrok/token/secret state.
2. Set **`WEBHOOK_SECRET_TELEGRAM`** in `.env` (restart Localforge if needed) or plan to pass `webhook_secret` on the next step.
3. Call `telegram` with `action: register_token, token: <your-token>`.
4. Call `telegram` with `action: start_ngrok` (with `webhook_secret` if the env var is not set).
5. The agent replies with the public URL and webhook registration status. Send a message to your bot to verify.

**Retroactive secret:** Add or change `WEBHOOK_SECRET_TELEGRAM` in `.env`, restart Localforge, then call **`set_webhook`**: with **`webhook_public_url`** set to your current HTTPS tunnel base (e.g. `https://xyz.ngrok.io`), or omit it if ngrok is running and the tool can read the tunnel from the local ngrok API. Telegram accepts `setWebhook` with the same URL and a new `secret_token`, so you do not need a new tunnel URL to rotate the secret.

### Environment variables

| Variable | Set by | Purpose |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | `register_token` action | Bot token used by `start_ngrok` / `set_webhook` and the webhook provider |
| `WEBHOOK_SECRET_TELEGRAM` | User / `.env` (required) | Must match Telegram’s `secret_token`; Localforge verifies `X-Telegram-Bot-Api-Secret-Token`. Omit only by passing `webhook_secret` on tool calls |

### Warning

Each call to `start_ngrok` spawns a new OS ngrok process. Stale processes from previous calls must be killed manually:

```bash
pkill ngrok
```

## See Also

- [Builder Configuration Guide](../src/builder/README.md)
- [Agent Configuration](CONFIG.md)
- [Examples Directory](../examples/)
