package searchlogs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thinktwiceco/agent-forge/src/sessionlog"
)

func TestSearchLogsTool_CurrentSession(t *testing.T) {
	dir := t.TempDir()
	chatId := "conv-abc"
	agentName := "Smith"
	path := sessionlog.LogPath(dir, agentName, chatId)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[ERROR] tool failed\n[INFO] ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewSearchLogsTool()
	ctx := map[string]any{
		"chatId":     chatId,
		"agentName":  agentName,
		"workingDir": dir,
	}
	result := tool.Call(ctx, map[string]any{"exp": `\[ERROR\]`})
	if !result.Success() {
		t.Fatalf("search_logs failed: %s", result.Error())
	}
	if !strings.Contains(result.Data(), "tool failed") {
		t.Fatalf("expected match in output: %s", result.Data())
	}
}

func TestSearchLogsTool_MissingExp(t *testing.T) {
	tool := NewSearchLogsTool()
	result := tool.Call(map[string]any{"chatId": "x", "agentName": "Smith"}, map[string]any{})
	if result.Success() {
		t.Fatal("expected error for missing exp")
	}
}
