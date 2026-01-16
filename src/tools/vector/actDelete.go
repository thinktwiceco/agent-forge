package vector

import (
	"fmt"

	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// delete handles the delete action for the vector tool.
func (v *Vector) delete(args map[string]any) llms.ToolReturn {
	// Extract document_id (required)
	documentID, ok := args["document_id"].(string)
	if !ok || documentID == "" {
		return core.NewErrorResponse("document_id parameter is required for delete action and must be a non-empty string")
	}

	// Delete the document
	err := v.vectorDB.Delete(documentID)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to delete document: %v", err))
	}

	return core.NewSuccessResponse(fmt.Sprintf("Document %s deleted successfully", documentID))
}
