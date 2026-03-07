package todo

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/plugins/registry"
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

var defaultOnTodoUpdate = func(todos []*TodoItem) {
	fmt.Println("========= Todo List ==========")
	for _, item := range todos {
		status := "[ ]"
		if item.Completed {
			status = "[x]"
		}
		fmt.Printf("%s %s: %s\n", status, item.Title, item.Description)
	}
	fmt.Println("========================")
}

// Name implements the core.Plugin interface
func (p *TodoPlugin) Name() string {
	return PLUGIN_NAME
}

// Hooks implements the core.HookProvider interface
func (p *TodoPlugin) Hooks() map[core.Event]core.AgentHookFn {
	return map[core.Event]core.AgentHookFn{
		core.EventToolExecution: agents.OnToolExecutionHook(p.handleToolExecution),
	}
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
	} else {
		defaultOnTodoUpdate(todoItems)
	}

	return nil
}

// Tools implements the core.ToolProvider interface
func (p *TodoPlugin) Tools() []llms.Tool {
	return []llms.Tool{
		newTodoHandlerTool(p),
	}
}

// SystemPrompt implements the core.PromptProvider interface
func (p *TodoPlugin) SystemPrompt() string {
	return `
[TODO]
- Tool: todo_handler
- Use with reasoning agent when present
- Create todo list from user request
- After adding todos, immediately start working on the first incomplete item without waiting for user input
- Complete tasks one by one, check as completed, then proceed to the next without pausing
- Use bulk action to add multiple items
- Never stop after creating or updating todos — always continue execution until all items are completed

[CLEANUP]
- Before new todo list: call clearTodos if any todos present
- After all completed: call clearTodos
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

func (p *TodoPlugin) clearTodos() {
	p.items = make([]*TodoItem, 0)
}

// Callback is the optional update handler called whenever todos change.
// Set this before building the agent (e.g. via builder.SetTodoCallback).
var Callback func(todos []*TodoItem)

func init() {
	registry.Register(PLUGIN_NAME, func(_ string) core.Plugin {
		return NewTodoPlugin(Callback)
	})
}
