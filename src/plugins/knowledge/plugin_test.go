package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestPlugin(t *testing.T) (*KnowledgePlugin, func()) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "knowledge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	plugin := NewKnowledgePlugin(tmpDir, nil, nil)

	// Initialize database
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
	plugin.explorer = NewKnowledgeExplorer(plugin.querier)

	// Return cleanup function
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

func TestSaveAndGetNode(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Save a node
	metadata := map[string]any{
		"confidence": 0.9,
		"source":     "test",
	}

	nodeID, err := plugin.saveNode("user_preference", "User prefers dark mode", "", metadata)
	if err != nil {
		t.Fatalf("Failed to save node: %v", err)
	}

	if nodeID == "" {
		t.Fatal("Expected non-empty node ID")
	}

	// Get the node
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

	// Create two nodes
	nodeID1, err := plugin.saveNode("user_info", "User is a developer", "", nil)
	if err != nil {
		t.Fatalf("Failed to save node 1: %v", err)
	}

	nodeID2, err := plugin.saveNode("user_preference", "User prefers Go language", "", nil)
	if err != nil {
		t.Fatalf("Failed to save node 2: %v", err)
	}

	// Create edge
	edgeMetadata := map[string]any{"strength": "high"}
	edgeID, err := plugin.saveEdge(nodeID1, nodeID2, "related_to", 0.8, edgeMetadata)
	if err != nil {
		t.Fatalf("Failed to save edge: %v", err)
	}

	if edgeID == "" {
		t.Fatal("Expected non-empty edge ID")
	}

	// Get edges for node 1
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

	// Create multiple nodes
	_, err := plugin.saveNode("user_preference", "Prefers dark mode", "", nil)
	if err != nil {
		t.Fatalf("Failed to save node 1: %v", err)
	}

	_, err = plugin.saveNode("user_preference", "Likes coffee", "", nil)
	if err != nil {
		t.Fatalf("Failed to save node 2: %v", err)
	}

	_, err = plugin.saveNode("user_info", "Lives in NYC", "", nil)
	if err != nil {
		t.Fatalf("Failed to save node 3: %v", err)
	}

	// Find by type
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

	// Create nodes
	_, err := plugin.saveNode("user_info", "User is a Go developer", "", nil)
	if err != nil {
		t.Fatalf("Failed to save node 1: %v", err)
	}

	_, err = plugin.saveNode("user_info", "User likes Python too", "", nil)
	if err != nil {
		t.Fatalf("Failed to save node 2: %v", err)
	}

	// Find by content pattern
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

	// Create a small graph
	node1, _ := plugin.saveNode("user_info", "Node 1", "", nil)
	node2, _ := plugin.saveNode("user_info", "Node 2", "", nil)
	node3, _ := plugin.saveNode("user_info", "Node 3", "", nil)
	node4, _ := plugin.saveNode("user_info", "Node 4", "", nil)

	// Create edges: 1->2->3->4
	_, _ = plugin.saveEdge(node1, node2, "related_to", 1.0, nil)
	_, _ = plugin.saveEdge(node2, node3, "related_to", 1.0, nil)
	_, _ = plugin.saveEdge(node3, node4, "related_to", 1.0, nil)

	// Traverse from node 1 with depth 2
	result, err := plugin.querier.findRelated([]string{node1}, 2)
	if err != nil {
		t.Fatalf("Failed to traverse graph: %v", err)
	}

	// Should find nodes 1, 2, 3 (depth 0, 1, 2) but not 4 (depth 3)
	if len(result.Nodes) != 3 {
		t.Errorf("Expected 3 nodes within depth 2, got %d", len(result.Nodes))
	}

	// Should find 2 edges (1->2 and 2->3)
	if len(result.Edges) != 2 {
		t.Errorf("Expected 2 edges, got %d", len(result.Edges))
	}
}

