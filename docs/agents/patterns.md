# Important Patterns

## Error Handling

```go
// ✅ Library code: Return errors
func DoSomething(input string) error {
    if err := validate(input); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    return nil
}

// ✅ Main/cmd: Can use log.Fatal
func main() {
    if err := setup(); err != nil {
        log.Fatalf("Setup failed: %v", err)
    }
}

// ❌ Don't panic in library code
```

## Interfaces Over Concrete Types

```go
// ✅ Good: Accept interface
func ProcessHistory(h history.Manager) error {
    // Implementation
}

// ❌ Avoid: Concrete type (tight coupling)
func ProcessHistory(h *history.ConversationHistory) error {
    // Implementation
}
```

## Factory Functions

```go
// ✅ Use New{Type}() pattern
tool := fs.NewFsTool("/path")
agent := agents.NewAgent(config)

// ✅ Builder for fluent construction
agent, err := agents.NewBuilder(llm, "name").
    WithTools(tool).
    Build()
```

## Nil Checks

```go
// ✅ Check before dereferencing
func ProcessAgent(a *agents.Agent) error {
    if a == nil {
        return fmt.Errorf("agent cannot be nil")
    }
    return nil
}

// ✅ Safe type assertions
if str, ok := value.(string); ok {
    // Use str safely
}
```

## Context Usage

```go
// ✅ Pass context as first parameter
func DoWork(ctx context.Context, data string) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    // Do work
    return nil
}
```
