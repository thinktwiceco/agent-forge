# Built-in Tools Guide

Agent Forge includes several built-in tools that extend agent capabilities. This guide provides detailed documentation for each tool.

## Table of Contents

- [Overview](#overview)
- [File System Tool](#file-system-tool)
- [Git Tool](#git-tool)
- [Postgres Tool](#postgres-tool)
- [API Tool](#api-tool)
- [Web Browser Tool](#web-browser-tool)
- [Vector Database Tool](#vector-database-tool)
- [Creating Custom Tools](#creating-custom-tools)

## Overview

Tools are the primary way agents interact with external systems. Each tool:
- Implements the `llms.Tool` interface
- Provides parameter validation
- Returns structured responses
- Can be configured via code or YAML

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

- **status**: Show working tree status
- **diff**: Show changes between commits
- **log**: Show commit logs
- **add**: Add files to staging area
- **commit**: Record changes to repository
- **push**: Update remote repository
- **pull**: Fetch and integrate changes
- **branch**: List, create, or delete branches
- **checkout**: Switch branches or restore files
- **reset**: Reset current HEAD
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

The web browser tool provides web automation using a headless browser.

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
- **get_content**: Get page content (HTML, text, or title)
- **save_content**: Save content to file in working directory

### YAML Configuration

```yaml
tools:
  - name: "web"
    root: "/path/to/working/dir"
```

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

- **index**: Index documents for semantic search
- **search**: Search indexed documents
- **list**: List all indexed documents
- **delete**: Delete indexed documents

### YAML Configuration

```yaml
tools:
  - name: "vector"
```

Note: Vector database and embedding generator must be configured separately at the agent builder level.

## Creating Custom Tools

To create a custom tool, implement the `llms.Tool` interface:

```go
import (
    "github.com/thinktwiceco/agent-forge/src/core"
    "github.com/thinktwiceco/agent-forge/src/llms"
)

func NewCustomTool() llms.Tool {
    return &core.Tool{
        Name:        "custom_tool",
        Description: "Brief description of the tool",
        AdvanceDesc: `Detailed description with:
- Usage instructions
- Available operations
- Examples`,
        TroubleshootingInfo: `Common issues and solutions`,
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
    }
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

## See Also

- [Builder Configuration Guide](../src/builder/README.md)
- [Agent Configuration](CONFIG.md)
- [Examples Directory](../examples/)
