package vector

import (
	"encoding/json"
	"fmt"

	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// listDocumentsResponse represents the response structure for listDocuments action.
type listDocumentsResponse struct {
	Documents []core.DocumentSummary `json:"documents"`
	Total     int                    `json:"total"`
}

// listDocuments handles the listDocuments action for the vector tool.
func (v *Vector) listDocuments(args map[string]any) llms.ToolReturn {
	// Extract offset (optional, default: 0)
	offset := 0
	if off, ok := args["offset"]; ok {
		switch v := off.(type) {
		case float64:
			offset = int(v)
		case int:
			offset = v
		case int64:
			offset = int(v)
		default:
			return core.NewErrorResponse("offset parameter must be a number")
		}
		if offset < 0 {
			return core.NewErrorResponse("offset must be greater than or equal to 0")
		}
	}

	// Extract limit (optional, default: 10)
	limit := 10
	if lim, ok := args["limit"]; ok {
		switch v := lim.(type) {
		case float64:
			limit = int(v)
		case int:
			limit = v
		case int64:
			limit = int(v)
		default:
			return core.NewErrorResponse("limit parameter must be a number")
		}
		if limit <= 0 {
			return core.NewErrorResponse("limit must be greater than 0")
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

	// Build ListOptions
	opts := core.ListOptions{
		Offset:  offset,
		Limit:   limit,
		Filters: filters,
	}

	if v.vectorDB == nil {
		return core.NewErrorResponse("vector database not configured for vector tool")
	}

	// List documents
	documents, total, err := v.vectorDB.ListDocuments(opts)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to list documents: %v", err))
	}

	// Build response
	response := listDocumentsResponse{
		Documents: documents,
		Total:     total,
	}

	// Format results as JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to serialize list results: %v", err))
	}

	return core.NewSuccessResponse(string(responseJSON))
}
