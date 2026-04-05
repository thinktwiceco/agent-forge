package brain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

func setupTestPlugin(t *testing.T) (*BrainPlugin, func()) {
	tmpDir, err := os.MkdirTemp("", "brain-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	plugin := NewBrainPlugin(tmpDir)

	if err := plugin.openDB(); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("Failed to open DB: %v", err)
	}

	if err := plugin.ensureSchema(); err != nil {
		_ = plugin.Close()
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	plugin.querier = NewGraphQuerier(plugin.db)

	cleanup := func() {
		_ = plugin.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return plugin, cleanup
}

func TestPluginName(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	if plugin.Name() != PLUGIN_NAME {
		t.Errorf("Expected plugin name %s, got %s", PLUGIN_NAME, plugin.Name())
	}
}

func TestPluginHooks(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	hooks := plugin.Hooks()

	if len(hooks) == 0 {
		t.Error("Expected at least one hook")
	}

	if hooks[core.EventAgentInitialized] == nil {
		t.Error("Expected non-nil hook for EventAgentInitialized")
	}

	if hooks[core.EventToolExecution] == nil {
		t.Error("Expected non-nil hook for EventToolExecution")
	}
}

func TestOnToolExecutionHook(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	toolResult := &llms.ToolResult{
		ToolCallID: "test-call-id",
		ToolName:   "test_tool",
		Success:    true,
		Result:     "Original result",
		Error:      "",
	}

	err := plugin.onToolExecution(nil, toolResult)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedReminder := "save_short_term_memory(topic, fact)"
	if !strings.Contains(toolResult.Result, expectedReminder) {
		t.Errorf("Expected reminder to mention save_short_term_memory, got: %s", toolResult.Result)
	}

	if !strings.Contains(toolResult.Result, "Original result") {
		t.Errorf("Expected original result to be preserved")
	}
}

func TestOnToolExecutionHook_FailedTool(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	toolResult := &llms.ToolResult{
		ToolCallID: "test-call-id",
		ToolName:   "test_tool",
		Success:    false,
		Result:     "Error result",
		Error:      "Something went wrong",
	}

	originalResult := toolResult.Result

	err := plugin.onToolExecution(nil, toolResult)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if toolResult.Result != originalResult {
		t.Errorf("Expected result to be unchanged for failed tool")
	}
}

func TestOnToolExecutionHook_EmptyResult(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	toolResult := &llms.ToolResult{
		ToolCallID: "test-call-id",
		ToolName:   "test_tool",
		Success:    true,
		Result:     "",
		Error:      "",
	}

	err := plugin.onToolExecution(nil, toolResult)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedReminder := "[Brain]: If something is worth remembering across sessions, use save_short_term_memory(topic, fact) — only for important facts; if unsure, ask the user."
	if toolResult.Result != expectedReminder {
		t.Errorf("Expected reminder to be the entire result for empty result, got: %s", toolResult.Result)
	}
}

func TestOnToolExecutionHook_BrainTool(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	toolResult := &llms.ToolResult{
		ToolCallID: "test-call-id",
		ToolName:   "get_conversation_topics",
		Success:    true,
		Result:     "Original result",
	}

	originalResult := toolResult.Result
	err := plugin.onToolExecution(nil, toolResult)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if toolResult.Result != originalResult {
		t.Errorf("Expected no reminder for brain tool, but result was modified")
	}
}

func TestSaveAndGetNode(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	metadata := map[string]any{
		"confidence": 0.9,
		"source":     "test",
	}

	nodeID, err := plugin.saveNode("user_preference", "User prefers dark mode", metadata)
	if err != nil {
		t.Fatalf("Failed to save node: %v", err)
	}

	if nodeID == "" {
		t.Fatal("Expected non-empty node ID")
	}

	node, err := plugin.getNode(nodeID)
	if err != nil {
		t.Fatalf("Failed to get node: %v", err)
	}

	if node.ID != nodeID {
		t.Errorf("Expected node ID %s, got %s", nodeID, node.ID)
	}

	if node.Type != "user_preference" {
		t.Errorf("Expected node type 'user_preference', got %s", node.Type)
	}

	if node.Content != "User prefers dark mode" {
		t.Errorf("Expected content 'User prefers dark mode', got %s", node.Content)
	}

	if node.Metadata["confidence"] != 0.9 {
		t.Errorf("Expected metadata confidence 0.9, got %v", node.Metadata["confidence"])
	}
}

func TestSaveAndGetEdge(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	nodeID1, err := plugin.saveNode("user_info", "User is a developer", nil)
	if err != nil {
		t.Fatalf("Failed to save node 1: %v", err)
	}

	nodeID2, err := plugin.saveNode("user_preference", "User prefers Go language", nil)
	if err != nil {
		t.Fatalf("Failed to save node 2: %v", err)
	}

	edgeMetadata := map[string]any{"strength": "high"}
	edgeID, err := plugin.saveEdge(nodeID1, nodeID2, "related_to", 0.8, edgeMetadata)
	if err != nil {
		t.Fatalf("Failed to save edge: %v", err)
	}

	if edgeID == "" {
		t.Fatal("Expected non-empty edge ID")
	}

	edges, err := plugin.getNodeEdges(nodeID1)
	if err != nil {
		t.Fatalf("Failed to get edges: %v", err)
	}

	if len(edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(edges))
	}

	edge := edges[0]
	if edge.FromNodeID != nodeID1 {
		t.Errorf("Expected from_node_id %s, got %s", nodeID1, edge.FromNodeID)
	}

	if edge.ToNodeID != nodeID2 {
		t.Errorf("Expected to_node_id %s, got %s", nodeID2, edge.ToNodeID)
	}

	if edge.RelationType != "related_to" {
		t.Errorf("Expected relation_type 'related_to', got %s", edge.RelationType)
	}

	if edge.Weight != 0.8 {
		t.Errorf("Expected weight 0.8, got %f", edge.Weight)
	}
}

func TestFindNodesByType(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	_, err := plugin.saveNode("user_preference", "Prefers dark mode", nil)
	if err != nil {
		t.Fatalf("Failed to save node 1: %v", err)
	}

	_, err = plugin.saveNode("user_preference", "Likes coffee", nil)
	if err != nil {
		t.Fatalf("Failed to save node 2: %v", err)
	}

	_, err = plugin.saveNode("user_info", "Lives in NYC", nil)
	if err != nil {
		t.Fatalf("Failed to save node 3: %v", err)
	}

	nodes, err := plugin.findNodesByType("user_preference", 0, 0)
	if err != nil {
		t.Fatalf("Failed to find nodes: %v", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("Expected 2 nodes, got %d", len(nodes))
	}

	for _, node := range nodes {
		if node.Type != "user_preference" {
			t.Errorf("Expected type 'user_preference', got %s", node.Type)
		}
	}
}

func TestFindNodesByContent(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	_, err := plugin.saveNode("user_info", "User is a Go developer", nil)
	if err != nil {
		t.Fatalf("Failed to save node 1: %v", err)
	}

	_, err = plugin.saveNode("user_info", "User likes Python too", nil)
	if err != nil {
		t.Fatalf("Failed to save node 2: %v", err)
	}

	nodes, err := plugin.findNodesByContent("Go developer", false)
	if err != nil {
		t.Fatalf("Failed to find nodes: %v", err)
	}

	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}

	if nodes[0].Content != "User is a Go developer" {
		t.Errorf("Expected content 'User is a Go developer', got %s", nodes[0].Content)
	}
}

func TestGraphTraversal(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	node1, _ := plugin.saveNode("user_info", "Node 1", nil)
	node2, _ := plugin.saveNode("user_info", "Node 2", nil)
	node3, _ := plugin.saveNode("user_info", "Node 3", nil)
	node4, _ := plugin.saveNode("user_info", "Node 4", nil)

	_, _ = plugin.saveEdge(node1, node2, "related_to", 1.0, nil)
	_, _ = plugin.saveEdge(node2, node3, "related_to", 1.0, nil)
	_, _ = plugin.saveEdge(node3, node4, "related_to", 1.0, nil)

	result, err := plugin.querier.findRelated([]string{node1}, 2)
	if err != nil {
		t.Fatalf("Failed to traverse graph: %v", err)
	}

	if len(result.Nodes) != 3 {
		t.Errorf("Expected 3 nodes within depth 2, got %d", len(result.Nodes))
	}

	if len(result.Edges) != 2 {
		t.Errorf("Expected 2 edges, got %d", len(result.Edges))
	}
}

func TestCyclePrevention(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	node1, _ := plugin.saveNode("user_info", "Node 1", nil)
	node2, _ := plugin.saveNode("user_info", "Node 2", nil)
	node3, _ := plugin.saveNode("user_info", "Node 3", nil)

	_, _ = plugin.saveEdge(node1, node2, "related_to", 1.0, nil)
	_, _ = plugin.saveEdge(node2, node3, "related_to", 1.0, nil)
	_, _ = plugin.saveEdge(node3, node1, "related_to", 1.0, nil)

	result, err := plugin.querier.findRelated([]string{node1}, 5)
	if err != nil {
		t.Fatalf("Failed to traverse graph: %v", err)
	}

	uniqueNodes := make(map[string]bool)
	for _, node := range result.Nodes {
		uniqueNodes[node.ID] = true
	}

	if len(uniqueNodes) != 3 {
		t.Errorf("Expected 3 unique nodes, got %d (total results: %d)", len(uniqueNodes), len(result.Nodes))
		for id := range uniqueNodes {
			t.Logf("  - %s", id)
		}
	}

	if len(result.Nodes) != len(uniqueNodes) {
		t.Logf("Note: Query returned %d results for %d unique nodes (DISTINCT may not be working as expected)", len(result.Nodes), len(uniqueNodes))
	}
}

func TestUpdateNode(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	nodeID, err := plugin.saveNode("user_info", "Original content", nil)
	if err != nil {
		t.Fatalf("Failed to save node: %v", err)
	}

	newMetadata := map[string]any{"updated": true}
	err = plugin.updateNode(nodeID, "user_preference", "Updated content", newMetadata)
	if err != nil {
		t.Fatalf("Failed to update node: %v", err)
	}

	node, err := plugin.getNode(nodeID)
	if err != nil {
		t.Fatalf("Failed to get node: %v", err)
	}

	if node.Type != "user_preference" {
		t.Errorf("Expected type 'user_preference', got %s", node.Type)
	}

	if node.Content != "Updated content" {
		t.Errorf("Expected content 'Updated content', got %s", node.Content)
	}

	if node.Metadata["updated"] != true {
		t.Errorf("Expected metadata updated=true, got %v", node.Metadata["updated"])
	}
}

func TestDeleteNode(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	nodeID, err := plugin.saveNode("user_info", "To be deleted", nil)
	if err != nil {
		t.Fatalf("Failed to save node: %v", err)
	}

	err = plugin.deleteNode(nodeID)
	if err != nil {
		t.Fatalf("Failed to delete node: %v", err)
	}

	_, err = plugin.getNode(nodeID)
	if err == nil {
		t.Error("Expected error when getting deleted node")
	}
}

func TestDeleteNodeCascade(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	node1, _ := plugin.saveNode("user_info", "Node 1", nil)
	node2, _ := plugin.saveNode("user_info", "Node 2", nil)
	_, _ = plugin.saveEdge(node1, node2, "related_to", 1.0, nil)

	err := plugin.deleteNode(node1)
	if err != nil {
		t.Fatalf("Failed to delete node: %v", err)
	}

	edges, err := plugin.getNodeEdges(node2)
	if err != nil {
		t.Fatalf("Failed to get edges: %v", err)
	}

	if len(edges) != 0 {
		t.Errorf("Expected 0 edges after cascade delete, got %d", len(edges))
	}
}

func TestGetNodeCountByType(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	_, _ = plugin.saveNode("user_preference", "Pref 1", nil)
	_, _ = plugin.saveNode("user_preference", "Pref 2", nil)
	_, _ = plugin.saveNode("user_info", "Info 1", nil)
	_, _ = plugin.saveNode("fact", "Fact 1", nil)

	counts, err := plugin.querier.getNodeCountByType()
	if err != nil {
		t.Fatalf("Failed to get counts: %v", err)
	}

	if counts["user_preference"] != 2 {
		t.Errorf("Expected 2 user_preference nodes, got %d", counts["user_preference"])
	}

	if counts["user_info"] != 1 {
		t.Errorf("Expected 1 user_info node, got %d", counts["user_info"])
	}

	if counts["fact"] != 1 {
		t.Errorf("Expected 1 fact node, got %d", counts["fact"])
	}
}

func TestDatabaseLocation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "brain-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	plugin := NewBrainPlugin(tmpDir)

	if err := plugin.openDB(); err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer func() { _ = plugin.Close() }()

	dbPath := filepath.Join(tmpDir, "brain", dbFileName)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("Expected database file at %s", dbPath)
	}
}

