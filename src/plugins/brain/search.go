package brain

import (
	"context"
	"fmt"
)

// findScored searches brain_nodes using FTS5 full-text search with BM25 ranking.
// Falls back to LIKE search when the query contains no FTS5-indexable tokens.
func (p *BrainPlugin) findScored(query string, limit int) ([]ScoredNode, error) {
	if limit <= 0 {
		limit = 10
	}

	ctx := context.Background()

	// FTS5 path: ranked by BM25 (lower bm25() value = better match in SQLite).
	// bm25() is negated so higher Score means better match.
	ftsQuery := fmt.Sprintf(`
		SELECT %s,
		       -bm25(brain_fts) AS score
		FROM brain_fts
		JOIN brain_nodes n ON n.id = brain_fts.node_id
		WHERE brain_fts MATCH ?
		  AND n.id != ?
		ORDER BY score DESC
		LIMIT ?
	`, sqlBrainNodeColumnsN)
	rows, err := p.db.QueryContext(ctx, ftsQuery, query, omniaNuncNodeID, limit)
	if err != nil {
		// FTS5 MATCH fails on some query strings (e.g. bare punctuation).
		// Fall back to LIKE.
		return p.findScoredLike(query, limit)
	}
	defer func() { _ = rows.Close() }()

	var results []ScoredNode
	for rows.Next() {
		node, score, err := scanNodeWithScore(rows)
		if err != nil {
			return nil, fmt.Errorf("brain find scan: %w", err)
		}
		results = append(results, ScoredNode{Node: node, Score: score})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("brain find rows: %w", err)
	}

	// If FTS returned nothing, fall back to LIKE so short/partial queries still work.
	if len(results) == 0 {
		return p.findScoredLike(query, limit)
	}
	return results, nil
}

// findScoredLike is the LIKE fallback. All results receive score 0.5 to signal
// lower confidence than ranked FTS results.
func (p *BrainPlugin) findScoredLike(query string, limit int) ([]ScoredNode, error) {
	nodes, err := p.querier.findNodesByContentPaginated(query, false, limit, 0)
	if err != nil {
		return nil, err
	}
	results := make([]ScoredNode, 0, len(nodes))
	for _, n := range nodes {
		results = append(results, ScoredNode{Node: n, Score: 0.5})
	}
	return results, nil
}
