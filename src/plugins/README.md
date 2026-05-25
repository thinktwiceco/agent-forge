# Plugins

## Table of Contents

- [How Plugins Work](#how-plugins-work)
- [Plugin Interface](#plugin-interface)
- [Available Plugins](#available-plugins)
- [Brain plugin](#brain-plugin)

## How Plugins Work

Plugins extend agent functionality by hooking into lifecycle events, providing tools, and adding system prompt instructions. They use a composable interface design following the Interface Segregation Principle - plugins only implement the capabilities they need.

### Registration Flow

1. Plugin implements `core.Plugin` interface (required)
2. Plugin optionally implements one or more provider interfaces:
   - `HookProvider` for event hooks
   - `ToolProvider` for tools
   - `PromptProvider` for system prompt additions
3. Plugin self-registers via `registry.Register(name, factory)` in its `init()` function. The factory receives `workingDir` and returns a `core.Plugin`.
4. [allplugins.go](../builder/allplugins.go) imports each plugin with a blank import to trigger `init()` when the builder loads.
5. Config lists plugin names (e.g. `plugins: ["todo", "vault"]`). The builder fetches plugins from the registry and instantiates them with the agent's working directory.
6. During `EventAgentInitialization`, the agent registers hooks, tools, and prompts from each plugin.

### Lifecycle Events

Plugins can hook into:
- `EventAgentInitialization` - Before agent initialization
- `EventAgentInitialized` - After agent initialization
- `EventContextBuild` - When building agent context
- `EventBeforeToolExecution` - Before tool execution
- `EventToolExecution` - After tool execution
- `EventNewUserMessage` - New user message received
- `EventNewChunk` - New streaming chunk created
- `EventNewAssistantMessage` - New assistant message
- `EventNewAssistantMessageWithToolCalls` - Assistant message with tool calls
- `EventAddedTools` - When tools are added
- `EventChatStart` - At the beginning of every ChatStream call, after the chatId is known and before the executor runs

## Plugin Interface

Plugins use a composable interface design where you only implement what you need:

### Base Interface (Required)

```go
type Plugin interface {
    Name() string
}
```

- **Name()**: Returns unique plugin identifier

### Optional Provider Interfaces

#### HookProvider - For Event Hooks

```go
type HookProvider interface {
    Plugin
    Hooks() map[Event]AgentHookFn
}
```

- **Hooks()**: Returns a map of event hooks that the plugin provides
- Only implement this if your plugin needs to respond to agent lifecycle events

**Example:**
```go
func (p *LoggerPlugin) Hooks() map[core.Event]core.AgentHookFn {
    return map[core.Event]core.AgentHookFn{
        core.EventNewChunk: agents.OnNewChunkHook(p.handleNewChunk),
    }
}
```

#### ToolProvider - For Adding Tools

```go
type ToolProvider interface {
    Plugin
    Tools() []llms.Tool
}
```

- **Tools()**: Returns list of tools provided to agents
- Only implement this if your plugin provides tools

**Example:**
```go
func (p *TodoPlugin) Tools() []llms.Tool {
    return []llms.Tool{
        newTodoHandlerTool(p),
    }
}
```

#### PromptProvider - For System Prompts

```go
type PromptProvider interface {
    Plugin
    SystemPrompt() string
}
```

- **SystemPrompt()**: Returns system prompt instructions that are automatically appended to the agent's system prompt
- Only implement this if your plugin needs to add instructions to the system prompt

**Example:**
```go
func (p *TodoPlugin) SystemPrompt() string {
    return `Use the todo_handler tool to manage tasks...`
}
```

### Benefits

- **Interface Segregation**: Plugins only implement capabilities they actually use
- **No Empty Methods**: No need to return empty values for unused features
- **Clear Intent**: Interface implementation clearly shows plugin capabilities
- **Flexible**: Mix and match interfaces as needed

### Optional Awareness Interfaces

Two additional interfaces let plugins receive agent infrastructure:

```go
// WorkingDirAware — called during EventAgentInitialization with the agent's working_dir.
type WorkingDirAware interface {
    SetWorkingDir(dir string)
}

// InboxAware — provides a reference to the agent's inbox queue so the plugin
// can inject autonomous messages (e.g. scheduled ticks, webhooks).
type InboxAware interface {
    SetInbox(q *queue.Queue)
}

// LLMEngineAware — provides direct LLM access for background tasks.
// The engine is injected during EventAgentInitialization.
// Used by the brain plugin's DreamingRunner for distilling conversation notes.
type LLMEngineAware interface {
    SetLLMEngine(engine llms.LLMEngine)
}
```

### Best Practices

- **Folder-based plugins**: If a plugin operates inside a folder (e.g. `vault/`, `skills/`), it should auto-create that folder at initialization time (e.g. in `EventAgentInitialized` or when the plugin is constructed). This avoids errors when the agent first uses the plugin.
- **Paths are relative to working_dir**: All plugin folder paths are relative to the agent's `working_dir`. The factory receives `workingDir` and should use it as the base (e.g. `filepath.Join(workingDir, "vault")`).
- **Goroutine lifecycle**: Plugins that start goroutines (e.g. tickers) must pair `context.WithCancel` with a stored `cancel` func so the goroutine stops cleanly on plugin teardown or config reload.
- **Inbox-injecting plugins**: Use `core.InboxAware` to receive the queue reference. Check `q.Len() > 0` before enqueueing to avoid flooding a busy agent.

## Available Plugins

| Plugin | Interfaces | Description |
|--------|-----------|-------------|
| [Logger](./logger/README.md) | `HookProvider` | Configurable output formatting for agent responses |
| [Todo](./todo/README.md) | `ToolProvider`, `PromptProvider` | Task management and todo list functionality |
| [Skills](./skills/plugin.go) | `HookProvider`, `ToolProvider`, `PromptProvider` | **Default plugin** — native `SKILL.md` packages from `working_dir/skills/`; `name` and `description` frontmatter are required, `references/` is optional, the `skill` tool can list/install/delete packages, and a built-in `web-navigation` skill is seeded automatically |
| [Vault](./vault/plugin.go) | `ToolProvider`, `PromptProvider`, `WorkingDirAware` | Encrypted secret storage in `working_dir/vault/` |
| [Brain](./brain/plugin.go) | `HookProvider`, `ToolProvider`, `PromptProvider`, `LLMEngineAware` | **Default plugin** — see [Brain plugin](#brain-plugin) below. |
| [Scheduler](./scheduler/plugin.go) | `HookProvider`, `ToolProvider`, `PromptProvider`, `WorkingDirAware`, `InboxAware` | One-shot task scheduling; fires reminders into the agent inbox |
| [Heartbeat](./heartbeat/plugin.go) | `HookProvider`, `PromptProvider`, `WorkingDirAware`, `InboxAware` | Interval-driven autonomous check-ins; creates default `HEARTBEAT.md` on workspace bind if missing; optional YAML `agent.heartbeat` (`every`, `prompt`, `ack_max_chars`, `active_hours`) — see [docs/agents/configuration.md](../../docs/agents/configuration.md#heartbeat-plugin-yaml) |

### Brain plugin

Long-term memory uses a SQLite graph at `brain/brain.db` (root → topic → conversation). Raw chats live in `data/conversations/<agent>/<id>.json`. `brain/MEMORY.md` holds editable daily notes.

**Dreaming** (scheduled via `agent.brain_plugin.dream` / `dreamTime`, or the `dream` tool) first distills pending JSON transcripts with the agent LLM. A transcript pass writes `brain/persistence/YYYY-MM-DD/<conv_id>.md` and updates the graph **only** when the model returns retainable JSON: non-empty `summary`, `title`, `description`, and `distillation_reason`. Those three fields are stored as SQLite columns on the conversation node; `topics` stay in metadata; `search_text` indexes summary + those fields for `find`. **Pending recategorize:** after that batch, any conversation that is dreamed (`dreamed_at` in metadata) and linked **only** to topic `pending` gets a second LLM call (title, description, distillation reason, summary) to assign 1–5 substantive topics from **CURRENT TOPICS** or new labels; graph edges are replaced so `pending` is no longer the sole topic. A further pass runs whenever `brain/MEMORY.md` exists with non-whitespace content: the model may rewrite that file into a shorter working-memory form and, **only if it explicitly chooses to**, promote **at most one** durable item into the same graph as a synthetic conversation node (`brain-memmd-` + content fingerprint in the id; metadata `source=memory_md`; optional matching persistence markdown). Long-term promotion from `MEMORY.md` is never required. New chats attach to topic `pending` until transcript distillation. Distillation prompts inject **CURRENT TOPICS** from the graph so labels prefer existing topics before new ones.

**Recall tools:** `get_conversation_topics`, `recall_recent_conversations`, `recall_older_conversation`, `retrieve_conversation`, `find`, plus `memory_read_short_term` / `memory_patch_short_term` / `save_short_term_memory` for `MEMORY.md`. **Forget:** `forget` removes a topic (cascade) or one conversation from the graph and deletes matching `brain/persistence/**/<conv_id>.md` and `data/conversations/*/<conv_id>.json`.

**Disable:** `agent.brain: false`. **Config:** [Brain plugin (YAML)](../../docs/agents/configuration.md#brain-plugin-yaml). **Spec pointer:** [memorySpec.md](../../memorySpec.md).