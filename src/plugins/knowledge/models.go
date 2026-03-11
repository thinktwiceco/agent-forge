package knowledge

// GraphResult represents the result of a graph traversal query
type GraphResult struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// LightNodeWithEdge is a minimal neighbor representation that includes the connecting edge type.
type LightNodeWithEdge struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	EdgeType string `json:"edge_type"`
}

// NodeNeighborsResult is returned by out_nodes and in_nodes tools.
type NodeNeighborsResult struct {
	Node      LightNode           `json:"node"`
	Neighbors []LightNodeWithEdge `json:"neighbors"`
	Count     int                 `json:"count"`
}
