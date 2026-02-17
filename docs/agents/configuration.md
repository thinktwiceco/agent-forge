# Configuration

See [docs/CONFIG.md](../CONFIG.md) and [docs/AGENT_BUILDER.md](../AGENT_BUILDER.md) for complete details.

## Builder Pattern (Recommended)

```go
agent, err := agents.NewBuilder(llmEngine, "agent-name").
    WithDescription("Brief description").
    WithSystemPrompt("You are...").
    WithTools(fsTool, gitTool).
    AsMainAgent().
    WithPersistence("json").
    WithContextWindow(128000).           // Auto-sets truncation
    WithTracer(telemetry.NewLogTracer()).
    Build()
```

## Manual AgentConfig (Advanced)

```go
agent := agents.NewAgent(&agents.AgentConfig{
    LLMEngine:            llmEngine,
    AgentName:            "agent-name",
    SystemPrompt:         "You are...",
    Tools:                []llms.Tool{fsTool, gitTool},
    MaxContextTokens:     128000,
    TruncationStrategy:   customStrategy,
    Tracer:               telemetry.NewLogTracer(),
})
```

## Environment Variables

```bash
# LLM API Keys
AF_TOGETHERAI_API_KEY=your_key
AF_OPENAI_API_KEY=your_key
AF_DEEPSEEK_API_KEY=your_key

# Logging
AF_LOG_LEVEL=INFO              # DEBUG, INFO, WARN, ERROR
AF_LOG_FILE=logs/app.log       # Optional
```

Set in `.env` file or system environment. System environment takes precedence.
