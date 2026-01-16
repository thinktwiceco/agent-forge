package todo

import (
	"encoding/json"
	"fmt"

	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

func newTodoHandlerTool(plugin *TodoPlugin) llms.Tool {
	return &core.Tool{
		Name: TODO_HANDLER_TOOL,
		Description: `
A tool that allows you to add, get, and update todo items. Use this tool to keep track of tasks and make sure you complete everything that has been requested
Expand on this tool to have a better description of how to use it.
Use it to keep track of your work in a multi step process, or a planning process, or anything
that can be broken down into smaller steps.
`,
		AdvanceDesc: `Advanced Details:
- Actions:
  * addTodo: Adds a new todo item,
  * addBulkTodos: Adds multiple todo items at once
  * getTodos: Gets all todo items
  * updateTodo: Updates the status of a todo item (can use either 'id' or 'title' to identify the item)

- Example:
You are asked to calculate the surface of a rectangle.
Your reasoning agents tells you that the steps are:
- Require the length to the user if not provided
- Require the width to the user if not provided
- Calculate the surface using the formula: length * width
- Return the result to the user

You can then add to your todo list the steps:
- Acquire the lenght
- Acquire the width
- Calculate the surface
- Return the result

While you proceed, you can use the action "update_todo" to update the status
of the todo items.
`,
		TroubleshootingInfo: `Troubleshooting:
- If the tool fails, ensure the 'action' parameter is provided as a string
- Empty strings are valid and will be returned as-is`,
		Parameters: []core.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "The action to perform: 'addTodo', 'addBulkTodos', 'getTodos', or 'updateTodo'",
				Required:    true,
			},
			{
				Name:        "title",
				Type:        "string",
				Description: "The title of the todo item (required for 'add_todo', optional for 'update_todo' - can be used instead of 'id')",
				Required:    false,
			},
			{
				Name:        "description",
				Type:        "string",
				Description: "The description of the todo item",
				Required:    false,
			},
			{
				Name:        "id",
				Type:        "string",
				Description: "The ID of the todo item (optional, can use 'title' instead)",
				Required:    false,
			},
			{
				Name:        "completed",
				Type:        "boolean",
				Description: "The completion status of the todo item",
				Required:    false,
			},
			{
				Name:        "todos",
				Type:        "array",
				Description: "Array of todo items to add. Each item should be an object with 'title' (string) and 'description' (string) fields",
				Required:    false,
				Items: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type":        "string",
							"description": "The title of the todo item",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "The description of the todo item",
						},
					},
					"required": []string{"title"},
				},
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			action, ok := args["action"].(string)
			if !ok {
				return core.NewErrorResponse("action parameter is required and must be a string")
			}

			switch action {
			case "addTodo":
				title, _ := args["title"].(string)
				description, _ := args["description"].(string)
				err := plugin.addTodoItem(title, description)
				if err != nil {
					return core.NewErrorResponse(fmt.Sprintf("failed to add todo: %v", err))
				}
				return core.NewSuccessResponse("Todo item added successfully")

			case "addBulkTodos":
				todosArray, ok := args["todos"].([]any)
				if !ok {
					return core.NewErrorResponse("todos parameter is required for addBulkTodos action and must be an array")
				}
				if len(todosArray) == 0 {
					return core.NewErrorResponse("todos array cannot be empty")
				}

				bulkTodos := make([]map[string]string, 0, len(todosArray))

				for i, todoItem := range todosArray {
					todoMap, ok := todoItem.(map[string]any)
					if !ok {
						return core.NewErrorResponse(fmt.Sprintf("todo item at index %d must be an object", i))
					}

					title, ok := todoMap["title"].(string)
					if !ok {
						return core.NewErrorResponse(fmt.Sprintf("todo item at index %d must have a 'title' field of type string", i))
					}

					description, _ := todoMap["description"].(string)
					bulkTodos = append(bulkTodos, map[string]string{
						"title":       title,
						"description": description,
					})
				}

				err := plugin.addBulkTodos(bulkTodos)
				if err != nil {
					return core.NewErrorResponse(fmt.Sprintf("failed to add bulk todos: %v", err))
				}
				return core.NewSuccessResponse(fmt.Sprintf("Successfully added %d todo items", len(bulkTodos)))

			case "getTodos":
				items := plugin.getTodoItems()
				itemsJSON, err := json.Marshal(items)
				if err != nil {
					return core.NewErrorResponse(fmt.Sprintf("failed to serialize todos: %v", err))
				}
				return core.NewEphemeralResponse(string(itemsJSON))

			case "updateTodo":
				completed, ok := args["completed"].(bool)
				if !ok {
					return core.NewErrorResponse("completed parameter is required for update_todo action and must be a boolean")
				}

				// Try to update by ID first, then by title
				if id, ok := args["id"].(string); ok && id != "" {
					err := plugin.updateTodoStatus(id, completed)
					if err != nil {
						return core.NewErrorResponse(fmt.Sprintf("failed to update todo: %v", err))
					}
					return core.NewSuccessResponse("Todo item updated successfully")
				}

				// If no ID provided, try to update by title
				if title, ok := args["title"].(string); ok && title != "" {
					err := plugin.updateTodoStatusByTitle(title, completed)
					if err != nil {
						return core.NewErrorResponse(fmt.Sprintf("failed to update todo: %v", err))
					}
					return core.NewSuccessResponse("Todo item updated successfully")
				}

				return core.NewErrorResponse("either 'id' or 'title' parameter is required for update_todo action")

			default:
				return core.NewErrorResponse(fmt.Sprintf("unknown action: %s. Valid actions are: 'add_todo', 'addBulkTodos', 'get_todos', or 'update_todo'", action))
			}
		},
	}
}