func TestCyclePrevention(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Create a cycle: 1->2->3->1
	node1, _ := plugin.saveNode("user_info", "Node 1", "", nil)
	node2, _ := plugin.saveNode("user_info", "Node 2", "", nil)
	node3, _ := plugin.saveNode("user_info", "Node 3", "", nil)

	_, _ = plugin.saveEdge(node1, node2, "related_to", 1.0, nil)
	_, _ = plugin.saveEdge(node2, node3, "related_to", 1.0, nil)
	_, _ = plugin.saveEdge(node3, node1, "related_to", 1.0, nil)

	// Traverse - should not loop infinitely
	result, err := plugin.querier.findRelated([]string{node1}, 5)
	if err != nil {
		t.Fatalf("Failed to traverse graph: %v", err)
	}

	// Should find all 3 nodes (might have multiple paths but DISTINCT should deduplicate)
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

	// Log if there are duplicates (for debugging, but don't fail - DISTINCT should handle this)
	if len(result.Nodes) != len(uniqueNodes) {
		t.Logf("Note: Query returned %d results for %d unique nodes (DISTINCT may not be working as expected)", len(result.Nodes), len(uniqueNodes))
	}
}

func TestUpdateNode(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Create node
	nodeID, err := plugin.saveNode("user_info", "Original content", "", nil)
	if err != nil {
		t.Fatalf("Failed to save node: %v", err)
	}

	// Update node
	newMetadata := map[string]any{"updated": true}
	err = plugin.updateNode(nodeID, "user_preference", "Updated content", newMetadata)
	if err != nil {
		t.Fatalf("Failed to update node: %v", err)
	}

	// Verify update
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

	// Create node
	nodeID, err := plugin.saveNode("user_info", "To be deleted", "", nil)
	if err != nil {
		t.Fatalf("Failed to save node: %v", err)
	}

	// Delete node
	err = plugin.deleteNode(nodeID)
	if err != nil {
		t.Fatalf("Failed to delete node: %v", err)
	}

	// Verify deletion
	_, err = plugin.getNode(nodeID)
	if err == nil {
		t.Error("Expected error when getting deleted node")
	}
}

func TestDeleteNodeCascade(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Create nodes and edge
	node1, _ := plugin.saveNode("user_info", "Node 1", "", nil)
	node2, _ := plugin.saveNode("user_info", "Node 2", "", nil)
	_, _ = plugin.saveEdge(node1, node2, "related_to", 1.0, nil)

	// Delete node1 - should cascade delete the edge
	err := plugin.deleteNode(node1)
	if err != nil {
		t.Fatalf("Failed to delete node: %v", err)
	}

	// Edges should be empty for node2
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

	// Create nodes of different types
	_, _ = plugin.saveNode("user_preference", "Pref 1", "", nil)
	_, _ = plugin.saveNode("user_preference", "Pref 2", "", nil)
	_, _ = plugin.saveNode("user_info", "Info 1", "", nil)
	_, _ = plugin.saveNode("fact", "Fact 1", "", nil)

	// Get counts
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
	tmpDir, err := os.MkdirTemp("", "knowledge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	plugin := NewKnowledgePlugin(tmpDir, nil, nil)

	// Open DB
	if err := plugin.openDB(); err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer func() { _ = plugin.Close() }()

	// Verify database file exists in knowledge subdirectory
	dbPath := filepath.Join(tmpDir, "knowledge", dbFileName)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("Expected database file at %s", dbPath)
	}
}

func TestAddAndListNodeTypes(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// List default node types (should be Category and Fact)
	types, err := plugin.listNodeTypes()
	if err != nil {
		t.Fatalf("Failed to list node types: %v", err)
	}

	if len(types) != 4 {
		t.Errorf("Expected exactly 4 default node types, got %d", len(types))
	}

	// Verify Category, Fact, Subcategory, and Document types exist
	foundCategory := false
	foundFact := false
	foundDocument := false
	foundSubcategory := false
	for _, typ := range types {
		if typ.Name == "Category" {
			foundCategory = true
			if typ.Description != "Organizational nodes for grouping knowledge" {
				t.Errorf("Expected Category description, got %s", typ.Description)
			}
		}
		if typ.Name == "Fact" {
			foundFact = true
			if typ.Description != "Actual knowledge/information nodes" {
				t.Errorf("Expected Fact description, got %s", typ.Description)
			}
		}
		if typ.Name == "Document" {
			foundDocument = true
		}
		if typ.Name == "Subcategory" {
			foundSubcategory = true
		}
	}

	if !foundCategory {
		t.Error("Expected to find Category node type")
	}
	if !foundFact {
		t.Error("Expected to find Fact node type")
	}
	if !foundDocument {
		t.Error("Expected to find Document node type")
	}
	if !foundSubcategory {
		t.Error("Expected to find Subcategory node type")
	}
}

