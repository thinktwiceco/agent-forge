# How-To: Create Plugins

See [src/plugins/README.md](../../src/plugins/README.md) for complete plugin documentation.

## Plugin Interface Design

Plugins use **interface segregation** - only implement what you need:

```go
// Base (required)
type Plugin interface {
    Name() string
}

// Optional: Add hooks
type HookProvider interface {
    Plugin
    Hooks() map[Event]AgentHookFn
}

// Optional: Add tools
type ToolProvider interface {
    Plugin
    Tools() []llms.Tool
}

// Optional: Add system prompt
type PromptProvider interface {
    Plugin
    SystemPrompt() string
}
```

## Create Plugin

```go
// src/plugins/myplugin/plugin.go
package myplugin

type MyPlugin struct {
    config string
}

func NewMyPlugin(config string) *MyPlugin {
    return &MyPlugin{config: config}
}

// Required: Base interface
func (p *MyPlugin) Name() string {
    return "my-plugin"
}

// Optional: Implement HookProvider
func (p *MyPlugin) Hooks() map[core.Event]core.AgentHookFn {
    return map[core.Event]core.AgentHookFn{
        core.EventToolExecution: agents.OnToolExecutionHook(p.handleToolExecution),
    }
}

func (p *MyPlugin) handleToolExecution(agent *agents.Agent, toolName string, args map[string]any, result llms.ToolReturn) error {
    // Hook logic
    return nil
}

// Optional: Implement ToolProvider
func (p *MyPlugin) Tools() []llms.Tool {
    return []llms.Tool{newPluginTool(p)}
}

// Optional: Implement PromptProvider
func (p *MyPlugin) SystemPrompt() string {
    return `[MY PLUGIN]
- Instructions for this plugin
- Use [square brackets] not <angle brackets>
- Ensures compatibility with all LLM providers`
}
```

## Register Plugin

```go
agent, err := agents.NewBuilder(llm, "agent").
    WithPlugins(myplugin.NewMyPlugin("config")).
    Build()
```

## System Prompt Best Practices

**CRITICAL: Do not use XML-style tags (`<tag>...</tag>`) in system prompts or tool descriptions.**

Some LLM providers filter or reject angle brackets. Always use square brackets instead:

✅ **Good:**
```go
func (p *MyPlugin) SystemPrompt() string {
    return `[PLUGIN NAME]
[INSTRUCTIONS]
- Step 1: Do this
- Step 2: Do that

[CONSTRAINTS]
- Never do X
- Always do Y`
}
```

❌ **Bad:**
```go
func (p *MyPlugin) SystemPrompt() string {
    return `<PLUGIN_NAME>
<INSTRUCTIONS>
- Step 1: Do this
</INSTRUCTIONS>
</PLUGIN_NAME>`
}
```

This applies to:
- System prompts (`SystemPrompt()`)
- Tool descriptions (`Description` field)
- Dynamic prompt sections

## Available Events

See [src/plugins/README.md](../../src/plugins/README.md#lifecycle-events) for the full event list. Key events:
- `EventAgentInitialization`, `EventAgentInitialized`
- `EventToolExecution`, `EventBeforeToolExecution`
- `EventNewUserMessage`, `EventNewAssistantMessage`
- `EventNewChunk`, `EventContextBuild`
- `EventAddSystemAgent`, `EventAddedSystemAgent`
- `EventNewAssistantMessageWithToolCalls`
- `EventAddedTools`
