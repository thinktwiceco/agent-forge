# Common Issues & Solutions

## Import Cycle Detected

**Symptoms:** `import cycle not allowed`

**Solution:** Check `src/agents` vs `src/agents/system` imports. System package should NOT import agents.

```go
// ❌ BAD: system importing agents
package system
import "github.com/thinktwiceco/agent-forge/src/agents"

// ✅ GOOD: system is self-contained
package system
// No agents import
```

## Interface Not Satisfied

**Symptoms:** `does not implement Interface (missing method)`

**Solution:** Use compile-time assertion:

```go
// In interfaces.go
var _ history.Manager = (*ConversationHistory)(nil)
var _ ExecutionEngine = (*execution.Executor)(nil)
```

## Nil Pointer Dereference

**Symptoms:** `panic: runtime error: invalid memory address`

**Solution:** Check initialization order. Example: PromptBuilder must be initialized before plugins.

```go
// ✅ Initialize promptBuilder BEFORE registerPlugins
func NewAgent(config *AgentConfig) *Agent {
    a := &Agent{config: config}
    a.promptBuilder = prompts.NewBuilder(...)
    a.registerPlugins()  // Now safe
    return a
}
```

## Context Window Overflow

**Symptoms:** LLM errors about token limit exceeded

**Solution:** Enable automatic truncation:

```go
agent, err := agents.NewBuilder(llm, "agent").
    WithContextWindow(128000).  // Enables truncation
    Build()
```

## Plugin Not Working

**Symptoms:** Hooks not called, tools not available

**Solution:** Implement correct interfaces:

```go
// For hooks: implement HookProvider
func (p *MyPlugin) Hooks() map[core.Event]core.AgentHookFn { ... }

// For tools: implement ToolProvider
func (p *MyPlugin) Tools() []llms.Tool { ... }
```

## Tool Validation Failing

**Symptoms:** Tool execution fails with validation errors

**Solution:** Implement proper parameter validation:

```go
Parameters: []core.Parameter{
    {
        Name:     "operation",
        Required: true,
        Validator: func(value any) error {
            str, ok := value.(string)
            if !ok || str == "" {
                return fmt.Errorf("operation must be non-empty string")
            }
            return nil
        },
    },
},
```