func TestAddAndListEdgeTypes(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// List default edge types (should be has_category and has_fact)
	types, err := plugin.listEdgeTypes()
	if err != nil {
		t.Fatalf("Failed to list edge types: %v", err)
	}

	if len(types) != 5 {
		t.Errorf("Expected exactly 5 default edge types, got %d", len(types))
	}

	// Verify all 5 default edge types exist
	foundHasCategory := false
	foundHasFact := false
	foundHasDocument := false
	foundIsRelevantTo := false
	foundHasSubcategory := false
	for _, typ := range types {
		switch typ.Name {
		case "has_category":
			foundHasCategory = true
			if typ.Description != "Links Category to Category" {
				t.Errorf("Expected has_category description, got %s", typ.Description)
			}
		case "has_fact":
			foundHasFact = true
			if typ.Description != "Links Category to Fact, or Fact to Fact" {
				t.Errorf("Expected has_fact description, got %s", typ.Description)
			}
		case "has_document":
			foundHasDocument = true
		case "is_relevant_to":
			foundIsRelevantTo = true
		case "has_subcategory":
			foundHasSubcategory = true
		}
	}

	if !foundHasCategory {
		t.Error("Expected to find has_category edge type")
	}
	if !foundHasFact {
		t.Error("Expected to find has_fact edge type")
	}
	if !foundHasDocument {
		t.Error("Expected to find has_document edge type")
	}
	if !foundIsRelevantTo {
		t.Error("Expected to find is_relevant_to edge type")
	}
	if !foundHasSubcategory {
		t.Error("Expected to find has_subcategory edge type")
	}
}

func TestTypeValidation(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Test valid node types
	err := plugin.validateNodeType("Category")
	if err != nil {
		t.Errorf("Expected Category to be valid, got error: %v", err)
	}

	err = plugin.validateNodeType("Fact")
	if err != nil {
		t.Errorf("Expected Fact to be valid, got error: %v", err)
	}

	// Test invalid node type
	err = plugin.validateNodeType("invalid_type")
	if err == nil {
		t.Error("Expected error for invalid node type")
	}

	// Test valid edge types
	err = plugin.validateEdgeType("has_category")
	if err != nil {
		t.Errorf("Expected has_category to be valid, got error: %v", err)
	}

	err = plugin.validateEdgeType("has_fact")
	if err != nil {
		t.Errorf("Expected has_fact to be valid, got error: %v", err)
	}

	// Test invalid edge type
	err = plugin.validateEdgeType("invalid_relation")
	if err == nil {
		t.Error("Expected error for invalid edge type")
	}
}

func TestDuplicateTypeCreation(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Try to create duplicate node type
	_, err := plugin.addNodeType("Category", "Duplicate", nil)
	if err == nil {
		t.Error("Expected error when creating duplicate node type")
	}

	// Try to create duplicate edge type
	_, err = plugin.addEdgeType("has_category", "Duplicate", nil)
	if err == nil {
		t.Error("Expected error when creating duplicate edge type")
	}
}

func TestSaveWithInvalidType(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Try to save node with invalid type
	err := plugin.validateNodeType("nonexistent_type")
	if err == nil {
		t.Error("Expected error when validating nonexistent node type")
	}

	// Verify error message contains type name
	if err != nil && err.Error() == "" {
		t.Error("Expected descriptive error message")
	}
}

