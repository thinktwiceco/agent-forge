package config

import (
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/plugins/registry"
)

const PLUGIN_NAME = "config"

type ConfigPlugin struct{}

func NewConfigPlugin() *ConfigPlugin {
	return &ConfigPlugin{}
}

func (p *ConfigPlugin) Name() string {
	return PLUGIN_NAME
}

func (p *ConfigPlugin) Tools() []llms.Tool {
	return []llms.Tool{newConfigTool()}
}

func (p *ConfigPlugin) SystemPrompt() string {
	return `Use the config tool to discover valid configuration options (get_config_reference or get_tools_reference) before changing tools or plugins.
Prefer get_tools_reference when adding tools. After mutations the agent reloads automatically in Localforge.`
}

func init() {
	registry.Register(PLUGIN_NAME, func(_ string) core.Plugin {
		return NewConfigPlugin()
	})
}
