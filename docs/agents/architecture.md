# Architecture

## Package Dependencies

```mermaid
graph TB
    cmd[cmd/app] --> agents[src/agents]
    agents --> core[src/core]
    agents --> llms[src/llms]
    agents --> history[src/history]
    agents --> telemetry[src/telemetry]
    agents --> execution[src/agents/execution]
    agents --> context[src/agents/context]
    agents --> prompts[src/agents/prompts]
    agents --> handlers[src/agents/handlers]
    agents --> system[src/agents/system]
    agents --> tools[src/tools]
    
    tools --> fs[tools/fs]
    tools --> git[tools/git]
    tools --> postgres[tools/postgres]
    tools --> api[tools/api]
    tools --> web[tools/web]
    tools --> vector[tools/vector]
```

## Key Packages

**Core Agent System:**
- `src/agents/` - Agent orchestrator (delegates to sub-packages)
- `src/agents/execution/` - Chat loop & tool execution engine
- `src/agents/context/` - Context management & truncation strategies
- `src/agents/prompts/` - Prompt building
- `src/agents/handlers/` - System event handlers
- `src/agents/system/` - System agent templates

**Extracted Packages (Reusable):**
- `src/history/` - Conversation history management
- `src/telemetry/` - Observability (tool execution, tokens, truncation)

**Core Abstractions:**
- `src/core/` - Core interfaces (SubAgent, Plugin, Tool)
- `src/llms/` - LLM engine implementations

**Extensions:**
- `src/tools/` - Built-in tools (fs, git, postgres, api, web, vector)
- `src/plugins/` - Plugin system (logger, todo)

## Import Patterns

```go
// ✅ Correct: agents package imports system package
import "github.com/thinktwiceco/agent-forge/src/agents/system"

// ❌ Never: system package MUST NOT import agents (circular dependency)
```

## Complete Structure

See [docs/FILE_STRUCTURE.md](../FILE_STRUCTURE.md) for detailed breakdown.
