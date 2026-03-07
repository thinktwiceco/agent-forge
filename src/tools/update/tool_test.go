package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateTool_Success(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, updateScriptName)

	script := "#!/bin/bash\nprintf 'updated successfully\\n'\nprintf 'warning stream\\n' >&2\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write update script: %v", err)
	}

	tool := NewUpdateTool(tmpDir)
	result := tool.Call(nil, nil)

	if !result.Success() {
		t.Fatalf("expected success, got error: %s", result.Error())
	}
	if !strings.Contains(result.Data(), "updated successfully") {
		t.Fatalf("expected stdout in result data, got: %s", result.Data())
	}
	if !strings.Contains(result.Data(), "warning stream") {
		t.Fatalf("expected stderr in result data, got: %s", result.Data())
	}
}

func TestUpdateTool_MissingScript(t *testing.T) {
	tmpDir := t.TempDir()

	tool := NewUpdateTool(tmpDir)
	result := tool.Call(nil, nil)

	if result.Success() {
		t.Fatal("expected failure when update script is missing")
	}
	if !strings.Contains(result.Error(), "not found") {
		t.Fatalf("expected missing script error, got: %s", result.Error())
	}
}

func TestUpdateTool_NonZeroExit(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, updateScriptName)

	script := "#!/bin/bash\nprintf 'partial output\\n'\nprintf 'update failed\\n' >&2\nexit 7\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write update script: %v", err)
	}

	tool := NewUpdateTool(tmpDir)
	result := tool.Call(nil, nil)

	if result.Success() {
		t.Fatal("expected failure for non-zero exit")
	}
	if !strings.Contains(result.Error(), "failed") {
		t.Fatalf("expected failure message, got: %s", result.Error())
	}
	if !strings.Contains(result.Error(), "partial output") {
		t.Fatalf("expected stdout in failure message, got: %s", result.Error())
	}
	if !strings.Contains(result.Error(), "update failed") {
		t.Fatalf("expected stderr in failure message, got: %s", result.Error())
	}
}
