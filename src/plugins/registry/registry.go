package registry

import (
	"fmt"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
)

// PluginFactory is a function that creates a plugin instance for the given working directory.
type PluginFactory func(workingDir string) core.Plugin

var plugins = map[string]PluginFactory{}

// Register adds a plugin factory under the given name.
// Plugins call this from their init() functions.
func Register(name string, factory PluginFactory) {
	agentforge.Debug(">>> [PluginRegistry] REGISTERING PLUGIN: %s", name)

	// Check if the plugin is already registered
	if _, ok := plugins[name]; ok {
		agentforge.Debug("🔌 Plugin already registered: %s", name)
		return
	}

	plugins[name] = factory
}

// Get returns the factory for the named plugin, or an error if it was never registered.
func Get(name string) (PluginFactory, error) {
	f, ok := plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin not registered: %s", name)
	}
	return f, nil
}
