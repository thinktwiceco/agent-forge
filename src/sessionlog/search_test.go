package sessionlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.logs")
	content := "[INFO] start\n[ERROR] boom\n[INFO] middle\n[ERROR] again\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := SearchFile(path, `\[ERROR\]`, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalMatches != 2 {
		t.Fatalf("TotalMatches = %d, want 2", result.TotalMatches)
	}
	if len(result.Matches) != 2 || result.Matches[0].LineNumber != 2 {
		t.Fatalf("unexpected matches: %+v", result.Matches)
	}

	result, err = SearchFile(path, `\[ERROR\]`, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.MatchCount != 1 || result.Matches[0].LineNumber != 4 {
		t.Fatalf("unexpected paginated match: %+v", result.Matches)
	}

	out := result.String()
	if !strings.Contains(out, "Showing 1 of 2 matches") {
		t.Fatalf("unexpected status text: %s", out)
	}
}

func TestSearchFile_InvalidRegex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.logs")
	if err := os.WriteFile(path, []byte("line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := SearchFile(path, "[unclosed", 0, 10)
	if err == nil || !strings.Contains(err.Error(), "invalid regex") {
		t.Fatalf("expected invalid regex error, got: %v", err)
	}
}

func TestSearchFile_NotFound(t *testing.T) {
	_, err := SearchFile(filepath.Join(t.TempDir(), "missing.logs"), "x", 0, 10)
	if err == nil || !strings.Contains(err.Error(), "session log not found") {
		t.Fatalf("expected not found error, got: %v", err)
	}
}
