package vector

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
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

	// Add file path to metadata
	metadata["_file_path"] = filePath

	// Extract or generate document_id (optional)
	documentID, ok := args["document_id"].(string)
	if !ok || documentID == "" {
		documentID = uuid.New().String()
	}

	if v.embeddingGenerator == nil {
		return core.NewErrorResponse("embedding generator not configured for vector tool")
	}

	if v.vectorDB == nil {
		return core.NewErrorResponse("vector database not configured for vector tool")
	}

	// Chunk the text if it's too large
	chunks := chunkText(text)

	if len(chunks) == 1 {
		// Single chunk - index normally
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

	// Multiple chunks - index each chunk with linking metadata
	var indexedIDs []string
	for i, chunk := range chunks {
		chunkMetadata := make(map[string]any)
		// Copy original metadata
		for k, v := range metadata {
			chunkMetadata[k] = v
		}
		// Add chunk-specific metadata
		chunkMetadata["_parent_document_id"] = documentID
		chunkMetadata["_chunk_index"] = i
		chunkMetadata["_total_chunks"] = len(chunks)

		embedding, modelName, err := v.embeddingGenerator.GenerateEmbedding(chunk)
		if err != nil {
			return core.NewErrorResponse(fmt.Sprintf("failed to generate embedding for chunk %d/%d: %v", i+1, len(chunks), err))
		}

		chunkMetadata["_embedding_model"] = modelName

		chunkID := fmt.Sprintf("%s_chunk_%d", documentID, i)
		resultID, err := v.vectorDB.Index(embedding, chunk, chunkMetadata, modelName)
		if err != nil {
			return core.NewErrorResponse(fmt.Sprintf("failed to index chunk %d/%d: %v", i+1, len(chunks), err))
		}

		if resultID != "" {
			chunkID = resultID
		}
		indexedIDs = append(indexedIDs, chunkID)
	}

	return core.NewSuccessResponse(fmt.Sprintf("Document from file '%s' indexed successfully as %d chunks with parent ID: %s (chunk IDs: %v)", filePath, len(chunks), documentID, indexedIDs))
}
