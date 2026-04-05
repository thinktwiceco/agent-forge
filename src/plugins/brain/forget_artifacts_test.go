package brain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveConversationArtifacts(t *testing.T) {
	tmp := t.TempDir()
	brain := filepath.Join(tmp, "brain")
	convID := "11111111-2222-3333-4444-555555555555"

	mdDir := filepath.Join(brain, "persistence", "2026-04-05")
	if err := os.MkdirAll(mdDir, 0755); err != nil {
		t.Fatal(err)
	}
	mdPath := filepath.Join(mdDir, convID+".md")
	if err := os.WriteFile(mdPath, []byte("# x"), 0644); err != nil {
		t.Fatal(err)
	}

	jsonPath := filepath.Join(tmp, "data", "conversations", "Smith", convID+".json")
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonPath, []byte("[]"), 0644); err != nil {
		t.Fatal(err)
	}

	removed, err := RemoveConversationArtifacts(tmp, brain, convID)
	if err != nil {
		t.Fatalf("RemoveConversationArtifacts: %v", err)
	}
	if len(removed) != 2 {
		t.Fatalf("expected 2 removed paths, got %d: %v", len(removed), removed)
	}
	if _, err := os.Stat(mdPath); !os.IsNotExist(err) {
		t.Errorf("expected md removed")
	}
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Errorf("expected json removed")
	}
}

func TestRemoveConversationArtifacts_InvalidID(t *testing.T) {
	_, err := RemoveConversationArtifacts("/tmp", "/tmp/brain", "../x")
	if err == nil {
		t.Fatal("expected error for invalid id")
	}
}
