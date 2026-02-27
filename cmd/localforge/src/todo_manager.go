package main

import (
	"sync"

	"github.com/thinktwiceco/agent-forge/src/builder"
	"github.com/thinktwiceco/agent-forge/src/plugins/todo"
)

// TodoManager manages the current state of todos for the app
type TodoManager struct {
	mu    sync.RWMutex
	todos []*todo.TodoItem
}

// NewTodoManager creates a new TodoManager instance
func NewTodoManager() *TodoManager {
	tm := &TodoManager{
		todos: []*todo.TodoItem{},
	}

	// Set the builder's global callback to update this manager
	builder.SetTodoCallback(tm.updateTodos)

	return tm
}

// updateTodos is the internal callback that updates the todo list
func (tm *TodoManager) updateTodos(todos []*todo.TodoItem) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Create a deep copy to avoid data races
	tm.todos = make([]*todo.TodoItem, len(todos))
	for i, t := range todos {
		copied := *t
		tm.todos[i] = &copied
	}
}

// GetTodos returns a copy of the current todos
func (tm *TodoManager) GetTodos() []*todo.TodoItem {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Return a copy to avoid external modifications
	result := make([]*todo.TodoItem, len(tm.todos))
	for i, t := range tm.todos {
		copied := *t
		result[i] = &copied
	}
	return result
}
