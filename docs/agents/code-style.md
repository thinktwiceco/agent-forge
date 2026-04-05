# Code Style & Conventions

## Go Standards

```bash
# Format code (always run before committing)
go fmt ./...

# Lint (uses golangci-lint)
./scripts/lint.sh
```

## Commit Conventions

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```bash
feat(agents): add custom prompt support
fix(web): resolve session cleanup
docs(readme): update installation instructions
refactor(llms): simplify factory pattern

# Breaking changes
feat(api)!: change tool validation signature
```

**Types:** `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for complete guidelines.

## Naming Conventions

**Interfaces:**
```go
// ✅ Good: Manager, Executor, Builder (no "I" prefix)
type Manager interface { ... }
type ExecutionEngine interface { ... }
```

**Implementations:**
```go
// ✅ Good: Concrete descriptive names
type ConversationHistory struct { ... }
type SlidingWindowStrategy struct { ... }
```

**Factory Functions:**
```go
// Pattern: New{Type}()
func NewAgent(config *AgentConfig) *Agent { ... }
func NewBuilder(llm llms.LLMEngine, name string) *Builder { ... }
```

## Interface Design Principles

**Interface Segregation:**
```go
// ✅ Good: Split large interface
type Plugin interface { Name() string }
type HookProvider interface { Plugin; Hooks() map[Event]AgentHookFn }
type ToolProvider interface { Plugin; Tools() []llms.Tool }
```

**Compile-Time Assertions:**
```go
// In interfaces.go
var _ HistoryManager = (*history.ConversationHistory)(nil)
var _ ExecutionEngine = (*execution.Executor)(nil)
```

**Mock Implementations:**
Always provide mocks in `src/agents/mocks/` for testing.

## Error Handling

```go
// ✅ Library code: Return errors, don't panic
func DoSomething(input string) error {
    if err := validate(input); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    return nil
}

// ✅ Main/cmd: Can use log.Fatal for setup failures
func main() {
    if err := setup(); err != nil {
        log.Fatalf("Setup failed: %v", err)
    }
}

// ❌ Don't panic in library code
func DoSomething(input string) {
    panic("bad input") // BAD
}
```