func TestAddAndListNodeTypes(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	types, err := plugin.listNodeTypes()
	if err != nil {
		t.Fatalf("Failed to list node types: %v", err)
	}

	if len(types) != 2 {
		t.Errorf("Expected exactly 2 default node types, got %d", len(types))
	}
	got := map[string]bool{}
	for _, typ := range types {
		got[typ.Name] = true
	}
	if !got[nodeTypeTopic] || !got[nodeTypeConversation] {
		t.Errorf("Expected node types %q and %q, got %+v", nodeTypeTopic, nodeTypeConversation, got)
	}
}

func TestAddAndListEdgeTypes(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	types, err := plugin.listEdgeTypes()
	if err != nil {
		t.Fatalf("Failed to list edge types: %v", err)
	}

	if len(types) != 3 {
		t.Errorf("Expected exactly 3 default edge types, got %d", len(types))
	}
	got := map[string]bool{}
	for _, typ := range types {
		got[typ.Name] = true
	}
	if !got[edgeTypeHasTopic] || !got[edgeTypeHasConversation] || !got[edgeTypeRelated] {
		t.Errorf("Expected edge types %q, %q, and %q, got %+v", edgeTypeHasTopic, edgeTypeHasConversation, edgeTypeRelated, got)
	}
}

func TestTypeValidation(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	err := plugin.validateNodeType(nodeTypeConversation)
	if err != nil {
		t.Errorf("Expected conversation to be valid, got error: %v", err)
	}
	err = plugin.validateNodeType(nodeTypeTopic)
	if err != nil {
		t.Errorf("Expected topic to be valid, got error: %v", err)
	}

	err = plugin.validateNodeType("invalid_type")
	if err == nil {
		t.Error("Expected error for invalid node type")
	}

	err = plugin.validateEdgeType(edgeTypeHasConversation)
	if err != nil {
		t.Errorf("Expected HAS_CONVERSATION to be valid, got error: %v", err)
	}
	err = plugin.validateEdgeType(edgeTypeHasTopic)
	if err != nil {
		t.Errorf("Expected HAS_TOPIC to be valid, got error: %v", err)
	}

	err = plugin.validateEdgeType("invalid_relation")
	if err == nil {
		t.Error("Expected error for invalid edge type")
	}
}

