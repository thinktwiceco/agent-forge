package builder

// Blank imports trigger each plugin's init(), which self-registers into the plugin registry.
// To add a new plugin, implement core.Plugin with an init() that calls registry.Register,
// then add a single blank import line here.
import (
	_ "github.com/thinktwiceco/agent-forge/src/plugins/brain"
	_ "github.com/thinktwiceco/agent-forge/src/plugins/heartbeat"
	_ "github.com/thinktwiceco/agent-forge/src/plugins/logger"
	_ "github.com/thinktwiceco/agent-forge/src/plugins/procedures"
	_ "github.com/thinktwiceco/agent-forge/src/plugins/scheduler"
	_ "github.com/thinktwiceco/agent-forge/src/plugins/todo"
	_ "github.com/thinktwiceco/agent-forge/src/plugins/vault"
)
