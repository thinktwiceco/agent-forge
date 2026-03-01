package knowledge

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// GraphQuerier handles database traversal and queries for the knowledge graph.
type GraphQuerier struct {
	db *sql.DB
}

// NewGraphQuerier creates a new graph querier.
func NewGraphQuerier(db *sql.DB) *GraphQuerier {
	return &GraphQuerier{db: db}
}

// queryNodes executes a SQL query and unmarshals returned nodes.
func (q *GraphQuerier) queryNodes(ctx context.Context, query string, args ...interface{}) ([]Node, error) {
	rows, err := q.db.QueryContext(ctx, query, args...)
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

// getDirectLightNodes returns light nodes reachable from fromID via a specific edge type.
func (q *GraphQuerier) getDirectLightNodes(fromID, edgeType string) ([]LightNode, error) {
	ctx := context.Background()
	query := `
		SELECT n.id, n.type, n.content, n.embedding_id, n.metadata, n.created_at, n.updated_at
		FROM knowledge_nodes n
		JOIN knowledge_edges e ON e.to_node_id = n.id
		WHERE e.from_node_id = ? AND e.relation_type = ?
		ORDER BY n.created_at ASC
	`
	nodes, err := q.queryNodes(ctx, query, fromID, edgeType)
	if err != nil {
		return nil, err
	}
	out := make([]LightNode, len(nodes))
	for i, n := range nodes {
		out[i] = toLightNode(n)
	}
	return out, nil
}

// getDirectLightParents returns light nodes that point TO toID via a specific edge type.
func (q *GraphQuerier) getDirectLightParents(toID, edgeType string) ([]LightNode, error) {
	ctx := context.Background()
	query := `
		SELECT n.id, n.type, n.content, n.embedding_id, n.metadata, n.created_at, n.updated_at
		FROM knowledge_nodes n
		JOIN knowledge_edges e ON e.from_node_id = n.id
		WHERE e.to_node_id = ? AND e.relation_type = ?
		ORDER BY n.created_at ASC
	`
	nodes, err := q.queryNodes(ctx, query, toID, edgeType)
	if err != nil {
		return nil, err
	}
	out := make([]LightNode, len(nodes))
	for i, n := range nodes {
		out[i] = toLightNode(n)
	}
	return out, nil
}

// getEdgesBetweenNodes retrieves all edges between a set of nodes
func (q *GraphQuerier) getEdgesBetweenNodes(nodeIDMap map[string]bool) ([]Edge, error) {
	if len(nodeIDMap) == 0 {
		return []Edge{}, nil
	}

	ctx := context.Background()

	nodeIDs := make([]string, 0, len(nodeIDMap))
	for id := range nodeIDMap {
		nodeIDs = append(nodeIDs, id)
	}

	placeholders := make([]string, len(nodeIDs))
	args := make([]interface{}, len(nodeIDs)*2)
	for i, id := range nodeIDs {
		placeholders[i] = "?"
		args[i] = id
		args[len(nodeIDs)+i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, from_node_id, to_node_id, relation_type, weight, metadata, created_at
		FROM knowledge_edges
		WHERE from_node_id IN (%s) AND to_node_id IN (%s)
	`, strings.Join(placeholders, ", "), strings.Join(placeholders, ", "))

	rows, err := q.db.QueryContext(ctx, query, args...)
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

// findRelated finds all nodes related to the given node IDs within the specified depth
func (q *GraphQuerier) findRelated(nodeIDs []string, depth int) (*GraphResult, error) {
	if len(nodeIDs) == 0 {
		return &GraphResult{Nodes: []Node{}, Edges: []Edge{}}, nil
	}

	if depth <= 0 {
		depth = 2 // Default depth
	}

	ctx := context.Background()

	placeholders := make([]string, len(nodeIDs))
	args := make([]interface{}, len(nodeIDs)+1)
	for i, id := range nodeIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	args[len(nodeIDs)] = depth

	query := fmt.Sprintf(`
		WITH RECURSIVE related(id, type, content, embedding_id, metadata, depth, path, created_at, updated_at) AS (
			SELECT id, type, content, embedding_id, metadata, 0 as depth, ',' || id || ',' as path, created_at, updated_at
			FROM knowledge_nodes
			WHERE id IN (%s)
			
			UNION ALL
			
			SELECT n.id, n.type, n.content, n.embedding_id, n.metadata, r.depth + 1, r.path || n.id || ',', n.created_at, n.updated_at
			FROM knowledge_nodes n
			JOIN knowledge_edges e ON (e.to_node_id = n.id OR e.from_node_id = n.id)
			JOIN related r ON (e.from_node_id = r.id OR e.to_node_id = r.id)
			WHERE r.depth < ? 
				AND NOT instr(r.path, ',' || n.id || ',')
		)
		SELECT id, type, content, embedding_id, metadata, MIN(depth) as depth, created_at, updated_at
		FROM related
		GROUP BY id
		ORDER BY depth, type, content
	`, strings.Join(placeholders, ", "))

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query related nodes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var nodes []Node
	nodeIDMap := make(map[string]bool)

	for rows.Next() {
		var node Node
		var embeddingID sql.NullString
		var metadataJSON sql.NullString
		var depth int
		var createdAt, updatedAt string

		err := rows.Scan(
			&node.ID, &node.Type, &node.Content, &embeddingID, &metadataJSON, &depth, &createdAt, &updatedAt,
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
		nodeIDMap[node.ID] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating nodes: %w", err)
	}

	edges, err := q.getEdgesBetweenNodes(nodeIDMap)
	if err != nil {
		return nil, fmt.Errorf("failed to get edges: %w", err)
	}

	return &GraphResult{Nodes: nodes, Edges: edges}, nil
}

// findNodesByTypeAndContent searches for nodes by type and an identifier (ID, content snippet, or title).
func (q *GraphQuerier) findNodesByTypeAndContent(nodeType, identifier string, limit, offset int) ([]Node, error) {
	ctx := context.Background()

	pattern := "%" + identifier + "%"

	query := `
		SELECT id, type, content, embedding_id, metadata, created_at, updated_at
		FROM knowledge_nodes
		WHERE type = ? AND (id = ? OR content LIKE ? OR json_extract(metadata, '$.title') LIKE ?)
		ORDER BY created_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	return q.queryNodes(ctx, query, nodeType, identifier, pattern, pattern)
}

// findNodesByContentPaginated searches for nodes by content with pagination
func (q *GraphQuerier) findNodesByContentPaginated(query string, exact bool, limit, offset int) ([]Node, error) {
	ctx := context.Background()

	var sqlQuery string
	var args []interface{}

	if exact {
		sqlQuery = `
			SELECT id, type, content, embedding_id, metadata, created_at, updated_at
			FROM knowledge_nodes
			WHERE content = ?
			ORDER BY created_at DESC
		`
		args = []interface{}{query}
	} else {
		sqlQuery = `
			SELECT id, type, content, embedding_id, metadata, created_at, updated_at
			FROM knowledge_nodes
			WHERE content LIKE ?
			ORDER BY created_at DESC
		`
		args = []interface{}{"%" + query + "%"}
	}

	if limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			sqlQuery += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	return q.queryNodes(ctx, sqlQuery, args...)
}

// findPath finds the shortest path between two nodes
//
//nolint:unused
func (q *GraphQuerier) findPath(fromID, toID string, maxDepth int) (*GraphResult, error) {
	if maxDepth <= 0 {
		maxDepth = 5
	}

	ctx := context.Background()

	query := `
		WITH RECURSIVE path(id, type, content, depth, path_ids, path_types) AS (
			SELECT id, type, content, 0 as depth, ',' || id || ',' as path_ids, type as path_types
			FROM knowledge_nodes
			WHERE id = ?
			
			UNION ALL
			
			SELECT n.id, n.type, n.content, p.depth + 1, p.path_ids || n.id || ',', p.path_types || ' -> ' || n.type
			FROM knowledge_nodes n
			JOIN knowledge_edges e ON (e.to_node_id = n.id OR e.from_node_id = n.id)
			JOIN path p ON (e.from_node_id = p.id OR e.to_node_id = p.id)
			WHERE p.depth < ?
				AND NOT instr(p.path_ids, ',' || n.id || ',')
				AND (p.id != ? OR n.id = ?)
		)
		SELECT id, type, content, depth, path_ids
		FROM path
		WHERE id = ?
		ORDER BY depth
		LIMIT 1
	`

	var resultID, nodeType, content, pathIDs string
	var depth int

	err := q.db.QueryRowContext(ctx, query, fromID, maxDepth, toID, toID, toID).Scan(
		&resultID, &nodeType, &content, &depth, &pathIDs,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &GraphResult{Nodes: []Node{}, Edges: []Edge{}}, nil
		}
		return nil, fmt.Errorf("failed to find path: %w", err)
	}

	pathIDList := strings.Split(strings.Trim(pathIDs, ","), ",")

	// We can't use q.queryNodes with an IN slice easily without formatting it.
	// We'll construct it exactly like findNodesByIDs
	placeholders := ""
	args := make([]interface{}, len(pathIDList))
	for i, id := range pathIDList {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
		args[i] = id
	}

	queryNodes := fmt.Sprintf(`
		SELECT id, type, content, embedding_id, metadata, created_at, updated_at
		FROM knowledge_nodes
		WHERE id IN (%s)
	`, placeholders)

	nodes, err := q.queryNodes(ctx, queryNodes, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get path nodes: %w", err)
	}

	nodeIDMap := make(map[string]bool)
	for _, id := range pathIDList {
		nodeIDMap[id] = true
	}

	edges, err := q.getEdgesBetweenNodes(nodeIDMap)
	if err != nil {
		return nil, fmt.Errorf("failed to get path edges: %w", err)
	}

	return &GraphResult{Nodes: nodes, Edges: edges}, nil
}

// getAllNodes retrieves all nodes (use with caution on large graphs)
//
//nolint:unused
func (q *GraphQuerier) getAllNodes(limit int) ([]Node, error) {
	ctx := context.Background()

	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, type, content, embedding_id, metadata, created_at, updated_at
		FROM knowledge_nodes
		ORDER BY created_at DESC
		LIMIT ?
	`

	return q.queryNodes(ctx, query, limit)
}

// getNeighborhood gets all nodes within N hops of the given node
//
//nolint:unused
func (q *GraphQuerier) getNeighborhood(nodeID string, depth int) (*GraphResult, error) {
	return q.findRelated([]string{nodeID}, depth)
}

// getNodeCountByType returns the count of nodes for each type
func (q *GraphQuerier) getNodeCountByType() (map[string]int, error) {
	ctx := context.Background()

	query := `
		SELECT type, COUNT(*) as count
		FROM knowledge_nodes
		GROUP BY type
	`

	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query node counts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	counts := make(map[string]int)
	for rows.Next() {
		var nodeType string
		var count int

		if err := rows.Scan(&nodeType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}

		counts[nodeType] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating counts: %w", err)
	}

	return counts, nil
}

// countNodesByType returns the count of nodes for a specific type
//
//nolint:unused // reserved for future stats features
func (q *GraphQuerier) countNodesByType(nodeType string) (int, error) {
	ctx := context.Background()

	query := `SELECT COUNT(*) FROM knowledge_nodes WHERE type = ?`

	var count int
	err := q.db.QueryRowContext(ctx, query, nodeType).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count nodes by type: %w", err)
	}

	return count, nil
}
