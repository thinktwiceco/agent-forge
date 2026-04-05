package brain

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

const dbFileName = "brain.db"

// parseSQLiteDatetime parses created_at/updated_at values from SQLite (layout varies by driver/build).
func parseSQLiteDatetime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	if len(s) >= 19 {
		if t, err := time.Parse("2006-01-02 15:04:05", s[:19]); err == nil {
			return t
		}
	}
	return time.Time{}
}

// Canonical graph vocabulary: root -> topic -> conversation.
const (
	nodeTypeTopic           = "topic"
	nodeTypeConversation    = "conversation"
	edgeTypeHasTopic        = "HAS_TOPIC"
	edgeTypeHasConversation = "HAS_CONVERSATION"
	// edgeTypeRelated is a generic associative edge (tests, ad-hoc links); not part of root→topic→conversation.
	edgeTypeRelated = "related_to"
)

// defaultConversationTopic is the topic label for sessions until dreaming assigns distilled labels (or if the model returns none).
const defaultConversationTopic = "pending"

// BrainPlugin is the persistent memory system for an agent.
//
// All brain data lives under a single directory tree rooted at:
//
//	<workingDir>/brain/
//	  brain.db                  — SQLite knowledge graph (nodes, edges, FTS index)
//	  MEMORY.md                 — daily working notes: quick facts, easy edit (agent-edited via tools)
//	  persistence/
//	    YYYY-MM-DD/
//	      <conv_id>.md          — distilled notes from the dreaming cron job
//	                              NOT written by the main agent during a conversation;
//	                              created only after the dreaming runner has analysed
//	                              the conversation JSON in data/conversations/.
//
// The dreaming cron (DreamingRunner) reads raw conversation JSON files, distils key
// points via an LLM call, and writes the result here; it may also rewrite brain/MEMORY.md
// and optionally upsert one synthetic conversation node from that pass. Each conversation
// graph node's content field stores the distilled summary for graph search and recall.
type BrainPlugin struct {
	db         *sql.DB
	dir        string // <workingDir>/brain/ — SQLite DB and all brain files
	workingDir string // agent working directory — base for data/conversations/ path
	querier    *GraphQuerier
	cfg        PluginConfig
	// llmEngine is injected by the agent via the LLMEngineAware interface.
	// Used exclusively by DreamingRunner; nil until injection completes.
	llmEngine llms.LLMEngine
	// dreamMu serializes RunPending (scheduled dreamTime run, dream tool).
	dreamMu sync.Mutex
	// memoryMu serializes brain/MEMORY.md reads and writes (tools, prompt injection, dreaming cleanup).
	memoryMu sync.RWMutex
}

// NewBrainPlugin creates a new brain graph plugin with default dreaming settings.
func NewBrainPlugin(workingDir string) *BrainPlugin {
	return NewBrainPluginWithConfig(workingDir, nil)
}

// NewBrainPluginWithConfig creates a brain plugin; cfg nil uses defaults.
func NewBrainPluginWithConfig(workingDir string, cfg *PluginConfig) *BrainPlugin {
	dir := filepath.Join(workingDir, "brain")
	return &BrainPlugin{dir: dir, workingDir: workingDir, cfg: MergePluginConfig(cfg)}
}

// openDB opens the SQLite database connection
func (p *BrainPlugin) openDB() error {
	// Data migration: move old knowledge DB to new brain location (best-effort).
	oldDir := filepath.Join(p.dir[:len(p.dir)-len("brain")], "knowledge")
	oldDB := filepath.Join(oldDir, dbFileName)
	newDB := filepath.Join(p.dir, dbFileName)
	if _, err := os.Stat(newDB); os.IsNotExist(err) {
		if _, err := os.Stat(oldDB); err == nil {
			_ = os.MkdirAll(p.dir, 0755)
			_ = os.Rename(oldDB, newDB)
		}
	}

	dbPath := filepath.Join(p.dir, dbFileName)

	if err := os.MkdirAll(p.dir, 0755); err != nil {
		return fmt.Errorf("failed to create brain directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=1")
	if err != nil {
		return fmt.Errorf("failed to open brain database: %w", err)
	}

	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect to brain database: %w", err)
	}

	p.db = db
	return nil
}

