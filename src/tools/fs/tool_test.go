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

	// 8. Ripgrep Search
	t.Run("ripgrep", func(t *testing.T) {
		// Create test files
		testContent1 := "This is a test file\nWith multiple lines\nContaining searchable content"
		testContent2 := "Another file\nWith different content\nBut also searchable"

		args1 := map[string]any{
			"operation": "write",
			"path":      "test1.txt",
			"content":   testContent1,
		}
		result1 := tool.Call(nil, args1)
		if !result1.Success() {
			t.Errorf("write test1.txt failed: %s", result1.Error())
		}

		args2 := map[string]any{
			"operation": "write",
			"path":      "test2.txt",
			"content":   testContent2,
		}
		result2 := tool.Call(nil, args2)
		if !result2.Success() {
			t.Errorf("write test2.txt failed: %s", result2.Error())
		}

		// Test ripgrep search
		args := map[string]any{
			"operation": "ripgrep",
			"path":      ".",
			"pattern":   "searchable",
		}
		result := tool.Call(nil, args)

		// Check if ripgrep is available
		if !result.Success() && strings.Contains(result.Error(), "not installed") {
			t.Skip("ripgrep not installed, skipping test")
		}

		if !result.Success() {
			t.Errorf("ripgrep failed: %s", result.Error())
		}

		// Should contain matches
		if !strings.Contains(result.Data(), "searchable") {
			t.Errorf("ripgrep should find 'searchable' in files")
		}

		// Test with flags
		argsWithFlags := map[string]any{
			"operation": "ripgrep",
			"path":      ".",
			"pattern":   "SEARCHABLE",
			"flags":     []interface{}{"-i"}, // case insensitive
		}
		resultWithFlags := tool.Call(nil, argsWithFlags)
		if resultWithFlags.Success() {
			if !strings.Contains(resultWithFlags.Data(), "searchable") && !strings.Contains(resultWithFlags.Data(), "SEARCHABLE") {
				t.Errorf("ripgrep with -i flag should find matches")
			}
		}

		// Test no matches
		argsNoMatch := map[string]any{
			"operation": "ripgrep",
			"path":      ".",
			"pattern":   "nonexistent_pattern_xyz",
		}
		resultNoMatch := tool.Call(nil, argsNoMatch)
		if !resultNoMatch.Success() {
			t.Errorf("ripgrep should succeed even with no matches: %s", resultNoMatch.Error())
		}
		if !strings.Contains(resultNoMatch.Data(), "No matches found") {
			t.Errorf("ripgrep should report no matches found")
		}
	})
}
