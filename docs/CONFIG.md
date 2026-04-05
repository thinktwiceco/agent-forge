# Configuration Guide

## Overview

Agent Forge provides configuration at two levels:

1. **Application Configuration** (`config.go`) - System-wide settings (logging, etc.)
2. **Agent Configuration** (`AgentConfig`) - Per-agent settings (LLM, tools, behavior, etc.)

## Application Configuration

The `Config` struct contains system-wide settings loaded from environment variables and `.env` files:

- `AF_LOG_LEVEL`: Logging level for the application
  - Valid values: `DEBUG`, `INFO`, `WARN`, `ERROR`
  - Default: `INFO`
  - See [Logger Documentation](LOGGER.md) for details on how logging works
- `AF_LOG_FILE`: Optional path to a log file. When set, logs are written to both stdout and the file

## Agent Configuration

The `AgentConfig` struct configures individual agents. Use the [Builder Pattern](AGENT_BUILDER.md) for easier construction:

### Required Fields
- `LLMEngine`: The LLM engine (OpenAI, DeepSeek, TogetherAI, OpenRouter, etc.)
- `AgentName`: Unique identifier for the agent

### Optional Fields

#### Basic Settings
- `Description`: Short description of the agent
- `AdvanceDescription`: Detailed capability information
- `Troubleshooting`: Common issues and debugging guidance
- `SystemPrompt`: System prompt for the agent
- `Trace`: Trace identifier (default: `{agentName}-trace`)

#### Behavioral Settings
- `MainAgent`: Mark as main coordinator (default: `false`)
- `Tone`: Response tone (`ToneKeepItShort`, `ToneSystemAgent`, or empty)
- `CanExpand`: Enable expand tool for discovering capabilities (default: `true`)
#### Tools & plugins
- `Tools`: List of tools available to the agent
- `Plugins`: List of plugins to load
- `MaxToolIterations`: Max tool execution iterations (default: `30`)

For YAML-based agent config (`builder.Config`):