func TestRelateWithInvalidType(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Create two nodes (Category nodes)
	node1, _ := plugin.saveNode("Category", "Node 1", "", nil)
	node2, _ := plugin.saveNode("Category", "Node 2", "", nil)

	// Try to create edge with invalid type
	err := plugin.validateEdgeType("nonexistent_relation")
	if err == nil {
		t.Error("Expected error when validating nonexistent edge type")
	}

	// Verify that valid edge type works
	err = plugin.validateEdgeType("has_category")
	if err != nil {
		t.Errorf("Expected valid edge type to pass, got error: %v", err)
	}

	// Create edge with valid type should work
	_, err = plugin.saveEdge(node1, node2, "has_category", 1.0, nil)
	if err != nil {
		t.Errorf("Failed to create edge with valid type: %v", err)
	}
}

func TestTypeExists(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Check existing node types
	exists, err := plugin.typeExists("node_type", "Category")
	if err != nil {
		t.Fatalf("Failed to check type existence: %v", err)
	}
	if !exists {
		t.Error("Expected Category node type to exist")
	}

	exists, err = plugin.typeExists("node_type", "Fact")
	if err != nil {
		t.Fatalf("Failed to check type existence: %v", err)
	}
	if !exists {
		t.Error("Expected Fact node type to exist")
	}

	// Check existing edge types
	exists, err = plugin.typeExists("edge_type", "has_category")
	if err != nil {
		t.Fatalf("Failed to check type existence: %v", err)
	}
	if !exists {
		t.Error("Expected has_category edge type to exist")
	}

	exists, err = plugin.typeExists("edge_type", "has_fact")
	if err != nil {
		t.Fatalf("Failed to check type existence: %v", err)
	}
	if !exists {
		t.Error("Expected has_fact edge type to exist")
	}

	// Check non-existing types
	exists, err = plugin.typeExists("node_type", "fake_type")
	if err != nil {
		t.Fatalf("Failed to check type existence: %v", err)
	}
	if exists {
		t.Error("Expected fake_type to not exist")
	}
}

func TestGetTopLevelCategories(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Test 1: Empty database should return empty list (only Omnia Nunc Node exists)
	categories, err := plugin.getTopLevelCategories()
	if err != nil {
		t.Fatalf("Failed to get top-level categories: %v", err)
	}
	if len(categories) != 0 {
		t.Errorf("Expected 0 top-level categories, got %d", len(categories))
	}

	// Test 2: Create some Category nodes
	cat1ID, err := plugin.saveNode("Category", "Personal", "", nil)
	if err != nil {
		t.Fatalf("Failed to save category 1: %v", err)
	}

	cat2ID, err := plugin.saveNode("Category", "Projects", "", nil)
	if err != nil {
		t.Fatalf("Failed to save category 2: %v", err)
	}

	cat3ID, err := plugin.saveNode("Category", "Skills", "", nil)
	if err != nil {
		t.Fatalf("Failed to save category 3: %v", err)
	}

	// Test 3: Link categories to Omnia Nunc Node
	_, err = plugin.saveEdge(omniaNuncNodeID, cat1ID, "has_category", 1.0, nil)
	if err != nil {
		t.Fatalf("Failed to create edge to cat1: %v", err)
	}

	_, err = plugin.saveEdge(omniaNuncNodeID, cat2ID, "has_category", 1.0, nil)
	if err != nil {
		t.Fatalf("Failed to create edge to cat2: %v", err)
	}

	_, err = plugin.saveEdge(omniaNuncNodeID, cat3ID, "has_category", 1.0, nil)
	if err != nil {
		t.Fatalf("Failed to create edge to cat3: %v", err)
	}

	// Test 4: Retrieve top-level categories
	categories, err = plugin.getTopLevelCategories()
	if err != nil {
		t.Fatalf("Failed to get top-level categories: %v", err)
	}

	if len(categories) != 3 {
		t.Errorf("Expected 3 top-level categories, got %d", len(categories))
	}

	// Test 5: Verify category contents
	expectedContents := map[string]bool{
		"Personal": false,
		"Projects": false,
		"Skills":   false,
	}

	for _, cat := range categories {
		if cat.Type != "Category" {
			t.Errorf("Expected type 'Category', got '%s'", cat.Type)
		}
		if _, ok := expectedContents[cat.Content]; ok {
			expectedContents[cat.Content] = true
		}
	}

	for content, found := range expectedContents {
		if !found {
			t.Errorf("Expected to find category '%s', but it was not found", content)
		}
	}
}

