# How-To: Add New Tools

See [docs/TOOLS.md](../TOOLS.md) for built-in tools documentation.

## Step 1: Create Tool Package

Create directory: `src/tools/{toolname}/`

## Step 2: Implement Tool

```go
// src/tools/mytool/tool.go
package mytool

import (
    "github.com/thinktwiceco/agent-forge/src/core"
    "github.com/thinktwiceco/agent-forge/src/llms"
)

func NewMyTool(config string) llms.Tool {
    return &core.Tool{
        Name:        "my_tool",
        Description: "Brief description",
        AdvanceDesc: `Detailed instructions and examples`,
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
    }
}
```

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
