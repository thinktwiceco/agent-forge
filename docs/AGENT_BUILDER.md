# Agent Builder Pattern

## Overview

The Builder pattern provides a fluent API for constructing Agent instances. Use it instead of manual `AgentConfig` when building agents programmatically.

## When to Use

- **Builder**: Preferred for programmatic agent creation (tests, custom agents, system agents)
- **AgentConfig + NewAgent**: Advanced use cases requiring full control over the config struct

## Basic Usage

```go
import (
    "github.com/thinktwiceco/agent-forge/src/agents"
    "github.com/thinktwiceco/agent-forge/src/llms"
)

// Fluent construction
agent, err := agents.NewBuilder(llmEngine, "assistant").
    WithDescription("Helpful assistant").
    WithSystemPrompt("You are helpful").
    WithTools(tool1, tool2).
    WithContextWindow(128000).
    Build()
if err != nil {
    log.Fatal(err)
}
```

## Comparison

**Before (verbose):**
```go
config := &agents.AgentConfig{
    LLMEngine:          engine,
    AgentName:          "assistant",
    Description:        "Helpful assistant",
    SystemPrompt:       "You are helpful",
    Tools:               []llms.Tool{tool1, tool2},
    MaxContextTokens:    128000,
    ReservedOutputTokens: 4000,
    MinRecentMessages:    10,
    CanExpand:           true,
    MaxToolIterations:   30,
}
agent := agents.NewAgent(config)
```

**After (fluent):**
```go
agent, err := agents.NewBuilder(engine, "assistant").
    WithDescription("Helpful assistant").
    WithSystemPrompt("You are helpful").
    WithTools(tool1, tool2).
    WithContextWindow(128000).  // Sets max tokens + defaults
    Build()
```

## Builder Methods

| Method | Description |
|--------|-------------|
| `WithDescription(s)` | Agent description |
| `WithAdvanceDescription(s)` | Detailed capability info |
| `WithTroubleshooting(s)` | Debugging guidance |
| `WithSystemPrompt(s)` | System prompt |
| `WithTrace(s)` | Trace identifier |
| `WithCanExpand(bool)` | Enable expand for tools |
| `WithTools(...llms.Tool)` | Add tools (append) |
| `WithMaxToolIterations(n)` | Max tool execution iterations |
| `AsMainAgent()` | Mark as main agent |
| `WithTone(s)` | Response tone (e.g., `ToneKeepItShort`) |
| `WithPersistence(s)` | Persistence layer ("", "json") |
| `WithPlugins(...)` | Add plugins |
| `WithContextWindow(n)` | Context size; sets reserved/minRecent defaults |
| `WithReservedOutputTokens(n)` | Token reserve for completion |
| `WithMinRecentMessages(n)` | Min recent messages to keep |
| `WithEnableSummarization(bool)` | Summarize truncated messages |
| `WithTruncationStrategy(s)` | Custom truncation strategy |
| `WithTracer(t)` | Telemetry tracer (tool exec, tokens, truncation) |

## Convenience: WithContextWindow

`WithContextWindow(128000)` automatically sets:
- `MaxContextTokens = 128000`
- `ReservedOutputTokens = 4000`
- `MinRecentMessages = 10`

## Main Agent Example

```go
agent, err := agents.NewBuilder(llmEngine, "main-agent").
    AsMainAgent().
    WithTone(agents.ToneKeepItShort).
    WithDescription("Main conversational agent").
    WithTools(fsTool, gitTool).
    WithContextWindow(128000).
    Build()
```

## Telemetry

Add structured observability for debugging agent behavior:

```go
import "github.com/thinktwiceco/agent-forge/src/telemetry"

// Log-based tracer (debug level)
agent, err := agents.NewBuilder(llm, "assistant").
    WithTracer(telemetry.NewLogTracer(nil)).
    Build()

// OpenTelemetry metrics
agent, err := agents.NewBuilder(llm, "assistant").
    WithTracer(telemetry.NewMetricsTracer(meter)).
    Build()
```

Events traced: tool execution (duration, success), token usage, history truncation, agent lifecycle.

## BuildConfig

To get the config without creating an agent:

```go
cfg, err := agents.NewBuilder(llmEngine, "name").
    WithSystemPrompt("...").
    BuildConfig()
if err != nil {
    return err
}
// Customize cfg further, then:
agent := agents.NewAgent(cfg)
```
