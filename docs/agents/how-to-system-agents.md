# System agents (removed)

The codebase previously supported **built-in specialist subagents** (YAML `subagents` roles, `core.SubAgent`, and template-driven `system-*` agents). That mechanism has been **removed**. The old **delegate** tool is also removed.

Use a **single main agent** with **tools** and **plugins** (for example the brain plugin: graph + `MEMORY.md` + dreaming distillation into `brain/persistence/` — see [src/plugins/README.md](../../src/plugins/README.md#brain-plugin) and [configuration](configuration.md#brain-plugin-yaml)).

## Ephemeral subagents (`spawn_subagent`)

When `CanSpawnSubagent` is true, the agent gets the built-in **spawn_subagent** tool. Enable it in YAML with `agent.spawn_subagent: true` ([`builder.Config`](../../src/builder/agentBuilder.go)), or in code with `agents.NewBuilder(...).WithSpawnSubagent().Build()`. It creates a **short-lived** child agent with the same LLM engine, a **subset** of the parent’s tools (by name), plus built-in **meta** / **expand**, and attaches the **todo** plugin when it is registered. The parent passes `prompt` and `tools: string[]`; execution is **synchronous** and the tool result is the subagent’s final reply. This replaces the removed async delegate flow.
