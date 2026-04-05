# Agent Builder Configuration Guide

The builder module provides a YAML-based configuration system for creating agents, tools, plugins, and vector storage components. This guide documents all available configuration options.

## Overview

Agents can be created from YAML configuration files using `NewAgentBuilderFromConfig()`. The configuration file defines the agent's capabilities, tools, plugins, and optional vector storage setup.

## Configuration File Structure

```yaml
agent:
  name: "my-agent"
  system_prompt: "You are a helpful assistant."
  model: "deepseek::deepseek-chat"
  working_dir: "/path/to/working/directory"  # Optional fallback for tools
  persistence: "json"
  tools:
    - name: fs
    - name: git
    - name: postgres
      postgresURL: "postgresql://user:password@localhost:5432/mydb"
      mode: "read"
      allowedTables: ["users", "orders"]
      allowedSchemas: ["public"]
  plugins:
    - "logger"
    - "todo"

vector-storage:
  vector_db: "sqlite"
  embedding_model: "openai::text-embedding-3-small"
  sqlite:
    db_path: "./vector.db"
    vector_dim: 1536
  # milvus:
  #   host: "localhost"
  #   port: 19530
  #   vector_dim: 1536
```

## Agent Configuration

The `agent` section defines the main agent configuration.

| Field | Type | Required | Description | Options |
|-------|------|----------|-------------|---------|
| `name` | string | Yes | Unique name for the agent | Any string |
| `system_prompt` | string | No | System prompt that defines agent behavior | Any string |
| `model` | string | Yes | LLM model in format `provider::model-name` | See [Model Format](#model-format) |
| `working_dir` | string | No | Working directory for file operations | Absolute or relative path |
| `persistence` | string | No | Conversation history persistence type | `"json"` or `""` (empty) |
| `tools` | array | No | List of tool names to enable | See [Tools Configuration](#tools-configuration) |
| `plugins` | array | No | List of plugin names to enable | See [Plugins Configuration](#plugins-configuration) |

### Agent Configuration Details

- **name**: Must be unique. Used for agent registration and identification.
- **system_prompt**: Optional custom prompt. If omitted, default system prompt is used.
- **model**: Required LLM model specification. Format: `provider::model-name` (e.g., `deepseek::deepseek-chat`).
- **working_dir**: Optional fallback directory for tools that don't specify their own `root` parameter. For backwards compatibility only.
- **persistence**: 
  - `"json"`: Stores conversation history as JSON files. When `working_dir` is set: `working_dir/data/conversations/{agentName}`. Otherwise: `data/conversations/{agentName}` relative to process CWD.
  - `""` (empty): No persistence, conversations are not saved
- **tools**: Array of tool configurations. Tools can be specified as:
  - **Object format** (recommended): `{name: "fs", root: "/path"}` - Allows per-tool configuration
  - **String format** (legacy): `"fs"` - Uses global `working_dir` as fallback
- **plugins**: Array of plugin identifiers that extend agent functionality.

## Tools Configuration

Tools extend agent capabilities with specific functionalities. Each tool can be configured with its own initialization parameters.

### Configuration Format

Tools support two configuration formats:

**Object format (recommended):**
```yaml
tools:
  - name: fs
    root: "/path/to/sandbox"
  - name: postgres
    postgresURL: "postgresql://..."
    mode: "read"
    allowedTables: ["users"]
```

**String format (legacy):**
```yaml
tools:
  - "fs"  # Uses agent.working_dir as fallback
  - "git"
```

### Available Tools

Most tools use `agent.working_dir` as their sandbox (fs roots at `working_dir`, git at `working_dir/repos`, web at `working_dir/web`, etc.). Set `working_dir` in YAML; do not rely on per-tool `root` fields (they are not part of the current `Tool` schema).

| Tool Name | Identifier | Description | YAML parameters |
|-----------|------------|-------------|-----------------|
| File System Tool | `fs` | File and directory operations | None (needs `working_dir`) |
| Git Tool | `git` | Git repository operations | None (needs `working_dir`) |
| Web Browser Tool | `web` | Web navigation, automation, content extraction | `headless` (optional bool) |
| API Tool | `api` | REST calls from JSON service definitions | `config_folder` (required): folder of `*.json` configs |
| Instagram Tool | `instagram` | Instagram Graph API (flat actions) | None; set `INSTAGRAM_ACCESS_TOKEN` in environment |
| Update Tool | `update` | Runs `update-release.sh` in agent root | None (needs `working_dir`) |
| Telegram Tool | `telegram` | Bot token registration + ngrok tunnel + `setWebhook` for Localforge | `port` (optional, default `8080`) |
| Vector DB Tool | `vector` | Semantic search and indexing | None (requires `vector-storage` section) |
| PostgreSQL Tool | `postgres` | Parameterized SQL | See [PostgreSQL Tool Configuration](#postgresql-tool-configuration) |

Full behaviour and env vars: [docs/TOOLS.md](../../docs/TOOLS.md).

### Tool Configuration Examples

**File System / Git / Web (working_dir only):**
```yaml
agent:
  working_dir: "/home/user/agent-data"
  tools:
    - name: fs
    - name: git
    - name: web
      # headless: false
```

**API Tool:**
```yaml
agent:
  working_dir: "/home/user/agent-data"
  tools:
    - name: api
      config_folder: "api_specs"   # relative to working_dir or absolute
```

**Telegram dev helper:**
```yaml
tools:
  - name: telegram
    port: "8080"
```

At runtime, Localforge can receive Telegram webhooks when `TELEGRAM_BOT_TOKEN` is set; **`/new_conversation`** starts a new JSON thread (see [docs/TOOLS.md](../../docs/TOOLS.md#telegram-webhook-threads)).

**Vector DB Tool:**
```yaml
tools:
  - name: vector  # requires vector-storage section
```

### PostgreSQL Tool Configuration

The PostgreSQL tool requires inline configuration parameters:

```yaml
tools:
  - name: postgres
    postgresURL: "postgresql://user:password@localhost:5432/mydb"  # Required
    mode: "read"  # Required: "read" or "write"
    allowedTables:  # Required: array of table names
      - "users"
      - "orders"
      - "products"
    allowedSchemas:  # Optional: array of schema names (defaults to public)
      - "public"
      - "analytics"
```

**Configuration parameters:**
- **postgresURL** (required): PostgreSQL connection URL in format `postgresql://user:pass@host:port/db`
- **mode** (required): Access mode - `"read"` for SELECT only, `"write"` for all operations
- **allowedTables** (required): Whitelist of table names the agent can access
- **allowedSchemas** (optional): Whitelist of schema names (defaults to `["public"]` if omitted)

**READ mode** allows:
- SELECT queries
- getTables operation
- getSchema operation

**WRITE mode** allows:
- All READ mode operations
- UPDATE operations
- INSERT operations

## Plugins Configuration

Plugins extend agent functionality with additional features and behaviors.

| Identifier | Notes |
|------------|--------|
| `logger` | Formatted, color-coded CLI output for responses and tool traces |
| `todo` | Task list CRUD for the agent |
| `vault` | Encrypted secret storage under the working directory |
| `scheduler` | Scheduled jobs |
| `procedures` | Procedure / checklist manifests |
| `heartbeat` | Proactive timed agent turns; optional `agent.heartbeat` YAML |
| `brain` | Memory graph + optional dreaming; omit `brain: false` to keep default on |

See [src/plugins/README.md](../plugins/README.md) and [docs/agents/configuration.md](../../docs/agents/configuration.md).

### Plugins Details

- **Logger**: Readable terminal output for agent and tool activity.
- **Todo**: In-conversation task tracking.
- **Brain**: Loads by default unless `brain: false`; optional `brain_plugin` for dreaming. See configuration docs.

## Vector Storage Configuration

The optional `vector-storage` section configures vector database and embedding generation for semantic search capabilities.

| Field | Type | Required | Description | Options |
|-------|------|----------|-------------|---------|
| `vector_db` | string | Yes* | Vector database type | `"sqlite"` or `"milvus"` |
| `embedding_model` | string | Yes* | Embedding model specification | Format: `provider::model-name` |
| `sqlite` | map | No | SQLite-specific configuration | See [SQLite Configuration](#sqlite-configuration) |
| `milvus` | map | No | Milvus-specific configuration | See [Milvus Configuration](#milvus-configuration) |

*Required if vector tool is used.

### SQLite Configuration

| Field | Type | Required | Description | Default |
|-------|------|----------|-------------|---------|
| `db_path` | string | Yes | Path to SQLite database file | None |
| `vector_dim` | integer | No | Vector dimension for embeddings | `1536` |

### Milvus Configuration

| Field | Type | Required | Description | Default |
|-------|------|----------|-------------|---------|
| `host` | string | No | Milvus server hostname | `"localhost"` |
| `port` | integer | No | Milvus server port | `19530` |
| `vector_dim` | integer | No | Vector dimension for embeddings | `1536` |

### Vector Storage Details

- **SQLite**: Lightweight, file-based vector database. Suitable for development and small-scale deployments. Requires `db_path` to be specified.
- **Milvus**: Production-grade vector database. Suitable for large-scale deployments. Requires Milvus server to be running.
- **Embedding Models**: Currently supports OpenAI embedding models. Format: `openai::model-name` (e.g., `openai::text-embedding-3-small`).

## Model Format

Models are specified in the format: `provider::model-name`

### Supported Providers

| Provider | Identifier | API Key Environment Variable | Description |
|----------|------------|------------------------------|-------------|
| OpenAI | `openai` | `AF_OPENAI_API_KEY` | OpenAI API models |
| DeepSeek | `deepseek` | `AF_DEEPSEEK_API_KEY` | DeepSeek API models |
| TogetherAI | `togetherai` | `AF_TOGETHERAI_API_KEY` | TogetherAI API models |
| OpenRouter | `openrouter` | `AP_OPENROUTER_API_KEY` | OpenAI-compatible chat via [OpenRouter](https://openrouter.ai/) |

### Example Models

**OpenAI:**
- `openai::gpt-5`
- `openai::gpt-5.1`
- `openai::gpt-5.2`

**DeepSeek:**
- `deepseek::deepseek-chat`

**TogetherAI:**
- `togetherai::meta-llama/Llama-3.2-3B-Instruct-Turbo`
- `togetherai::meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo`
- `togetherai::Qwen/Qwen2.5-7B-Instruct-Turbo`
- `togetherai::zai-org/GLM-4.7`
- `togetherai::moonshotai/Kimi-K2.5`

**OpenRouter** (model id is the OpenRouter slug, e.g. from their model directory):
- `openrouter::openai/gpt-4o`
- `openrouter::openai/gpt-4o-mini`

### Embedding Models

Currently supported embedding providers:

| Provider | Example Models |
|----------|----------------|
| OpenAI | `openai::text-embedding-3-small` |

## Usage Examples

### Basic Agent

```yaml
agent:
  name: "basic-agent"
  model: "deepseek::deepseek-chat"
  system_prompt: "You are a helpful assistant."
```

### Agent with Tools

```yaml
agent:
  name: "developer-agent"
  model: "deepseek::deepseek-chat"
  working_dir: "/path/to/project"
  tools:
    - "fs"
    - "git"
```

### Agent with Vector Storage

```yaml
agent:
  name: "knowledge-agent"
  model: "deepseek::deepseek-chat"
  working_dir: "/path/to/docs"
  tools:
    - "vector"

vector-storage:
  vector_db: "sqlite"
  embedding_model: "openai::text-embedding-3-small"
  sqlite:
    db_path: "./knowledge.db"
```

### Agent with PostgreSQL Database Access

**Example configuration (READ mode):**
```yaml
agent:
  name: "database-agent"
  model: "deepseek::deepseek-chat"
  tools:
    - name: postgres
      postgresURL: "postgresql://user:password@localhost:5432/mydb"
      mode: "read"
      allowedTables:
        - "users"
        - "orders"
        - "products"
      allowedSchemas:
        - "public"
```

**Agent usage (READ mode):**

Query data:
```
Tool: postgres
Parameters:
  operation: "select"
  table: "users"
  select: "name, email WHERE age > 25"
  limit: 50
```

List tables:
```
Tool: postgres
Parameters:
  operation: "getTables"
  schema: "public"
```

Get table schema:
```
Tool: postgres
Parameters:
  operation: "getSchema"
  table: "users"
  schema: "public"
```

**Example configuration (WRITE mode):**
```yaml
agent:
  name: "database-agent"
  model: "deepseek::deepseek-chat"
  tools:
    - name: postgres
      postgresURL: "postgresql://user:password@localhost:5432/mydb"
      mode: "write"
      allowedTables:
        - "users"
      allowedSchemas:
        - "public"
```

**Agent usage (WRITE mode):**

Update data:
```
Tool: postgres
Parameters:
  operation: "update"
  table: "users"
  update: "last_login = NOW()"
  select: "WHERE user_id = 123"
```

Insert data:
```
Tool: postgres
Parameters:
  operation: "insert"
  table: "users"
  add: "name, email VALUES ('John', 'john@example.com')"
```

**Key features:**
- SQL injection prevention through parameterized queries
- Table and schema whitelist enforcement for access control
- Per-tool configuration embedded in the tools array  
- Mode-based operation restrictions configured by developer
- Database introspection via getTables and getSchema operations
- Agent provides SQL fragments, not complete statements
- Automatic validation of SQL fragments to prevent injection attempts
- Connection credentials managed by developer, not exposed to agent
- Filtered results show only accessible tables and schemas

### Complete Example

```yaml
agent:
  name: "full-featured-agent"
  system_prompt: "You are an advanced AI assistant with multiple capabilities."
  model: "deepseek::deepseek-chat"
  working_dir: "/home/user/projects"
  persistence: "json"
  tools:
    - "fs"
    - "git"
    - "web"
    - "vector"
    - "postgres"
  plugins:
    - "logger"
    - "todo"

vector-storage:
  vector_db: "sqlite"
  embedding_model: "openai::text-embedding-3-small"
  sqlite:
    db_path: "./vector.db"
    vector_dim: 1536
```

## Programmatic Usage

### Using AgentBuilder

```go
import "github.com/thinktwiceco/agent-forge/src/builder"

// Create agent builder from config file
agentBuilder, err := builder.NewAgentBuilderFromConfig("agent_config.yaml")
if err != nil {
    log.Fatalf("Failed to create agent builder: %v", err)
}

// Build the agent
agent, err := agentBuilder.Build()
if err != nil {
    log.Fatalf("Failed to build agent: %v", err)
}
```

## Environment Variables

The following environment variables are required for LLM providers:

- `AF_OPENAI_API_KEY`: OpenAI API key
- `AF_DEEPSEEK_API_KEY`: DeepSeek API key
- `AF_TOGETHERAI_API_KEY`: TogetherAI API key
- `AP_OPENROUTER_API_KEY`: OpenRouter API key (uses the `AP_` prefix; required for `openrouter::...` models)

Set these in your `.env` file or as system environment variables.

## Validation

The builder validates configuration files and will return errors for:

- Missing required fields (`name`, `model`)
- Invalid model format (must be `provider::model-name`)
- Invalid persistence type (must be `"json"` or empty)
- Missing requirements for tools (e.g., `working_dir` for file system tools)
- Invalid vector storage configuration

## See Also

- [Main README](../../README.md) - General Agent Forge documentation
- [Logger Plugin Documentation](../plugins/logger/README.md) - Logger plugin details
- [Todo Plugin Documentation](../plugins/todo/README.md) - Todo plugin details
- [Configuration Guide](../../docs/CONFIG.md) - Environment variable configuration



