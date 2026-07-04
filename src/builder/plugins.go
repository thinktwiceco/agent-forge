package builder

import (
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/plugins/config"
	"github.com/thinktwiceco/agent-forge/src/plugins/registry"
	"github.com/thinktwiceco/agent-forge/src/plugins/todo"
)

// SetTodoCallback sets the callback invoked whenever the todo list changes.
// Must be called before Build().
func SetTodoCallback(callback func(todos []*todo.TodoItem)) {
	todo.Callback = callback
}

// SetConfigWriter registers the runtime hook used by the config tool to persist YAML changes.
func SetConfigWriter(w core.ConfigWriter) {
	config.SetWriter(w)
}

func getPlugin(p string, workingDir string) (core.Plugin, error) {
	factory, err := registry.Get(string(p))
	if err != nil {
		return nil, err
	}
	return factory(workingDir), nil
}
