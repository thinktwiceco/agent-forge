# Agent Configuration Reference

This document describes all `config.yaml` options for Agent Forge / Localforge agents.

## Agent fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Agent display name |
| `model` | Yes | LLM as `provider::model-name` |
| `system_prompt` | No | Multi-line system prompt |
| `working_dir` | Yes for most tools | Sandbox root; supports `${ENV_VAR}` |
| `persistence` | No | `"json"` (default conversations) or `""` |
| `tools` | No | Tool list (see below) |
| `plugins` | No | Plugin names (see below) |
| `spawn_subagent` | No | `true` enables `spawn_subagent` tool |
| `brain` | No | `false` disables default brain plugin |
| `config_tool` | No | `false` disables default config tool |
| `brain_plugin` | No | Dreaming schedule (see below) |
| `heartbeat` | No | Heartbeat schedule (see below) |

### Model format

`provider::model-name` — providers: `openai`, `deepseek`, `togetherai`, `openrouter`.

Examples: `deepseek::deepseek-chat`, `openrouter::openai/gpt-4o`.

API keys go in `.env`: `AF_OPENAI_API_KEY`, `AF_DEEPSEEK_API_KEY`, `AF_TOGETHERAI_API_KEY`, `AP_OPENROUTER_API_KEY`.

## Tools Configuration

Tools extend agent capabilities. Each entry is a string (`- fs`) or object (`- name: fs`).

### Available Tools

| Tool | Identifier | YAML parameters |
|------|------------|-----------------|
| File System | `fs` | None (needs `working_dir`) |
| Git | `git` | None (needs `working_dir`) |
| Web Browser | `web` | `headless` (optional bool) |
| API | `api` | `config_folder` (required) |
| Instagram | `instagram` | None; `INSTAGRAM_ACCESS_TOKEN` in env |
| Update | `update` | None (needs `working_dir`) |
| Telegram | `telegram` | `port` (optional, default `8080`) |
| PostgreSQL | `postgres` | `postgresURL`, `mode`, `allowedTables`, `allowedSchemas` |

### Tool examples

```yaml
agent:
  working_dir: "${AGENT_WORKING_DIR}"
  tools:
    - name: fs
    - name: web
      headless: true
    - name: api
      config_folder: "api_config/"
    - name: telegram
      port: "8080"
    - name: postgres
      postgresURL: "${DATABASE_URL}"
      mode: "read"
      allowedTables: ["users"]
      allowedSchemas: ["public"]
```

### PostgreSQL parameters

- **postgresURL** (required): `postgresql://user:pass@host:port/db`
- **mode** (required): `read` or `write`
- **allowedTables** (required): table whitelist
- **allowedSchemas** (optional): schema whitelist

## Plugins Configuration

Default plugins (no YAML entry needed): **brain**, **skills**, **config**.

| Plugin | Notes |
|--------|--------|
| `logger` | Formatted CLI output |
| `todo` | Task list tool |
| `vault` | Encrypted secrets under `vault/` |
| `scheduler` | One-shot scheduled reminders |
| `heartbeat` | Autonomous check-ins; use with `agent.heartbeat` |
| `brain` | Memory graph; disable with `brain: false` |
| `skills` | SKILL.md packages under `skills/` |
| `config` | Read/mutate this config; disable with `config_tool: false` |

```yaml
agent:
  plugins:
    - todo
    - vault
    - scheduler
```

## Heartbeat

Optional block under `agent.heartbeat`. Heartbeat plugin auto-activates when this block exists.

```yaml
agent:
  heartbeat:
    every: "30m"
    ack_max_chars: 300
    active_hours:
      start: "08:00"
      end: "22:00"
      timezone: "America/New_York"
```

## Brain plugin (dreaming)

```yaml
agent:
  brain_plugin:
    dream: "on"
    dreamTime: "02:00"
```

- **dream**: `on` / `off` (also true/false, yes/no)
- **dreamTime**: local wall time `HH:MM` or `HH:MM:SS`

## Vector storage (optional)

Top-level `vector-storage` section for semantic search backends.

```yaml
vector-storage:
  vector_db: "sqlite"
  embedding_model: "openai::text-embedding-3-small"
  sqlite:
    db_path: "./vector.db"
    vector_dim: 1536
```

## Environment variables

| Variable | Purpose |
|----------|---------|
| `AF_LOG_LEVEL` | DEBUG, INFO, WARN, ERROR |
| `AF_BRAVE_API_KEY` | Web search |
| `INSTAGRAM_ACCESS_TOKEN` | Instagram tool/provider |
| `TELEGRAM_BOT_TOKEN` | Telegram provider |
| `WEBHOOK_SECRET_TELEGRAM` | Telegram webhook validation |
| `TELEGRAM_ALLOWED_USER_IDS` | Telegram username allowlist |
| `AUTH_USERNAME` / `AUTH_PASSWORD_HASH` | Localforge login |

After config changes via the config tool, the agent reloads automatically in Localforge.
