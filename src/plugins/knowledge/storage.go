package knowledge

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const dbFileName = "knowledge.db"

// Node represents a knowledge node in the graph
type Node struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Content     string         `json:"content"`
	EmbeddingID string         `json:"embedding_id,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// Edge represents a relationship between two nodes
type Edge struct {
	ID           string         `json:"id"`
	FromNodeID   string         `json:"from_node_id"`
	ToNodeID     string         `json:"to_node_id"`
	RelationType string         `json:"relation_type"`
	Weight       float64        `json:"weight"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// Type represents a node or edge type definition
type Type struct {
	ID          string         `json:"id"`
	Category    string         `json:"category"` // "node_type" or "edge_type"
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// ScoredNode represents a node with a relevance score
type ScoredNode struct {
	Node  Node    `json:"node"`
	Score float32 `json:"score"`
}

// openDB opens the SQLite database connection
func (p *KnowledgePlugin) openDB() error {
	dbPath := filepath.Join(p.dir, dbFileName)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(p.dir, 0755); err != nil {
		return fmt.Errorf("failed to create knowledge directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=1")
	if err != nil {
		return fmt.Errorf("failed to open knowledge database: %w", err)
	}

	// Test connection
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect to knowledge database: %w", err)
	}

	p.db = db
	return nil
}

// ensureSchema creates the database schema if it doesn't exist
func (p *KnowledgePlugin) ensureSchema() error {
	ctx := context.Background()

	// Create nodes table
	createNodesTable := `
		CREATE TABLE IF NOT EXISTS knowledge_nodes (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			content TEXT NOT NULL,
			embedding_id TEXT,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
	if _, err := p.db.ExecContext(ctx, createNodesTable); err != nil {
		return fmt.Errorf("failed to create nodes table: %w", err)
	}

	// Create edges table
	createEdgesTable := `
		CREATE TABLE IF NOT EXISTS knowledge_edges (
			id TEXT PRIMARY KEY,
			from_node_id TEXT NOT NULL,
			to_node_id TEXT NOT NULL,
			relation_type TEXT NOT NULL,
			weight REAL DEFAULT 1.0,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(from_node_id) REFERENCES knowledge_nodes(id) ON DELETE CASCADE,
			FOREIGN KEY(to_node_id) REFERENCES knowledge_nodes(id) ON DELETE CASCADE
		)
	`
	if _, err := p.db.ExecContext(ctx, createEdgesTable); err != nil {
		return fmt.Errorf("failed to create edges table: %w", err)
	}

	// Create types table
	createTypesTable := `
		CREATE TABLE IF NOT EXISTS knowledge_types (
			id TEXT PRIMARY KEY,
			category TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(category, name)
		)
	`
	if _, err := p.db.ExecContext(ctx, createTypesTable); err != nil {
		return fmt.Errorf("failed to create types table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_nodes_type ON knowledge_nodes(type)",
		"CREATE INDEX IF NOT EXISTS idx_nodes_content ON knowledge_nodes(content)",
		"CREATE INDEX IF NOT EXISTS idx_edges_from ON knowledge_edges(from_node_id)",
		"CREATE INDEX IF NOT EXISTS idx_edges_to ON knowledge_edges(to_node_id)",
		"CREATE INDEX IF NOT EXISTS idx_edges_type ON knowledge_edges(relation_type)",
		"CREATE INDEX IF NOT EXISTS idx_types_category ON knowledge_types(category)",
		"CREATE INDEX IF NOT EXISTS idx_types_name ON knowledge_types(name)",
	}

	for _, idx := range indexes {
		if _, err := p.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Seed default types
	if err := p.seedDefaultTypes(); err != nil {
		return fmt.Errorf("failed to seed default types: %w", err)
	}

	return nil
}

// seedDefaultTypes populates the types table with default types and creates Omnia Nunc Node
func (p *KnowledgePlugin) seedDefaultTypes() error {
	defaultNodeTypes := []struct {
		name        string
		description string
	}{
		{"Category", "Organizational nodes for grouping knowledge"},
		{"Fact", "Actual knowledge/information nodes"},
	}

	defaultEdgeTypes := []struct {
		name        string
		description string
	}{
		{"has_category", "Links Category to Category"},
		{"has_fact", "Links Category to Fact, or Fact to Fact"},
	}

	for _, nt := range defaultNodeTypes {
		exists, _ := p.typeExists("node_type", nt.name)
		if !exists {
			_, err := p.addNodeType(nt.name, nt.description, nil)
			if err != nil {
				return fmt.Errorf("failed to seed node type %s: %w", nt.name, err)
			}
		}
	}

	for _, et := range defaultEdgeTypes {
		exists, _ := p.typeExists("edge_type", et.name)
		if !exists {
			_, err := p.addEdgeType(et.name, et.description, nil)
			if err != nil {
				return fmt.Errorf("failed to seed edge type %s: %w", et.name, err)
			}
		}
	}

	// Create the central Omnia Nunc Node if it doesn't exist
	if err := p.ensureOmniaNuncNode(); err != nil {
		return fmt.Errorf("failed to create Omnia Nunc Node: %w", err)
	}

	return nil
}

const omniaNuncNodeID = "omnia-nunc-root"

// ensureOmniaNuncNode creates the central Omnia Nunc Node if it doesn't exist
func (p *KnowledgePlugin) ensureOmniaNuncNode() error {
	ctx := context.Background()

	// Check if it already exists
	query := `SELECT COUNT(*) FROM knowledge_nodes WHERE id = ?`
	var count int
	err := p.db.QueryRowContext(ctx, query, omniaNuncNodeID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for Omnia Nunc Node: %w", err)
	}

	if count > 0 {
		return nil
	}

	// Create the Omnia Nunc Node with fixed ID
	now := time.Now()
	insertQuery := `
		INSERT INTO knowledge_nodes (id, type, content, embedding_id, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	metadata := map[string]any{
		"system": true,
		"root":   true,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = p.db.ExecContext(ctx, insertQuery, omniaNuncNodeID, "Category", "Omnia Nunc Node", nil, string(metadataJSON), now, now)
	if err != nil {
		return fmt.Errorf("failed to insert Omnia Nunc Node: %w", err)
	}

	return nil
}

// isOmniaNuncNode checks if a node ID is the Omnia Nunc Node
func (p *KnowledgePlugin) isOmniaNuncNode(nodeID string) bool {
	return nodeID == omniaNuncNodeID
}

// validateEdgeRules enforces strict edge validation based on node types
//
//nolint:unused // reserved for strict validation
func (p *KnowledgePlugin) validateEdgeRules(fromNodeID, toNodeID, relationType string) error {
	// Get the source node
	fromNode, err := p.getNode(fromNodeID)
	if err != nil {
		return fmt.Errorf("failed to get source node: %w", err)
	}

	// Get the target node
	toNode, err := p.getNode(toNodeID)
	if err != nil {
		return fmt.Errorf("failed to get target node: %w", err)
	}

	// Validate edge rules based on relation type
	switch relationType {
	case "has_category":
		// Both nodes must be Category type
		if fromNode.Type != "Category" {
			return fmt.Errorf("has_category edge requires source node to be Category type, got: %s", fromNode.Type)
		}
		if toNode.Type != "Category" {
			return fmt.Errorf("has_category edge requires target node to be Category type, got: %s", toNode.Type)
		}

	case "has_fact":
		// Source can be Category or Fact, target must be Fact
		if fromNode.Type != "Category" && fromNode.Type != "Fact" {
			return fmt.Errorf("has_fact edge requires source node to be Category or Fact type, got: %s", fromNode.Type)
		}
		if toNode.Type != "Fact" {
			return fmt.Errorf("has_fact edge requires target node to be Fact type, got: %s", toNode.Type)
		}

	default:
		return fmt.Errorf("unknown edge type: %s (valid types: has_category, has_fact)", relationType)
	}

	// Additional rule: Facts cannot have Category children (regardless of edge type)
	if fromNode.Type == "Fact" && toNode.Type == "Category" {
		return fmt.Errorf("fact nodes cannot have category children")
	}

	return nil
}

// saveNode stores a new node in the graph
func (p *KnowledgePlugin) saveNode(nodeType, content string, embeddingID string, metadata map[string]any) (string, error) {
	ctx := context.Background()

	nodeID := uuid.New().String()
	now := time.Now()

	var metadataJSON []byte
	var err error
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return "", fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO knowledge_nodes (id, type, content, embedding_id, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = p.db.ExecContext(ctx, query, nodeID, nodeType, content, embeddingID, string(metadataJSON), now, now)
	if err != nil {
		return "", fmt.Errorf("failed to insert node: %w", err)
	}

	return nodeID, nil
}

// saveEdge stores a new edge in the graph
func (p *KnowledgePlugin) saveEdge(fromID, toID, relationType string, weight float64, metadata map[string]any) (string, error) {
	ctx := context.Background()

	edgeID := uuid.New().String()

	var metadataJSON []byte
	var err error
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return "", fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO knowledge_edges (id, from_node_id, to_node_id, relation_type, weight, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = p.db.ExecContext(ctx, query, edgeID, fromID, toID, relationType, weight, string(metadataJSON))
	if err != nil {
		return "", fmt.Errorf("failed to insert edge: %w", err)
	}

	return edgeID, nil
}

// getNode retrieves a node by ID
func (p *KnowledgePlugin) getNode(nodeID string) (*Node, error) {
	ctx := context.Background()

	query := `
		SELECT id, type, content, embedding_id, metadata, created_at, updated_at
		FROM knowledge_nodes
		WHERE id = ?
	`

	var node Node
	var embeddingID sql.NullString
	var metadataJSON sql.NullString
	var createdAt, updatedAt string

	err := p.db.QueryRowContext(ctx, query, nodeID).Scan(
		&node.ID, &node.Type, &node.Content, &embeddingID, &metadataJSON, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("node not found: %s", nodeID)
		}
		return nil, fmt.Errorf("failed to query node: %w", err)
	}

	if embeddingID.Valid {
		node.EmbeddingID = embeddingID.String
	}

	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &node.Metadata); err != nil {
			node.Metadata = make(map[string]any)
		}
	}

	node.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	node.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return &node, nil
}

// findNodesByContent searches for nodes by content (exact match or LIKE)
func (p *KnowledgePlugin) findNodesByContent(query string, exact bool) ([]Node, error) {
	ctx := context.Background()

	var sqlQuery string
	var args []interface{}

	if exact {
		sqlQuery = `
			SELECT id, type, content, embedding_id, metadata, created_at, updated_at
			FROM knowledge_nodes
			WHERE content = ?
		`
		args = []interface{}{query}
	} else {
		sqlQuery = `
			SELECT id, type, content, embedding_id, metadata, created_at, updated_at
			FROM knowledge_nodes
			WHERE content LIKE ?
		`
		args = []interface{}{"%" + query + "%"}
	}

	return p.queryNodes(ctx, sqlQuery, args...)
}

// findNodesByType searches for nodes by type with optional pagination
func (p *KnowledgePlugin) findNodesByType(nodeType string, limit, offset int) ([]Node, error) {
	ctx := context.Background()

	query := `
		SELECT id, type, content, embedding_id, metadata, created_at, updated_at
		FROM knowledge_nodes
		WHERE type = ?
		ORDER BY created_at DESC
	`

	// Add pagination if limit is specified
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	return p.queryNodes(ctx, query, nodeType)
}

// findNodesByIDs retrieves multiple nodes by their IDs
//
//nolint:unused // reserved for batch operations
func (p *KnowledgePlugin) findNodesByIDs(nodeIDs []string) ([]Node, error) {
	if len(nodeIDs) == 0 {
		return []Node{}, nil
	}

	ctx := context.Background()

	// Build placeholders for IN clause
	placeholders := ""
	args := make([]interface{}, len(nodeIDs))
	for i, id := range nodeIDs {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, type, content, embedding_id, metadata, created_at, updated_at
		FROM knowledge_nodes
		WHERE id IN (%s)
	`, placeholders)

	return p.queryNodes(ctx, query, args...)
}

// queryNodes is a helper function to execute a query and return nodes
func (p *KnowledgePlugin) queryNodes(ctx context.Context, query string, args ...interface{}) ([]Node, error) {
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var nodes []Node
	for rows.Next() {
		var node Node
		var embeddingID sql.NullString
		var metadataJSON sql.NullString
		var createdAt, updatedAt string

		err := rows.Scan(
			&node.ID, &node.Type, &node.Content, &embeddingID, &metadataJSON, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}

		if embeddingID.Valid {
			node.EmbeddingID = embeddingID.String
		}

		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &node.Metadata); err != nil {
				node.Metadata = make(map[string]any)
			}
		}

		node.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		node.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

		nodes = append(nodes, node)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating nodes: %w", err)
	}

	return nodes, nil
}

// updateNode updates an existing node
func (p *KnowledgePlugin) updateNode(nodeID, nodeType, content string, metadata map[string]any) error {
	ctx := context.Background()

	var metadataJSON []byte
	var err error
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		UPDATE knowledge_nodes
		SET type = ?, content = ?, metadata = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := p.db.ExecContext(ctx, query, nodeType, content, string(metadataJSON), time.Now(), nodeID)
	if err != nil {
		return fmt.Errorf("failed to update node: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	return nil
}

// deleteNode deletes a node and its edges
func (p *KnowledgePlugin) deleteNode(nodeID string) error {
	ctx := context.Background()

	query := `DELETE FROM knowledge_nodes WHERE id = ?`

	result, err := p.db.ExecContext(ctx, query, nodeID)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	return nil
}

// getNodeEdges retrieves all edges connected to a node
func (p *KnowledgePlugin) getNodeEdges(nodeID string) ([]Edge, error) {
	ctx := context.Background()

	query := `
		SELECT id, from_node_id, to_node_id, relation_type, weight, metadata, created_at
		FROM knowledge_edges
		WHERE from_node_id = ? OR to_node_id = ?
	`

	rows, err := p.db.QueryContext(ctx, query, nodeID, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query edges: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var edges []Edge
	for rows.Next() {
		var edge Edge
		var metadataJSON sql.NullString
		var createdAt string

		err := rows.Scan(
			&edge.ID, &edge.FromNodeID, &edge.ToNodeID, &edge.RelationType, &edge.Weight, &metadataJSON, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan edge: %w", err)
		}

		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &edge.Metadata); err != nil {
				edge.Metadata = make(map[string]any)
			}
		}

		edge.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)

		edges = append(edges, edge)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating edges: %w", err)
	}

	return edges, nil
}

// deleteEdge deletes an edge by ID
//
//nolint:unused // reserved for edge management
func (p *KnowledgePlugin) deleteEdge(edgeID string) error {
	ctx := context.Background()

	query := `DELETE FROM knowledge_edges WHERE id = ?`

	result, err := p.db.ExecContext(ctx, query, edgeID)
	if err != nil {
		return fmt.Errorf("failed to delete edge: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("edge not found: %s", edgeID)
	}

	return nil
}

// addNodeType creates a new node type
func (p *KnowledgePlugin) addNodeType(name, description string, metadata map[string]any) (string, error) {
	return p.addType("node_type", name, description, metadata)
}

// addEdgeType creates a new edge type
func (p *KnowledgePlugin) addEdgeType(name, description string, metadata map[string]any) (string, error) {
	return p.addType("edge_type", name, description, metadata)
}

// addType creates a new type (node or edge)
func (p *KnowledgePlugin) addType(category, name, description string, metadata map[string]any) (string, error) {
	ctx := context.Background()

	// Check if type already exists
	exists, err := p.typeExists(category, name)
	if err != nil {
		return "", fmt.Errorf("failed to check if type exists: %w", err)
	}
	if exists {
		return "", fmt.Errorf("type '%s' already exists in category '%s'", name, category)
	}

	typeID := uuid.New().String()
	now := time.Now()

	var metadataJSON []byte
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return "", fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO knowledge_types (id, category, name, description, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = p.db.ExecContext(ctx, query, typeID, category, name, description, string(metadataJSON), now)
	if err != nil {
		return "", fmt.Errorf("failed to insert type: %w", err)
	}

	return typeID, nil
}

// listNodeTypes retrieves all node types
func (p *KnowledgePlugin) listNodeTypes() ([]Type, error) {
	return p.listTypes("node_type")
}

// listEdgeTypes retrieves all edge types
func (p *KnowledgePlugin) listEdgeTypes() ([]Type, error) {
	return p.listTypes("edge_type")
}

// listTypes retrieves all types for a given category
func (p *KnowledgePlugin) listTypes(category string) ([]Type, error) {
	ctx := context.Background()

	query := `
		SELECT id, category, name, description, metadata, created_at
		FROM knowledge_types
		WHERE category = ?
		ORDER BY name
	`

	rows, err := p.db.QueryContext(ctx, query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to query types: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var types []Type
	for rows.Next() {
		var t Type
		var metadataJSON sql.NullString
		var createdAt string

		err := rows.Scan(
			&t.ID, &t.Category, &t.Name, &t.Description, &metadataJSON, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan type: %w", err)
		}

		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &t.Metadata); err != nil {
				t.Metadata = make(map[string]any)
			}
		}

		t.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)

		types = append(types, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating types: %w", err)
	}

	return types, nil
}

// getType retrieves a specific type by category and name
//
//nolint:unused // reserved for type lookup
func (p *KnowledgePlugin) getType(category, name string) (*Type, error) {
	ctx := context.Background()

	query := `
		SELECT id, category, name, description, metadata, created_at
		FROM knowledge_types
		WHERE category = ? AND name = ?
	`

	var t Type
	var metadataJSON sql.NullString
	var createdAt string

	err := p.db.QueryRowContext(ctx, query, category, name).Scan(
		&t.ID, &t.Category, &t.Name, &t.Description, &metadataJSON, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("type not found: category=%s, name=%s", category, name)
		}
		return nil, fmt.Errorf("failed to query type: %w", err)
	}

	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &t.Metadata); err != nil {
			t.Metadata = make(map[string]any)
		}
	}

	t.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)

	return &t, nil
}

// typeExists checks if a type exists
func (p *KnowledgePlugin) typeExists(category, name string) (bool, error) {
	ctx := context.Background()

	query := `SELECT COUNT(*) FROM knowledge_types WHERE category = ? AND name = ?`

	var count int
	err := p.db.QueryRowContext(ctx, query, category, name).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check type existence: %w", err)
	}

	return count > 0, nil
}

// validateNodeType checks if a node type is valid
func (p *KnowledgePlugin) validateNodeType(nodeType string) error {
	exists, err := p.typeExists("node_type", nodeType)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("node type '%s' does not exist", nodeType)
	}
	return nil
}

// validateEdgeType checks if an edge type is valid
func (p *KnowledgePlugin) validateEdgeType(edgeType string) error {
	exists, err := p.typeExists("edge_type", edgeType)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("edge type '%s' does not exist", edgeType)
	}
	return nil
}

// getTopLevelCategories retrieves all Category nodes directly connected to Omnia Nunc Node
func (p *KnowledgePlugin) getTopLevelCategories() ([]Node, error) {
	// Return empty list if database is not initialized
	if p.db == nil {
		return []Node{}, nil
	}

	ctx := context.Background()

	query := `
		SELECT n.id, n.type, n.content, n.embedding_id, n.metadata, n.created_at, n.updated_at
		FROM knowledge_nodes n
		JOIN knowledge_edges e ON e.to_node_id = n.id
		WHERE e.from_node_id = ? AND e.relation_type = ?
		ORDER BY n.created_at ASC
	`

	rows, err := p.db.QueryContext(ctx, query, omniaNuncNodeID, "has_category")
	if err != nil {
		return nil, fmt.Errorf("failed to query top-level categories: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var nodes []Node
	for rows.Next() {
		var node Node
		var embeddingID sql.NullString
		var metadataJSON sql.NullString
		var createdAt, updatedAt string

		err := rows.Scan(
			&node.ID, &node.Type, &node.Content, &embeddingID, &metadataJSON, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}

		if embeddingID.Valid {
			node.EmbeddingID = embeddingID.String
		}

		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &node.Metadata); err != nil {
				node.Metadata = make(map[string]any)
			}
		}

		node.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		node.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

		nodes = append(nodes, node)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating nodes: %w", err)
	}

	return nodes, nil
}

// remember saves a fact under a specific category
func (p *KnowledgePlugin) remember(category string, fact string) (string, error) {
	// Find the category node
	categoryNodes, err := p.findNodesByTypeAndContent("Category", category, 10, 0)
	if err != nil {
		return "", fmt.Errorf("failed to search for category: %w", err)
	}

	if len(categoryNodes) == 0 {
		return "", fmt.Errorf("category not found: %s", category)
	}

	categoryNode := categoryNodes[0]

	// Create the fact node with embedding
	factID, err := p.saveWithEmbedding("Fact", fact, nil)
	if err != nil {
		return "", fmt.Errorf("failed to save fact: %w", err)
	}

	// Create has_fact edge from category to fact
	_, err = p.saveEdge(categoryNode.ID, factID, "has_fact", 1.0, nil)
	if err != nil {
		// Clean up the fact node if edge creation fails
		_ = p.deleteNode(factID)
		return "", fmt.Errorf("failed to create relationship: %w", err)
	}

	return factID, nil
}

// addCategory creates a new category node
func (p *KnowledgePlugin) addCategory(category string) (string, error) {
	// Create the category node with embedding
	categoryID, err := p.saveWithEmbedding("Category", category, nil)
	if err != nil {
		return "", fmt.Errorf("failed to save category: %w", err)
	}

	// Check if category has any incoming edges
	edges, err := p.getNodeEdges(categoryID)
	if err != nil {
		return categoryID, nil // Return the ID even if edge check fails
	}

	// If no incoming edges, link to Omnia Nunc Node
	hasIncoming := false
	for _, edge := range edges {
		if edge.ToNodeID == categoryID {
			hasIncoming = true
			break
		}
	}

	if !hasIncoming {
		_, err = p.saveEdge(omniaNuncNodeID, categoryID, "has_category", 1.0, nil)
		if err != nil {
			// Log but don't fail - the category is already created
			return categoryID, nil
		}
	}

	return categoryID, nil
}

// getCategories returns all Category nodes
func (p *KnowledgePlugin) getCategories() ([]Node, error) {
	nodes, err := p.findNodesByType("Category", 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	return nodes, nil
}

// getCategoryFacts returns all Fact nodes directly connected to a category
func (p *KnowledgePlugin) getCategoryFacts(category string) ([]Node, error) {
	// Find the category node
	categoryNodes, err := p.findNodesByTypeAndContent("Category", category, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to search for category: %w", err)
	}

	if len(categoryNodes) == 0 {
		return nil, fmt.Errorf("category not found: %s", category)
	}

	categoryNode := categoryNodes[0]

	// Query all facts connected via has_fact edge
	ctx := context.Background()
	query := `
		SELECT n.id, n.type, n.content, n.embedding_id, n.metadata, n.created_at, n.updated_at
		FROM knowledge_nodes n
		JOIN knowledge_edges e ON e.to_node_id = n.id
		WHERE e.from_node_id = ? AND e.relation_type = ? AND n.type = ?
		ORDER BY n.created_at DESC
	`

	return p.queryNodes(ctx, query, categoryNode.ID, "has_fact", "Fact")
}

// forgetCascade deletes a node and all its dependents (cascade delete)
func (p *KnowledgePlugin) forgetCascade(identifier string) (int, error) {
	// Try to find node by ID first
	node, err := p.getNode(identifier)
	if err != nil {
		// Try to find by content
		nodes, err := p.findNodesByContent(identifier, false)
		if err != nil {
			return 0, fmt.Errorf("failed to find node: %w", err)
		}
		if len(nodes) == 0 {
			return 0, fmt.Errorf("node not found: %s", identifier)
		}
		node = &nodes[0]
	}

	// Prevent deletion of Omnia Nunc Node
	if p.isOmniaNuncNode(node.ID) {
		return 0, fmt.Errorf("cannot delete the Omnia Nunc Node (system root node)")
	}

	// Get all dependent nodes using recursive query
	dependentIDs, err := p.getCascadeDeleteNodes(node.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to get dependent nodes: %w", err)
	}

	// Delete nodes in reverse order (leaves first, then parents)
	// SQLite foreign keys will handle edge cleanup
	deletedCount := 0
	for i := len(dependentIDs) - 1; i >= 0; i-- {
		err := p.deleteNode(dependentIDs[i])
		if err != nil {
			// Continue even if some deletes fail
			continue
		}
		deletedCount++
	}

	return deletedCount, nil
}

// getCascadeDeleteNodes returns all nodes that should be deleted in cascade
func (p *KnowledgePlugin) getCascadeDeleteNodes(nodeID string) ([]string, error) {
	ctx := context.Background()

	// Recursive CTE to find all dependent nodes (following outgoing edges)
	query := `
		WITH RECURSIVE dependents(id, depth) AS (
			-- Base case: start with the given node
			SELECT id, 0 as depth
			FROM knowledge_nodes
			WHERE id = ?
			
			UNION ALL
			
			-- Recursive case: find all children (nodes pointed to by outgoing edges)
			SELECT n.id, d.depth + 1
			FROM knowledge_nodes n
			JOIN knowledge_edges e ON e.to_node_id = n.id
			JOIN dependents d ON e.from_node_id = d.id
			WHERE d.depth < 100  -- Safety limit
		)
		SELECT DISTINCT id
		FROM dependents
		ORDER BY depth ASC
	`

	rows, err := p.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query dependent nodes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var nodeIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan node ID: %w", err)
		}
		nodeIDs = append(nodeIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dependent nodes: %w", err)
	}

	return nodeIDs, nil
}

// Close closes the database connection
func (p *KnowledgePlugin) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}
