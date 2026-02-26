package vector

import (
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// NewVectorTool creates a new vector database tool that allows indexing, searching, and deleting documents.
//
// The tool requires a VectorDB implementation for storage and an EmbeddingGenerator for creating embeddings.
//
// Available actions:
//   - index: Index a document with text and optional metadata
//   - indexFile: Index a document from a file path
//   - search: Perform semantic search with optional filters
//   - listDocuments: List documents with pagination and optional filters
//   - delete: Delete a document by ID
func NewVectorTool(vectorDB core.VectorDB, embeddingGenerator core.EmbeddingGenerator) llms.Tool {
	v := &Vector{
		vectorDB:           vectorDB,
		embeddingGenerator: embeddingGenerator,
	}

	detailsAbout := func(item string) string {
		switch item {
		case "index":
			return `index: Index a document with text content and optional metadata.
- Required: text (string) — document text to embed and store
- Optional: document_id (string) — custom ID; auto-generated UUID if omitted
- Optional: metadata (object) — free-form key-value pairs stored alongside the document
- The embedding model name is automatically stored in metadata as "_embedding_model"`
		case "indexFile":
			return `indexFile: Read a file and index its content as a document.
- Required: file_path (string) — path to the file to read and index
- Optional: document_id (string) — custom ID; auto-generated UUID if omitted
- Optional: metadata (object) — free-form key-value pairs`
		case "search":
			return `search: Perform semantic search using a query string.
- Required: query (string) — search query text
- Optional: top_k (number, default 10) — number of results to return
- Optional: filters (object) — exact-match metadata filters, e.g. {"category": "docs"}
- Returns: results with text, metadata, and similarity scores (highest first)`
		case "listDocuments":
			return `listDocuments: List documents from the vector database with pagination.
- Optional: offset (number, default 0) — number of documents to skip
- Optional: limit (number, default 10) — maximum number of documents to return
- Optional: filters (object) — exact-match metadata filters
- Returns: documents with text, metadata, and total count`
		case "delete":
			return `delete: Remove a document from the vector database by its ID.
- Required: document_id (string) — the ID of the document to delete`
		default:
			return fmt.Sprintf("Nothing to add about %s", item)
		}
	}

	return &core.Tool{
		Name:        "vector_db",
		Description: "Index, search, and delete documents in a vector database using semantic search.",
		AdvanceDesc: `Advanced Details:
- Available actions: index, indexFile, search, listDocuments, delete
  Use expand tool with details_about="<action>" for full parameter details on any action.
- Common parameters:
  * action (required): the action to perform
- Behavior:
  * Embeddings are generated automatically; model name is stored in metadata as "_embedding_model"
  * search results are ordered by similarity score (highest first)
  * Metadata filtering uses exact match on key-value pairs`,
		DetailsAboutFunc: detailsAbout,
		TroubleshootingInfo: `Troubleshooting:
- If index fails: Ensure text parameter is provided and non-empty
- If indexFile fails: Ensure file_path parameter is provided, file exists, and is readable
- If search fails: Ensure query parameter is provided and non-empty
- If listDocuments fails: Ensure offset and limit are non-negative numbers
- If delete fails: Ensure document_id exists in the database
- If embedding generation fails: Check that the embedding generator is properly configured
- If no results found: Try adjusting top_k/limit or removing filters
- Ensure action parameter is exactly one of: index, indexFile, search, listDocuments, delete`,
		Parameters: []core.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "The action to perform: 'index', 'indexFile', 'search', 'listDocuments', or 'delete'",
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
				Name:        "file_path",
				Type:        "string",
				Description: "The file path to read and index (required for 'indexFile' action)",
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
				Description: "Document ID - auto-generated UUID if not provided for 'index'/'indexFile', required for 'delete'",
				Required:    false,
			},
			{
				Name:        "top_k",
				Type:        "number",
				Description: "Number of results to return for search (default: 10, optional)",
				Required:    false,
			},
			{
				Name:        "offset",
				Type:        "number",
				Description: "Number of documents to skip for listDocuments (default: 0, optional)",
				Required:    false,
			},
			{
				Name:        "limit",
				Type:        "number",
				Description: "Maximum number of documents to return for listDocuments (default: 10, optional)",
				Required:    false,
			},
			{
				Name:        "filters",
				Type:        "object",
				Description: "Metadata filters for search/listDocuments - exact match on key-value pairs (optional)",
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
			case "indexFile":
				return v.indexFile(args)
			case "search":
				return v.search(args)
			case "listDocuments":
				return v.listDocuments(args)
			case "delete":
				return v.delete(args)
			default:
				return core.NewErrorResponse(fmt.Sprintf("unknown action: %s. Valid actions are: index, indexFile, search, listDocuments, delete", action))
			}
		},
	}
}
