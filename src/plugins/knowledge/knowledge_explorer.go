package knowledge

import (
	"fmt"
)

// KnowledgeExplorer provides domain-specific operations for exploring the knowledge graph
// including finding categories, subcategories, facts and joining them with related semantic units.
type KnowledgeExplorer struct {
	querier *GraphQuerier
}

// NewKnowledgeExplorer creates a new KnowledgeExplorer.
func NewKnowledgeExplorer(querier *GraphQuerier) *KnowledgeExplorer {
	return &KnowledgeExplorer{querier: querier}
}

// ExploreCategory finds a category node and returns a structured exploration result.
// Sub-categories, facts, documents, and relevant nodes are returned as light nodes
// so the agent can browse titles and decide what to explore fully.
func (e *KnowledgeExplorer) ExploreCategory(category string) (*CategoryExploreResult, error) {
	nodes, err := e.querier.findNodesByTypeAndContent("Category", category, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to find category: %w", err)
	}
	if len(nodes) == 0 {
		return &CategoryExploreResult{}, nil
	}
	cat := nodes[0]

	subCats, err := e.querier.getDirectLightNodes(cat.ID, "has_subcategory")
	if err != nil {
		return nil, err
	}
	facts, err := e.querier.getDirectLightNodes(cat.ID, "has_fact")
	if err != nil {
		return nil, err
	}
	docs, err := e.querier.getDirectLightNodes(cat.ID, "has_document")
	if err != nil {
		return nil, err
	}
	relevant, err := e.querier.getDirectLightNodes(cat.ID, "is_relevant_to")
	if err != nil {
		return nil, err
	}

	return &CategoryExploreResult{
		Category:      cat,
		SubCategories: subCats,
		Facts:         facts,
		Documents:     docs,
		RelevantTo:    relevant,
	}, nil
}

// ExploreSubcategory finds a subcategory node and returns a structured exploration result.
func (e *KnowledgeExplorer) ExploreSubcategory(subcategory string) (*SubcategoryExploreResult, error) {
	nodes, err := e.querier.findNodesByTypeAndContent("Subcategory", subcategory, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to find subcategory: %w", err)
	}
	if len(nodes) == 0 {
		return &SubcategoryExploreResult{}, nil
	}
	subCat := nodes[0]

	subCats, err := e.querier.getDirectLightNodes(subCat.ID, "has_subcategory")
	if err != nil {
		return nil, err
	}
	facts, err := e.querier.getDirectLightNodes(subCat.ID, "has_fact")
	if err != nil {
		return nil, err
	}
	docs, err := e.querier.getDirectLightNodes(subCat.ID, "has_document")
	if err != nil {
		return nil, err
	}
	relevant, err := e.querier.getDirectLightNodes(subCat.ID, "is_relevant_to")
	if err != nil {
		return nil, err
	}

	return &SubcategoryExploreResult{
		Subcategory:   subCat,
		SubCategories: subCats,
		Facts:         facts,
		Documents:     docs,
		RelevantTo:    relevant,
	}, nil
}

// ExploreFact finds a fact node and returns its full content plus light-node neighbours.
func (e *KnowledgeExplorer) ExploreFact(fact string) (*FactExploreResult, error) {
	nodes, err := e.querier.findNodesByTypeAndContent("Fact", fact, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to find fact: %w", err)
	}
	if len(nodes) == 0 {
		return &FactExploreResult{}, nil
	}
	factNode := nodes[0]

	parents, err := e.querier.getDirectLightParents(factNode.ID, "has_fact")
	if err != nil {
		return nil, err
	}
	docs, err := e.querier.getDirectLightNodes(factNode.ID, "has_document")
	if err != nil {
		return nil, err
	}
	relevant, err := e.querier.getDirectLightNodes(factNode.ID, "is_relevant_to")
	if err != nil {
		return nil, err
	}

	return &FactExploreResult{
		Fact:             factNode,
		ParentCategories: parents,
		Documents:        docs,
		RelevantTo:       relevant,
	}, nil
}
