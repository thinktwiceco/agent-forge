package core

// EmbeddingGenerator is an interface for generating embeddings from text.
// Implementations should handle API calls to embedding models and return
// the embedding vector along with the model name used.
type EmbeddingGenerator interface {
	// GenerateEmbedding generates an embedding vector for the given text.
	// Returns the embedding vector, the model name used, and any error that occurred.
	GenerateEmbedding(text string) ([]float32, string, error)
}
