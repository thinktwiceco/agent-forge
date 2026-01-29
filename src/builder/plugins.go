package builder

import (
	"fmt"
	"os"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/plugins/logger"
	"github.com/thinktwiceco/agent-forge/src/plugins/todo"
)

type Plugin string

const (
	LOGGER_PLUGIN Plugin = "logger"
	TODO_PLUGIN   Plugin = "todo"
)

func (p Plugin) getPlugin() (core.Plugin, error) {
	switch p {
	case LOGGER_PLUGIN:
		return logger.NewPlugin(
			logger.DefaultColorRules(),
			logger.DefaultLabelRules(),
			os.Stdout,
		), nil
	case TODO_PLUGIN:
		return todo.NewTodoPlugin(nil), nil
	}
	return nil, fmt.Errorf("invalid plugin: %s", p)
}
