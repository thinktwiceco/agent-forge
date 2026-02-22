package builder

import (
	"fmt"
	"os"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/plugins/logger"
	"github.com/thinktwiceco/agent-forge/src/plugins/procedures"
	"github.com/thinktwiceco/agent-forge/src/plugins/todo"
)

type Plugin string

const (
	LOGGER_PLUGIN     Plugin = "logger"
	TODO_PLUGIN       Plugin = "todo"
	PROCEDURES_PLUGIN Plugin = "procedures"
)

// Global callback for todo plugin (can be set by applications)
var globalTodoCallback func(todos []*todo.TodoItem)

// SetTodoCallback sets the global callback for the todo plugin
func SetTodoCallback(callback func(todos []*todo.TodoItem)) {
	globalTodoCallback = callback
}

func (p Plugin) getPlugin() (core.Plugin, error) {
	switch p {
	case LOGGER_PLUGIN:
		return logger.NewPlugin(
			logger.DefaultColorRules(),
			logger.DefaultLabelRules(),
			os.Stdout,
		), nil
	case TODO_PLUGIN:
		// Use global callback if set, otherwise use nil
		return todo.NewTodoPlugin(globalTodoCallback), nil
	case PROCEDURES_PLUGIN:
		return procedures.NewProceduresPlugin(""), nil
	}
	return nil, fmt.Errorf("invalid plugin: %s", p)
}
