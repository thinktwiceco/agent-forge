# Discoverable Interface

## Overview

The `Discoverable` interface enables progressive discovery of Agents and Tools through three levels of information disclosure:

1. **BasicDescription**: A short one-line description
2. **AdvanceDescription**: Detailed capabilities, parameters, and usage patterns
3. **Troubleshooting**: Common issues, debugging tips, and configuration guidance

## Interface Definition

```go
type Discoverable interface {
    BasicDescription() string
    AdvanceDescription() string
    Troubleshooting() string
}
```

## Implementation

### Agents

Agents implement the `Discoverable` interface through the `AgentConfig`:

```go
config := agents.AgentConfig{
    LLMEngine:   engine,
    AgentName:   "my-agent",
    Description: "Basic one-line description",
    AdvanceDescription: `Detailed information:
- Capabilities
- Usage patterns
- Performance characteristics`,
    Troubleshooting: `Common issues:
- Issue 1 and solution
- Issue 2 and solution`,
    SystemPrompt: "You are a helpful assistant",
}

agent := agents.NewAgent(config)

// Access descriptions
basic := agent.BasicDescription()       // Returns Description field
advanced := agent.AdvanceDescription()  // Returns AdvanceDescription field
troubleshooting := agent.Troubleshooting() // Returns Troubleshooting field
```

### Tools

Tools implement the `Discoverable` interface through the universal tool constructor:

```go
tool := core.NewTool(
    "tool-name",
    "Basic one-line description",
    "Advanced description with detailed parameters and usage",
    "Troubleshooting information and common issues",
    []core.Parameter{
        {Name: "param1", Type: "string", Required: true},
    },
    handlerFunc,
)

// Access descriptions
basic := tool.BasicDescription()       // Returns basic description
advanced := tool.AdvanceDescription()  // Returns advanced description
troubleshooting := tool.Troubleshooting() // Returns troubleshooting info
```

## System Prompt Integration

**Important**: Only `BasicDescription()` is injected into the system prompt for both sub-agents and tools.

### Sub-Agents in System Prompt

When a main agent has sub-agents, only their basic descriptions are included:

```
=== SUB AGENTS ===
📌 reasoning-agent: Breaks down complex problems into logical steps
📌 calculator-agent: Performs mathematical calculations
```

The advanced descriptions and troubleshooting information are **not** included in the system prompt to keep it concise.

### Tools in Function Definitions

When tools are sent to the LLM via the OpenAI API, only the basic description is included in the function definition:

```json
{
  "name": "foo",
  "description": "A test tool that returns the echo argument",
  "parameters": { ... }
}
```

## Progressive Discovery Pattern

The intended use case is for a special discovery tool that allows agents to progressively learn more about other agents or tools:

```go
// Level 1: Quick overview
description := agent.BasicDescription()

// Level 2: Detailed information (if needed)
if needMoreInfo {
    details := agent.AdvanceDescription()
}

// Level 3: Troubleshooting (if issues arise)
if hasIssue {
    help := agent.Troubleshooting()
}
```

## Examples

### Example Agent with All Descriptions

```go
config := agents.AgentConfig{
    LLMEngine:   llm,
    AgentName:   "data-analyzer",
    Description: "Analyzes data and generates insights",
    AdvanceDescription: `Advanced Details:
- Supports CSV, JSON, and Parquet formats
- Can handle datasets up to 1GB
- Provides statistical analysis and visualizations
- Integrates with pandas and numpy
- Streaming support for large datasets`,
    Troubleshooting: `Common Issues:
- Memory errors: Reduce dataset size or use streaming mode
- Format errors: Ensure data is properly formatted
- Performance: Use sampling for initial analysis
- Dependencies: Requires pandas>=1.5.0`,
    SystemPrompt: "You analyze data and provide insights",
    MainAgent:    false,
}
```

### Example Tool with All Descriptions

```go
calculatorTool := core.NewTool(
    "calculate",
    "Performs mathematical calculations",
    `Advanced Details:
- Supports basic operations: +, -, *, /
- Handles floating point numbers
- Expression format: "2 + 2" or "10 * 5"
- Returns numeric result as string`,
    `Troubleshooting:
- Invalid expression: Check syntax (e.g., "2+2" not "2 plus 2")
- Division by zero: Returns error message
- Large numbers: May lose precision beyond 15 digits`,
    []core.Parameter{
        {Name: "expression", Type: "string", Required: true},
    },
    calculatorHandler,
)
```

## System Agent Templates

The `SystemAgentTemplate` also supports discoverable descriptions:

```go
template := agents.NewSystemAgentTemplate("agent-name", "trace")
template.
    AddSystemPrompt(incipit, steps, output, examples, critical).
    AddDescription(incipit, examples).
    AddAdvanceDescription("Detailed capabilities...").
    AddTroubleshooting("Common issues...")

config := template.ToAgentConfig(llmEngine)
agent := agents.NewAgent(config)
```

## Testing

Tests verify that only BasicDescription is used in system prompts:

```bash
go test ./src/agents -v -run TestAgent_BasicDescriptionInSystemPrompt
```

The test confirms:
- ✅ BasicDescription is present in system prompt
- ✅ AdvanceDescription is NOT in system prompt
- ✅ Troubleshooting is NOT in system prompt

## Design Rationale

1. **Concise System Prompts**: Including only basic descriptions keeps system prompts short and focused
2. **Progressive Discovery**: Agents can request more details only when needed
3. **Separation of Concerns**: Basic info for LLM, detailed info for discovery tools
4. **Backward Compatible**: Existing code works without changes; new fields are optional

