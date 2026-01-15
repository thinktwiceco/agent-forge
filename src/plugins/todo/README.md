# Todo Plugin

A plugin that provides task management capabilities to agents through a `todo_handler` tool. The plugin allows agents to create, track, and manage todo items, making it ideal for multi-step processes, planning, and task breakdown.

## Features

- **Add Todo Items**: Create individual todo items with title and description
- **Bulk Operations**: Add multiple todo items at once
- **Status Tracking**: Mark todos as completed or incomplete
- **Retrieve Todos**: Get all todo items in JSON format
- **Update Callbacks**: Receive notifications when todos are updated
- **Flexible Identification**: Update todos by ID or title

## How to Use

### Basic Setup

Add the todo plugin to your agent configuration:

```go
import "github.com/thinktwice/agentForge/src/plugins/todo"

// Optional: Define a callback function to be notified when todos are updated
onTodoUpdate := func(todos []*todo.TodoItem) {
    fmt.Printf("Todo list updated: %d items\n", len(todos))
    for _, item := range todos {
        status := "✓"
        if !item.Completed {
            status = "○"
        }
        fmt.Printf("  %s %s\n", status, item.Title)
    }
}

// Create plugin instance
todoPlugin := todo.NewTodoPlugin(onTodoUpdate)

// Add to agent config
config := agents.AgentConfig{
    LLMEngine: llmEngine,
    AgentName: "Assistant",
    Plugins:   []core.Plugin{todoPlugin},
}

agent := agents.NewAgent(&config)
```

### Without Update Callback

If you don't need update notifications, pass `nil`:

```go
todoPlugin := todo.NewTodoPlugin(nil)
```

## Todo Handler Tool

The plugin provides a `todo_handler` tool with the following actions:

### Actions

1. **`addTodo`** - Add a single todo item
2. **`addBulkTodos`** - Add multiple todo items at once
3. **getTodos** - Retrieve all todo items
4. **updateTodo** - Update the completion status of a todo item

### Tool Parameters

- **`action`** (required, string): The action to perform: `"addTodo"`, `"addBulkTodos"`, `"getTodos"`, or `"updateTodo"`
- **`title`** (optional, string): The title of the todo item
- **`description`** (optional, string): The description of the todo item
- **`id`** (optional, string): The ID of the todo item (for updates)
- **`completed`** (optional, boolean): The completion status (for updates)
- **`todos`** (optional, array): Array of todo items for bulk operations

## Usage Examples

### Example 1: Adding a Single Todo

```go
// The agent can use the tool like this:
todo_handler(action="addTodo", title="Review documentation", description="Check README for accuracy")
```

### Example 2: Adding Multiple Todos

```go
// Bulk add todos:
todo_handler(
    action="addBulkTodos",
    todos=[
        {"title": "Step 1", "description": "Gather requirements"},
        {"title": "Step 2", "description": "Design solution"},
        {"title": "Step 3", "description": "Implement changes"}
    ]
)
```

### Example 3: Getting All Todos

```go
// Retrieve all todos:
todo_handler(action="getTodos")
// Returns: JSON array of all todo items
```

### Example 4: Updating Todo Status by ID

```go
// Mark a todo as completed using its ID:
todo_handler(action="updateTodo", id="uuid-here", completed=true)
```

### Example 5: Updating Todo Status by Title

```go
// Mark a todo as completed using its title:
todo_handler(action="updateTodo", title="Review documentation", completed=true)
```

## TodoItem Structure

Each todo item has the following structure:

```go
type TodoItem struct {
    ID          string    `json:"id"`          // Unique identifier (UUID)
    Title       string    `json:"title"`       // Todo title
    Description string    `json:"description"` // Todo description
    Completed   bool      `json:"completed"`   // Completion status
    CreatedAt   time.Time `json:"createdAt"`   // Creation timestamp
    UpdatedAt   time.Time `json:"updatedAt"`   // Last update timestamp
}
```

## System Prompt Integration

The plugin automatically enhances the agent's system prompt with instructions:

```
Use the todo_handler tool in combination with the reasoning agent.
Create a todo list based on the user request. You can then complete each task
one by one and check them as completed!

Use the bulk action to add multiple todo items at once.
Use it after consulting with the reasoning agent, if present.
```

## Use Cases

### Multi-Step Process Planning

When an agent needs to break down a complex task:

1. Agent receives a complex request
2. Uses reasoning agent to break it down into steps
3. Creates todo items for each step using `addBulkTodos`
4. Executes each step and marks todos as completed
5. Reports progress to the user

