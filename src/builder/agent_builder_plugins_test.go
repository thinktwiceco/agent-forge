package builder

import (
	"testing"

	"github.com/thinktwiceco/agent-forge/src/core"
)

func TestBuildPlugins_DefaultIncludesBrainAndSkills(t *testing.T) {
	t.Parallel()

	plugins, err := (&AgentFactory{workingDir: t.TempDir()}).buildPlugins()
	if err != nil {
		t.Fatal(err)
	}

	if countPluginName(plugins, "brain") != 1 {
		t.Fatalf("expected one default brain plugin, got %d", countPluginName(plugins, "brain"))
	}
	if countPluginName(plugins, "skills") != 1 {
		t.Fatalf("expected one default skills plugin, got %d", countPluginName(plugins, "skills"))
	}
}

func TestBuildPlugins_DoesNotDuplicateSkills(t *testing.T) {
	t.Parallel()

	plugins, err := (&AgentFactory{
		workingDir: t.TempDir(),
		plugins:    []string{"skills", "logger"},
	}).buildPlugins()
	if err != nil {
		t.Fatal(err)
	}

	if countPluginName(plugins, "skills") != 1 {
		t.Fatalf("expected one skills plugin, got %d", countPluginName(plugins, "skills"))
	}
	if countPluginName(plugins, "logger") != 1 {
		t.Fatalf("expected one logger plugin, got %d", countPluginName(plugins, "logger"))
	}
}

func countPluginName(plugins []core.Plugin, name string) int {
	count := 0
	for _, plugin := range plugins {
		if plugin.Name() == name {
			count++
		}
	}
	return count
}
