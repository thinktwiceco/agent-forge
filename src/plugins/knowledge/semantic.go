package knowledge

import "fmt"

// findScored searches for nodes matching query using text search.
func (p *KnowledgePlugin) findScored(query string, limit int) ([]ScoredNode, error) {
	if limit <= 0 {
		limit = 10
	}

	nodes, err := p.querier.findNodesByContentPaginated(query, false, limit, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to search nodes: %w", err)
	}

	scoredNodes := make([]ScoredNode, 0, len(nodes))
	for _, node := range nodes {
		scoredNodes = append(scoredNodes, ScoredNode{
			Node:  node,
			Score: 1.0,
		})
	}

	return scoredNodes, nil
}
