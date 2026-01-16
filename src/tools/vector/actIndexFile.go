package vector

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// indexFile handles the indexFile action for the vector tool.
// It reads content from a file path and indexes it.
func (v *Vector) indexFile(args map[string]any) llms.ToolReturn {
	// Extract file_path (required)
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return core.NewErrorResponse("file_path parameter is required for indexFile action and must be a non-empty string")
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return core.NewErrorResponse(fmt.Sprintf("file not found: %s", filePath))
		}
		return core.NewErrorResponse(fmt.Sprintf("failed to read file '%s': %v", filePath, err))
	}

	text := string(content)
	if text == "" {
		return core.NewErrorResponse(fmt.Sprintf("file '%s' is empty", filePath))
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

	return core.NewSuccessResponse(fmt.Sprintf("Document from file '%s' indexed successfully with ID: %s", filePath, documentID))
}
