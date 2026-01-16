package integrations

import (
	"context"
	"fmt"
	"strings"

	chromago "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/types"
	"github.com/google/uuid"
	"github.com/thinktwice/agentForge/src/core"
)

// ChromaDB implements the core.VectorDB interface using ChromaDB as the backend.
type ChromaDB struct {
	client         *chromago.Client
	collection     *chromago.Collection
	collectionName string
	config         ChromaDBConfig
}

// NewChromaDB creates a new ChromaDB instance that implements the VectorDB interface.
//
// Parameters:
//   - config: ChromaDBConfig with connection and collection settings
//
// Returns:
//   - *ChromaDB: A new ChromaDB instance
//   - error: Any error that occurred during initialization
func NewChromaDB(config ChromaDBConfig) (*ChromaDB, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create ChromaDB client
	baseURL := fmt.Sprintf("http://%s:%d", config.Host, config.Port)
	client, err := chromago.NewClient(
		chromago.WithBasePath(baseURL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChromaDB client: %w", err)
	}

	cdb := &ChromaDB{
		client:         client,
		collectionName: config.CollectionName,
		config:         config,
	}

	// Verify ChromaDB is accessible with a health check
	ctx := context.Background()
	if err := cdb.healthCheck(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("ChromaDB is not accessible at %s:%d - please ensure ChromaDB is running. Error: %w", config.Host, config.Port, err)
	}

	// Get or create collection
	collection, err := cdb.getOrCreateCollection(ctx)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to get or create collection: %w", err)
	}

	cdb.collection = collection
	return cdb, nil
}

// healthCheck verifies that ChromaDB is accessible by calling the version endpoint.
func (c *ChromaDB) healthCheck(ctx context.Context) error {
	_, err := c.client.Version(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}

// getOrCreateCollection gets an existing collection or creates a new one if it doesn't exist.
// Important: We pass nil for embedding function since we provide embeddings directly.
func (c *ChromaDB) getOrCreateCollection(ctx context.Context) (*chromago.Collection, error) {
	// Try to get existing collection (pass nil for embedding function since we provide embeddings)
	collection, err := c.client.GetCollection(ctx, c.collectionName, nil)
	if err == nil && collection != nil {
		// Collection exists - return it
		// Note: If the collection was created with an embedding function, queries might fail
		// In that case, delete and recreate the collection
		return collection, nil
	}

	// Collection doesn't exist, create it
	// Pass nil for embedding function, false for createOrGet, and L2 for distance function
	// The nil embedding function means we'll provide embeddings directly
	collection, err = c.client.CreateCollection(ctx, c.collectionName, nil, false, nil, types.L2)
	if err != nil {
		// Check if error is related to ONNX Runtime version mismatch
		errStr := err.Error()
		if strings.Contains(strings.ToLower(errStr), "ort") ||
			strings.Contains(strings.ToLower(errStr), "api version") ||
			strings.Contains(strings.ToLower(errStr), "platform-specific initialization") {
			// ONNX Runtime version mismatch - try to get collection again in case it was created
			// despite the error, or return a more helpful error message
			retryCollection, retryErr := c.client.GetCollection(ctx, c.collectionName, nil)
			if retryErr == nil && retryCollection != nil {
				// Collection was actually created, return it
				return retryCollection, nil
			}
			return nil, fmt.Errorf("failed to create collection due to ONNX Runtime version mismatch. "+
				"ChromaDB server requires ONNX Runtime API version 22, but only versions 1-21 are available. "+
				"Please update ONNX Runtime or use a ChromaDB version that doesn't require it. "+
				"Original error: %w", err)
		}
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	return collection, nil
}

// Index stores a document with its embedding, text, metadata, and embedding model.
// Implements core.VectorDB interface.
func (c *ChromaDB) Index(embedding []float32, text string, metadata map[string]any, embeddingModel string) (string, error) {
	ctx := context.Background()

	// Generate a document ID using UUID
	documentID := uuid.New().String()

	// Convert metadata to ChromaDB format
	chromaMetadata := make(map[string]interface{})
	for k, v := range metadata {
		chromaMetadata[k] = v
	}
	// Ensure embedding model is in metadata
	chromaMetadata["_embedding_model"] = embeddingModel

	// Create embedding object
	emb := types.NewEmbeddingFromFloat32(embedding)

	// Create record set with options
	recordSet, err := types.NewRecordSet()
	if err != nil {
		return "", fmt.Errorf("failed to create record set: %w", err)
	}

	// Build record options
	opts := []types.Option{
		types.WithID(documentID),
		types.WithEmbedding(*emb),
		types.WithDocument(text),
	}
	// Add metadata key-value pairs
	for k, v := range chromaMetadata {
		opts = append(opts, types.WithMetadata(k, v))
	}

	// Add record to record set
	recordSet = recordSet.WithRecord(opts...)

	// Add document to collection
	_, err = c.collection.AddRecords(ctx, recordSet)
	if err != nil {
		return "", fmt.Errorf("failed to add document to ChromaDB: %w", err)
	}

	return documentID, nil
}

// Search performs semantic search with optional metadata filters.
// Implements core.VectorDB interface.
func (c *ChromaDB) Search(queryEmbedding []float32, topK int, filters map[string]any) ([]core.SearchResult, error) {
	ctx := context.Background()

	if topK <= 0 {
		topK = c.config.DefaultTopK
	}

	// Convert query embedding to ChromaDB format
	queryEmb := types.NewEmbeddingFromFloat32(queryEmbedding)

	// Build query options
	queryOptions := []types.CollectionQueryOption{
		types.WithQueryEmbedding(queryEmb),
		types.WithNResults(int32(topK)),
		types.WithInclude(types.IDocuments, types.IMetadatas, types.IDistances),
	}

	// Add metadata filters if provided
	if len(filters) > 0 {
		whereMap := make(map[string]interface{})
		for k, v := range filters {
			whereMap[k] = v
		}
		queryOptions = append(queryOptions, types.WithWhereMap(whereMap))
	}

	// Query the collection
	queryResults, err := c.collection.QueryWithOptions(ctx, queryOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to query ChromaDB: %w", err)
	}

	// Convert QueryResults to SearchResult slice
	results := make([]core.SearchResult, 0)

	// QueryResults has arrays of arrays (one per query)
	// Since we're doing a single query, we use the first element
	if len(queryResults.Ids) == 0 || len(queryResults.Ids[0]) == 0 {
		return results, nil
	}

	ids := queryResults.Ids[0]
	documents := queryResults.Documents[0]
	metadatas := queryResults.Metadatas[0]
	distances := queryResults.Distances[0]

	// Convert distances to similarity scores (ChromaDB returns distances, lower is better)
	// We'll convert to similarity scores where higher is better
	for i := range ids {
		// Convert distance to similarity score (1 / (1 + distance))
		// Or use negative distance as similarity (higher is better)
		similarity := -distances[i] // Negative distance = similarity (higher is better)

		result := core.SearchResult{
			DocumentID: ids[i],
			Text:       documents[i],
			Metadata:   metadatas[i],
			Score:      similarity,
		}
		results = append(results, result)
	}

	return results, nil
}

// Delete removes a document by ID.
// Implements core.VectorDB interface.
func (c *ChromaDB) Delete(documentID string) error {
	ctx := context.Background()

	ids := []string{documentID}
	deletedIDs, err := c.collection.Delete(ctx, ids, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete document from ChromaDB: %w", err)
	}

	if len(deletedIDs) == 0 {
		return fmt.Errorf("document with ID %s not found", documentID)
	}

	return nil
}

// Close closes the ChromaDB client connection.
func (c *ChromaDB) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
