package builder

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigYAMLSpawnSubagent(t *testing.T) {
	const y = `
agent:
  name: "t"
  model: "openrouter::x/y"
  working_dir: "/tmp"
  spawn_subagent: true
`
	var cfg Config
	if err := yaml.Unmarshal([]byte(y), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !cfg.Agent.SpawnSubagent {
		t.Fatalf("SpawnSubagent: got false, want true")
	}
	f, err := newAgentFactoryFromConfigStruct(cfg)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	if !f.canSpawnSubagent {
		t.Fatalf("factory.canSpawnSubagent: got false, want true")
	}
}

func TestConfigYAMLSpawnSubagentOmittedFalse(t *testing.T) {
	const y = `
agent:
  name: "t"
  model: "openrouter::x/y"
  working_dir: "/tmp"
`
	var cfg Config
	if err := yaml.Unmarshal([]byte(y), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.Agent.SpawnSubagent {
		t.Fatalf("SpawnSubagent: got true, want false when omitted")
	}
	f, err := newAgentFactoryFromConfigStruct(cfg)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	if f.canSpawnSubagent {
		t.Fatalf("factory.canSpawnSubagent: got true, want false")
	}
}
