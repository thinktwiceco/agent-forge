package knowledge

import (
	"context"
	"fmt"
)

// SemanticSearchResult combines semantic search results with graph context
type SemanticSearchResult struct {
	Nodes           []Node  `json:"nodes"`
	Edges           []Edge  `json:"edges"`
	SemanticMatches []Match `json:"semantic_matches"`
}

// Match represents a semantic search match
type Match struct {
	NodeID string  `json:"node_id"`
	Score  float32 `json:"score"`
	Text   string  `json:"text"`
}

// saveWithEmbedding saves a node and its embedding to both graph DB and vector DB
func (p *KnowledgePlugin) saveWithEmbedding(nodeType, content string, metadata map[string]any) (string, error) {
	// Check if we have vector DB and embedding generator
	if p.vectorDB == nil || p.embeddingGen == nil {
		// Fallback: save without embedding
		return p.saveNode(nodeType, content, "", metadata)
	}

	// Generate embedding for the content
	embedding, modelName, err := p.embeddingGen.GenerateEmbedding(content)
	if err != nil {
		return "", fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Store in vector DB
	vectorMetadata := make(map[string]any)
	for k, v := range metadata {
		vectorMetadata[k] = v
	}
	vectorMetadata["node_type"] = nodeType

	embeddingID, err := p.vectorDB.Index(embedding, content, vectorMetadata, modelName)
	if err != nil {
		return "", fmt.Errorf("failed to index in vector DB: %w", err)
	}

	// Save node with embedding ID
	nodeID, err := p.saveNode(nodeType, content, embeddingID, metadata)
	if err != nil {
		// Attempt to clean up vector DB entry
		_ = p.vectorDB.Delete(embeddingID)
		return "", fmt.Errorf("failed to save node: %w", err)
	}

	return nodeID, nil
}

// semanticSearch performs semantic search and returns matching nodes
func (p *KnowledgePlugin) semanticSearch(query string, topK int, filters map[string]any) ([]Match, error) {
	if p.vectorDB == nil || p.embeddingGen == nil {
		return nil, fmt.Errorf("semantic search not available: vector DB or embedding generator not configured")
	}

	// Generate query embedding
	queryEmbedding, _, err := p.embeddingGen.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search in vector DB
	results, err := p.vectorDB.Search(queryEmbedding, topK, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector DB: %w", err)
	}

	// Convert results to matches
	matches := make([]Match, 0, len(results))
	for _, result := range results {
		// Try to find the corresponding node by embedding_id
		nodes, err := p.findNodesByEmbeddingID(result.DocumentID)
		if err != nil || len(nodes) == 0 {
			continue
		}

		matches = append(matches, Match{
			NodeID: nodes[0].ID,
			Score:  result.Score,
			Text:   result.Text,
		})
	}

	return matches, nil
}

// findWithSemanticAndGraph performs hybrid search: semantic + graph traversal
//
//nolint:unused // reserved for future hybrid search
func (p *KnowledgePlugin) findWithSemanticAndGraph(query string, topK int, traverseDepth int, filters map[string]any, limit, offset int) (*SemanticSearchResult, error) {
	// Perform semantic search
	matches, err := p.semanticSearch(query, topK, filters)
	if err != nil {
		// Fallback to text search if semantic search fails
		return p.findWithTextSearch(query, traverseDepth, filters, limit, offset)
	}

	if len(matches) == 0 {
		return &SemanticSearchResult{
			Nodes:           []Node{},
			Edges:           []Edge{},
			SemanticMatches: []Match{},
		}, nil
	}

	// Extract node IDs from matches
	nodeIDs := make([]string, len(matches))
	for i, match := range matches {
		nodeIDs[i] = match.NodeID
	}

	// Perform graph traversal from matched nodes
	graphResult, err := p.querier.findRelated(nodeIDs, traverseDepth)
	if err != nil {
		return nil, fmt.Errorf("failed to traverse graph: %w", err)
	}

	return &SemanticSearchResult{
		Nodes:           graphResult.Nodes,
		Edges:           graphResult.Edges,
		SemanticMatches: matches,
	}, nil
}

// findWithTextSearch is a fallback when semantic search is not available
//
//nolint:unused // reserved for fallback search
func (p *KnowledgePlugin) findWithTextSearch(query string, traverseDepth int, filters map[string]any, limit, offset int) (*SemanticSearchResult, error) {
	// Perform text-based search with pagination
	var nodes []Node
	var err error

	// Default limit if not specified
	if limit <= 0 {
		limit = 10
	}

	if nodeType, ok := filters["type"].(string); ok {
		if query == "" {
			// If no query, get all nodes of this type
			nodes, err = p.findNodesByType(nodeType, limit, offset)
		} else {
			// Search within specific type
			nodes, err = p.querier.findNodesByTypeAndContent(nodeType, query, limit, offset)
		}
	} else {
		if query == "" {
			// No query and no type filter - get recent nodes
			nodes, err = p.querier.getAllNodes(limit)
		} else {
			// General text search
			nodes, err = p.querier.findNodesByContentPaginated(query, false, limit, offset)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to search by content: %w", err)
	}

	if len(nodes) == 0 {
		return &SemanticSearchResult{
			Nodes:           []Node{},
			Edges:           []Edge{},
			SemanticMatches: []Match{},
		}, nil
	}

	// Extract node IDs
	nodeIDs := make([]string, len(nodes))
	for i, node := range nodes {
		nodeIDs[i] = node.ID
	}

	// Perform graph traversal
	graphResult, err := p.querier.findRelated(nodeIDs, traverseDepth)
	if err != nil {
		return nil, fmt.Errorf("failed to traverse graph: %w", err)
	}

	// Create matches from text search results
	matches := make([]Match, len(nodes))
	for i, node := range nodes {
		matches[i] = Match{
			NodeID: node.ID,
			Score:  1.0, // No real score for text search
			Text:   node.Content,
		}
	}

	return &SemanticSearchResult{
		Nodes:           graphResult.Nodes,
		Edges:           graphResult.Edges,
		SemanticMatches: matches,
	}, nil
}

// findNodesByEmbeddingID finds nodes by their embedding ID
func (p *KnowledgePlugin) findNodesByEmbeddingID(embeddingID string) ([]Node, error) {
	query := `
		SELECT id, type, content, embedding_id, metadata, created_at, updated_at
		FROM knowledge_nodes
		WHERE embedding_id = ?
	`

	return p.queryNodes(context.TODO(), query, embeddingID)
}

// updateEmbedding updates the embedding for a node
//
//nolint:unused // reserved for embedding refresh
func (p *KnowledgePlugin) updateEmbedding(nodeID string) error {
	if p.vectorDB == nil || p.embeddingGen == nil {
		return fmt.Errorf("cannot update embedding: vector DB or embedding generator not configured")
	}

	// Get the node
	node, err := p.getNode(nodeID)
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	// Delete old embedding if it exists
	if node.EmbeddingID != "" {
		_ = p.vectorDB.Delete(node.EmbeddingID)
	}

	// Generate new embedding
	embedding, modelName, err := p.embeddingGen.GenerateEmbedding(node.Content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Store in vector DB
	vectorMetadata := make(map[string]any)
	if node.Metadata != nil {
		for k, v := range node.Metadata {
			vectorMetadata[k] = v
		}
	}
	vectorMetadata["node_type"] = node.Type

	embeddingID, err := p.vectorDB.Index(embedding, node.Content, vectorMetadata, modelName)
	if err != nil {
		return fmt.Errorf("failed to index in vector DB: %w", err)
	}

	// Update node with new embedding ID
	query := `UPDATE knowledge_nodes SET embedding_id = ? WHERE id = ?`
	_, err = p.db.Exec(query, embeddingID, nodeID)
	if err != nil {
		// Clean up vector DB entry
		_ = p.vectorDB.Delete(embeddingID)
		return fmt.Errorf("failed to update node embedding ID: %w", err)
	}

	return nil
}

// findScored searches for nodes matching query and returns scored results
func (p *KnowledgePlugin) findScored(query string, limit int) ([]ScoredNode, error) {
	if limit <= 0 {
		limit = 10
	}

	// Try semantic search first if available
	if p.vectorDB != nil && p.embeddingGen != nil {
		matches, err := p.semanticSearch(query, limit, nil)
		if err == nil && len(matches) > 0 {
			// Convert matches to ScoredNode
			scoredNodes := make([]ScoredNode, 0, len(matches))
			for _, match := range matches {
				node, err := p.getNode(match.NodeID)
				if err != nil {
					continue
				}
				scoredNodes = append(scoredNodes, ScoredNode{
					Node:  *node,
					Score: match.Score,
				})
			}
			return scoredNodes, nil
		}
	}

	// Fallback to text search
	nodes, err := p.querier.findNodesByContentPaginated(query, false, limit, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to search nodes: %w", err)
	}

	// Convert to ScoredNode with fixed score
	scoredNodes := make([]ScoredNode, 0, len(nodes))
	for _, node := range nodes {
		scoredNodes = append(scoredNodes, ScoredNode{
			Node:  node,
			Score: 1.0, // Fixed score for text search
		})
	}

	return scoredNodes, nil
}
