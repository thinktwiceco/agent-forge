package vector

import (
	"fmt"

	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// NewVectorTool creates a new vector database tool that allows indexing, searching, and deleting documents.
//
// The tool requires a VectorDB implementation for storage and an EmbeddingGenerator for creating embeddings.
//
// Available actions:
//   - index: Index a document with text and optional metadata
//   - search: Perform semantic search with optional filters
//   - delete: Delete a document by ID
func NewVectorTool(vectorDB core.VectorDB, embeddingGenerator core.EmbeddingGenerator) llms.Tool {
	v := &Vector{
		vectorDB:           vectorDB,
		embeddingGenerator: embeddingGenerator,
	}

	return &core.Tool{
		Name:        "vector_db",
		Description: "Index, search, and delete documents in a vector database using semantic search.",
		AdvanceDesc: `Advanced Details:
- Actions:
  * index: Index a document with text content and optional metadata. Automatically generates embeddings and tracks the embedding model used.
  * search: Perform semantic search using a query string. Returns results with text, metadata, and similarity scores.
  * delete: Remove a document from the vector database by its document ID.
- Parameters:
  * action (required): The action to perform: "index", "search", or "delete"
  * text (required for index): The document text to index
  * query (required for search): The search query text
  * metadata (optional): Free-form key-value pairs for document metadata
  * document_id (optional for index): Document ID - auto-generated UUID if not provided
  * top_k (optional for search): Number of results to return (default: 10)
  * filters (optional for search): Metadata filters for exact-match filtering
- Behavior:
  * Embeddings are generated automatically using the configured embedding generator
  * The embedding model name is automatically stored in metadata as "_embedding_model"
  * Search results include similarity scores ordered from highest to lowest
  * Metadata filtering uses exact match on key-value pairs
- Usage:
  * Use index to store documents for later retrieval
  * Use search to find semantically similar documents
  * Use delete to remove documents when no longer needed`,
		TroubleshootingInfo: `Troubleshooting:
- If index fails: Ensure text parameter is provided and non-empty
- If search fails: Ensure query parameter is provided and non-empty
- If delete fails: Ensure document_id exists in the database
- If embedding generation fails: Check that the embedding generator is properly configured
- If no results found: Try adjusting top_k or removing filters
- Ensure action parameter is exactly one of: index, search, delete`,
		Parameters: []core.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "The action to perform: 'index', 'search', or 'delete'",
				Required:    true,
				Validator:   validateAction,
			},
			{
				Name:        "text",
				Type:        "string",
				Description: "The document text to index (required for 'index' action)",
				Required:    false,
			},
			{
				Name:        "query",
				Type:        "string",
				Description: "The search query text (required for 'search' action)",
				Required:    false,
			},
			{
				Name:        "metadata",
				Type:        "object",
				Description: "Free-form key-value pairs for document metadata (optional)",
				Required:    false,
			},
			{
				Name:        "document_id",
				Type:        "string",
				Description: "Document ID - auto-generated UUID if not provided for 'index', required for 'delete'",
				Required:    false,
			},
			{
				Name:        "top_k",
				Type:        "number",
				Description: "Number of results to return for search (default: 10, optional)",
				Required:    false,
			},
			{
				Name:        "filters",
				Type:        "object",
				Description: "Metadata filters for search - exact match on key-value pairs (optional)",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			action, ok := args["action"].(string)
			if !ok {
				return core.NewErrorResponse("action parameter is required and must be a string")
			}

			switch action {
			case "index":
				return v.index(args)
			case "search":
				return v.search(args)
			case "delete":
				return v.delete(args)
			default:
				return core.NewErrorResponse(fmt.Sprintf("unknown action: %s. Valid actions are: index, search, delete", action))
			}
		},
	}
}
