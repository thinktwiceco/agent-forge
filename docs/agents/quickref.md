# Quick Reference

Essential commands for working with Agent Forge.

## Build & Run

```bash
# Build entire project
go build ./...

# Run dev server
./scripts/start-dev.sh

# Run with custom config
AF_LOG_LEVEL=DEBUG go run ./cmd/localforge -config custom.yaml -port 8080
```

## Testing

```bash
# Run all tests with race detection
./scripts/test.sh --unit

# Quick single-file test (fast feedback)
go test ./src/agents/agent_test.go -v

# Test specific package
go test ./src/agents/... -v

# Coverage report
go test ./src/agents/... -cover
```

## Code Quality

```bash
# Format code
go fmt ./...

# Lint
./scripts/lint.sh
```

## Git Workflow

```bash
# Branch naming
feature/<name>    # New features
fix/<name>        # Bug fixes
docs/<name>       # Documentation
refactor/<name>   # Refactoring

# Commit format (Conventional Commits)
feat(agents): add custom prompt support
fix(web): resolve session cleanup
docs(readme): update install instructions
```

## Environment Variables

```bash
# Required for LLM access
AF_TOGETHERAI_API_KEY=your_key
AF_OPENAI_API_KEY=your_key
AF_DEEPSEEK_API_KEY=your_key
AP_OPENROUTER_API_KEY=your_key

# Logging
AF_LOG_LEVEL=INFO              # DEBUG, INFO, WARN, ERROR
AF_LOG_FILE=logs/app.log       # Optional
```
