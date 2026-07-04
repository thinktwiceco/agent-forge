# Architecture

## Turn queue

All agent turns — web chat, webhooks, heartbeat, scheduler — enter through a single `TurnQueue` (`src/agents/turn_queue.go`) before the executor runs.

```mermaid
flowchart LR
    UI[ChatStream callers] --> TQ[TurnQueue]
    PL[InboxAware plugins] --> TQ
    TQ -->|per-chatId lock| EX[executeTurn → executor → LLM]
```

- `ChatStream` enqueues a turn and returns a `ResponseCh` immediately.
- Plugins implement `core.InboxAware` and receive `queue.Inbox` (implemented by `TurnQueue`).
- Async **spawn_subagent** completions re-enter the parent conversation via `TurnQueue.submitSpawnResult` (same autonomous path as heartbeat/scheduler).
- Same `chatId` turns are serialized; different conversations may run concurrently.
- `Len()` reflects ingress depth only (turns waiting, not in-flight).

## Package Dependencies

```mermaid
graph TB
    cmd[cmd/localforge] --> agents[src/agents]
    agents --> core[src/core]
    agents --> llms[src/llms]
    agents --> history[src/history]
    agents --> telemetry[src/telemetry]
    agents --> execution[src/agents/execution]
    agents --> context[src/agents/context]
    agents --> prompts[src/agents/prompts]
    agents --> handlers[src/agents/handlers]
    agents --> tools[src/tools]
    
    tools --> fs[tools/fs]
    tools --> git[tools/git]
    tools --> postgres[tools/postgres]
    tools --> api[tools/api]
    tools --> web[tools/web]
    tools --> image[tools/image]
    tools --> instagram[tools/instagram]
    tools --> telegram[tools/telegram]
    tools --> meta[tools/meta]
    tools --> expand[tools/expand]
    tools --> update[tools/update]
    tools --> spawn[tools/spawn]
```

## Key Packages

**Core Agent System:**
- `src/agents/` - Agent orchestrator (delegates to sub-packages)
- `src/agents/execution/` - Chat loop & tool execution engine
- `src/agents/context/` - Context management & truncation strategies
- `src/agents/prompts/` - Prompt building
- `src/agents/handlers/` - System event handlers

**Extracted Packages (Reusable):**
- `src/history/` - Conversation history management
- `src/telemetry/` - Observability (tool execution, tokens, truncation)

**Core Abstractions:**
- `src/core/` - Core interfaces (Plugin, Tool, Executable)
- `src/llms/` - LLM engine implementations

**Extensions:**
- `src/tools/` - Built-in tools (fs, git, postgres, api, web, image, instagram, telegram, meta, expand, update, spawn)
- `src/plugins/` - Plugin system (logger, todo, vault, skills, scheduler, heartbeat, brain)

## Import Patterns

```go
// ✅ Correct: agents package imports system package
import "github.com/thinktwiceco/agent-forge/src/agents/system"

// ❌ Never: system package MUST NOT import agents (circular dependency)
```

## Complete Structure

See [docs/FILE_STRUCTURE.md](../FILE_STRUCTURE.md) for detailed breakdown.