func TestDuplicateTypeCreation(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	_, err := plugin.addNodeType(nodeTypeConversation, "Duplicate", nil)
	if err == nil {
		t.Error("Expected error when creating duplicate node type")
	}
	_, err = plugin.addNodeType(nodeTypeTopic, "Duplicate", nil)
	if err == nil {
		t.Error("Expected error when creating duplicate topic node type")
	}

	_, err = plugin.addEdgeType(edgeTypeHasConversation, "Duplicate", nil)
	if err == nil {
		t.Error("Expected error when creating duplicate edge type")
	}
	_, err = plugin.addEdgeType(edgeTypeHasTopic, "Duplicate", nil)
	if err == nil {
		t.Error("Expected error when creating duplicate HAS_TOPIC edge type")
	}
}

func TestSaveWithInvalidType(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	err := plugin.validateNodeType("nonexistent_type")
	if err == nil {
		t.Error("Expected error when validating nonexistent node type")
	}

	if err != nil && err.Error() == "" {
		t.Error("Expected descriptive error message")
	}
}

func TestSaveEdgeFromRoot(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	child, _ := plugin.saveNode(nodeTypeTopic, "science", map[string]any{"title": "science", "topic_name": "science"})

	err := plugin.validateEdgeType("nonexistent_relation")
	if err == nil {
		t.Error("Expected error when validating nonexistent edge type")
	}

	err = plugin.validateEdgeType(edgeTypeHasTopic)
	if err != nil {
		t.Errorf("Expected valid edge type to pass, got error: %v", err)
	}

	_, err = plugin.saveEdge(omniaNuncNodeID, child, edgeTypeHasTopic, 1.0, nil)
	if err != nil {
		t.Errorf("Failed to create HAS_TOPIC edge: %v", err)
	}
}

