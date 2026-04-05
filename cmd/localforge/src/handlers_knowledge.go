package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type BrainNode struct {
	ID                 string         `json:"id"`
	Type               string         `json:"type"`
	Content            string         `json:"content"`
	Title              string         `json:"title,omitempty"`
	Description        string         `json:"description,omitempty"`
	DistillationReason string         `json:"distillation_reason,omitempty"`
	SearchText         string         `json:"search_text,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	CreatedAt          string         `json:"created_at"`
	UpdatedAt          string         `json:"updated_at"`
}

type BrainEdge struct {
	ID           string         `json:"id"`
	FromNodeID   string         `json:"from_node_id"`
	ToNodeID     string         `json:"to_node_id"`
	RelationType string         `json:"relation_type"`
	Weight       float64        `json:"weight"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    string         `json:"created_at"`
}

type BrainType struct {
	ID          string         `json:"id"`
	Category    string         `json:"category"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   string         `json:"created_at"`
}

type GraphResponse struct {
	Nodes []BrainNode `json:"nodes"`
	Edges []BrainEdge `json:"edges"`
	Stats GraphStats  `json:"stats"`
	Types []BrainType `json:"types"`
}

type GraphStats struct {
	TotalNodes int            `json:"total_nodes"`
	TotalEdges int            `json:"total_edges"`
	ByType     map[string]int `json:"by_type"`
}

type NodeDetailResponse struct {
	Node      BrainNode     `json:"node"`
	Neighbors GraphResponse `json:"neighbors"`
}

// brainDBConn returns the shared DB connection or an error if it was never opened.
func (s *Server) brainDBConn() (*sql.DB, error) {
	if s.brainDB != nil {
		return s.brainDB, nil
	}
	return nil, fmt.Errorf("brain database is not available")
}

func (s *Server) handleGetKnowledgeGraph(c *gin.Context) {
	db, err := s.brainDBConn()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "brain database not available"})
		return
	}

	typeFilter := c.Query("type")
	limit := c.DefaultQuery("limit", "1000")

	nodes, err := s.queryNodes(db, typeFilter, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query nodes"})
		return
	}

	edges, err := s.queryEdges(db, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query edges"})
		return
	}

	stats, err := s.getGraphStats(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}

	types, err := s.queryTypes(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query types"})
		return
	}

	response := GraphResponse{
		Nodes: nodes,
		Edges: edges,
		Stats: stats,
		Types: types,
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) handleGetKnowledgeStats(c *gin.Context) {
	db, err := s.brainDBConn()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "brain database not available"})
		return
	}

	stats, err := s.getGraphStats(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (s *Server) handleGetKnowledgeNode(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node id is required"})
		return
	}

	db, err := s.brainDBConn()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "brain database not available"})
		return
	}

	node, err := s.getNodeByID(db, nodeID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get node"})
		return
	}

	neighborNodes, neighborEdges, err := s.getNodeNeighborhood(db, nodeID, 1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get neighbors"})
		return
	}

	response := NodeDetailResponse{
		Node: *node,
		Neighbors: GraphResponse{
			Nodes: neighborNodes,
			Edges: neighborEdges,
			Stats: GraphStats{
				TotalNodes: len(neighborNodes),
				TotalEdges: len(neighborEdges),
			},
		},
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) queryNodes(db *sql.DB, typeFilter, limit string) ([]BrainNode, error) {
	query := `
		SELECT id, type, content, COALESCE(metadata, '{}'),
		       title, description, distillation_reason, COALESCE(search_text, ''),
		       created_at, updated_at
		FROM brain_nodes
	`

	args := []interface{}{}
	if typeFilter != "" {
		query += " WHERE type = ?"
		args = append(args, typeFilter)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var nodes []BrainNode
	for rows.Next() {
		var node BrainNode
		var metadataJSON string
		var title, description, distillationReason, searchText sql.NullString

		err := rows.Scan(
			&node.ID, &node.Type, &node.Content,
			&metadataJSON, &title, &description, &distillationReason, &searchText,
			&node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if metadataJSON != "" && metadataJSON != "{}" {
			_ = json.Unmarshal([]byte(metadataJSON), &node.Metadata)
		}
		if title.Valid {
			node.Title = title.String
		}
		if description.Valid {
			node.Description = description.String
		}
		if distillationReason.Valid {
			node.DistillationReason = distillationReason.String
		}
		if searchText.Valid {
			node.SearchText = searchText.String
		}

		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

func (s *Server) queryEdges(db *sql.DB, nodeIDFilter string) ([]BrainEdge, error) {
	query := `
		SELECT id, from_node_id, to_node_id, relation_type, weight, 
		       COALESCE(metadata, '{}'), created_at
		FROM brain_edges
	`

	args := []interface{}{}
	if nodeIDFilter != "" {
		query += " WHERE from_node_id = ? OR to_node_id = ?"
		args = append(args, nodeIDFilter, nodeIDFilter)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var edges []BrainEdge
	for rows.Next() {
		var edge BrainEdge
		var metadataJSON string

		err := rows.Scan(
			&edge.ID, &edge.FromNodeID, &edge.ToNodeID, &edge.RelationType,
			&edge.Weight, &metadataJSON, &edge.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if metadataJSON != "" && metadataJSON != "{}" {
			_ = json.Unmarshal([]byte(metadataJSON), &edge.Metadata)
		}

		edges = append(edges, edge)
	}

	return edges, rows.Err()
}

func (s *Server) queryTypes(db *sql.DB) ([]BrainType, error) {
	query := `
		SELECT id, category, name, description, COALESCE(metadata, '{}'), created_at
		FROM brain_types
		ORDER BY category, name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var types []BrainType
	for rows.Next() {
		var t BrainType
		var metadataJSON string

		err := rows.Scan(
			&t.ID, &t.Category, &t.Name, &t.Description,
			&metadataJSON, &t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if metadataJSON != "" && metadataJSON != "{}" {
			_ = json.Unmarshal([]byte(metadataJSON), &t.Metadata)
		}

		types = append(types, t)
	}

	return types, rows.Err()
}

func (s *Server) getGraphStats(db *sql.DB) (GraphStats, error) {
	var stats GraphStats

	err := db.QueryRow("SELECT COUNT(*) FROM brain_nodes").Scan(&stats.TotalNodes)
	if err != nil {
		return stats, err
	}

	err = db.QueryRow("SELECT COUNT(*) FROM brain_edges").Scan(&stats.TotalEdges)
	if err != nil {
		return stats, err
	}

	rows, err := db.Query(`
		SELECT type, COUNT(*) as count
		FROM brain_nodes
		GROUP BY type
	`)
	if err != nil {
		return stats, err
	}
	defer func() { _ = rows.Close() }()

	stats.ByType = make(map[string]int)
	for rows.Next() {
		var nodeType string
		var count int
		if err := rows.Scan(&nodeType, &count); err != nil {
			return stats, err
		}
		stats.ByType[nodeType] = count
	}

	return stats, rows.Err()
}

func (s *Server) getNodeByID(db *sql.DB, nodeID string) (*BrainNode, error) {
	query := `
		SELECT id, type, content, COALESCE(metadata, '{}'),
		       title, description, distillation_reason, COALESCE(search_text, ''),
		       created_at, updated_at
		FROM brain_nodes
		WHERE id = ?
	`

	var node BrainNode
	var metadataJSON string
	var title, description, distillationReason, searchText sql.NullString

	err := db.QueryRow(query, nodeID).Scan(
		&node.ID, &node.Type, &node.Content,
		&metadataJSON, &title, &description, &distillationReason, &searchText,
		&node.CreatedAt, &node.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if metadataJSON != "" && metadataJSON != "{}" {
		_ = json.Unmarshal([]byte(metadataJSON), &node.Metadata)
	}
	if title.Valid {
		node.Title = title.String
	}
	if description.Valid {
		node.Description = description.String
	}
	if distillationReason.Valid {
		node.DistillationReason = distillationReason.String
	}
	if searchText.Valid {
		node.SearchText = searchText.String
	}

	return &node, nil
}

func (s *Server) getNodeNeighborhood(db *sql.DB, nodeID string, depth int) ([]BrainNode, []BrainEdge, error) {
	query := `
		WITH RECURSIVE related(id, depth, path) AS (
			SELECT id, 0, ',' || id || ','
			FROM brain_nodes
			WHERE id = ?
			UNION ALL
			SELECT n.id, r.depth + 1, r.path || n.id || ','
			FROM brain_nodes n
			JOIN brain_edges e ON (e.to_node_id = n.id OR e.from_node_id = n.id)
			JOIN related r ON (e.from_node_id = r.id OR e.to_node_id = r.id)
			WHERE r.depth < ?
				AND NOT instr(r.path, ',' || n.id || ',')
		)
		SELECT n.id, n.type, n.content, COALESCE(n.metadata, '{}'),
		       n.title, n.description, n.distillation_reason, COALESCE(n.search_text, ''),
		       MIN(r.depth), n.created_at, n.updated_at
		FROM related r
		JOIN brain_nodes n ON n.id = r.id
		GROUP BY n.id
		ORDER BY MIN(r.depth), n.type, n.content
	`

	rows, err := db.Query(query, nodeID, depth)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()

	var nodes []BrainNode
	nodeIDs := make(map[string]bool)

	for rows.Next() {
		var node BrainNode
		var metadataJSON string
		var title, description, distillationReason, searchText sql.NullString
		var nodeDepth int

		err := rows.Scan(
			&node.ID, &node.Type, &node.Content,
			&metadataJSON, &title, &description, &distillationReason, &searchText,
			&nodeDepth, &node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, nil, err
		}

		if metadataJSON != "" && metadataJSON != "{}" {
			_ = json.Unmarshal([]byte(metadataJSON), &node.Metadata)
		}
		if title.Valid {
			node.Title = title.String
		}
		if description.Valid {
			node.Description = description.String
		}
		if distillationReason.Valid {
			node.DistillationReason = distillationReason.String
		}
		if searchText.Valid {
			node.SearchText = searchText.String
		}

		nodes = append(nodes, node)
		nodeIDs[node.ID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	edges, err := s.queryEdgesBetweenNodes(db, nodeIDs)
	if err != nil {
		return nil, nil, err
	}

	return nodes, edges, nil
}

func (s *Server) queryEdgesBetweenNodes(db *sql.DB, nodeIDs map[string]bool) ([]BrainEdge, error) {
	if len(nodeIDs) == 0 {
		return []BrainEdge{}, nil
	}

	placeholders := ""
	args := make([]interface{}, 0, len(nodeIDs)*2)

	i := 0
	for id := range nodeIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, id)
		i++
	}

	args = append(args, args...)

	query := `
		SELECT id, from_node_id, to_node_id, relation_type, weight, 
		       COALESCE(metadata, '{}'), created_at
		FROM brain_edges
		WHERE from_node_id IN (` + placeholders + `) AND to_node_id IN (` + placeholders + `)
	`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var edges []BrainEdge
	for rows.Next() {
		var edge BrainEdge
		var metadataJSON string

		err := rows.Scan(
			&edge.ID, &edge.FromNodeID, &edge.ToNodeID, &edge.RelationType,
			&edge.Weight, &metadataJSON, &edge.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if metadataJSON != "" && metadataJSON != "{}" {
			_ = json.Unmarshal([]byte(metadataJSON), &edge.Metadata)
		}

		edges = append(edges, edge)
	}

	return edges, rows.Err()
}
