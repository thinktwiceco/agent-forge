# Expand Tool

## Overview

The `Expand` tool enables progressive discovery of detailed information about tools and sub-agents. It allows agents to retrieve `AdvanceDescription` and `Troubleshooting` information on-demand, keeping system prompts concise while providing access to detailed documentation when needed.

## Purpose

- **Concise System Prompts**: Only basic descriptions are included in system prompts
- **On-Demand Discovery**: Agents can request detailed information when they need it
- **Troubleshooting Support**: Access to debugging information when issues arise
- **Self-Documentation**: Tools and agents can be fully self-documenting

## Tool Signature

```go
expand(
    subject_type: "tool" | "agent",
    subject_name: string,
    troubleshoot: bool = false
) -> string
```

### Parameters

- **subject_type** (string, required): The type of subject to expand
  - `"tool"`: Expand information about a tool
  - `"agent"`: Expand information about a sub-agent

- **subject_name** (string, required): The exact name of the tool or agent
  - Must match the name exactly (case-sensitive)
  - For tools: Use the name returned by `GetName()`
  - For agents: Use the agent name from configuration

- **troubleshoot** (boolean, optional): Whether to include troubleshooting information
  - Default: `false`
  - When `true`: Includes the `Troubleshooting()` section
  - When `false`: Only includes basic and advanced descriptions

### Returns

A formatted string containing:
- **Basic Description**: One-line overview (always included)
- **Advanced Description**: Detailed capabilities and usage (always included)
- **Troubleshooting**: Common issues and solutions (only if `troubleshoot=true`)

## Usage Examples

### Example 1: Expanding a Tool

```go
// Agent context automatically includes tools
result := expandTool.Call(agentContext, map[string]any{
    "subject_type": "tool",
    "subject_name": "foo",
    "troubleshoot": false,
})

// Output:
// === TOOL: foo ===
//
// 📄 Basic Description:
// A test tool that returns the echo argument. Use this to test only.
//
// 📚 Advanced Description:
// Advanced Details:
// - Parameters: 
//   * echo (string, required): The text to echo back
// - Behavior: Returns the input string unchanged
// - Usage: Primarily for testing tool invocation and response handling
// - Performance: Instant response with no side effects
```

### Example 2: Expanding with Troubleshooting

```go
result := expandTool.Call(agentContext, map[string]any{
    "subject_type": "tool",
    "subject_name": "foo",
    "troubleshoot": true,
})

// Output includes additional troubleshooting section:
// 🔧 Troubleshooting:
// - If the tool fails, ensure the 'echo' parameter is provided as a string
// - Empty strings are valid and will be returned as-is
// - This tool has no external dependencies and should always succeed if called correctly
```

### Example 3: Expanding a Sub-Agent

```go
result := expandTool.Call(agentContext, map[string]any{
    "subject_type": "agent",
    "subject_name": "system-reasoning",
    "troubleshoot": true,
})

// Output:
// === AGENT: system-reasoning ===
//
// 📄 Basic Description:
// Breaks down complex problems into logical steps
//
// 📚 Advanced Description:
// Advanced Details:
// - Purpose: Breaks down complex problems into logical, actionable steps
// - Reasoning Style: Uses 🔎 markers to show thought process
// ...
//
// 🔧 Troubleshooting:
// - "Rejection response": If the agent rejects your query, ensure you're asking a "how to" question
// ...
```

## Integration

### Adding to an Agent

```go
agent := agents.NewAgent(agents.AgentConfig{
    LLMEngine: llm,
    AgentName: "my-agent",
    Tools: []llms.Tool{
        tools.NewFooTool(),
        tools.NewExpandTool(),  // Add the expand tool
        // ... other tools
    },
})
```

### How It Works

1. **Agent Context**: The expand tool receives `agentContext` which includes:
   - `tools`: List of available tools
   - `subAgents`: List of available sub-agents

2. **Discovery**: The tool searches for the requested subject by name

3. **Casting**: The tool casts the found subject to `Discoverable` interface

4. **Formatting**: Returns formatted information with emoji markers

## Agent Context Requirements

The expand tool requires the following in `agentContext`:

```go
agentContext := map[string]any{
    "tools":     []llms.Tool{...},      // Available tools
    "subAgents": []core.SubAgent{...}, // Available sub-agents
}
```

**Note**: The agent framework automatically provides this context when executing tools.

## Error Handling

### Tool Not Found

```
Error: Tool 'nonexistent' not found in context. 
Available tools can be seen in your system prompt or by listing your tools.
```

### Agent Not Found

```
Error: Agent 'nonexistent' not found in context. 
Available agents are listed in your system prompt under [SUB AGENTS].
```

### Invalid Subject Type

```
Error: Invalid subject_type 'invalid'. Must be either 'tool' or 'agent'
```

## Use Cases

### 1. Learning About Tools

An agent can discover what a tool does before using it:

```
Agent: "I need to calculate something. Let me first understand the calculator tool."
Action: expand(subject_type="tool", subject_name="calculator", troubleshoot=false)
```

### 2. Troubleshooting Failures

When a tool fails, the agent can get troubleshooting information:

```
Agent: "The calculator tool failed. Let me check the troubleshooting guide."
Action: expand(subject_type="tool", subject_name="calculator", troubleshoot=true)
```

### 3. Understanding Sub-Agents

Before delegating, an agent can learn about a sub-agent's capabilities:

```
Agent: "Should I delegate this to the reasoning agent? Let me check its capabilities."
Action: expand(subject_type="agent", subject_name="system-reasoning", troubleshoot=false)
```

### 4. Self-Documentation

The expand tool can even expand itself:

```
Action: expand(subject_type="tool", subject_name="expand", troubleshoot=true)
```

## Best Practices

1. **Start Without Troubleshooting**: Only request troubleshooting when needed
2. **Cache Information**: If an agent expands a tool once, it has the information
3. **Use for Complex Tools**: Simple tools may not need expansion
4. **Combine with Delegation**: Expand sub-agents before delegating complex tasks

## Implementation Details

### Discoverable Interface

Both tools and agents must implement the `Discoverable` interface:

```go
type Discoverable interface {
    BasicDescription() string
    AdvanceDescription() string
    Troubleshooting() string
}
```

### Agent Implementation

Agents directly implement the `core.SubAgent` interface which includes `Discoverable` methods:

```go
// Agent implements core.SubAgent interface
func (a *Agent) ChatStream(message string) core.IResponseChStarter { ... }
func (a *Agent) Name() string { return a.agentName }
func (a *Agent) BasicDescription() string { return a.description }
func (a *Agent) AdvanceDescription() string { return a.advanceDescription }
func (a *Agent) Troubleshooting() string { return a.troubleshooting }
```

No adapter is needed - agents can be used directly as sub-agents.

## Testing

Run the expand tool tests:

```bash
go test ./src/tools -v -run TestExpandTool
```

Run the example:

```bash
go run examples/expand_tool_example.go
```

## See Also

- [Discoverable Interface Documentation](DISCOVERABLE.md)
- [Tool Development Guide](../README.md)
- [Agent Configuration Guide](CONFIG.md)