func TestTypeExists(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	exists, err := plugin.typeExists("node_type", nodeTypeConversation)
	if err != nil {
		t.Fatalf("Failed to check type existence: %v", err)
	}
	if !exists {
		t.Error("Expected conversation node type to exist")
	}
	exists, err = plugin.typeExists("node_type", nodeTypeTopic)
	if err != nil {
		t.Fatalf("Failed to check topic type existence: %v", err)
	}
	if !exists {
		t.Error("Expected topic node type to exist")
	}

	exists, err = plugin.typeExists("edge_type", edgeTypeHasConversation)
	if err != nil {
		t.Fatalf("Failed to check type existence: %v", err)
	}
	if !exists {
		t.Error("Expected HAS_CONVERSATION edge type to exist")
	}
	exists, err = plugin.typeExists("edge_type", edgeTypeHasTopic)
	if err != nil {
		t.Fatalf("Failed to check HAS_TOPIC existence: %v", err)
	}
	if !exists {
		t.Error("Expected HAS_TOPIC edge type to exist")
	}

	exists, err = plugin.typeExists("node_type", "fake_type")
	if err != nil {
		t.Fatalf("Failed to check type existence: %v", err)
	}
	if exists {
		t.Error("Expected fake_type to not exist")
	}
}

func TestGetTopicsUnderRoot(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	topics, err := plugin.getTopicsUnderRoot()
	if err != nil {
		t.Fatalf("Failed to get topics under root: %v", err)
	}
	if len(topics) != 0 {
		t.Errorf("Expected 0 topics, got %d", len(topics))
	}

	c1, err := plugin.saveNode(nodeTypeTopic, "science", map[string]any{"title": "science", "topic_name": "science"})
	if err != nil {
		t.Fatalf("Failed to save node: %v", err)
	}
	c2, err := plugin.saveNode(nodeTypeTopic, "health", map[string]any{"title": "health", "topic_name": "health"})
	if err != nil {
		t.Fatalf("Failed to save node: %v", err)
	}
	c3, err := plugin.saveNode(nodeTypeTopic, "travel", map[string]any{"title": "travel", "topic_name": "travel"})
	if err != nil {
		t.Fatalf("Failed to save node: %v", err)
	}

	_, err = plugin.saveEdge(omniaNuncNodeID, c1, edgeTypeHasTopic, 1.0, nil)
	if err != nil {
		t.Fatalf("Failed to create edge: %v", err)
	}
	_, err = plugin.saveEdge(omniaNuncNodeID, c2, edgeTypeHasTopic, 1.0, nil)
	if err != nil {
		t.Fatalf("Failed to create edge: %v", err)
	}
	_, err = plugin.saveEdge(omniaNuncNodeID, c3, edgeTypeHasTopic, 1.0, nil)
	if err != nil {
		t.Fatalf("Failed to create edge: %v", err)
	}

	topics, err = plugin.getTopicsUnderRoot()
	if err != nil {
		t.Fatalf("Failed to get topics under root: %v", err)
	}

	if len(topics) != 3 {
		t.Errorf("Expected 3 topics, got %d", len(topics))
	}

	expectedContents := map[string]bool{"science": false, "health": false, "travel": false}
	for _, n := range topics {
		if n.Type != nodeTypeTopic {
			t.Errorf("Expected type %q, got %q", nodeTypeTopic, n.Type)
		}
		if _, ok := expectedContents[n.Content]; ok {
			expectedContents[n.Content] = true
		}
	}
	for content, found := range expectedContents {
		if !found {
			t.Errorf("Expected to find conversation content %q", content)
		}
	}
}

