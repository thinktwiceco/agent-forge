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
AP_OPENROUTER_API_KEY=your_key

# Logging
AF_LOG_LEVEL=INFO              # DEBUG, INFO, WARN, ERROR
AF_LOG_FILE=logs/app.log       # Optional
```

Set in `.env` file or system environment. System environment takes precedence.

## Heartbeat plugin (YAML)

When loading [`builder.Config`](../../src/builder/agentBuilder.go) from YAML (e.g. Localforge `config.yaml`), include `heartbeat` under `agent.plugins` and optionally override defaults with a sibling `agent.heartbeat` block:

```yaml
agent:
  plugins:
    - todo
    - heartbeat
  heartbeat:
    every: "30m"           # optional; omit → 30m; use "0m" to disable
    prompt: ""             # optional; empty → default injected prompt
    ack_max_chars: 300     # optional; 0 → 300
    active_hours:          # optional; omit → 24/7
      start: "08:00"
      end: "22:00"
      timezone: "America/New_York"
```

If `heartbeat` is not listed in `plugins`, `agent.heartbeat` is ignored. Merge rules are implemented in [`MergeConfig`](../../src/plugins/heartbeat/config.go) in the heartbeat package.

## Brain plugin (YAML)

The **brain** plugin loads by default (you do not need to list it under `plugins`). Opt out with `brain: false`.

Optional **dreaming** (distils `data/conversations/**/*.json`, recategorizes conversations that are only on topic `pending`, then cleans `brain/MEMORY.md` when that file has content; writes `brain/persistence/` and updates graph nodes when the model returns retainable JSON) is configured under `brain_plugin`. Defaults: dreaming **on**, daily at **02:00** local time.

```yaml
agent:
  # brain: false          # uncomment to disable the entire brain plugin
  brain_plugin:
    dream: "on"            # or "off"
    dreamTime: "02:00"     # local wall time, HH:MM or HH:MM:SS
```

**Dreaming (distillation):** Scheduled and on-demand runs read conversation JSON under `data/conversations/`, call the configured LLM, and may write `brain/persistence/YYYY-MM-DD/<conv_id>.md` and update graph nodes only when the model returns retainable fields (`summary`, `title`, `description`, `distillation_reason`). Retained `title`, `description`, and `distillation_reason` are stored as columns on the conversation node; summary is `content`; `topics` remain in metadata; full-text search uses a denormalized `search_text` column. A follow-up pass then re-tags any **dreamed** conversation that is linked **only** to topic `pending`, using a separate LLM call with the distilled fields and **CURRENT TOPICS**. The same run then processes `brain/MEMORY.md` if it is non-empty: the model may rewrite short-term notes and may optionally promote **at most one** durable item into the graph (synthetic id prefix `brain-memmd-`, same recall tools as other conversation nodes). The prompts include **CURRENT TOPICS** from the graph so labels prefer existing topic names. See [`src/plugins/brain/dreaming.go`](../../src/plugins/brain/dreaming.go) and [`src/plugins/brain/dreaming_recategorize.go`](../../src/plugins/brain/dreaming_recategorize.go).

See [`src/plugins/brain/config.go`](../../src/plugins/brain/config.go) for merge rules and [`src/plugins/README.md`](../../src/plugins/README.md#brain-plugin) for tools and on-disk layout.
