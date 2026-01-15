package todo

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/thinktwice/agentForge/src/agents"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

const (
	PLUGIN_NAME       = "todo"
	TODO_HANDLER_TOOL = "todo_handler"
)

type TodoItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type TodoPlugin struct {
	items        []*TodoItem
	onTodoUpdate func(todos []*TodoItem)
}

func (p *TodoPlugin) Name() string {
	return PLUGIN_NAME
}

func (p *TodoPlugin) On(event core.Event) core.AgentHookFn {
	switch event {
	case core.EventToolExecution:
		return agents.OnToolExecutionHook(p.handleToolExecution)
	}
	return nil
}
func (p *TodoPlugin) handleToolExecution(a *agents.Agent, toolResult *llms.ToolResult) error {
	// Only handle todo_handler tool executions
	if toolResult.ToolName != TODO_HANDLER_TOOL {
		return nil
	}

	// Get the todo items
	todoItems := p.getTodoItems()

	if p.onTodoUpdate != nil {
		p.onTodoUpdate(todoItems)
	}

	return nil
}

func (p *TodoPlugin) Tools() []llms.Tool {
	return []llms.Tool{
		newTodoHandlerTool(p),
	}
}

func (p *TodoPlugin) SystemPrompt() string {
	return `
	Use the todo_handler tool in combination with the reasoning agent.
	Create a todo list based on the user request. You can then complete each task
	one by one and check them as completed!

	Use the bulk action to add multiple todo items at once.
	Use it after consulting with the reasoning agent, if present.
	`
}

func NewTodoPlugin(onTodoUpdate func(todos []*TodoItem)) *TodoPlugin {
	return &TodoPlugin{
		items:        make([]*TodoItem, 0),
		onTodoUpdate: onTodoUpdate,
	}
}

func (p *TodoPlugin) addTodoItem(title, description string) error {

	id := uuid.New().String()
	item := &TodoItem{
		ID:          id,
		Title:       title,
		Description: description,
		Completed:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	p.items = append(p.items, item)
	return nil
}

func (p *TodoPlugin) getTodoItems() []*TodoItem {
	return p.items
}

func (p *TodoPlugin) getTodoItem(id string) *TodoItem {
	for _, item := range p.items {
		if item.ID == id {
			return item
		}
	}
	return nil
}

func (p *TodoPlugin) getTodoItemByTitle(title string) *TodoItem {
	for _, item := range p.items {
		if item.Title == title {
			return item
		}
	}
	return nil
}

func (p *TodoPlugin) updateTodoStatus(id string, completed bool) error {
	item := p.getTodoItem(id)
	if item == nil {
		return fmt.Errorf("todo item not found")
	}
	item.Completed = completed
	item.UpdatedAt = time.Now()
	return nil
}

func (p *TodoPlugin) updateTodoStatusByTitle(title string, completed bool) error {
	item := p.getTodoItemByTitle(title)
	if item == nil {
		return fmt.Errorf("todo item not found")
	}
	item.Completed = completed
	item.UpdatedAt = time.Now()
	return nil
}

func (p *TodoPlugin) addBulkTodos(todos []map[string]string) error {
	for _, todo := range todos {
		title := todo["title"]
		description := todo["description"]
		if err := p.addTodoItem(title, description); err != nil {
			return fmt.Errorf("failed to add todo item '%s': %w", title, err)
		}
	}
	return nil
}
