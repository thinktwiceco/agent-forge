package config

import (
	"strings"
	"testing"
)

func TestHandleGetConfigReference(t *testing.T) {
	resp := handleGetConfigReference()
	if !resp.Success() {
		t.Fatalf("expected success: %q", resp.Error())
	}
	if !strings.Contains(resp.Data(), "## Agent fields") {
		t.Fatalf("missing agent fields section")
	}
}

func TestHandleGetToolsReference(t *testing.T) {
	resp := handleGetToolsReference()
	if !resp.Success() {
		t.Fatalf("expected success: %q", resp.Error())
	}
	if !strings.Contains(resp.Data(), "## Tools Configuration") {
		t.Fatalf("missing tools section")
	}
	if strings.Contains(resp.Data(), "## Plugins Configuration") {
		t.Fatal("should not include plugins section")
	}
}

func TestMutationsRequireWriter(t *testing.T) {
	Writer = nil
	resp := handleAddTool(map[string]any{"name": "fs"})
	if resp.Success() {
		t.Fatal("expected error without writer")
	}
	if !strings.Contains(resp.Error(), mutationsUnavailable) {
		t.Fatalf("unexpected error: %q", resp.Error())
	}
}

func TestAddToolValidatesName(t *testing.T) {
	Writer = &stubWriter{}
	resp := handleAddTool(map[string]any{"name": "not-a-tool"})
	if resp.Success() {
		t.Fatal("expected validation error")
	}
}

type stubWriter struct{}

func (stubWriter) AddTool(string, map[string]any) error              { return nil }
func (stubWriter) RemoveTool(string) error                           { return nil }
func (stubWriter) AddPlugin(string) error                            { return nil }
func (stubWriter) RemovePlugin(string) error                         { return nil }
func (stubWriter) SetHeartbeat(string) error                         { return nil }
func (stubWriter) SetDream(string, string) error                     { return nil }
