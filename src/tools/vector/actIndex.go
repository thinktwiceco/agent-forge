package vector

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// index handles the index action for the vector tool.
func (v *Vector) index(args map[string]any) llms.ToolReturn {
	// Extract text (required)
	text, ok := args["text"].(string)
	if !ok || text == "" {
		return core.NewErrorResponse("text parameter is required for index action and must be a non-empty string")
	}

	// Extract metadata (optional)
	var metadata map[string]any
	if md, ok := args["metadata"]; ok {
		metadata, ok = md.(map[string]any)
		if !ok {
			return core.NewErrorResponse("metadata parameter must be an object")
		}
	} else {
		metadata = make(map[string]any)
	}

	// Extract or generate document_id (optional)
	documentID, ok := args["document_id"].(string)
	if !ok || documentID == "" {
		documentID = uuid.New().String()
	}

	// Generate embedding
	embedding, modelName, err := v.embeddingGenerator.GenerateEmbedding(text)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to generate embedding: %v", err))
	}

	// Add embedding model to metadata
	metadata["_embedding_model"] = modelName

	// Index the document
	resultID, err := v.vectorDB.Index(embedding, text, metadata, modelName)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to index document: %v", err))
	}

	// Use the returned ID if provided, otherwise use the one we generated
	if resultID != "" {
		documentID = resultID
	}

	return core.NewSuccessResponse(fmt.Sprintf("Document indexed successfully with ID: %s", documentID))
}