- **Heartbeat:** optional `agent.heartbeat` (`every`, `prompt`, `ack_max_chars`, `active_hours`) when `heartbeat` is listed under `plugins`. See [docs/agents/configuration.md](agents/configuration.md#heartbeat-plugin-yaml).
- **Brain:** loaded by default; `brain: false` opts out. Optional `agent.brain_plugin` sets `dream` (`on`/`off`) and `dreamTime` (local `HH:MM` when pending conversations are distilled). Distillation writes graph + `brain/persistence/` only when the model returns full retainable fields; see [docs/agents/configuration.md](agents/configuration.md#brain-plugin-yaml) and [src/plugins/README.md](../src/plugins/README.md#brain-plugin).
- **Spawn subagent:** optional `agent.spawn_subagent: true` enables the built-in `spawn_subagent` tool. See [docs/agents/how-to-system-agents.md](agents/how-to-system-agents.md#ephemeral-subagents-spawn_subagent).

#### History & Persistence
- `Persistence`: Persistence layer (`"json"` or `""` for none)
- `MaxContextTokens`: Context window size (enables truncation when > 0)
- `ReservedOutputTokens`: Token reserve for completion (default: `4000`)
- `MinRecentMessages`: Minimum recent messages to keep (default: `10`)
- `EnableSummarization`: Summarize truncated messages (default: `false`)
- `TruncationStrategy`: Custom truncation strategy (optional)

#### Observability
- `Tracer`: Telemetry tracer for tool execution, tokens, truncation events

## Usage

### Application Configuration

```go
import agentforge "github.com/thinktwiceco/agent-forge/src"

// Load configuration from .env file (if present) and environment variables
config, err := agentforge.NewConfig()
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

fmt.Printf("Log Level: %s\n", config.AFLogLevel)

// Initialize the logger with the config
agentforge.InitLogger(config)

// Now you can use the logger
agentforge.Info("Application started with log level: %s", config.AFLogLevel)
```

### Agent Configuration (Builder Pattern)

**Recommended approach:**

```go
import (
    "github.com/thinktwiceco/agent-forge/src/agents"
    "github.com/thinktwiceco/agent-forge/src/telemetry"
)

agent, err := agents.NewBuilder(llmEngine, "my-agent").
    WithDescription("Helpful assistant").
    WithSystemPrompt("You are helpful and concise").
    WithTools(tool1, tool2).
    AsMainAgent().
    WithTone(agents.ToneKeepItShort).
    WithPersistence("json").
    WithContextWindow(128000).           // Enables truncation
    WithTracer(telemetry.NewLogTracer()). // Optional telemetry
    Build()
if err != nil {
    log.Fatal(err)
}
```

### Agent Configuration (Manual)

**For advanced use cases:**

```go
import "github.com/thinktwiceco/agent-forge/src/agents"

agent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine:            llmEngine,
    AgentName:            "my-agent",
    Description:          "Helpful assistant",
    SystemPrompt:         "You are helpful and concise",
    Tools:                []llms.Tool{tool1, tool2},
    MainAgent:            true,
    Tone:                 agents.ToneKeepItShort,
    Persistence:          "json",
    MaxContextTokens:     128000,
    ReservedOutputTokens: 4000,
    MinRecentMessages:    10,
    Tracer:              telemetry.NewLogTracer(),
})
```

## Environment Variables

### Setting Environment Variables

You can set environment variables in several ways:

#### 1. Using a .env file

Create a `.env` file in the project root:

```bash
# .env file
AF_LOG_LEVEL=DEBUG
```

#### 2. Using shell export

```bash
export AF_LOG_LEVEL=DEBUG
```

#### 3. Inline with command

```bash
AF_LOG_LEVEL=DEBUG go run main.go
```

## Configuration Priority

Configuration values are loaded in the following order (later sources override earlier ones):

1. Default values (hardcoded in the application)
2. `.env` file values
3. System environment variables (highest priority)

## Validation

The configuration system automatically validates all values:

- `AF_LOG_LEVEL` must be one of: `DEBUG`, `INFO`, `WARN`, `ERROR`
- Invalid values will cause `NewConfig()` to return an error

## Example .env File

```bash
# Agent Forge Configuration
# Copy this to .env and update the values as needed

# Logging level for the application
# Valid values: DEBUG, INFO, WARN, ERROR
# Default: INFO
AF_LOG_LEVEL=INFO

# Optional: stream logs to a file (in addition to stdout)
# AF_LOG_FILE=logs/app.log

# Optional: LLM providers (set the keys you use)
# AF_OPENAI_API_KEY=
# AF_DEEPSEEK_API_KEY=
# AF_TOGETHERAI_API_KEY=
# AP_OPENROUTER_API_KEY=
```

## Adding New Configuration Fields

To add a new configuration field:

1. Add the field to the `Config` struct in `config.go`
2. Load the field in `NewConfig()` using `getEnv()`
3. Add validation logic in the `validate()` method if needed
4. Update this documentation

Example:

```go
type Config struct {
    AFLogLevel string
    // Add new field here
    NewField string
}

func NewConfig() (*Config, error) {
    _ = godotenv.Load()
    
    config := &Config{
        AFLogLevel: getEnv("AF_LOG_LEVEL", "INFO"),
        NewField:   getEnv("NEW_FIELD", "default_value"),
    }
    
    if err := config.validate(); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }
    
    return config, nil
}
```

## Truncation Strategies

When `MaxContextTokens` is set, the agent automatically truncates conversation history to fit within the context window:

```go
import "github.com/thinktwiceco/agent-forge/src/agents/context"

// Use default sliding window strategy
agent, err := agents.NewBuilder(llm, "agent").
    WithContextWindow(128000).  // Automatically sets defaults
    Build()

// Or use a custom strategy
customStrategy := context.NewSlidingWindowStrategy(
    tokenCounter,
    10,      // min recent messages
    context.ToolCallPairPreservation(), // preserve tool call/result pairs
)

agent, err := agents.NewBuilder(llm, "agent").
    WithTruncationStrategy(customStrategy).
    Build()
```

**Built-in Strategies:**
- `SlidingWindowStrategy`: Keeps recent N messages (default)
- `NoTruncationStrategy`: Disables truncation

## Telemetry Configuration

Add observability for debugging and monitoring:

```go
import "github.com/thinktwiceco/agent-forge/src/telemetry"

// Log-based tracer
agent, err := agents.NewBuilder(llm, "agent").
    WithTracer(telemetry.NewLogTracer()).
    Build()

// Noop tracer (no overhead)
agent, err := agents.NewBuilder(llm, "agent").
    WithTracer(telemetry.NewNoopTracer()).
    Build()

// Custom tracer
type CustomTracer struct{}
func (t *CustomTracer) TraceToolExecution(ctx context.Context, event telemetry.ToolExecutionEvent) {
    // Your implementation
}
// ... implement other methods
```

## See Also

- [Agent Builder Documentation](AGENT_BUILDER.md) - Fluent API for agent construction
- [Interfaces Documentation](INTERFACES.md) - Internal interfaces and architecture
- [Logger Documentation](LOGGER.md) - Comprehensive guide to using the logger system

