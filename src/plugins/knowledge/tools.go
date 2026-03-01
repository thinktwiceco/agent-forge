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
		p.newExploreCategoryTool(),
		p.newExploreSubcategoryTool(),
		p.newExploreFactTool(),
		p.newFindTool(),
		p.newRememberTool(),
		p.newAttachDocumentTool(),
		p.newLinkRelevantTool(),
		p.newAddCategoryTool(),
		p.newAddSubcategoryTool(),
		p.newGetCategoriesTool(),
		p.newGetCategoryFactsTool(),
		p.newForgetTool(),
	}
}

// newExploreCategoryTool creates the explore_category tool
func (p *KnowledgePlugin) newExploreCategoryTool() llms.Tool {
	return &core.Tool{
		Name:        "explore_category",
		Description: "<ACTION>Retrieve topology of Category.</ACTION> <RETURNS>Children (Subcategories, Facts [title-only], Documents), and relevant Nodes.</RETURNS> <INSTRUCTION>You MUST use explore_fact afterward to expand text content of Facts.</INSTRUCTION>",
		Parameters: []core.Parameter{
			{
				Name:        "category",
				Type:        "string",
				Description: "The category name to explore",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			category, ok := args["category"].(string)
			if !ok || category == "" {
				return core.NewErrorResponse("category parameter is required")
			}
			result, err := p.ExploreCategory(category)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to explore category: %v", err))
			}
			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	}
}

// newExploreSubcategoryTool creates the explore_subcategory tool
func (p *KnowledgePlugin) newExploreSubcategoryTool() llms.Tool {
	return &core.Tool{
		Name:        "explore_subcategory",
		Description: "<ACTION>Retrieve topology of Subcategory.</ACTION> <RETURNS>Children (Subcategories, Facts [title-only], Documents), and relevant Nodes.</RETURNS> <INSTRUCTION>You MUST use explore_fact afterward to expand text content of Facts.</INSTRUCTION>",
		Parameters: []core.Parameter{
			{
				Name:        "subcategory",
				Type:        "string",
				Description: "The subcategory name to explore",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			subcategory, ok := args["subcategory"].(string)
			if !ok || subcategory == "" {
				return core.NewErrorResponse("subcategory parameter is required")
			}
			result, err := p.ExploreSubcategory(subcategory)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to explore subcategory: %v", err))
			}
			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	}
}

// newExploreFactTool creates the explore_fact tool
func (p *KnowledgePlugin) newExploreFactTool() llms.Tool {
	return &core.Tool{
		Name:        "explore_fact",
		Description: "<ACTION>Expand Fact content.</ACTION> <RETURNS>Full content body + parent Categories, attached Documents, and relevant Nodes.</RETURNS>",
		Parameters: []core.Parameter{
			{
				Name:        "fact",
				Type:        "string",
				Description: "The fact title or content snippet to explore (use the id or title from explore_category)",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			fact, ok := args["fact"].(string)
			if !ok || fact == "" {
				return core.NewErrorResponse("fact parameter is required")
			}
			result, err := p.ExploreFact(fact)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to explore fact: %v", err))
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
		Description: "Search for nodes matching a query using semantic search (if available) or text search",
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

// newRememberTool creates the remember tool
func (p *KnowledgePlugin) newRememberTool() llms.Tool {
	return &core.Tool{
		Name:        "remember",
		Description: "<ACTION>Ingest knowledge Fact.</ACTION> <INSTRUCTION>Evaluate existing Categories/Subcategories before insertion. You MUST construct a short 'title' (3-8 words) for optimal traversal.</INSTRUCTION>",
		Parameters: []core.Parameter{
			{
				Name:        "category",
				Type:        "string",
				Description: "The category to store the fact under",
				Required:    true,
			},
			{
				Name:        "title",
				Type:        "string",
				Description: "Short label for the fact (3-8 words), e.g. \"Prefers dark mode\"",
				Required:    false,
			},
			{
				Name:        "fact",
				Type:        "string",
				Description: "The full fact content to remember",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			category, ok := args["category"].(string)
			if !ok || category == "" {
				return core.NewErrorResponse("category parameter is required")
			}
			fact, ok := args["fact"].(string)
			if !ok || fact == "" {
				return core.NewErrorResponse("fact parameter is required")
			}
			title, _ := args["title"].(string)

			var factID string
			var err error
			if title != "" {
				factID, err = p.RememberWithTitle(category, title, fact)
			} else {
				factID, err = p.Remember(category, fact)
			}
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to remember fact: %v", err))
			}
			result := map[string]any{
				"fact_id":  factID,
				"category": category,
				"title":    title,
				"fact":     fact,
			}
			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	}
}

// newAttachDocumentTool creates the attach_document tool
func (p *KnowledgePlugin) newAttachDocumentTool() llms.Tool {
	return &core.Tool{
		Name:        "attach_document",
		Description: "Attach a filesystem document path to a Category or Fact node via a has_document edge",
		Parameters: []core.Parameter{
			{
				Name:        "parent",
				Type:        "string",
				Description: "Node name (or ID) to attach the document to (Category or Fact)",
				Required:    true,
			},
			{
				Name:        "file_path",
				Type:        "string",
				Description: "Absolute or relative filesystem path to the document",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			parent, ok := args["parent"].(string)
			if !ok || parent == "" {
				return core.NewErrorResponse("parent parameter is required")
			}
			filePath, ok := args["file_path"].(string)
			if !ok || filePath == "" {
				return core.NewErrorResponse("file_path parameter is required")
			}
			docID, err := p.AttachDocument(parent, filePath)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to attach document: %v", err))
			}
			result := map[string]any{
				"document_id": docID,
				"parent":      parent,
				"file_path":   filePath,
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
		Description: "<ACTION>Construct horizontal relationship (is_relevant_to edge) between any two nodes.</ACTION> <INSTRUCTION>Consider categories and subcategories relationships when organizing multidimensional data.</INSTRUCTION>",
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

// newAddCategoryTool creates the add_category tool
func (p *KnowledgePlugin) newAddCategoryTool() llms.Tool {
	return &core.Tool{
		Name:        "add_category",
		Description: "Create a new top-level category node",
		Parameters: []core.Parameter{
			{
				Name:        "category",
				Type:        "string",
				Description: "The category name to create",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			category, ok := args["category"].(string)
			if !ok || category == "" {
				return core.NewErrorResponse("category parameter is required")
			}

			categoryID, err := p.AddCategory(category)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to add category: %v", err))
			}

			result := map[string]any{
				"category_id": categoryID,
				"category":    category,
			}

			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	}
}

// newAddSubcategoryTool creates the add_subcategory tool
func (p *KnowledgePlugin) newAddSubcategoryTool() llms.Tool {
	return &core.Tool{
		Name:        "add_subcategory",
		Description: "Create a new subcategory node under an existing category or subcategory",
		Parameters: []core.Parameter{
			{
				Name:        "parent",
				Type:        "string",
				Description: "The name or ID of the parent category or subcategory",
				Required:    true,
			},
			{
				Name:        "subcategory",
				Type:        "string",
				Description: "The subcategory name to create",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			parent, ok := args["parent"].(string)
			if !ok || parent == "" {
				return core.NewErrorResponse("parent parameter is required")
			}
			subcategory, ok := args["subcategory"].(string)
			if !ok || subcategory == "" {
				return core.NewErrorResponse("subcategory parameter is required")
			}

			subCatID, err := p.AddSubcategory(parent, subcategory)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to add subcategory: %v", err))
			}

			result := map[string]any{
				"subcategory_id": subCatID,
				"subcategory":    subcategory,
				"parent":         parent,
			}

			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	}
}

// newGetCategoriesTool creates the get_categories tool
func (p *KnowledgePlugin) newGetCategoriesTool() llms.Tool {
	return &core.Tool{
		Name:        "get_categories",
		Description: "Get all category nodes",
		Parameters:  []core.Parameter{},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			categories, err := p.GetCategories()
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to get categories: %v", err))
			}

			result := map[string]any{
				"categories": categories,
				"count":      len(categories),
			}

			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	}
}

// newGetCategoryFactsTool creates the get_category_facts tool
func (p *KnowledgePlugin) newGetCategoryFactsTool() llms.Tool {
	return &core.Tool{
		Name:        "get_category_facts",
		Description: "Get all facts directly connected to a category",
		Parameters: []core.Parameter{
			{
				Name:        "category",
				Type:        "string",
				Description: "The category name to get facts from",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			category, ok := args["category"].(string)
			if !ok || category == "" {
				return core.NewErrorResponse("category parameter is required")
			}

			facts, err := p.GetCategoryFacts(category)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to get category facts: %v", err))
			}

			result := map[string]any{
				"category": category,
				"facts":    facts,
				"count":    len(facts),
			}

			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	}
}

// newForgetTool creates the forget tool
func (p *KnowledgePlugin) newForgetTool() llms.Tool {
	return &core.Tool{
		Name:        "forget",
		Description: "Delete a node and all its dependents (cascade delete)",
		Parameters: []core.Parameter{
			{
				Name:        "identifier",
				Type:        "string",
				Description: "Node ID or content to delete",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			identifier, ok := args["identifier"].(string)
			if !ok || identifier == "" {
				return core.NewErrorResponse("identifier parameter is required")
			}

			deletedCount, err := p.Forget(identifier)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to forget: %v", err))
			}

			result := map[string]any{
				"deleted_count": deletedCount,
				"identifier":    identifier,
			}

			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	}
}