func TestSystemPromptWithIndexedTopics(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	prompt := plugin.SystemPrompt()
	if prompt == "" {
		t.Fatal("Expected non-empty system prompt")
	}

	if len(prompt) > 5000 {
		t.Errorf("SystemPrompt should not be excessive, got %d chars", len(prompt))
	}

	if !contains(prompt, "[SHORT_TERM_MEMORY]") {
		t.Error("Expected system prompt to mention short-term memory injection")
	}

	if !contains(prompt, "Use memory tools silently") {
		t.Error("Expected system prompt to contain silent execution guidance")
	}

	if !contains(prompt, "get_conversation_topics") {
		t.Error("Expected system prompt to contain topic discovery guidance")
	}

	if !contains(prompt, "[INDEXED TOPICS]") {
		t.Error("Expected system prompt to contain indexed topics section header")
	}
	if !contains(prompt, "call get_conversation_topics") {
		t.Error("Expected system prompt to name get_conversation_topics for topic index retrieval")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestAddConversationUnderTopic(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	topicID, err := plugin.AddNode("", edgeTypeHasTopic, nodeTypeTopic, "science", "")
	if err != nil {
		t.Fatalf("Failed to add topic node: %v", err)
	}
	id, err := plugin.AddNode(topicID, edgeTypeHasConversation, nodeTypeConversation, "Manual session", "Manual body")
	if err != nil {
		t.Fatalf("Failed to add conversation node: %v", err)
	}
	if id == "" {
		t.Fatal("Expected non-empty node ID")
	}

	node, err := plugin.getNode(id)
	if err != nil {
		t.Fatalf("Failed to get node: %v", err)
	}
	if node.Type != nodeTypeConversation {
		t.Errorf("Expected type %q, got %q", nodeTypeConversation, node.Type)
	}
	if node.Content != "Manual body" {
		t.Errorf("Expected content 'Manual body', got %q", node.Content)
	}

	edges, err := plugin.getNodeEdges(id)
	if err != nil {
		t.Fatalf("Failed to get edges: %v", err)
	}
	linked := false
	for _, edge := range edges {
		if edge.FromNodeID == topicID && edge.ToNodeID == id && edge.RelationType == edgeTypeHasConversation {
			linked = true
		}
	}
	if !linked {
		t.Error("Expected conversation to be linked from topic via HAS_CONVERSATION")
	}
}

func TestOutNodesFromRootListsTopics(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	root, err := plugin.OutNodes("")
	if err != nil {
		t.Fatalf("Failed to get root out nodes: %v", err)
	}
	initialCount := root.Count

	_, err = plugin.AddNode("", edgeTypeHasTopic, nodeTypeTopic, "work", "")
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}
	_, err = plugin.AddNode("", edgeTypeHasTopic, nodeTypeTopic, "personal", "")
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	root, err = plugin.OutNodes("")
	if err != nil {
		t.Fatalf("Failed to get root out nodes: %v", err)
	}
	if root.Count != initialCount+2 {
		t.Errorf("Expected neighbor count %d, got %d", initialCount+2, root.Count)
	}
	found := map[string]bool{"work": false, "personal": false}
	for _, n := range root.Neighbors {
		if n.Title == "work" || n.Title == "personal" {
			found[n.Title] = true
		}
	}
	for name, wasFound := range found {
		if !wasFound {
			t.Errorf("Expected neighbor title %q", name)
		}
	}
}

func TestAddNodeRejectsNonRootParent(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	leaf, err := plugin.AddNode("", edgeTypeHasTopic, nodeTypeTopic, "leaf", "")
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}
	_, err = plugin.AddNode(leaf, edgeTypeHasTopic, nodeTypeTopic, "child", "")
	if err == nil {
		t.Fatal("Expected error when trying to attach a topic under a topic")
	}
}

