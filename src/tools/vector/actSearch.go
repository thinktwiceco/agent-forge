package vector

import (
	"encoding/json"
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// search handles the search action for the vector tool.
func (v *Vector) search(args map[string]any) llms.ToolReturn {
	// Extract query (required)
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return core.NewErrorResponse("query parameter is required for search action and must be a non-empty string")
	}

	// Extract top_k (optional, default: 10)
	topK := 10
	if tk, ok := args["top_k"]; ok {
		switch v := tk.(type) {
		case float64:
			topK = int(v)
		case int:
			topK = v
		case int64:
			topK = int(v)
		default:
			return core.NewErrorResponse("top_k parameter must be a number")
		}
		if topK <= 0 {
			return core.NewErrorResponse("top_k must be greater than 0")
		}
	}

	// Extract filters (optional)
	var filters map[string]any
	if f, ok := args["filters"]; ok {
		filters, ok = f.(map[string]any)
		if !ok {
			return core.NewErrorResponse("filters parameter must be an object")
		}
	}

	// Generate embedding for query
	if v.embeddingGenerator == nil {
		return core.NewErrorResponse("embedding generator not configured for vector tool")
	}
	queryEmbedding, _, err := v.embeddingGenerator.GenerateEmbedding(query)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to generate query embedding: %v", err))
	}

	// Perform search
	if v.vectorDB == nil {
		return core.NewErrorResponse("vector database not configured for vector tool")
	}
	results, err := v.vectorDB.Search(queryEmbedding, topK, filters)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to search: %v", err))
	}

	// Format results as JSON
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to serialize search results: %v", err))
	}

	return core.NewSuccessResponse(string(resultsJSON))
}
