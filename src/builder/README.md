# Agent Builder Configuration Guide

The builder module provides a YAML-based configuration system for creating agents, tools, subagents, plugins, and vector storage components. This guide documents all available configuration options.

## Overview

Agents can be created from YAML configuration files using `NewAgentBuilderFromConfig()`. The configuration file defines the agent's capabilities, tools, subagents, plugins, and optional vector storage setup.

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
      root: "/path/to/sandbox"
    - name: git
      root: "/path/to/repo"
    - name: postgres
      postgresURL: "postgresql://user:password@localhost:5432/mydb"
      mode: "read"
      allowedTables: ["users", "orders"]
      allowedSchemas: ["public"]
  subagents:
    reasoning: "deepseek::deepseek-chat"
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
| `subagents` | map | No | Map of subagent names to their models | See [Subagents Configuration](#subagents-configuration) |
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
- **subagents**: Map where keys are subagent names and values are their model specifications.
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

| Tool Name | Identifier | Description | Configuration Parameters |
|-----------|------------|-------------|--------------------------|
| File System Tool | `fs` | File and directory operations (read, write, list, delete) | `root` (required): Root directory path |
| Git Tool | `git` | Git repository operations (add, commit, push, pull, status, log) | `root` (required): Repository directory path |
| Web Browser Tool | `web` | Web navigation, automation, and content extraction | `root` (required): Working directory path |
| Vector DB Tool | `vector` | Semantic search and document indexing | No parameters (requires `vector-storage` section) |
| PostgreSQL Tool | `postgres` | Secure database operations with parameterized queries | See [PostgreSQL Tool Configuration](#postgresql-tool-configuration) |

### Tool Configuration Examples

**File System Tool:**
```yaml
tools:
  - name: fs
    root: "/home/user/sandbox"  # Required: root directory for file operations
```

**Git Tool:**
```yaml
tools:
  - name: git
    root: "/home/user/my-repo"  # Required: git repository directory
```

**Web Browser Tool:**
```yaml
tools:
  - name: web
    root: "/home/user/downloads"  # Required: working directory for downloads/screenshots
```

**Vector DB Tool:**
```yaml
tools:
  - name: vector  # No additional parameters, requires vector-storage section
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

## Subagents Configuration

Subagents are specialized agents that can be delegated tasks by the main agent.

| Subagent Name | Identifier | Description | Requirements |
|---------------|------------|-------------|--------------|
| Reasoning Agent | `reasoning` | Analyzes questions for ambiguities and assumptions | Model specification required |
| OS Agent | `os` | Handles OS-level operations and file system tasks | Model specification and `working_dir` required |
| Git Agent | `git` | Specialized Git repository management | Model specification and `working_dir` required |
| Web Agent | `web` | Web automation and browser operations | Model specification and `working_dir` required |

### Subagents Details

- **Reasoning Agent**: Analyzes user questions before the main agent responds, helping identify ambiguities, detect assumptions, and guide objective responses.
- **OS Agent**: Handles file system operations and OS-related tasks within a restricted directory. All paths are validated for security.
- **Git Agent**: Provides specialized Git repository operations including branch management, commit history, and version control workflows.
- **Web Agent**: Manages browser sessions for web navigation, form interaction, content extraction, and JavaScript execution.

## Plugins Configuration

Plugins extend agent functionality with additional features and behaviors.

| Plugin Name | Identifier | Description | Features |
|-------------|------------|-------------|----------|
| Logger Plugin | `"logger"` | Configurable output formatting for agent responses | Color-coded output, trace labels, formatted tool execution |
| Todo Plugin | `"todo"` | Task management and todo list functionality | Create, update, and manage todo items |

### Plugins Details

- **Logger Plugin**: Provides formatted, color-coded output for agent responses, tool executions, and trace information. Enhances readability of agent interactions.
- **Todo Plugin**: Adds task management capabilities, allowing agents to create, track, and manage todo lists during conversations.

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

### Agent with Subagents

```yaml
agent:
  name: "reasoning-agent"
  model: "deepseek::deepseek-chat"
  subagents:
    reasoning: "deepseek::deepseek-chat"
    os: "deepseek::deepseek-chat"
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
  subagents:
    reasoning: "deepseek::deepseek-chat"
    os: "deepseek::deepseek-chat"
    git: "deepseek::deepseek-chat"
    web: "deepseek::deepseek-chat"
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

Set these in your `.env` file or as system environment variables.

## Validation

The builder validates configuration files and will return errors for:

- Missing required fields (`name`, `model`)
- Invalid model format (must be `provider::model-name`)
- Invalid persistence type (must be `"json"` or empty)
- Missing requirements for tools/subagents (e.g., `working_dir` for file system tools)
- Invalid vector storage configuration

## See Also

- [Main README](../../README.md) - General Agent Forge documentation
- [Logger Plugin Documentation](../plugins/logger/README.md) - Logger plugin details
- [Todo Plugin Documentation](../plugins/todo/README.md) - Todo plugin details
- [Configuration Guide](../../docs/CONFIG.md) - Environment variable configuration



