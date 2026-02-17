# Testing Guidelines

## Test Commands

```bash
# Run all tests with race detection
./scripts/test.sh --unit

# Run specific package
go test ./src/agents/... -v

# Quick single-file test (fast feedback)
go test ./src/agents/agent_test.go -v

# Coverage
go test ./src/agents/... -cover

# HTML coverage report
go test ./src/agents/... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Test specific function
go test ./src/agents/... -run TestAgent_ChatStream -v
```

## Testing Patterns

**Test Files:** `*_test.go` in same package

**Test Function Names:** `TestFunctionName_Scenario`

**Table-Driven Tests:**
```go
func TestAgent_AddTools(t *testing.T) {
    tests := []struct {
        name      string
        tools     []llms.Tool
        wantCount int
    }{
        {"single tool", []llms.Tool{tool1}, 1},
        {"multiple tools", []llms.Tool{tool1, tool2}, 2},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Using Mocks

```go
import "github.com/thinktwiceco/agent-forge/src/agents/mocks"

func TestAgent_Feature(t *testing.T) {
    // Create mocks
    mockExecutor := mocks.NewMockExecutionEngine()
    mockHistory := mocks.NewMockHistoryManager()
    
    // Configure behavior
    mockExecutor.ExecuteFunc = func(...) error {
        return nil
    }
    
    // Use in test
}
```

Available mocks in `src/agents/mocks/`:
- `MockExecutionEngine`
- `MockHistoryManager`
- `MockPromptBuilder`
- `MockContextManager`