func TestOutNodesConversationNodeHasNoChildrenByDefault(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	topicID, err := plugin.AddNode("", edgeTypeHasTopic, nodeTypeTopic, "solo", "")
	if err != nil {
		t.Fatalf("Failed to add topic: %v", err)
	}
	_, err = plugin.AddNode(topicID, edgeTypeHasConversation, nodeTypeConversation, "Solo", "Solo content")
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	result, err := plugin.OutNodes("Solo")
	if err != nil {
		t.Fatalf("Failed to get out nodes: %v", err)
	}
	if result.Node.Type != nodeTypeConversation {
		t.Errorf("Expected type %q, got %s", nodeTypeConversation, result.Node.Type)
	}
	if result.Count != 0 {
		t.Errorf("Expected no children, got %d", result.Count)
	}
}

func TestGetNodeContentAndInNodesFromRoot(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	topicID, err := plugin.AddNode("", edgeTypeHasTopic, nodeTypeTopic, "health", "")
	if err != nil {
		t.Fatalf("Failed to add topic: %v", err)
	}
	_, err = plugin.AddNode(topicID, edgeTypeHasConversation, nodeTypeConversation, "Health check", "Exercise daily")
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	node, err := plugin.GetNodeContent("Exercise daily")
	if err != nil {
		t.Fatalf("Failed to get node content: %v", err)
	}
	if node.Type != nodeTypeConversation || node.Content != "Exercise daily" {
		t.Errorf("Unexpected node type=%s content=%s", node.Type, node.Content)
	}

	parents, err := plugin.InNodes(node.ID)
	if err != nil {
		t.Fatalf("Failed to get in nodes: %v", err)
	}
	hasTopic := false
	for _, n := range parents.Neighbors {
		if n.ID == topicID {
			hasTopic = true
		}
	}
	if !hasTopic {
		t.Error("Expected topic as incoming neighbor")
	}
}

func TestFind(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	topicID, err := plugin.AddNode("", edgeTypeHasTopic, nodeTypeTopic, "science", "")
	if err != nil {
		t.Fatalf("Failed to add topic: %v", err)
	}
	_, err = plugin.AddNode(topicID, edgeTypeHasConversation, nodeTypeConversation, "Science thread", "Physics studies matter and energy")
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	results, err := plugin.Find("Physics", 10)
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected to find matching nodes")
	}

	foundPhysics := false
	for _, result := range results {
		if contains(result.Node.Content, "Physics") {
			foundPhysics = true
		}
		if result.Score <= 0 {
			t.Error("Expected positive score")
		}
	}

	if !foundPhysics {
		t.Error("Expected to find Physics in search results")
	}
}

func TestForgetCascade(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	parentID, err := plugin.AddNode("", edgeTypeHasTopic, nodeTypeTopic, "todelete", "")
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	childID, err := plugin.saveNode(nodeTypeConversation, "child", map[string]any{"title": "child"})
	if err != nil {
		t.Fatalf("Failed to save child: %v", err)
	}
	_, err = plugin.saveEdge(parentID, childID, edgeTypeHasConversation, 1.0, nil)
	if err != nil {
		t.Fatalf("Failed to link child: %v", err)
	}

	deletedCount, err := plugin.DeleteNode(parentID)
	if err != nil {
		t.Fatalf("Failed to forget: %v", err)
	}
	if deletedCount < 1 {
		t.Errorf("Expected at least 1 deletion, got %d", deletedCount)
	}
	_, err = plugin.getNode(parentID)
	if err == nil {
		t.Error("Expected parent to be deleted")
	}
}

func TestForgetOmniaNuncNode(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	_, err := plugin.DeleteNode(omniaNuncNodeID)
	if err == nil {
		t.Fatal("Expected error when trying to delete Omnia Nunc Node")
	}
}

func TestListConversationsInTimeRange(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	_, err := plugin.AddNode("", edgeTypeHasTopic, nodeTypeTopic, "science", "")
	if err != nil {
		t.Fatalf("add topic: %v", err)
	}
	recentConvID := "recent-conv-range-1"
	if err := plugin.upsertConversationNode(recentConvID, time.Now()); err != nil {
		t.Fatalf("upsert recent: %v", err)
	}
	if err := plugin.updateConversationNodeSummary(recentConvID, "body", []string{"science"}, "Recent", "Recent description", "Distillation reason recent.", time.Now()); err != nil {
		t.Fatalf("update recent: %v", err)
	}
	recentNode, err := plugin.getConversationNodeByConvID(recentConvID)
	if err != nil {
		t.Fatalf("get recent: %v", err)
	}
	recentID := recentNode.ID

	oldConvID := "old-conv-range-1"
	if err := plugin.upsertConversationNode(oldConvID, time.Now()); err != nil {
		t.Fatalf("upsert old: %v", err)
	}
	if err := plugin.updateConversationNodeSummary(oldConvID, "oldbody", []string{"science"}, "Old", "Old description", "Distillation reason old.", time.Now()); err != nil {
		t.Fatalf("update old: %v", err)
	}
	oldNode, err := plugin.getConversationNodeByConvID(oldConvID)
	if err != nil {
		t.Fatalf("get old: %v", err)
	}
	oldID := oldNode.ID
	_, err = plugin.db.Exec(`UPDATE brain_nodes SET updated_at = ?, metadata = json_set(coalesce(metadata,'{}'),'$.last_access','2020-06-01T12:00:00Z') WHERE id = ?`,
		"2020-06-01 12:00:00", oldID)
	if err != nil {
		t.Fatalf("stamp old: %v", err)
	}

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	end := time.Now().Add(24 * time.Hour)
	items, err := plugin.listConversationsInTimeRange(start, end, "science")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	seenRecent := false
	for _, it := range items {
		if it.ID == recentID {
			seenRecent = true
		}
		if it.ID == oldID {
			t.Error("old node should be excluded by date filter")
		}
	}
	if !seenRecent {
		t.Error("expected recent node in range")
	}
}

