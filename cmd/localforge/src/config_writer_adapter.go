package main

import "github.com/thinktwiceco/agent-forge/src/core"

type configWriterAdapter struct {
	cm *ConfigManager
	am *AgentManager
}

func newConfigWriterAdapter(cm *ConfigManager, am *AgentManager) core.ConfigWriter {
	return &configWriterAdapter{cm: cm, am: am}
}

func (a *configWriterAdapter) reload() error {
	return a.am.Reload()
}

func (a *configWriterAdapter) AddTool(name string, params map[string]any) error {
	if err := a.cm.AddTool(name, params); err != nil {
		return err
	}
	return a.reload()
}

func (a *configWriterAdapter) RemoveTool(name string) error {
	if err := a.cm.RemoveTool(name); err != nil {
		return err
	}
	return a.reload()
}

func (a *configWriterAdapter) AddPlugin(name string) error {
	if err := a.cm.AddPlugin(name); err != nil {
		return err
	}
	return a.reload()
}

func (a *configWriterAdapter) RemovePlugin(name string) error {
	if err := a.cm.RemovePlugin(name); err != nil {
		return err
	}
	return a.reload()
}

func (a *configWriterAdapter) SetHeartbeat(every string) error {
	if err := a.cm.SetHeartbeat(every); err != nil {
		return err
	}
	return a.reload()
}

func (a *configWriterAdapter) SetDream(status, dreamTime string) error {
	if err := a.cm.SetDream(status, dreamTime); err != nil {
		return err
	}
	return a.reload()
}