### Task Tracking

Track progress through long-running operations:

```go
// Agent workflow:
1. todo_handler(action="addTodo", title="Fetch data", description="Get data from API")
2. // ... perform fetch ...
3. todo_handler(action="updateTodo", title="Fetch data", completed=true)
4. todo_handler(action="addTodo", title="Process data", description="Transform and validate")
5. // ... process data ...
6. todo_handler(action="updateTodo", title="Process data", completed=true)
```

### Workflow Management

Coordinate multiple sub-tasks:

```go
// Create initial todo list
todo_handler(action="addBulkTodos", todos=[
    {"title": "Validate input", "description": "Check user input format"},
    {"title": "Query database", "description": "Fetch relevant records"},
    {"title": "Generate report", "description": "Create summary document"},
    {"title": "Send notification", "description": "Notify user of completion"}
])

// As each step completes, update status
todo_handler(action="updateTodo", title="Validate input", completed=true)
```

## Update Callbacks

The `onTodoUpdate` callback function is called whenever a todo item is modified through the `todo_handler` tool. This allows you to:

- Display todo list updates in real-time
- Persist todos to external storage
- Trigger notifications or alerts
- Update UI components

```go
onTodoUpdate := func(todos []*todo.TodoItem) {
    // Save to database
    saveTodosToDB(todos)
    
    // Update UI
    updateTodoListUI(todos)
    
    // Send notification if all completed
    if allCompleted(todos) {
        sendCompletionNotification()
    }
}
```

## Integration with Reasoning Agent

The plugin is designed to work seamlessly with the reasoning sub-agent:

1. User asks a complex question
2. Main agent delegates to reasoning agent for planning
3. Reasoning agent breaks down the problem
4. Main agent creates todo items based on the plan
5. Main agent executes tasks and updates todos

```go
config := agents.AgentConfig{
    LLMEngine: llmEngine,
    AgentName: "Assistant",
    Reasoning: true,  // Enable reasoning agent
    Plugins:   []core.Plugin{todoPlugin},
}
```

## Error Handling

The tool returns appropriate error messages for:

- Missing required parameters
- Invalid action names
- Todo items not found (for updates)
- Invalid data types
- Empty arrays for bulk operations

Example error responses:
- `"action parameter is required and must be a string"`
- `"todo item not found"`
- `"todos array cannot be empty"`
- `"unknown action: invalid_action"`

## Best Practices

1. **Use Bulk Operations**: When creating multiple todos, use `addBulkTodos` instead of multiple `addTodo` calls
2. **Combine with Reasoning**: Use the reasoning agent to plan before creating todos
3. **Update Progress**: Regularly update todo status as tasks complete
4. **Use Descriptive Titles**: Make todo titles clear and actionable
5. **Track Updates**: Implement the callback function to monitor todo changes

## Complete Example

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/thinktwice/agentForge/src/agents"
    "github.com/thinktwice/agentForge/src/core"
    "github.com/thinktwice/agentForge/src/llms"
    "github.com/thinktwice/agentForge/src/plugins/logger"
    "github.com/thinktwice/agentForge/src/plugins/todo"
)

func main() {
    // Initialize LLM
    llm, _ := llms.NewOpenAILLMBuilder("togetherai").
        SetModel(llms.TOGETHERAI_Llama3170BInstructTurbo).
        Build()
    
    // Todo update callback
    onTodoUpdate := func(todos []*todo.TodoItem) {
        fmt.Println("\n=== Todo List Update ===")
        for _, item := range todos {
            status := "○"
            if item.Completed {
                status = "✓"
            }
            fmt.Printf("%s %s: %s\n", status, item.Title, item.Description)
        }
    }
    
    // Create plugins
    loggerPlugin := logger.NewPlugin(
        logger.DefaultColorRules(),
        logger.DefaultLabelRules(),
        os.Stdout,
    )
    todoPlugin := todo.NewTodoPlugin(onTodoUpdate)
    
    // Create agent
    agent := agents.NewAgent(&agents.AgentConfig{
        LLMEngine:   llm,
        AgentName:   "TaskManager",
        Description: "An agent that manages tasks using todo lists",
        Reasoning:   true,
        MainAgent:   true,
        Plugins:     []core.Plugin{loggerPlugin, todoPlugin},
    })
    
    // Use the agent
    responseCh := agent.ChatStream("Plan and execute a code review process")
    
    for chunk := range responseCh.Start() {
        if chunk.Content != "" {
            fmt.Print(chunk.Content)
        }
    }
}
```

