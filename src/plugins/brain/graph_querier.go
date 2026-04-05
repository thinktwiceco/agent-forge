package brain

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// GraphQuerier handles database traversal and queries for the brain graph.
type GraphQuerier struct {
	db *sql.DB
}

// NewGraphQuerier creates a new graph querier.
func NewGraphQuerier(db *sql.DB) *GraphQuerier {
	return &GraphQuerier{db: db}
}

// queryNodes executes a SQL query that selects sqlBrainNodeColumns (or sqlBrainNodeColumnsN) in order.
func (q *GraphQuerier) queryNodes(ctx context.Context, query string, args ...interface{}) ([]Node, error) {
	rows, err := q.db.QueryContext(ctx, query, args...)
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

// scanNodeWithScore scans the FTS query result: sqlBrainNodeColumnsN + score.
func scanNodeWithScore(rows *sql.Rows) (Node, float64, error) {
	var node Node
	var meta sql.NullString
	var title, desc, reason, searchText sql.NullString
	var score float64
	var createdAt, updatedAt string

	if err := rows.Scan(
		&node.ID, &node.Type, &node.Content, &meta,
		&title, &desc, &reason, &searchText,
		&createdAt, &updatedAt, &score,
	); err != nil {
		return Node{}, 0, err
	}

	applyNodeMetadataJSON(meta, &node)
	if title.Valid {
		node.Title = title.String
	}
	if desc.Valid {
		node.Description = desc.String
	}
	if reason.Valid {
		node.DistillationReason = reason.String
	}
	if searchText.Valid {
		node.SearchText = searchText.String
	}
	node.CreatedAt = parseSQLiteDatetime(createdAt)
	node.UpdatedAt = parseSQLiteDatetime(updatedAt)

	return node, score, nil
}

// getOutNodesWithEdge returns all nodes reachable via outgoing edges from nodeID,
// each annotated with the connecting edge type.
func (q *GraphQuerier) getOutNodesWithEdge(nodeID string) ([]LightNodeWithEdge, error) {
	ctx := context.Background()
	const query = `
		SELECT n.id, n.type, n.content, n.metadata, n.title, e.relation_type
		FROM brain_nodes n
		JOIN brain_edges e ON e.to_node_id = n.id
		WHERE e.from_node_id = ?
		ORDER BY e.relation_type, n.created_at ASC
	`
	return q.scanLightNodesWithEdge(ctx, query, nodeID)
}

// getInNodesWithEdge returns all nodes that have an outgoing edge pointing TO nodeID,
// each annotated with the connecting edge type.
func (q *GraphQuerier) getInNodesWithEdge(nodeID string) ([]LightNodeWithEdge, error) {
	ctx := context.Background()
	const query = `
		SELECT n.id, n.type, n.content, n.metadata, n.title, e.relation_type
		FROM brain_nodes n
		JOIN brain_edges e ON e.from_node_id = n.id
		WHERE e.to_node_id = ?
		ORDER BY e.relation_type, n.created_at ASC
	`
	return q.scanLightNodesWithEdge(ctx, query, nodeID)
}

// scanLightNodesWithEdge selects id, type, content, metadata, title column, relation_type.
func (q *GraphQuerier) scanLightNodesWithEdge(ctx context.Context, query string, args ...interface{}) ([]LightNodeWithEdge, error) {
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query neighbors: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []LightNodeWithEdge
	for rows.Next() {
		var id, nodeType, content, edgeType string
		var metadataJSON sql.NullString
		var titleCol sql.NullString
		if err := rows.Scan(&id, &nodeType, &content, &metadataJSON, &titleCol, &edgeType); err != nil {
			return nil, fmt.Errorf("failed to scan neighbor: %w", err)
		}
		title := content
		if len(title) > 80 {
			title = title[:80] + "..."
		}
		if titleCol.Valid && strings.TrimSpace(titleCol.String) != "" {
			title = strings.TrimSpace(titleCol.String)
		} else if metadataJSON.Valid && metadataJSON.String != "" {
			var meta map[string]any
			if err := json.Unmarshal([]byte(metadataJSON.String), &meta); err == nil {
				if t, ok := meta["title"].(string); ok && t != "" {
					title = t
				}
			}
		}
		result = append(result, LightNodeWithEdge{ID: id, Type: nodeType, Title: title, EdgeType: edgeType})
	}
	return result, rows.Err()
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
		FROM brain_edges
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

		edge.CreatedAt = parseSQLiteDatetime(createdAt)

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
		depth = 2
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
		WITH RECURSIVE related(id, depth, path) AS (
			SELECT id, 0, ',' || id || ','
			FROM brain_nodes
			WHERE id IN (%s)
			UNION ALL
			SELECT n.id, r.depth + 1, r.path || n.id || ','
			FROM brain_nodes n
			JOIN brain_edges e ON (e.to_node_id = n.id OR e.from_node_id = n.id)
			JOIN related r ON (e.from_node_id = r.id OR e.to_node_id = r.id)
			WHERE r.depth < ?
				AND NOT instr(r.path, ',' || n.id || ',')
		)
		SELECT %s, MIN(r.depth) AS min_depth
		FROM related r
		JOIN brain_nodes n ON n.id = r.id
		GROUP BY n.id
		ORDER BY min_depth, n.type, n.content
	`, strings.Join(placeholders, ", "), sqlBrainNodeColumnsN)

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query related nodes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var nodes []Node
	nodeIDMap := make(map[string]bool)

	for rows.Next() {
		var node Node
		var meta sql.NullString
		var title, desc, reason, searchText sql.NullString
		var createdAt, updatedAt string
		var minDepth int
		err := rows.Scan(
			&node.ID, &node.Type, &node.Content, &meta,
			&title, &desc, &reason, &searchText,
			&createdAt, &updatedAt, &minDepth,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}
		applyNodeMetadataJSON(meta, &node)
		if title.Valid {
			node.Title = title.String
		}
		if desc.Valid {
			node.Description = desc.String
		}
		if reason.Valid {
			node.DistillationReason = reason.String
		}
		if searchText.Valid {
			node.SearchText = searchText.String
		}
		node.CreatedAt = parseSQLiteDatetime(createdAt)
		node.UpdatedAt = parseSQLiteDatetime(updatedAt)
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

// findNodesByContentPaginated searches for nodes by content with pagination
func (q *GraphQuerier) findNodesByContentPaginated(query string, exact bool, limit, offset int) ([]Node, error) {
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
			ORDER BY created_at DESC
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
			ORDER BY created_at DESC
		`, sqlBrainNodeColumns)
		args = []interface{}{like, like, like, like, like}
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
			FROM brain_nodes
			WHERE id = ?
			
			UNION ALL
			
			SELECT n.id, n.type, n.content, p.depth + 1, p.path_ids || n.id || ',', p.path_types || ' -> ' || n.type
			FROM brain_nodes n
			JOIN brain_edges e ON (e.to_node_id = n.id OR e.from_node_id = n.id)
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
		SELECT %s
		FROM brain_nodes
		WHERE id IN (%s)
	`, sqlBrainNodeColumns, placeholders)

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

	query := fmt.Sprintf(`
		SELECT %s
		FROM brain_nodes
		ORDER BY created_at DESC
		LIMIT ?
	`, sqlBrainNodeColumns)

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
		FROM brain_nodes
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

	query := `SELECT COUNT(*) FROM brain_nodes WHERE type = ?`

	var count int
	err := q.db.QueryRowContext(ctx, query, nodeType).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count nodes by type: %w", err)
	}

	return count, nil
}
