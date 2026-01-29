# Logger Plugin

Configurable output formatting for agent streaming responses with color and label rules.

## How to Use

Add the logger plugin to your agent configuration:

```go
import "github.com/thinktwiceco/agent-forge/src/plugins/logger"

// Create plugin with default rules
loggerPlugin := logger.NewPlugin(
    logger.DefaultColorRules(),
    logger.DefaultLabelRules(),
    os.Stdout,
)

// Add to agent config
config := agents.AgentConfig{
    LLMEngine: llmEngine,
    AgentName: "Assistant",
    Plugins:   []core.Plugin{loggerPlugin},
}

agent := agents.NewAgent(&config)
```

## How to Configure

### Color Rules

Define colors based on agent name and trace patterns:

```go
colorRules := []logger.ColorRule{
    {
        AgentNamePattern: "reasoning",
        TracePattern:     "",
        Color:            "\033[33m", // Yellow
    },
    {
        AgentNamePattern: "",
        TracePattern:     "thinking",
        Color:            "\033[33m",
    },
}
```

### Label Rules

Define labels with emojis and formatting:

```go
labelRules := []logger.LabelRule{
    {
        AgentNamePattern: "reasoning",
        TracePattern:     "",
        Emoji:            "🧠",
        IsSubAgent:       false,
        Format:           "%s",
    },
    {
        AgentNamePattern: "",
        TracePattern:     "",
        Emoji:            "→",
        IsSubAgent:       true,
        Format:           "%s",
    },
}
```

### Output Writer

Set custom output destination:

```go
// Write to file
file, _ := os.Create("output.log")
loggerPlugin := logger.NewPlugin(colorRules, labelRules, file)

// Write to stderr
loggerPlugin := logger.NewPlugin(colorRules, labelRules, os.Stderr)

// Disable output
loggerPlugin := logger.NewPlugin(colorRules, labelRules, nil)
```

## Examples

### Default Configuration

```go
loggerPlugin := logger.NewPlugin(
    logger.DefaultColorRules(),
    logger.DefaultLabelRules(),
    os.Stdout,
)
```

### Custom Colors

```go
colorRules := []logger.ColorRule{
    {AgentNamePattern: "assistant", Color: "\033[36m"}, // Cyan
    {TracePattern: "reasoning", Color: "\033[33m"},      // Yellow
}

loggerPlugin := logger.NewPlugin(colorRules, logger.DefaultLabelRules(), os.Stdout)
```

### File Logging

```go
file, _ := os.OpenFile("agent.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
loggerPlugin := logger.NewPlugin(
    logger.DefaultColorRules(),
    logger.DefaultLabelRules(),
    file,
)
```


