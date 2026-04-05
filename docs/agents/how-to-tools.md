# How-To: Add New Tools

See [docs/TOOLS.md](../TOOLS.md) for built-in tools documentation.

## Step 1: Create Tool Package

Create directory: `src/tools/{toolname}/`

## Step 2: Implement Tool

```go
// src/tools/mytool/tool.go
package mytool

import (
    "fmt"

    "github.com/thinktwiceco/agent-forge/src/core"
    "github.com/thinktwiceco/agent-forge/src/llms"
)

func NewMyTool(config string) llms.Tool {
    return core.NewTool(core.ToolConfig{
        Name:        "my_tool",
        Description: "Brief description",
        AdvanceDesc: `Detailed instructions and examples`,
        TroubleshootingInfo: `Common issues and solutions`,
        // Optional: called by the expand tool to get per-item details
        DetailsAboutFunc: func(item string) string {
            return fmt.Sprintf("Details about %s: ...", item)
        },
        Parameters: []core.Parameter{
            {
                Name:        "operation",
                Type:        "string",
                Description: "Operation to perform",
                Required:    true,
                Validator:   validateOperation,
            },
        },
        Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
            operation := args["operation"].(string)
            
            result, err := performOperation(operation)
            if err != nil {
                return core.NewErrorResponse(err.Error())
            }
            
            return core.NewSuccessResponse(result)
        },
    })
}
```

Keep `Description` brief. It is used in provider tool metadata, while richer guidance should live in `AdvanceDesc`, `TroubleshootingInfo`, and `DetailsAboutFunc` so the always-on prompt can stay lean.

## Step 3: Add Tests

```go
// src/tools/mytool/tool_test.go
func TestMyTool_Operation(t *testing.T) {
    tool := NewMyTool("config")
    // Test implementation
}
```

## Step 4: Use Tool

```go
// Using Builder
agent, err := agents.NewBuilder(llm, "agent").
    WithTools(mytool.NewMyTool("config")).
    Build()
```

## Response Types

```go
core.NewSuccessResponse(result)           // Success with data
core.NewErrorResponse("error")            // Error
core.NewFailureResponse(err, partial)     // Error with partial data
core.NewEphemeralResponse(result)         // Success, don't store in history
```

## Hooks (Optional)

Implement `core.Hooks` to add path or command sandboxing:

```go
type core.Hooks interface {
    IsSafePath(path string) bool
    IsSafeCommand(cmd string) bool
}

tool.(*core.Tool).SetHooks(myHooks)
```

See [TOOLS.md](../TOOLS.md#hooks) for details.