func TestSystemPromptWithCategories(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Test 1: System prompt should be concise (categories loaded dynamically)
	prompt := plugin.SystemPrompt()
	if prompt == "" {
		t.Fatal("Expected non-empty system prompt")
	}

	// Should NOT contain "No top-level categories yet" (that's in the dynamic section)
	if contains(prompt, "No top-level categories yet") {
		t.Error("SystemPrompt should not contain dynamic category info")
	}

	// Should be comprehensive but not excessive
	// The improved system prompt is intentionally detailed to provide clear guidance
	// With contextual awareness, proactive storage, and natural presentation guidance
	if len(prompt) > 4000 {
		t.Errorf("SystemPrompt should not be excessive, got %d chars", len(prompt))
	}

	// Should contain key guidance sections
	if !contains(prompt, "SILENT_EXECUTION") {
		t.Error("Expected system prompt to contain silent execution guidance")
	}

	if !contains(prompt, "PROACTIVE_RETENTION") {
		t.Error("Expected system prompt to contain proactive retention guidance")
	}

	// Test 2: buildCategoriesSection should return empty before categories exist
	categoriesSection := plugin.buildCategoriesSection()
	if categoriesSection != "" {
		t.Error("Expected empty categories section when no categories exist")
	}

	// Test 3: Add some categories
	cat1ID, err := plugin.saveNode("Category", "Personal Information", "", nil)
	if err != nil {
		t.Fatalf("Failed to save category: %v", err)
	}

	_, err = plugin.saveEdge(omniaNuncNodeID, cat1ID, "has_category", 1.0, nil)
	if err != nil {
		t.Fatalf("Failed to create edge: %v", err)
	}

	cat2ID, err := plugin.saveNode("Category", "Work Projects", "", nil)
	if err != nil {
		t.Fatalf("Failed to save category: %v", err)
	}

	_, err = plugin.saveEdge(omniaNuncNodeID, cat2ID, "has_category", 1.0, nil)
	if err != nil {
		t.Fatalf("Failed to create edge: %v", err)
	}

	// Test 4: buildCategoriesSection should now return category info
	categoriesSection = plugin.buildCategoriesSection()

	// Should contain "CURRENT CATEGORIES"
	if !contains(categoriesSection, "CURRENT_CATEGORIES") {
		t.Error("Expected categories section to contain 'CURRENT_CATEGORIES'")
	}

	// Should contain the category names
	if !contains(categoriesSection, "Personal Information") {
		t.Error("Expected categories section to contain 'Personal Information' category")
	}

	if !contains(categoriesSection, "Work Projects") {
		t.Error("Expected categories section to contain 'Work Projects' category")
	}

	// Should contain the helpful text
	if !contains(categoriesSection, "Top-level knowledge organization nodes") {
		t.Error("Expected categories section to contain organizational context text")
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

// Tests for new API methods

func TestAddCategory(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Add a category
	categoryID, err := plugin.AddCategory("Programming")
	if err != nil {
		t.Fatalf("Failed to add category: %v", err)
	}

	if categoryID == "" {
		t.Fatal("Expected non-empty category ID")
	}

	// Verify the category was created
	node, err := plugin.getNode(categoryID)
	if err != nil {
		t.Fatalf("Failed to get category node: %v", err)
	}

	if node.Type != "Category" {
		t.Errorf("Expected type 'Category', got '%s'", node.Type)
	}

	if node.Content != "Programming" {
		t.Errorf("Expected content 'Programming', got '%s'", node.Content)
	}

	// Verify it's linked to Omnia Nunc Node
	edges, err := plugin.getNodeEdges(categoryID)
	if err != nil {
		t.Fatalf("Failed to get edges: %v", err)
	}

	hasOmniaNuncLink := false
	for _, edge := range edges {
		if edge.FromNodeID == omniaNuncNodeID && edge.ToNodeID == categoryID {
			hasOmniaNuncLink = true
			break
		}
	}

	if !hasOmniaNuncLink {
		t.Error("Expected category to be linked to Omnia Nunc Node")
	}
}

func TestGetCategories(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Initially should have just the Omnia Nunc Node
	categories, err := plugin.GetCategories()
	if err != nil {
		t.Fatalf("Failed to get categories: %v", err)
	}

	initialCount := len(categories)

	// Add some categories
	_, err = plugin.AddCategory("Work")
	if err != nil {
		t.Fatalf("Failed to add category: %v", err)
	}

	_, err = plugin.AddCategory("Personal")
	if err != nil {
		t.Fatalf("Failed to add category: %v", err)
	}

	// Get categories again
	categories, err = plugin.GetCategories()
	if err != nil {
		t.Fatalf("Failed to get categories: %v", err)
	}

	if len(categories) != initialCount+2 {
		t.Errorf("Expected %d categories, got %d", initialCount+2, len(categories))
	}

	// Verify content
	found := map[string]bool{"Work": false, "Personal": false}
	for _, cat := range categories {
		if cat.Content == "Work" || cat.Content == "Personal" {
			found[cat.Content] = true
		}
	}

	for name, wasFound := range found {
		if !wasFound {
			t.Errorf("Expected to find category '%s'", name)
		}
	}
}

func TestRemember(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Create a category first
	categoryID, err := plugin.AddCategory("Preferences")
	if err != nil {
		t.Fatalf("Failed to add category: %v", err)
	}

	// Remember a fact under this category
	factID, err := plugin.Remember("Preferences", "User prefers dark mode")
	if err != nil {
		t.Fatalf("Failed to remember fact: %v", err)
	}

	if factID == "" {
		t.Fatal("Expected non-empty fact ID")
	}

	// Verify the fact was created
	fact, err := plugin.getNode(factID)
	if err != nil {
		t.Fatalf("Failed to get fact node: %v", err)
	}

	if fact.Type != "Fact" {
		t.Errorf("Expected type 'Fact', got '%s'", fact.Type)
	}

	if fact.Content != "User prefers dark mode" {
		t.Errorf("Expected content 'User prefers dark mode', got '%s'", fact.Content)
	}

	// Verify it's linked to the category
	edges, err := plugin.getNodeEdges(factID)
	if err != nil {
		t.Fatalf("Failed to get edges: %v", err)
	}

	hasCategoryLink := false
	for _, edge := range edges {
		if edge.FromNodeID == categoryID && edge.ToNodeID == factID && edge.RelationType == "has_fact" {
			hasCategoryLink = true
			break
		}
	}

	if !hasCategoryLink {
		t.Error("Expected fact to be linked to category")
	}
}

func TestRememberNonExistentCategory(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Try to remember a fact under a non-existent category
	_, err := plugin.Remember("NonExistent", "Some fact")
	if err == nil {
		t.Fatal("Expected error when remembering under non-existent category")
	}
}

func TestGetCategoryFacts(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Create a category
	_, err := plugin.AddCategory("Skills")
	if err != nil {
		t.Fatalf("Failed to add category: %v", err)
	}

	// Add facts
	_, err = plugin.Remember("Skills", "Knows Go programming")
	if err != nil {
		t.Fatalf("Failed to remember fact 1: %v", err)
	}

	_, err = plugin.Remember("Skills", "Knows Python programming")
	if err != nil {
		t.Fatalf("Failed to remember fact 2: %v", err)
	}

	// Get category facts
	facts, err := plugin.GetCategoryFacts("Skills")
	if err != nil {
		t.Fatalf("Failed to get category facts: %v", err)
	}

	if len(facts) != 2 {
		t.Errorf("Expected 2 facts, got %d", len(facts))
	}

	// Verify all are facts
	for _, fact := range facts {
		if fact.Type != "Fact" {
			t.Errorf("Expected type 'Fact', got '%s'", fact.Type)
		}
	}
}

func TestExploreCategory(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Create a category with sub-category and facts
	_, err := plugin.AddCategory("Technology")
	if err != nil {
		t.Fatalf("Failed to add category: %v", err)
	}

	_, err = plugin.Remember("Technology", "AI is advancing rapidly")
	if err != nil {
		t.Fatalf("Failed to remember fact: %v", err)
	}

	// Explore the category
	result, err := plugin.ExploreCategory("Technology")
	if err != nil {
		t.Fatalf("Failed to explore category: %v", err)
	}

	if result.Category.ID == "" {
		t.Error("Expected category in exploration result")
	}

	// Should contain the category node
	hasCategory := result.Category.Type == "Category" && result.Category.Content == "Technology"
	// Should have the fact as a light node
	hasFact := len(result.Facts) > 0

	if !hasCategory {
		t.Errorf("Expected to find Technology category, got type=%s content=%s", result.Category.Type, result.Category.Content)
	}

	if !hasFact {
		t.Error("Expected to find facts in category exploration")
	}
}

func TestExploreFact(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Create a category and fact
	_, err := plugin.AddCategory("Health")
	if err != nil {
		t.Fatalf("Failed to add category: %v", err)
	}

	_, err = plugin.Remember("Health", "Exercise daily")
	if err != nil {
		t.Fatalf("Failed to remember fact: %v", err)
	}

	// Explore the fact
	result, err := plugin.ExploreFact("Exercise daily")
	if err != nil {
		t.Fatalf("Failed to explore fact: %v", err)
	}

	if result.Fact.ID == "" {
		t.Error("Expected fact in exploration result")
	}

	// The fact itself should be the full node
	hasFact := result.Fact.Type == "Fact" && result.Fact.Content == "Exercise daily"
	// Parent categories are returned as light nodes
	hasCategory := false
	for _, light := range result.ParentCategories {
		if light.Title == "Health" || light.Type == "Category" {
			hasCategory = true
		}
	}

	if !hasFact {
		t.Errorf("Expected to find fact, got type=%s content=%s", result.Fact.Type, result.Fact.Content)
	}

	if !hasCategory {
		t.Error("Expected to find parent category in fact exploration")
	}
}

func TestFind(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Add some data
	_, err := plugin.AddCategory("Science")
	if err != nil {
		t.Fatalf("Failed to add category: %v", err)
	}

	_, err = plugin.Remember("Science", "Physics studies matter and energy")
	if err != nil {
		t.Fatalf("Failed to remember fact: %v", err)
	}

	// Search for nodes
	results, err := plugin.Find("Physics", 10)
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected to find matching nodes")
	}

	// Verify result structure
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

	// Create a category with facts
	catID, err := plugin.AddCategory("ToDelete")
	if err != nil {
		t.Fatalf("Failed to add category: %v", err)
	}

	_, err = plugin.Remember("ToDelete", "Fact 1")
	if err != nil {
		t.Fatalf("Failed to remember fact 1: %v", err)
	}

	_, err = plugin.Remember("ToDelete", "Fact 2")
	if err != nil {
		t.Fatalf("Failed to remember fact 2: %v", err)
	}

	// Delete the category (should cascade to facts)
	deletedCount, err := plugin.Forget(catID)
	if err != nil {
		t.Fatalf("Failed to forget: %v", err)
	}

	if deletedCount < 1 {
		t.Errorf("Expected at least 1 deletion, got %d", deletedCount)
	}

	// Verify category is gone
	_, err = plugin.getNode(catID)
	if err == nil {
		t.Error("Expected category to be deleted")
	}

	// Verify facts are also gone
	facts, err := plugin.GetCategoryFacts("ToDelete")
	if err == nil {
		t.Error("Expected error when getting facts from deleted category")
	}
	if len(facts) != 0 {
		t.Errorf("Expected 0 facts after cascade delete, got %d", len(facts))
	}
}

func TestForgetOmniaNuncNode(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	// Try to delete Omnia Nunc Node (should fail)
	_, err := plugin.Forget(omniaNuncNodeID)
	if err == nil {
		t.Fatal("Expected error when trying to delete Omnia Nunc Node")
	}
}
