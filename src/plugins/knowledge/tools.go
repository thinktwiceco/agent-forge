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
		p.newExploreFactTool(),
		p.newFindTool(),
		p.newRememberTool(),
		p.newAddCategoryTool(),
		p.newGetCategoriesTool(),
		p.newGetCategoryFactsTool(),
		p.newForgetTool(),
	}
}

// newExploreCategoryTool creates the explore_category tool
func (p *KnowledgePlugin) newExploreCategoryTool() llms.Tool {
	return &core.Tool{
		Name:        "explore_category",
		Description: "Explore a category and get its full hierarchy including all child nodes",
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

// newExploreFactTool creates the explore_fact tool
func (p *KnowledgePlugin) newExploreFactTool() llms.Tool {
	return &core.Tool{
		Name:        "explore_fact",
		Description: "Explore a fact and get its full context including related facts and parent categories",
		Parameters: []core.Parameter{
			{
				Name:        "fact",
				Type:        "string",
				Description: "The fact content to explore",
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
		Description: "Save a fact under a specific category",
		Parameters: []core.Parameter{
			{
				Name:        "category",
				Type:        "string",
				Description: "The category to store the fact under",
				Required:    true,
			},
			{
				Name:        "fact",
				Type:        "string",
				Description: "The fact content to remember",
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

			factID, err := p.Remember(category, fact)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to remember fact: %v", err))
			}

			result := map[string]any{
				"fact_id":  factID,
				"category": category,
				"fact":     fact,
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
		Description: "Create a new category node",
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
