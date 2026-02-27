package builder

import (
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/plugins/registry"
	"github.com/thinktwiceco/agent-forge/src/plugins/todo"
)

// SetTodoCallback sets the callback invoked whenever the todo list changes.
// Must be called before Build().
func SetTodoCallback(callback func(todos []*todo.TodoItem)) {
	todo.Callback = callback
}

func getPlugin(p string, workingDir string) (core.Plugin, error) {
	factory, err := registry.Get(string(p))
	if err != nil {
		return nil, err
	}
	return factory(workingDir), nil
}
