package fs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFsTool_Integration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fs-tool-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tool := NewFsTool(tmpDir)

	// 1. Get Root
	t.Run("get_root", func(t *testing.T) {
		args := map[string]any{"operation": "get_root"}
		result := tool.Call(nil, args)
		if !result.Success() {
			t.Errorf("get_root failed: %s", result.Error())
		}
		// Should contain tmpDir path
	})

	// 2. Write File
	t.Run("write", func(t *testing.T) {
		args := map[string]any{
			"operation": "write",
			"path":      "hello.txt",
			"content":   "Hello World",
		}
		result := tool.Call(nil, args)
		if !result.Success() {
			t.Errorf("write failed: %s", result.Error())
		}

		// Verify on disk
		content, err := os.ReadFile(filepath.Join(tmpDir, "hello.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != "Hello World" {
			t.Errorf("File content mismatch")
		}
	})

	// 3. Read File
	t.Run("read", func(t *testing.T) {
		args := map[string]any{
			"operation": "read",
			"path":      "hello.txt",
		}
		result := tool.Call(nil, args)
		if !result.Success() {
			t.Errorf("read failed: %s", result.Error())
		}
		// Output is formatted, so check containment
		if !strings.Contains(result.Data(), "Hello World") {
			t.Errorf("read content mismatch: got %s", result.Data())
		}
	})

	// 4. Get File Info
	t.Run("get_file_info", func(t *testing.T) {
		args := map[string]any{
			"operation": "get_file_info",
			"path":      "hello.txt",
		}
		result := tool.Call(nil, args)
		if !result.Success() {
			t.Errorf("get_file_info failed: %s", result.Error())
		}
		// Should return info string
	})

	// 5. List
	t.Run("list", func(t *testing.T) {
		args := map[string]any{
			"operation": "list",
			"path":      ".",
		}
		result := tool.Call(nil, args)
		if !result.Success() {
			t.Errorf("list failed: %s", result.Error())
		}
		// Should contain hello.txt (depends on format)
	})

	// 6. Delete
	t.Run("delete", func(t *testing.T) {
		args := map[string]any{
			"operation": "delete",
			"path":      "hello.txt",
		}
		result := tool.Call(nil, args)
		if !result.Success() {
			t.Errorf("delete failed: %s", result.Error())
		}

		if _, err := os.Stat(filepath.Join(tmpDir, "hello.txt")); !os.IsNotExist(err) {
			t.Error("file still exists after delete")
		}
	})

	// 7. Path Traversal
	t.Run("traversal", func(t *testing.T) {
		args := map[string]any{
			"operation": "read",
			"path":      "../secret.txt",
		}
		result := tool.Call(nil, args)
		if result.Success() {
			t.Error("Expected failure for path traversal")
		}
	})
}
