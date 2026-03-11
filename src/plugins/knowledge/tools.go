package knowledge

import (
	"encoding/json"
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// Tools implements core.ToolProvider
func (p *KnowledgePlugin) Tools() []llms.Tool {
	return []llms.Tool{
		p.newOutNodesTool(),
		p.newInNodesTool(),
		p.newGetNodeContentTool(),
		p.newFindTool(),
		p.newAddNodeTool(),
		p.newLinkRelevantTool(),
		p.newDeleteNodeTool(),
	}
}

// newOutNodesTool lists all outgoing neighbors of a node.
// An empty node defaults to the graph root, returning all top-level categories.
func (p *KnowledgePlugin) newOutNodesTool() llms.Tool {
	return &core.Tool{
		Name:        "out_nodes",
		Description: "List outgoing neighbors of a node (edges FROM the node). Pass node=\"\" to start from the graph root and discover top-level categories. Each neighbor includes id, type, title, and edge_type.",
		Parameters: []core.Parameter{
			{
				Name:        "node",
				Type:        "string",
				Description: "Node id, name, or title. Empty string defaults to the graph root.",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			node, _ := args["node"].(string)
			result, err := p.OutNodes(node)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to get out nodes: %v", err))
			}
			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	}
}

// newInNodesTool lists all incoming neighbors of a node (nodes that point TO it).
func (p *KnowledgePlugin) newInNodesTool() llms.Tool {
	return &core.Tool{
		Name:        "in_nodes",
		Description: "List incoming neighbors of a node (edges TO the node). Returns parent nodes and cross-references that point to this node. Each neighbor includes id, type, title, and edge_type.",
		Parameters: []core.Parameter{
			{
				Name:        "node",
				Type:        "string",
				Description: "Node id, name, or title to look up incoming edges for.",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			node, ok := args["node"].(string)
			if !ok || node == "" {
				return core.NewErrorResponse("node parameter is required")
			}
			result, err := p.InNodes(node)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to get in nodes: %v", err))
			}
			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	}
}

// newGetNodeContentTool returns the full content of a node.
func (p *KnowledgePlugin) newGetNodeContentTool() llms.Tool {
	return &core.Tool{
		Name:        "get_node_content",
		Description: "Retrieve the full content and metadata of a node (Fact, Category, Subcategory, or Document).",
		Parameters: []core.Parameter{
			{
				Name:        "node",
				Type:        "string",
				Description: "Node id, name, or title.",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			node, ok := args["node"].(string)
			if !ok || node == "" {
				return core.NewErrorResponse("node parameter is required")
			}
			result, err := p.GetNodeContent(node)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to get node content: %v", err))
			}
			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	}
}

// newFindTool creates the find tool
func (p *KnowledgePlugin) newFindTool() llms.Tool {
	return &core.Tool{
		Name:        "find",
		Description: "Search for nodes matching a query using text search",
		Parameters: []core.Parameter{
			{
				Name:        "query",
				Type:        "string",
				Description: "Search query to match against node content",
				Required:    true,
			},
			{
				Name:        "limit",
				Type:        "number",
				Description: "Maximum number of results to return (default: 10)",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			query, ok := args["query"].(string)
			if !ok || query == "" {
				return core.NewErrorResponse("query parameter is required")
			}

			limit := 10
			if l, ok := args["limit"].(float64); ok {
				limit = int(l)
			}

			result, err := p.Find(query, limit)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to search: %v", err))
			}

			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	}
}

// newAddNodeTool creates the add_node tool — generic node creation.
func (p *KnowledgePlugin) newAddNodeTool() llms.Tool {
	return &core.Tool{
		Name: "add_node",
		Description: "Create a node and attach it to a parent via an edge. " +
			"parent=\"\" targets the graph root. " +
			"name is the short label (3-8 words) surfaced during traversal. " +
			"content is the full body; defaults to name when omitted. " +
			"Common edges: has_category, has_subcategory, has_fact, has_document.",
		Parameters: []core.Parameter{
			{
				Name:        "parent",
				Type:        "string",
				Description: "Parent node id, name, or title. Empty string = graph root.",
				Required:    true,
			},
			{
				Name:        "edge",
				Type:        "string",
				Description: "Edge type connecting parent to the new node (e.g. has_category, has_subcategory, has_fact, has_document).",
				Required:    true,
			},
			{
				Name:        "type",
				Type:        "string",
				Description: "Node type: Category, Subcategory, Fact, or Document.",
				Required:    true,
			},
			{
				Name:        "name",
				Type:        "string",
				Description: "Short label for the node (3-8 words). Stored as metadata.title for traversal.",
				Required:    true,
			},
			{
				Name:        "content",
				Type:        "string",
				Description: "Full content body. Defaults to name when empty.",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			parent, _ := args["parent"].(string)
			edge, ok := args["edge"].(string)
			if !ok || edge == "" {
				return core.NewErrorResponse("edge parameter is required")
			}
			nodeType, ok := args["type"].(string)
			if !ok || nodeType == "" {
				return core.NewErrorResponse("type parameter is required")
			}
			name, ok := args["name"].(string)
			if !ok || name == "" {
				return core.NewErrorResponse("name parameter is required")
			}
			content, _ := args["content"].(string)

			nodeID, err := p.AddNode(parent, edge, nodeType, name, content)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to add node: %v", err))
			}
			result := map[string]any{
				"node_id": nodeID,
				"parent":  parent,
				"edge":    edge,
				"type":    nodeType,
				"name":    name,
			}
			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	}
}

// newLinkRelevantTool creates the link_relevant tool
func (p *KnowledgePlugin) newLinkRelevantTool() llms.Tool {
	return &core.Tool{
		Name:        "link_relevant",
		Description: "Construct horizontal relationship (is_relevant_to edge) between any two nodes. Instruction: Consider categories and subcategories relationships when organizing multidimensional data.",
		Parameters: []core.Parameter{
			{
				Name:        "node_a",
				Type:        "string",
				Description: "First node name or ID",
				Required:    true,
			},
			{
				Name:        "node_b",
				Type:        "string",
				Description: "Second node name or ID to link to node_a",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			nodeA, ok := args["node_a"].(string)
			if !ok || nodeA == "" {
				return core.NewErrorResponse("node_a parameter is required")
			}
			nodeB, ok := args["node_b"].(string)
			if !ok || nodeB == "" {
				return core.NewErrorResponse("node_b parameter is required")
			}
			edgeAB, edgeBA, err := p.LinkRelevant(nodeA, nodeB)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to link nodes: %v", err))
			}
			result := map[string]any{
				"edge_ab": edgeAB,
				"edge_ba": edgeBA,
				"node_a":  nodeA,
				"node_b":  nodeB,
			}
			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	}
}

// newDeleteNodeTool creates the delete_node tool.
func (p *KnowledgePlugin) newDeleteNodeTool() llms.Tool {
	return &core.Tool{
		Name:        "delete_node",
		Description: "Delete a node and all its descendants (cascade). Accepts node id, name, or title.",
		Parameters: []core.Parameter{
			{
				Name:        "node",
				Type:        "string",
				Description: "Node id, name, or title to delete.",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			node, ok := args["node"].(string)
			if !ok || node == "" {
				return core.NewErrorResponse("node parameter is required")
			}

			deletedCount, err := p.DeleteNode(node)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to delete node: %v", err))
			}

			result := map[string]any{
				"deleted_count": deletedCount,
				"node":          node,
			}
			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	}
}
