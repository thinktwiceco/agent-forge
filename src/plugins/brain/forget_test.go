package brain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestForgetTopic_RemovesArtifactsAndGraph(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	convID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	topicID, err := plugin.AddNode("", edgeTypeHasTopic, nodeTypeTopic, "forgettest", "")
	if err != nil {
		t.Fatalf("add topic: %v", err)
	}
	meta := map[string]any{"conv_id": convID, "title": "t"}
	convNodeID, err := plugin.saveNode(nodeTypeConversation, "body", meta)
	if err != nil {
		t.Fatalf("save conv: %v", err)
	}
	if err := plugin.ensureEdge(topicID, convNodeID, edgeTypeHasConversation); err != nil {
		t.Fatalf("edge: %v", err)
	}

	mdDir := filepath.Join(plugin.dir, "persistence", "2026-04-05")
	if err := os.MkdirAll(mdDir, 0755); err != nil {
		t.Fatal(err)
	}
	mdPath := filepath.Join(mdDir, convID+".md")
	if err := os.WriteFile(mdPath, []byte("# distilled"), 0644); err != nil {
		t.Fatal(err)
	}
	jsonPath := filepath.Join(plugin.workingDir, "data", "conversations", "Smith", convID+".json")
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonPath, []byte("[]"), 0644); err != nil {
		t.Fatal(err)
	}

	res, err := plugin.Forget("topic", "forgettest")
	if err != nil {
		t.Fatalf("Forget: %v", err)
	}
	if res.Scope != "topic" || res.DeletedNodes < 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if len(res.RemovedFiles) < 2 {
		t.Fatalf("expected artifact paths removed, got %+v", res.RemovedFiles)
	}

	if _, err := os.Stat(mdPath); !os.IsNotExist(err) {
		t.Error("expected distilled file removed")
	}
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Error("expected json removed")
	}
	if _, err := plugin.getNode(topicID); err == nil {
		t.Error("expected topic node deleted")
	}
	if _, err := plugin.getNode(convNodeID); err == nil {
		t.Error("expected conversation node deleted")
	}
}

func TestForgetConversation_ByConvID(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	topicID, err := plugin.AddNode("", edgeTypeHasTopic, nodeTypeTopic, "solo", "")
	if err != nil {
		t.Fatalf("add topic: %v", err)
	}
	convID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	meta := map[string]any{"conv_id": convID, "title": "t"}
	convNodeID, err := plugin.saveNode(nodeTypeConversation, "body", meta)
	if err != nil {
		t.Fatalf("save conv: %v", err)
	}
	if err := plugin.ensureEdge(topicID, convNodeID, edgeTypeHasConversation); err != nil {
		t.Fatalf("edge: %v", err)
	}

	res, err := plugin.Forget("conversation", convID)
	if err != nil {
		t.Fatalf("Forget: %v", err)
	}
	if res.Scope != "conversation" || res.DeletedNodes < 1 {
		t.Fatalf("unexpected: %+v", res)
	}
	if _, err := plugin.getNode(convNodeID); err == nil {
		t.Error("expected conversation deleted")
	}
}

func TestForget_InvalidScope(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()
	_, err := plugin.Forget("other", "x")
	if err == nil {
		t.Fatal("expected error")
	}
}
