package knowledge

// GraphResult represents the result of a graph traversal query
type GraphResult struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// CategoryExploreResult is the structured result of exploring a category.
type CategoryExploreResult struct {
	Category      Node        `json:"category"`
	SubCategories []LightNode `json:"sub_categories"`
	Facts         []LightNode `json:"facts"`
	Documents     []LightNode `json:"documents"`
	RelevantTo    []LightNode `json:"relevant_to"`
}

// SubcategoryExploreResult is the structured result of exploring a subcategory.
type SubcategoryExploreResult struct {
	Subcategory   Node        `json:"subcategory"`
	SubCategories []LightNode `json:"sub_categories"`
	Facts         []LightNode `json:"facts"`
	Documents     []LightNode `json:"documents"`
	RelevantTo    []LightNode `json:"relevant_to"`
}

// FactExploreResult is the structured result of exploring a fact.
type FactExploreResult struct {
	Fact             Node        `json:"fact"`
	ParentCategories []LightNode `json:"parent_categories"`
	Documents        []LightNode `json:"documents"`
	RelevantTo       []LightNode `json:"relevant_to"`
}