func TestUpsertConversationNodeLinksDefaultTopic(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	convID := "conv-default-topic-1"
	if err := plugin.upsertConversationNode(convID, time.Now()); err != nil {
		t.Fatalf("upsert conversation: %v", err)
	}
	node, err := plugin.getConversationNodeByConvID(convID)
	if err != nil {
		t.Fatalf("get conversation node: %v", err)
	}
	topics, err := plugin.getTopicsForConversationNodeID(node.ID)
	if err != nil {
		t.Fatalf("get topics for conversation: %v", err)
	}
	if len(topics) != 1 {
		t.Fatalf("expected 1 topic, got %d", len(topics))
	}
	if normalizeTopicName(topics[0].GetTitle()) != defaultConversationTopic {
		t.Fatalf("expected default topic %q, got %q", defaultConversationTopic, topics[0].GetTitle())
	}
}

func TestUpdateConversationNodeSummaryAssignsTopics(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	convID := "conv-topics-1"
	if err := plugin.upsertConversationNode(convID, time.Now()); err != nil {
		t.Fatalf("upsert conversation: %v", err)
	}
	node, err := plugin.getConversationNodeByConvID(convID)
	if err != nil {
		t.Fatalf("get conversation node: %v", err)
	}
	pendingTopics, err := plugin.getTopicsForConversationNodeID(node.ID)
	if err != nil {
		t.Fatalf("get topics after upsert: %v", err)
	}
	if len(pendingTopics) != 1 || normalizeTopicName(pendingTopics[0].GetTitle()) != defaultConversationTopic {
		t.Fatalf("expected single pending topic before dream update, got %+v", pendingTopics)
	}

	if err := plugin.updateConversationNodeSummary(convID, "- Discussed physics", []string{"science", "research"}, "Physics discussion", "Session covered physics concepts", "Meets criterion 2: concrete topic exploration.", time.Now()); err != nil {
		t.Fatalf("update summary: %v", err)
	}

	node, err = plugin.getConversationNodeByConvID(convID)
	if err != nil {
		t.Fatalf("get conversation node: %v", err)
	}
	topics, err := plugin.getTopicsForConversationNodeID(node.ID)
	if err != nil {
		t.Fatalf("get topics for conversation: %v", err)
	}
	if len(topics) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(topics))
	}
	for _, tn := range topics {
		if normalizeTopicName(tn.GetTitle()) == defaultConversationTopic {
			t.Fatalf("did not expect default topic after distilled topics assigned")
		}
	}
}

func TestUpdateConversationNodeSummaryEmptyTopicsUsesDefault(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	convID := "conv-empty-topics-1"
	if err := plugin.upsertConversationNode(convID, time.Now()); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := plugin.updateConversationNodeSummary(convID, "- Kept summary only", []string{}, "Summary line", "Retained one bullet", "Meets criterion 1: user-directed retention test.", time.Now()); err != nil {
		t.Fatalf("update summary: %v", err)
	}
	node, err := plugin.getConversationNodeByConvID(convID)
	if err != nil {
		t.Fatalf("get node: %v", err)
	}
	topics, err := plugin.getTopicsForConversationNodeID(node.ID)
	if err != nil {
		t.Fatalf("get topics: %v", err)
	}
	if len(topics) != 1 || normalizeTopicName(topics[0].GetTitle()) != defaultConversationTopic {
		t.Fatalf("expected default topic when model returned none, got %+v", topics)
	}
}

func TestGetConversationTopicsTool(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	convID := "conv-topics-2"
	if err := plugin.upsertConversationNode(convID, time.Now()); err != nil {
		t.Fatalf("upsert conversation: %v", err)
	}
	if err := plugin.updateConversationNodeSummary(convID, "- Discussed fitness", []string{"health"}, "Fitness chat", "Exercise and health notes", "Meets criterion 3: user preference context.", time.Now()); err != nil {
		t.Fatalf("update summary: %v", err)
	}

	var matchedTool *core.Tool
	for _, tool := range plugin.Tools() {
		if ct, ok := tool.(*core.Tool); ok && ct.GetName() == "get_conversation_topics" {
			matchedTool = ct
			break
		}
	}
	if matchedTool == nil {
		t.Fatal("get_conversation_topics tool not found")
	}

	ret := matchedTool.Call(nil, nil)
	if !ret.Success() {
		t.Fatalf("expected success: %s", ret.Error())
	}
	if !contains(ret.Data(), "health") {
		t.Fatalf("expected health topic in tool output, got %s", ret.Data())
	}
}

