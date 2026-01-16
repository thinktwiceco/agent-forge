package core

// VectorDB is an interface for vector database operations.
// Implementations should handle storing and retrieving documents with their embeddings.
type VectorDB interface {
	// Index stores a document with its embedding, text, metadata, and embedding model.
	// Returns the document ID and any error that occurred.
	Index(embedding []float32, text string, metadata map[string]any, embeddingModel string) (string, error)

	// Search performs semantic search with optional metadata filters.
	// Returns a slice of SearchResult ordered by similarity score (highest first).
	Search(queryEmbedding []float32, topK int, filters map[string]any) ([]SearchResult, error)

	// Delete removes a document by ID.
	// Returns an error if the document doesn't exist or deletion fails.
	Delete(documentID string) error
}

// SearchResult represents a single search result from a vector database query.
type SearchResult struct {
	DocumentID string         // Unique identifier of the document
	Text       string         // Original text content
	Metadata   map[string]any // Document metadata
	Score      float32        // Similarity score (higher is more similar)
}
