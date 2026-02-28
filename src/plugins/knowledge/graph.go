package knowledge

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// GraphResult represents the result of a graph traversal query
type GraphResult struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// findRelated finds all nodes related to the given node IDs within the specified depth
func (p *KnowledgePlugin) findRelated(nodeIDs []string, depth int) (*GraphResult, error) {
	if len(nodeIDs) == 0 {
		return &GraphResult{Nodes: []Node{}, Edges: []Edge{}}, nil
	}

	if depth <= 0 {
		depth = 2 // Default depth
	}

	ctx := context.Background()

	// Build placeholders for IN clause
	placeholders := make([]string, len(nodeIDs))
	args := make([]interface{}, len(nodeIDs)+1)
	for i, id := range nodeIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	args[len(nodeIDs)] = depth

	// Recursive CTE to find related nodes
	// Note: We use GROUP BY to ensure each node appears only once, taking the minimum depth
	query := fmt.Sprintf(`
		WITH RECURSIVE related(id, type, content, embedding_id, metadata, depth, path, created_at, updated_at) AS (
			-- Base case: start with the given nodes
			SELECT id, type, content, embedding_id, metadata, 0 as depth, ',' || id || ',' as path, created_at, updated_at
			FROM knowledge_nodes
			WHERE id IN (%s)
			
			UNION ALL
			
			-- Recursive case: find connected nodes
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

	rows, err := p.db.QueryContext(ctx, query, args...)
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

	// Now fetch edges between all discovered nodes
	edges, err := p.getEdgesBetweenNodes(nodeIDMap)
	if err != nil {
		return nil, fmt.Errorf("failed to get edges: %w", err)
	}

	return &GraphResult{Nodes: nodes, Edges: edges}, nil
}

// findPath finds the shortest path between two nodes
//
//nolint:unused // reserved for future path-finding features
func (p *KnowledgePlugin) findPath(fromID, toID string, maxDepth int) (*GraphResult, error) {
	if maxDepth <= 0 {
		maxDepth = 5 // Default max depth for path finding
	}

	ctx := context.Background()

	// Recursive CTE to find path
	query := `
		WITH RECURSIVE path(id, type, content, depth, path_ids, path_types) AS (
			-- Base case: start with the source node
			SELECT id, type, content, 0 as depth, ',' || id || ',' as path_ids, type as path_types
			FROM knowledge_nodes
			WHERE id = ?
			
			UNION ALL
			
			-- Recursive case: follow edges
			SELECT n.id, n.type, n.content, p.depth + 1, p.path_ids || n.id || ',', p.path_types || ' -> ' || n.type
			FROM knowledge_nodes n
			JOIN knowledge_edges e ON (e.to_node_id = n.id OR e.from_node_id = n.id)
			JOIN path p ON (e.from_node_id = p.id OR e.to_node_id = p.id)
			WHERE p.depth < ?
				AND NOT instr(p.path_ids, ',' || n.id || ',')
				AND (p.id != ? OR n.id = ?)  -- Stop when we reach the target
		)
		SELECT id, type, content, depth, path_ids
		FROM path
		WHERE id = ?
		ORDER BY depth
		LIMIT 1
	`

	var resultID, nodeType, content, pathIDs string
	var depth int

	err := p.db.QueryRowContext(ctx, query, fromID, maxDepth, toID, toID, toID).Scan(
		&resultID, &nodeType, &content, &depth, &pathIDs,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &GraphResult{Nodes: []Node{}, Edges: []Edge{}}, nil
		}
		return nil, fmt.Errorf("failed to find path: %w", err)
	}

	// Extract node IDs from path
	pathIDList := strings.Split(strings.Trim(pathIDs, ","), ",")

	// Get full node details
	nodes, err := p.findNodesByIDs(pathIDList)
	if err != nil {
		return nil, fmt.Errorf("failed to get path nodes: %w", err)
	}

	// Get edges between path nodes
	nodeIDMap := make(map[string]bool)
	for _, id := range pathIDList {
		nodeIDMap[id] = true
	}

	edges, err := p.getEdgesBetweenNodes(nodeIDMap)
	if err != nil {
		return nil, fmt.Errorf("failed to get path edges: %w", err)
	}

	return &GraphResult{Nodes: nodes, Edges: edges}, nil
}

// getNeighborhood gets all nodes within N hops of the given node
func (p *KnowledgePlugin) getNeighborhood(nodeID string, depth int) (*GraphResult, error) {
	return p.findRelated([]string{nodeID}, depth)
}

// getEdgesBetweenNodes retrieves all edges between a set of nodes
func (p *KnowledgePlugin) getEdgesBetweenNodes(nodeIDMap map[string]bool) ([]Edge, error) {
	if len(nodeIDMap) == 0 {
		return []Edge{}, nil
	}

	ctx := context.Background()

	// Build placeholders for IN clause
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

	rows, err := p.db.QueryContext(ctx, query, args...)
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

// findNodesByTypeAndContent searches for nodes by both type and content pattern
func (p *KnowledgePlugin) findNodesByTypeAndContent(nodeType, contentPattern string, limit, offset int) ([]Node, error) {
	ctx := context.Background()

	query := `
		SELECT id, type, content, embedding_id, metadata, created_at, updated_at
		FROM knowledge_nodes
		WHERE type = ? AND content LIKE ?
		ORDER BY created_at DESC
	`

	// Add pagination if limit is specified
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	return p.queryNodes(ctx, query, nodeType, "%"+contentPattern+"%")
}

// getAllNodes retrieves all nodes (use with caution on large graphs)
//
//nolint:unused // reserved for future bulk operations
func (p *KnowledgePlugin) getAllNodes(limit int) ([]Node, error) {
	ctx := context.Background()

	if limit <= 0 {
		limit = 100 // Default limit
	}

	query := `
		SELECT id, type, content, embedding_id, metadata, created_at, updated_at
		FROM knowledge_nodes
		ORDER BY created_at DESC
		LIMIT ?
	`

	return p.queryNodes(ctx, query, limit)
}

// getNodeCountByType returns the count of nodes for each type
func (p *KnowledgePlugin) getNodeCountByType() (map[string]int, error) {
	ctx := context.Background()

	query := `
		SELECT type, COUNT(*) as count
		FROM knowledge_nodes
		GROUP BY type
	`

	rows, err := p.db.QueryContext(ctx, query)
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
func (p *KnowledgePlugin) countNodesByType(nodeType string) (int, error) {
	ctx := context.Background()

	query := `SELECT COUNT(*) FROM knowledge_nodes WHERE type = ?`

	var count int
	err := p.db.QueryRowContext(ctx, query, nodeType).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count nodes by type: %w", err)
	}

	return count, nil
}

// findNodesByContentPaginated searches for nodes by content with pagination
func (p *KnowledgePlugin) findNodesByContentPaginated(query string, exact bool, limit, offset int) ([]Node, error) {
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

	// Add pagination if limit is specified
	if limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			sqlQuery += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	return p.queryNodes(ctx, sqlQuery, args...)
}

// exploreCategory finds a category and returns its full hierarchy
func (p *KnowledgePlugin) exploreCategory(category string) (*GraphResult, error) {
	// Search for category node by content
	nodes, err := p.findNodesByTypeAndContent("Category", category, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to find category: %w", err)
	}

	if len(nodes) == 0 {
		return &GraphResult{Nodes: []Node{}, Edges: []Edge{}}, nil
	}

	// Use the first matching category
	categoryNode := nodes[0]

	// Get full hierarchy using graph traversal (depth 10 to get all children)
	result, err := p.findRelated([]string{categoryNode.ID}, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to traverse category hierarchy: %w", err)
	}

	return result, nil
}

// exploreFact finds a fact and returns its full context
func (p *KnowledgePlugin) exploreFact(fact string) (*GraphResult, error) {
	// Search for fact node by content
	nodes, err := p.findNodesByTypeAndContent("Fact", fact, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to find fact: %w", err)
	}

	if len(nodes) == 0 {
		return &GraphResult{Nodes: []Node{}, Edges: []Edge{}}, nil
	}

	// Use the first matching fact
	factNode := nodes[0]

	// Get neighborhood with bidirectional traversal (depth 2 for context)
	result, err := p.getNeighborhood(factNode.ID, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to get fact context: %w", err)
	}

	return result, nil
}