// ensureSchema creates the database schema if it doesn't exist
func (p *BrainPlugin) ensureSchema() error {
	ctx := context.Background()

	createNodesTable := `
		CREATE TABLE IF NOT EXISTS brain_nodes (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			content TEXT NOT NULL,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
	if _, err := p.db.ExecContext(ctx, createNodesTable); err != nil {
		return fmt.Errorf("failed to create nodes table: %w", err)
	}

	createEdgesTable := `
		CREATE TABLE IF NOT EXISTS brain_edges (
			id TEXT PRIMARY KEY,
			from_node_id TEXT NOT NULL,
			to_node_id TEXT NOT NULL,
			relation_type TEXT NOT NULL,
			weight REAL DEFAULT 1.0,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(from_node_id) REFERENCES brain_nodes(id) ON DELETE CASCADE,
			FOREIGN KEY(to_node_id) REFERENCES brain_nodes(id) ON DELETE CASCADE
		)
	`
	if _, err := p.db.ExecContext(ctx, createEdgesTable); err != nil {
		return fmt.Errorf("failed to create edges table: %w", err)
	}

	createTypesTable := `
		CREATE TABLE IF NOT EXISTS brain_types (
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

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_nodes_type ON brain_nodes(type)",
		"CREATE INDEX IF NOT EXISTS idx_nodes_content ON brain_nodes(content)",
		"CREATE INDEX IF NOT EXISTS idx_edges_from ON brain_edges(from_node_id)",
		"CREATE INDEX IF NOT EXISTS idx_edges_to ON brain_edges(to_node_id)",
		"CREATE INDEX IF NOT EXISTS idx_edges_type ON brain_edges(relation_type)",
		"CREATE INDEX IF NOT EXISTS idx_types_category ON brain_types(category)",
		"CREATE INDEX IF NOT EXISTS idx_types_name ON brain_types(name)",
	}

	for _, idx := range indexes {
		if _, err := p.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	if err := p.ensureBrainNodeExtendedColumns(ctx); err != nil {
		return fmt.Errorf("brain node columns: %w", err)
	}
	if err := p.backfillSearchTextIfNeeded(ctx); err != nil {
		return fmt.Errorf("brain search_text backfill: %w", err)
	}
	if err := p.migrateFTSToSearchText(ctx); err != nil {
		// FTS5 may be unavailable in some SQLite builds; find falls back to LIKE.
		agentforge.Debug("brain: FTS5 search_text migration skipped: %v", err)
	}

	if err := p.seedDefaultTypes(); err != nil {
		return fmt.Errorf("failed to seed default types: %w", err)
	}

	return nil
}

// seedDefaultTypes populates the types table with default types and creates Omnia Nunc Node
func (p *BrainPlugin) seedDefaultTypes() error {
	defaultNodeTypes := []struct {
		name        string
		description string
	}{
		{nodeTypeTopic, "Topical grouping for long-term memory conversations"},
		{nodeTypeConversation, "Indexed chat session; metadata holds conv_id, topics, access times; content is distilled summary; title/description/distillation_reason are columns when dreamed"},
	}

	defaultEdgeTypes := []struct {
		name        string
		description string
	}{
		{edgeTypeHasTopic, "Links graph root to a topic node"},
		{edgeTypeHasConversation, "Links a topic node to a conversation node"},
		{edgeTypeRelated, "Generic link between two nodes"},
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

	if err := p.ensureOmniaNuncNode(); err != nil {
		return fmt.Errorf("failed to create Omnia Nunc Node: %w", err)
	}

	return nil
}

const omniaNuncNodeID = "omnia-nunc-root"

// ensureOmniaNuncNode creates the central Omnia Nunc Node if it doesn't exist
func (p *BrainPlugin) ensureOmniaNuncNode() error {
	ctx := context.Background()

	query := `SELECT COUNT(*) FROM brain_nodes WHERE id = ?`
	var count int
	err := p.db.QueryRowContext(ctx, query, omniaNuncNodeID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for Omnia Nunc Node: %w", err)
	}

	if count > 0 {
		return nil
	}

	now := time.Now()
	rootContent := "Omnia Nunc Node"
	st := buildSearchText(rootContent, "", "", "")
	insertQuery := `
		INSERT INTO brain_nodes (id, type, content, metadata, title, description, distillation_reason, search_text, created_at, updated_at)
		VALUES (?, ?, ?, ?, NULL, NULL, NULL, ?, ?, ?)
	`

	metadata := map[string]any{
		"system": true,
		"root":   true,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = p.db.ExecContext(ctx, insertQuery, omniaNuncNodeID, nodeTypeTopic, rootContent, string(metadataJSON), st, now, now)
	if err != nil {
		return fmt.Errorf("failed to insert Omnia Nunc Node: %w", err)
	}

	return nil
}

// isOmniaNuncNode checks if a node ID is the Omnia Nunc Node
func (p *BrainPlugin) isOmniaNuncNode(nodeID string) bool {
	return nodeID == omniaNuncNodeID
}

// validateEdgeRules enforces strict edge validation based on node types
//
//nolint:unused // reserved for strict validation
func (p *BrainPlugin) validateEdgeRules(fromNodeID, toNodeID, relationType string) error {
	fromNode, err := p.getNode(fromNodeID)
	if err != nil {
		return fmt.Errorf("failed to get source node: %w", err)
	}

	toNode, err := p.getNode(toNodeID)
	if err != nil {
		return fmt.Errorf("failed to get target node: %w", err)
	}

	switch relationType {
	case edgeTypeHasTopic:
		if fromNode.ID != omniaNuncNodeID {
			return fmt.Errorf("%s edge requires source to be graph root, got: %s", edgeTypeHasTopic, fromNode.ID)
		}
		if toNode.Type != nodeTypeTopic || p.isOmniaNuncNode(toNode.ID) {
			return fmt.Errorf("%s edge requires target type %s", edgeTypeHasTopic, nodeTypeTopic)
		}
	case edgeTypeHasConversation:
		if fromNode.Type != nodeTypeTopic || p.isOmniaNuncNode(fromNode.ID) {
			return fmt.Errorf("%s edge requires source type %s", edgeTypeHasConversation, nodeTypeTopic)
		}
		if toNode.Type != nodeTypeConversation {
			return fmt.Errorf("%s edge requires target type %s, got: %s", edgeTypeHasConversation, nodeTypeConversation, toNode.Type)
		}
	case edgeTypeRelated:
		if fromNode.ID == toNode.ID {
			return fmt.Errorf("%s edge cannot be a self-loop", edgeTypeRelated)
		}
	default:
		return fmt.Errorf("unknown edge type: %s (valid: %s, %s, %s)", relationType, edgeTypeHasTopic, edgeTypeHasConversation, edgeTypeRelated)
	}

	return nil
}

// saveNode stores a new node in the graph (no display columns; search_text from content only).
func (p *BrainPlugin) saveNode(nodeType, content string, metadata map[string]any) (string, error) {
	return p.saveNodeWithDisplay(nodeType, content, metadata, "", "", "")
}

// saveNodeWithDisplay inserts a node with optional title/description/distillation_reason columns and computed search_text.
func (p *BrainPlugin) saveNodeWithDisplay(nodeType, content string, metadata map[string]any, title, description, distillationReason string) (string, error) {
	ctx := context.Background()

	nodeID := uuid.New().String()
	now := time.Now()

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var metadataValue sql.NullString
	if metadata != nil {
		metadataValue.String = string(metadataJSON)
		metadataValue.Valid = true
	}

	st := buildSearchText(content, title, description, distillationReason)

	var t, d, r sql.NullString
	if strings.TrimSpace(title) != "" {
		t.String = strings.TrimSpace(title)
		t.Valid = true
	}
	if strings.TrimSpace(description) != "" {
		d.String = strings.TrimSpace(description)
		d.Valid = true
	}
	if strings.TrimSpace(distillationReason) != "" {
		r.String = strings.TrimSpace(distillationReason)
		r.Valid = true
	}

	query := `
		INSERT INTO brain_nodes (id, type, content, metadata, title, description, distillation_reason, search_text, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = p.db.ExecContext(ctx, query, nodeID, nodeType, content, metadataValue, t, d, r, st, now, now)
	if err != nil {
		return "", fmt.Errorf("failed to insert node: %w", err)
	}

	return nodeID, nil
}

// saveEdge stores a new edge in the graph
func (p *BrainPlugin) saveEdge(fromID, toID, relationType string, weight float64, metadata map[string]any) (string, error) {
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
		INSERT INTO brain_edges (id, from_node_id, to_node_id, relation_type, weight, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = p.db.ExecContext(ctx, query, edgeID, fromID, toID, relationType, weight, string(metadataJSON))
	if err != nil {
		return "", fmt.Errorf("failed to insert edge: %w", err)
	}

	return edgeID, nil
}

func (p *BrainPlugin) edgeExists(fromID, toID, relationType string) (bool, error) {
	ctx := context.Background()
	query := `
		SELECT COUNT(*)
		FROM brain_edges
		WHERE from_node_id = ? AND to_node_id = ? AND relation_type = ?
	`
	var count int
	if err := p.db.QueryRowContext(ctx, query, fromID, toID, relationType).Scan(&count); err != nil {
		return false, fmt.Errorf("failed to query edge existence: %w", err)
	}
	return count > 0, nil
}

func (p *BrainPlugin) ensureEdge(fromID, toID, relationType string) error {
	if err := p.validateEdgeType(relationType); err != nil {
		return err
	}
	if err := p.validateEdgeRules(fromID, toID, relationType); err != nil {
		return err
	}
	exists, err := p.edgeExists(fromID, toID, relationType)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = p.saveEdge(fromID, toID, relationType, 1.0, nil)
	return err
}

func (p *BrainPlugin) deleteEdges(fromID, toID, relationType string) error {
	ctx := context.Background()
	query := `
		DELETE FROM brain_edges
		WHERE from_node_id = ? AND to_node_id = ? AND relation_type = ?
	`
	_, err := p.db.ExecContext(ctx, query, fromID, toID, relationType)
	if err != nil {
		return fmt.Errorf("failed to delete edges: %w", err)
	}
	return nil
}

// getNode retrieves a node by ID
func (p *BrainPlugin) getNode(nodeID string) (*Node, error) {
	ctx := context.Background()
	query := fmt.Sprintf(`SELECT %s FROM brain_nodes WHERE id = ?`, sqlBrainNodeColumns)
	n, err := scanBrainNodeFromScanner(p.db.QueryRowContext(ctx, query, nodeID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("node not found: %s", nodeID)
		}
		return nil, fmt.Errorf("failed to query node: %w", err)
	}
	return &n, nil
}

// findNodesByContent searches by content, title, description, distillation_reason, or search_text.
func (p *BrainPlugin) findNodesByContent(query string, exact bool) ([]Node, error) {
	ctx := context.Background()

	var sqlQuery string
	var args []interface{}

	if exact {
		sqlQuery = fmt.Sprintf(`
			SELECT %s
			FROM brain_nodes
			WHERE content = ?
			   OR title = ?
			   OR description = ?
			   OR distillation_reason = ?
			   OR search_text = ?
		`, sqlBrainNodeColumns)
		args = []interface{}{query, query, query, query, query}
	} else {
		like := "%" + query + "%"
		sqlQuery = fmt.Sprintf(`
			SELECT %s
			FROM brain_nodes
			WHERE content LIKE ?
			   OR title LIKE ?
			   OR description LIKE ?
			   OR distillation_reason LIKE ?
			   OR search_text LIKE ?
		`, sqlBrainNodeColumns)
		args = []interface{}{like, like, like, like, like}
	}

	return p.queryNodes(ctx, sqlQuery, args...)
}

// findNodesByType searches for nodes by type with optional pagination
func (p *BrainPlugin) findNodesByType(nodeType string, limit, offset int) ([]Node, error) {
	ctx := context.Background()

	query := fmt.Sprintf(`
		SELECT %s
		FROM brain_nodes
		WHERE type = ?
		ORDER BY created_at DESC
	`, sqlBrainNodeColumns)

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
func (p *BrainPlugin) findNodesByIDs(nodeIDs []string) ([]Node, error) {
	if len(nodeIDs) == 0 {
		return []Node{}, nil
	}

	ctx := context.Background()

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
		SELECT %s
		FROM brain_nodes
		WHERE id IN (%s)
	`, sqlBrainNodeColumns, placeholders)

	return p.queryNodes(ctx, query, args...)
}

// queryNodes executes a query that selects sqlBrainNodeColumns in order.
func (p *BrainPlugin) queryNodes(ctx context.Context, query string, args ...interface{}) ([]Node, error) {
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var nodes []Node
	for rows.Next() {
		node, err := scanBrainNodeFromScanner(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}
		nodes = append(nodes, node)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating nodes: %w", err)
	}

	return nodes, nil
}

// updateNode updates an existing node (preserves title/description/distillation_reason columns; recomputes search_text).
func (p *BrainPlugin) updateNode(nodeID, nodeType, content string, metadata map[string]any) error {
	ctx := context.Background()

	existing, err := p.getNode(nodeID)
	if err != nil {
		return err
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var metadataValue sql.NullString
	if metadata != nil {
		metadataValue.String = string(metadataJSON)
		metadataValue.Valid = true
	}

	st := buildSearchText(content, existing.Title, existing.Description, existing.DistillationReason)

	query := `
		UPDATE brain_nodes
		SET type = ?, content = ?, metadata = ?, search_text = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := p.db.ExecContext(ctx, query, nodeType, content, metadataValue, st, time.Now(), nodeID)
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
func (p *BrainPlugin) deleteNode(nodeID string) error {
	ctx := context.Background()

	query := `DELETE FROM brain_nodes WHERE id = ?`

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
func (p *BrainPlugin) getNodeEdges(nodeID string) ([]Edge, error) {
	ctx := context.Background()

	query := `
		SELECT id, from_node_id, to_node_id, relation_type, weight, metadata, created_at
		FROM brain_edges
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
func (p *BrainPlugin) deleteEdge(edgeID string) error {
	ctx := context.Background()

	query := `DELETE FROM brain_edges WHERE id = ?`

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
func (p *BrainPlugin) addNodeType(name, description string, metadata map[string]any) (string, error) {
	return p.addType("node_type", name, description, metadata)
}

// addEdgeType creates a new edge type
func (p *BrainPlugin) addEdgeType(name, description string, metadata map[string]any) (string, error) {
	return p.addType("edge_type", name, description, metadata)
}

// addType creates a new type (node or edge)
func (p *BrainPlugin) addType(category, name, description string, metadata map[string]any) (string, error) {
	ctx := context.Background()

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
		INSERT INTO brain_types (id, category, name, description, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = p.db.ExecContext(ctx, query, typeID, category, name, description, string(metadataJSON), now)
	if err != nil {
		return "", fmt.Errorf("failed to insert type: %w", err)
	}

	return typeID, nil
}

// listNodeTypes retrieves all node types
func (p *BrainPlugin) listNodeTypes() ([]Type, error) {
	return p.listTypes("node_type")
}

// listEdgeTypes retrieves all edge types
func (p *BrainPlugin) listEdgeTypes() ([]Type, error) {
	return p.listTypes("edge_type")
}

// listTypes retrieves all types for a given category
func (p *BrainPlugin) listTypes(category string) ([]Type, error) {
	ctx := context.Background()

	query := `
		SELECT id, category, name, description, metadata, created_at
		FROM brain_types
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
func (p *BrainPlugin) getType(category, name string) (*Type, error) {
	ctx := context.Background()

	query := `
		SELECT id, category, name, description, metadata, created_at
		FROM brain_types
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
func (p *BrainPlugin) typeExists(category, name string) (bool, error) {
	ctx := context.Background()

	query := `SELECT COUNT(*) FROM brain_types WHERE category = ? AND name = ?`

	var count int
	err := p.db.QueryRowContext(ctx, query, category, name).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check type existence: %w", err)
	}

	return count > 0, nil
}

// validateNodeType checks if a node type is valid
func (p *BrainPlugin) validateNodeType(nodeType string) error {
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
func (p *BrainPlugin) validateEdgeType(edgeType string) error {
	exists, err := p.typeExists("edge_type", edgeType)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("edge type '%s' does not exist", edgeType)
	}
	return nil
}

// getTopicsUnderRoot returns topic nodes linked from the graph root via HAS_TOPIC.
func (p *BrainPlugin) getTopicsUnderRoot() ([]Node, error) {
	if p.db == nil {
		return []Node{}, nil
	}

	ctx := context.Background()

	query := fmt.Sprintf(`
		SELECT %s
		FROM brain_nodes n
		JOIN brain_edges e ON e.to_node_id = n.id
		WHERE e.from_node_id = ? AND e.relation_type = ?
		ORDER BY n.created_at ASC
	`, sqlBrainNodeColumnsN)

	return p.queryNodes(ctx, query, omniaNuncNodeID, edgeTypeHasTopic)
}

func (p *BrainPlugin) getTopicNodeByName(topic string) (*Node, error) {
	if p.db == nil {
		return nil, fmt.Errorf("brain database not initialized")
	}
	norm := normalizeTopicName(topic)
	if norm == "" {
		return nil, fmt.Errorf("topic name is required")
	}
	ctx := context.Background()
	query := fmt.Sprintf(`
		SELECT %s
		FROM brain_nodes
		WHERE type = ?
		  AND id != ?
		  AND (
			content = ?
			OR (json_valid(metadata) AND json_extract(metadata, '$.topic_name') = ?)
		  )
		LIMIT 1
	`, sqlBrainNodeColumns)
	nodes, err := p.queryNodes(ctx, query, nodeTypeTopic, omniaNuncNodeID, norm, norm)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("topic not found: %s", norm)
	}
	return &nodes[0], nil
}

func (p *BrainPlugin) ensureTopicNode(topic string) (*Node, error) {
	norm := normalizeTopicName(topic)
	if norm == "" {
		return nil, fmt.Errorf("topic name is required")
	}
	if existing, err := p.getTopicNodeByName(norm); err == nil {
		if err := p.ensureEdge(omniaNuncNodeID, existing.ID, edgeTypeHasTopic); err != nil {
			return nil, err
		}
		return existing, nil
	} else if !strings.Contains(err.Error(), "topic not found") {
		return nil, err
	}

	metadata := map[string]any{
		"topic_name": norm,
	}
	nodeID, err := p.saveNodeWithDisplay(nodeTypeTopic, norm, metadata, norm, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to save topic node: %w", err)
	}
	if err := p.ensureEdge(omniaNuncNodeID, nodeID, edgeTypeHasTopic); err != nil {
		_ = p.deleteNode(nodeID)
		return nil, fmt.Errorf("failed to link topic to root: %w", err)
	}
	return p.getNode(nodeID)
}

func (p *BrainPlugin) getConversationsForTopic(topic string) ([]Node, error) {
	topicNode, err := p.getTopicNodeByName(topic)
	if err != nil {
		return nil, err
	}
	return p.getConversationsForTopicID(topicNode.ID)
}

func (p *BrainPlugin) getConversationsForTopicID(topicID string) ([]Node, error) {
	if p.db == nil {
		return []Node{}, nil
	}
	ctx := context.Background()
	query := fmt.Sprintf(`
		SELECT %s
		FROM brain_nodes n
		JOIN brain_edges e ON e.to_node_id = n.id
		WHERE e.from_node_id = ? AND e.relation_type = ?
		ORDER BY n.updated_at DESC
	`, sqlBrainNodeColumnsN)
	return p.queryNodes(ctx, query, topicID, edgeTypeHasConversation)
}

func (p *BrainPlugin) getTopicsForConversationNodeID(conversationID string) ([]Node, error) {
	if p.db == nil {
		return []Node{}, nil
	}
	ctx := context.Background()
	query := fmt.Sprintf(`
		SELECT %s
		FROM brain_nodes n
		JOIN brain_edges e ON e.from_node_id = n.id
		WHERE e.to_node_id = ? AND e.relation_type = ?
		ORDER BY n.content ASC
	`, sqlBrainNodeColumnsN)
	return p.queryNodes(ctx, query, conversationID, edgeTypeHasConversation)
}

// ensureConversationHasDefaultTopic links conversationNodeID to defaultConversationTopic when it has no topic edges (new sessions or legacy orphans).
func (p *BrainPlugin) ensureConversationHasDefaultTopic(conversationNodeID string) error {
	if p.db == nil {
		return nil
	}
	topics, err := p.getTopicsForConversationNodeID(conversationNodeID)
	if err != nil {
		return err
	}
	if len(topics) > 0 {
		return nil
	}
	topicNode, err := p.ensureTopicNode(defaultConversationTopic)
	if err != nil {
		return err
	}
	return p.ensureEdge(topicNode.ID, conversationNodeID, edgeTypeHasConversation)
}

// listAllConversationNodes returns every session conversation node (excludes graph root).
func (p *BrainPlugin) listAllConversationNodes() ([]Node, error) {
	if p.db == nil {
		return []Node{}, nil
	}
	ctx := context.Background()
	query := fmt.Sprintf(`
		SELECT %s
		FROM brain_nodes
		WHERE type = ? AND id != ?
		ORDER BY updated_at DESC
	`, sqlBrainNodeColumns)
	return p.queryNodes(ctx, query, nodeTypeConversation, omniaNuncNodeID)
}

// listConversationsInTimeRange returns recall rows whose effectiveRecallTime lies in [start, end] inclusive.
func (p *BrainPlugin) listConversationsInTimeRange(start, end time.Time, topic string) ([]ConversationRecallItem, error) {
	var (
		nodes []Node
		err   error
	)
	if normalizeTopicName(topic) != "" {
		nodes, err = p.getConversationsForTopic(topic)
	} else {
		nodes, err = p.listAllConversationNodes()
	}
	if err != nil {
		return nil, err
	}
	var out []ConversationRecallItem
	for i := range nodes {
		n := &nodes[i]
		if err := n.Validate(); err != nil {
			return nil, fmt.Errorf("conversation node %s failed validation: %w", n.ID, err)
		}
		t, err := effectiveRecallTime(n)
		if err != nil {
			return nil, fmt.Errorf("conversation node %s has invalid recall time: %w", n.ID, err)
		}
		if t.Before(start) || t.After(end) {
			continue
		}
		out = append(out, ConversationRecallItem{
			ID:          n.ID,
			Title:       recallTitle(n),
			Description: recallDescription(n),
			Topics:      recallTopics(n),
		})
	}
	return out, nil
}

func (p *BrainPlugin) listConversationTopics() ([]ConversationTopicItem, error) {
	topics, err := p.getTopicsUnderRoot()
	if err != nil {
		return nil, err
	}
	out := make([]ConversationTopicItem, 0, len(topics))
	for _, topic := range topics {
		conversations, err := p.getConversationsForTopicID(topic.ID)
		if err != nil {
			return nil, err
		}
		for i := range conversations {
			if err := conversations[i].Validate(); err != nil {
				return nil, fmt.Errorf("conversation node %s failed validation: %w", conversations[i].ID, err)
			}
		}
		if len(conversations) == 0 {
			continue
		}
		out = append(out, ConversationTopicItem{
			ID:                topic.ID,
			Name:              normalizeTopicName(topic.GetTitle()),
			ConversationCount: len(conversations),
		})
	}
	return out, nil
}

func (p *BrainPlugin) replaceConversationTopics(conversationNodeID string, topics []string) error {
	normalized := normalizeTopicNames(topics)
	currentTopics, err := p.getTopicsForConversationNodeID(conversationNodeID)
	if err != nil {
		return err
	}

	currentByName := make(map[string]Node, len(currentTopics))
	for _, topic := range currentTopics {
		currentByName[normalizeTopicName(topic.GetTitle())] = topic
	}
	desired := make(map[string]bool, len(normalized))
	for _, topic := range normalized {
		desired[topic] = true
		topicNode, err := p.ensureTopicNode(topic)
		if err != nil {
			return err
		}
		if err := p.ensureEdge(topicNode.ID, conversationNodeID, edgeTypeHasConversation); err != nil {
			return err
		}
	}

	for topicName, topicNode := range currentByName {
		if desired[topicName] {
			continue
		}
		if err := p.deleteEdges(topicNode.ID, conversationNodeID, edgeTypeHasConversation); err != nil {
			return err
		}
		if err := p.pruneTopicIfEmpty(topicNode.ID); err != nil {
			return err
		}
	}

	return nil
}

func (p *BrainPlugin) pruneTopicIfEmpty(topicNodeID string) error {
	if topicNodeID == "" || p.isOmniaNuncNode(topicNodeID) {
		return nil
	}
	conversations, err := p.getConversationsForTopicID(topicNodeID)
	if err != nil {
		return err
	}
	if len(conversations) > 0 {
		return nil
	}
	if err := p.deleteEdges(omniaNuncNodeID, topicNodeID, edgeTypeHasTopic); err != nil {
		return err
	}
	return p.deleteNode(topicNodeID)
}

// addNode creates a topic or conversation node following the root -> topic -> conversation graph.
func (p *BrainPlugin) addNode(parentIdentifier, edgeType, nodeType, name, content string) (string, error) {
	if err := p.validateEdgeType(edgeType); err != nil {
		return "", err
	}
	if err := p.validateNodeType(nodeType); err != nil {
		return "", err
	}

	parent, err := p.resolveNode(parentIdentifier)
	if err != nil {
		return "", fmt.Errorf("parent not found: %w", err)
	}
	if content == "" {
		content = name
	}

	switch {
	case p.isOmniaNuncNode(parent.ID):
		if edgeType != edgeTypeHasTopic || nodeType != nodeTypeTopic {
			return "", fmt.Errorf("root may only attach %q nodes via %q", nodeTypeTopic, edgeTypeHasTopic)
		}
		topicNode, err := p.ensureTopicNode(name)
		if err != nil {
			return "", err
		}
		return topicNode.ID, nil
	case parent.Type == nodeTypeTopic:
		if edgeType != edgeTypeHasConversation || nodeType != nodeTypeConversation {
			return "", fmt.Errorf("topic nodes may only attach %q nodes via %q", nodeTypeConversation, edgeTypeHasConversation)
		}
		nodeID, err := p.saveNodeWithDisplay(nodeTypeConversation, content, map[string]any{}, name, "", "")
		if err != nil {
			return "", fmt.Errorf("failed to save node: %w", err)
		}
		if err := p.ensureEdge(parent.ID, nodeID, edgeTypeHasConversation); err != nil {
			_ = p.deleteNode(nodeID)
			return "", fmt.Errorf("failed to create edge: %w", err)
		}
		return nodeID, nil
	default:
		return "", fmt.Errorf("parent must be graph root or topic node")
	}
}

// forgetCascade deletes a node and all its dependents (cascade delete)
func (p *BrainPlugin) forgetCascade(identifier string) (int, error) {
	node, err := p.getNode(identifier)
	if err != nil {
		nodes, err := p.findNodesByContent(identifier, false)
		if err != nil {
			return 0, fmt.Errorf("failed to find node: %w", err)
		}
		if len(nodes) == 0 {
			return 0, fmt.Errorf("node not found: %s", identifier)
		}
		node = &nodes[0]
	}

	if p.isOmniaNuncNode(node.ID) {
		return 0, fmt.Errorf("cannot delete the Omnia Nunc Node (system root node)")
	}

	dependentIDs, err := p.getCascadeDeleteNodes(node.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to get dependent nodes: %w", err)
	}

	deletedCount := 0
	for i := len(dependentIDs) - 1; i >= 0; i-- {
		err := p.deleteNode(dependentIDs[i])
		if err != nil {
			continue
		}
		deletedCount++
	}

	return deletedCount, nil
}

// getCascadeDeleteNodes returns all nodes that should be deleted in cascade
func (p *BrainPlugin) getCascadeDeleteNodes(nodeID string) ([]string, error) {
	ctx := context.Background()

	query := `
		WITH RECURSIVE dependents(id, depth) AS (
			SELECT id, 0 as depth
			FROM brain_nodes
			WHERE id = ?
			
			UNION ALL
			
			SELECT n.id, d.depth + 1
			FROM brain_nodes n
			JOIN brain_edges e ON e.to_node_id = n.id
			JOIN dependents d ON e.from_node_id = d.id
			WHERE d.depth < 100
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

// ─── Filesystem path helpers ──────────────────────────────────────────────────
//
// All brain file paths live under <workingDir>/brain/:
//
//	brain/
//	  brain.db          — SQLite graph
//	  MEMORY.md         — daily working notes (agent-edited; quick read/write)
//	  persistence/
//	    YYYY-MM-DD/
//	      <conv_id>.md  — distilled notes written by DreamingRunner (not the main agent)

// distilledDir returns the persistence directory for a given date, creating it if needed.
func (p *BrainPlugin) distilledDir(date time.Time) (string, error) {
	dir := filepath.Join(p.dir, "persistence", date.Format("2006-01-02"))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create persistence directory: %w", err)
	}
	return dir, nil
}

// distilledPath returns the path to the dreaming-output file for a conversation.
// Does not create the file.
func (p *BrainPlugin) distilledPath(convID string, date time.Time) string {
	return filepath.Join(p.dir, "persistence", date.Format("2006-01-02"), convID+".md")
}

func (p *BrainPlugin) getConversationNodeByConvID(convID string) (*Node, error) {
	if p.db == nil {
		return nil, fmt.Errorf("brain database not initialized")
	}
	ctx := context.Background()
	query := fmt.Sprintf(`
		SELECT %s
		FROM brain_nodes
		WHERE type = ?
		  AND (id = ?
		    OR (json_valid(metadata) AND json_extract(metadata, '$.conv_id') = ?))
		LIMIT 1
	`, sqlBrainNodeColumns)
	nodes, err := p.queryNodes(ctx, query, nodeTypeConversation, convID, convID)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("conversation node not found for %s", convID)
	}
	return &nodes[0], nil
}

// memoryMDPath returns the absolute path to brain/MEMORY.md (does not create directories).
func (p *BrainPlugin) memoryMDPath() string {
	return filepath.Join(p.dir, "MEMORY.md")
}

// readMemoryMDForInjection returns MEMORY.md content for per-turn system injection, or "" if missing or whitespace-only.
func (p *BrainPlugin) readMemoryMDForInjection() string {
	if p.workingDir == "" {
		return ""
	}
	p.memoryMu.RLock()
	defer p.memoryMu.RUnlock()
	data, err := os.ReadFile(p.memoryMDPath())
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(data))
	if s == "" {
		return ""
	}
	return s
}

// readMemoryMDRaw returns the full file bytes and any read error (NotExist if missing). Caller must not hold memoryMu.
func (p *BrainPlugin) readMemoryMDRaw() ([]byte, error) {
	p.memoryMu.RLock()
	defer p.memoryMu.RUnlock()
	return os.ReadFile(p.memoryMDPath())
}

// writeMemoryMDFull replaces MEMORY.md with content (creates brain/ if needed). Caller must not hold memoryMu.
func (p *BrainPlugin) writeMemoryMDFull(content string) error {
	p.memoryMu.Lock()
	defer p.memoryMu.Unlock()
	if err := os.MkdirAll(p.dir, 0755); err != nil {
		return fmt.Errorf("failed to ensure brain directory: %w", err)
	}
	return os.WriteFile(p.memoryMDPath(), []byte(content), 0644)
}

// writeMemoryMDFullIfUnchanged replaces MEMORY.md only when its current bytes
// still match expectedCurrent. This prevents the dreaming cleanup pass from
// clobbering newer tool or user edits made while the LLM was generating.
func (p *BrainPlugin) writeMemoryMDFullIfUnchanged(expectedCurrent []byte, newContent string) (bool, error) {
	p.memoryMu.Lock()
	defer p.memoryMu.Unlock()
	if err := os.MkdirAll(p.dir, 0755); err != nil {
		return false, fmt.Errorf("failed to ensure brain directory: %w", err)
	}
	current, err := os.ReadFile(p.memoryMDPath())
	if err != nil {
		if os.IsNotExist(err) {
			if len(expectedCurrent) != 0 {
				return false, nil
			}
			current = nil
		} else {
			return false, err
		}
	}
	if string(current) != string(expectedCurrent) {
		return false, nil
	}
	if err := os.WriteFile(p.memoryMDPath(), []byte(newContent), 0644); err != nil {
		return false, err
	}
	return true, nil
}

// appendShortTermMemoryLineLocked appends one "topic: fact" line to MEMORY.md under memoryMu.
func (p *BrainPlugin) appendShortTermMemoryLineLocked(topic, fact string) error {
	p.memoryMu.Lock()
	defer p.memoryMu.Unlock()
	if err := os.MkdirAll(p.dir, 0755); err != nil {
		return fmt.Errorf("failed to ensure brain directory: %w", err)
	}
	path := p.memoryMDPath()
	entry := topic + ": " + fact + "\n"
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return os.WriteFile(path, []byte(entry), 0644)
	}
	if err != nil {
		return err
	}
	trimmed := strings.TrimRight(string(data), "\n\r")
	if trimmed == "" {
		return os.WriteFile(path, []byte(entry), 0644)
	}
	out := trimmed + "\n\n" + topic + ": " + fact + "\n"
	return os.WriteFile(path, []byte(out), 0644)
}

// writeMemoryPromotionPersistence writes a markdown artifact for a synthetic MEMORY.md promotion node.
func (p *BrainPlugin) writeMemoryPromotionPersistence(convID string, date time.Time, title, description, distillationReason string, topics []string, summary string) error {
	destDir, err := p.distilledDir(date)
	if err != nil {
		return err
	}
	destPath := filepath.Join(destDir, convID+".md")
	content := fmt.Sprintf("# Promoted from MEMORY.md — %s\n\nConversation: %s\nSource: memory_md\n\n## Title\n\n%s\n\n## Description\n\n%s\n\n## Distillation reason\n\n%s\n\nTopics: %s\n\n## Summary\n\n%s\n",
		date.Format("2006-01-02"), convID, title, description, distillationReason, strings.Join(topics, ", "), summary)
	return os.WriteFile(destPath, []byte(content), 0644)
}

// upsertConversationNode creates a conversation graph node for convID if it does
// not exist, or updates its last_access timestamp if it does. The node stores the
// relative persistence path so the graph can locate promoted long-term notes.
//
// Each conversation is linked to defaultConversationTopic immediately; dreaming
// replaces topic edges when distilled labels are available.
func (p *BrainPlugin) upsertConversationNode(convID string, t time.Time) error {
	return p.upsertConversationNodeWithMetadata(convID, t, nil)
}

// upsertConversationNodeWithMetadata is like upsertConversationNode but merges extra
// keys into metadata on insert (e.g. source=memory_md for synthetic promotion nodes).
func (p *BrainPlugin) upsertConversationNodeWithMetadata(convID string, t time.Time, extraMeta map[string]any) error {
	if p.db == nil {
		return nil
	}
	ctx := context.Background()
	dateStr := t.Format("2006-01-02")
	stmPath := filepath.Join("brain", "persistence", dateStr, convID+".md")

	// Check whether a conversation node for this convID already exists.
	var nodeID string
	err := p.db.QueryRowContext(ctx, `
		SELECT id FROM brain_nodes
		WHERE type = ?
		  AND (id = ?
		    OR (json_valid(metadata) AND json_extract(metadata, '$.conv_id') = ?))
		LIMIT 1
	`, nodeTypeConversation, convID, convID).Scan(&nodeID)

	if err == sql.ErrNoRows {
		meta := map[string]any{
			"conv_id":       convID,
			"creation_date": t.Format(time.RFC3339),
			"last_access":   t.Format(time.RFC3339),
			"stm_path":      stmPath,
		}
		for k, v := range extraMeta {
			meta[k] = v
		}
		newID, err := p.saveNode(nodeTypeConversation, convID, meta)
		if err != nil {
			return fmt.Errorf("failed to create conversation node: %w", err)
		}
		if err := p.ensureConversationHasDefaultTopic(newID); err != nil {
			return err
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to query conversation node: %w", err)
	}

	// Node exists — update last_access so recent conversations sort first.
	updateQuery := `
		UPDATE brain_nodes
		SET metadata = json_set(coalesce(metadata, '{}'), '$.last_access', ?),
		    updated_at = ?
		WHERE id = ?
	`
	_, err = p.db.ExecContext(ctx, updateQuery, t.Format(time.RFC3339), t, nodeID)
	if err != nil {
		return err
	}
	return p.ensureConversationHasDefaultTopic(nodeID)
}

// updateConversationNodeSummary sets title/description/distillation_reason columns and
// topics/dreamed_at/last_access in metadata. Content is intentionally cleared — the actual
// conversation text lives in the persistence markdown file (stm_path). Validate() is called
// before writing to enforce all required fields.
func (p *BrainPlugin) updateConversationNodeSummary(convID, summary string, topics []string, title, description, distillationReason string, dreamedAt time.Time) error {
	ctx := context.Background()

	node, err := p.getConversationNodeByConvID(convID)
	if err != nil {
		return err
	}

	topics = normalizeTopicNames(topics)
	if len(topics) == 0 {
		topics = []string{defaultConversationTopic}
	}
	distillationReason = strings.TrimSpace(distillationReason)
	title = truncateRunes(strings.TrimSpace(title), maxRecallTitleRunes)
	description = truncateRunes(strings.TrimSpace(description), maxRecallDescriptionRunes)
	now := time.Now()

	meta := make(map[string]any)
	if node.Metadata != nil {
		for k, v := range node.Metadata {
			meta[k] = v
		}
	}
	delete(meta, "title")
	delete(meta, "description")
	delete(meta, "distillation_reason")
	meta["dreamed_at"] = dreamedAt.Format(time.RFC3339)
	meta["last_access"] = now.Format(time.RFC3339)
	meta["topics"] = topics

	candidate := &Node{
		ID:                 node.ID,
		Type:               nodeTypeConversation,
		Title:              title,
		Description:        description,
		DistillationReason: distillationReason,
		Metadata:           meta,
	}
	if err := candidate.Validate(); err != nil {
		return fmt.Errorf("brain: updateConversationNodeSummary: %w", err)
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	st := buildSearchText(summary, title, description, distillationReason)

	updateQuery := `
		UPDATE brain_nodes
		SET content = '',
		    title = ?,
		    description = ?,
		    distillation_reason = ?,
		    search_text = ?,
		    metadata = ?,
		    updated_at = ?
		WHERE id = ?
	`
	_, err = p.db.ExecContext(ctx, updateQuery,
		title,
		description,
		distillationReason,
		st,
		string(metaJSON),
		now,
		node.ID,
	)
	if err != nil {
		return err
	}
	return p.replaceConversationTopics(node.ID, topics)
}

// readSummaryFromFile reads the content of the persistence markdown file for a
// conversation node. It is used instead of the content column, which is cleared after
// dreaming. Returns an empty string if the file does not exist or cannot be read.
func (p *BrainPlugin) readSummaryFromFile(convID string, dreamedAt time.Time) string {
	if p.dir == "" || convID == "" {
		return ""
	}
	path := p.distilledPath(convID, dreamedAt)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// Close closes the database connection
func (p *BrainPlugin) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}
