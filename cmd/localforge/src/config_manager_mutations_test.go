package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigManagerAddRemoveTool(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	base := `agent:
  name: "Test"
  model: "deepseek::deepseek-chat"
  working_dir: "/tmp"
  persistence: "json"
  tools:
    - name: fs
`
	if err := os.WriteFile(path, []byte(base), 0644); err != nil {
		t.Fatal(err)
	}

	cm, err := NewConfigManager(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := cm.AddTool("web", map[string]any{"headless": true}); err != nil {
		t.Fatalf("AddTool: %v", err)
	}
	cfg := cm.GetConfig()
	if len(cfg.Agent.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(cfg.Agent.Tools))
	}

	if err := cm.RemoveTool("web"); err != nil {
		t.Fatalf("RemoveTool: %v", err)
	}
	cfg = cm.GetConfig()
	if len(cfg.Agent.Tools) != 1 || cfg.Agent.Tools[0].Name != "fs" {
		t.Fatalf("unexpected tools after remove: %+v", cfg.Agent.Tools)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if !strings.Contains(content, "fs") {
		t.Fatalf("yaml should still contain fs tool: %s", raw)
	}
}

func TestConfigManagerAddRemovePlugin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	base := `agent:
  name: "Test"
  model: "deepseek::deepseek-chat"
  working_dir: "/tmp"
  persistence: "json"
  plugins:
    - todo
`
	if err := os.WriteFile(path, []byte(base), 0644); err != nil {
		t.Fatal(err)
	}

	cm, err := NewConfigManager(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := cm.AddPlugin("vault"); err != nil {
		t.Fatalf("AddPlugin: %v", err)
	}
	if err := cm.AddPlugin("vault"); err == nil {
		t.Fatal("expected duplicate plugin error")
	}
	if err := cm.RemovePlugin("todo"); err != nil {
		t.Fatalf("RemovePlugin: %v", err)
	}
	cfg := cm.GetConfig()
	if len(cfg.Agent.Plugins) != 1 || cfg.Agent.Plugins[0] != "vault" {
		t.Fatalf("unexpected plugins: %+v", cfg.Agent.Plugins)
	}
}

func TestConfigManagerSetHeartbeatAndDream(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	base := `agent:
  name: "Test"
  model: "deepseek::deepseek-chat"
  working_dir: "/tmp"
  persistence: "json"
`
	if err := os.WriteFile(path, []byte(base), 0644); err != nil {
		t.Fatal(err)
	}

	cm, err := NewConfigManager(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := cm.SetHeartbeat("45m"); err != nil {
		t.Fatalf("SetHeartbeat: %v", err)
	}
	if err := cm.SetDream("off", "03:30"); err != nil {
		t.Fatalf("SetDream: %v", err)
	}

	cfg := cm.GetConfig()
	if cfg.Agent.Heartbeat == nil || cfg.Agent.Heartbeat.Every != "45m" {
		t.Fatalf("heartbeat: %+v", cfg.Agent.Heartbeat)
	}
	if cfg.Agent.BrainPlugin == nil || cfg.Agent.BrainPlugin.Dream != "off" || cfg.Agent.BrainPlugin.DreamTime != "03:30" {
		t.Fatalf("brain_plugin: %+v", cfg.Agent.BrainPlugin)
	}
}