func TestRetrieveConversationToolEphemeral(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	convID := "retrieve-conv-ephemeral-1"
	now := time.Now()
	if err := plugin.upsertConversationNode(convID, now); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := plugin.updateConversationNodeSummary(convID, "secret-content-xyz", []string{"science"}, "T", "Desc", "Reason.", now); err != nil {
		t.Fatalf("update: %v", err)
	}
	node, err := plugin.getConversationNodeByConvID(convID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	// Write the persistence file so readConversationNodeFile can serve the content.
	stmPath, _ := node.Metadata["stm_path"].(string)
	fullPath := filepath.Join(plugin.workingDir, stmPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("secret-content-xyz"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var matchedTool *core.Tool
	for _, tool := range plugin.Tools() {
		if ct, ok := tool.(*core.Tool); ok && ct.GetName() == "retrieve_conversation" {
			matchedTool = ct
			break
		}
	}
	if matchedTool == nil {
		t.Fatal("retrieve_conversation tool not found")
	}
	ret := matchedTool.Call(nil, map[string]any{"id": node.ID})
	if !ret.Success() {
		t.Fatalf("expected success: %s", ret.Error())
	}
	if !strings.Contains(ret.Data(), "secret-content-xyz") {
		t.Errorf("unexpected data: %q", ret.Data())
	}
	if !ret.Ephemeral() {
		t.Error("retrieve_conversation must return ephemeral tool result")
	}
}

func TestDreamToolRegistered(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()
	found := false
	for _, tool := range plugin.Tools() {
		if ct, ok := tool.(*core.Tool); ok && ct.GetName() == "dream" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("dream tool not found")
	}
}

func TestDreamToolNilEngine(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()
	var matchedTool *core.Tool
	for _, tool := range plugin.Tools() {
		if ct, ok := tool.(*core.Tool); ok && ct.GetName() == "dream" {
			matchedTool = ct
			break
		}
	}
	if matchedTool == nil {
		t.Fatal("dream handler not found")
	}
	ret := matchedTool.Call(nil, nil)
	if ret.Success() {
		t.Fatal("expected error without LLM engine")
	}
}

func TestDreamToolEmptyWorkingDir(t *testing.T) {
	tmpDir := t.TempDir()
	plugin := NewBrainPlugin(tmpDir)
	plugin.workingDir = ""
	var matchedTool *core.Tool
	for _, tool := range plugin.Tools() {
		if ct, ok := tool.(*core.Tool); ok && ct.GetName() == "dream" {
			matchedTool = ct
			break
		}
	}
	if matchedTool == nil {
		t.Fatal("dream handler not found")
	}
	ret := matchedTool.Call(nil, nil)
	if ret.Success() {
		t.Fatal("expected error with empty workingDir")
	}
}

func TestAppendShortTermMemoryLine_CreateAndAppend(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	if err := plugin.appendShortTermMemoryLineLocked("Focus", "ship MVP"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(plugin.dir, "MEMORY.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(b)); got != "Focus: ship MVP" {
		t.Errorf("first write: %q", got)
	}

	if err := plugin.appendShortTermMemoryLineLocked("Stack", "Go 1.22"); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(path)
	wantSub := "Focus: ship MVP"
	if !strings.Contains(string(b), wantSub) || !strings.Contains(string(b), "Stack: Go 1.22") {
		t.Errorf("append: %q", string(b))
	}
}

func TestSaveShortTermMemoryTool_Validation(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	var matchedTool *core.Tool
	for _, tool := range plugin.Tools() {
		if ct, ok := tool.(*core.Tool); ok && ct.GetName() == "save_short_term_memory" {
			matchedTool = ct
			break
		}
	}
	if matchedTool == nil {
		t.Fatal("save_short_term_memory tool not found")
	}

	ret := matchedTool.Call(nil, map[string]any{"topic": "", "fact": "x"})
	if ret.Success() {
		t.Fatal("expected error for empty topic")
	}
	ret = matchedTool.Call(nil, map[string]any{"topic": "a", "fact": "bad\nline"})
	if ret.Success() {
		t.Fatal("expected error for newline in fact")
	}
}

func TestOnToolExecutionHook_SaveShortTermMemory_NoReminder(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	toolResult := &llms.ToolResult{
		ToolCallID: "id",
		ToolName:   "save_short_term_memory",
		Success:    true,
		Result:     "Saved to MEMORY.md.",
	}
	_ = plugin.onToolExecution(nil, toolResult)
	if strings.Contains(toolResult.Result, "worth remembering across sessions") {
		t.Error("should not append reminder after save_short_term_memory")
	}
}
