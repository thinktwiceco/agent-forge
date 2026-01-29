package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/thinktwiceco/agent-forge/src/core"
)

// MilvusDB implements the core.VectorDB interface using Milvus as the backend.
type MilvusDB struct {
	client         *milvusclient.Client
	collectionName string
	vectorDim      int
	config         MilvusConfig
}

// NewMilvusDB creates a new MilvusDB instance that implements the VectorDB interface.
//
// Parameters:
//   - config: MilvusConfig with connection and collection settings
//
// Returns:
//   - *MilvusDB: A new MilvusDB instance
//   - error: Any error that occurred during initialization
func NewMilvusDB(config MilvusConfig) (*MilvusDB, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create Milvus client
	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	clientConfig := &milvusclient.ClientConfig{
		Address: address,
	}

	// Add authentication if provided
	if config.Username != "" && config.Password != "" {
		clientConfig.Username = config.Username
		clientConfig.Password = config.Password
	}

	ctx := context.Background()
	client, err := milvusclient.New(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Milvus client: %w", err)
	}

	mdb := &MilvusDB{
		client:         client,
		collectionName: config.CollectionName,
		vectorDim:      config.VectorDim,
		config:         config,
	}

	// Verify Milvus is accessible with a health check
	if err := mdb.healthCheck(ctx); err != nil {
		_ = client.Close(ctx)
		return nil, fmt.Errorf("milvus is not accessible at %s - please ensure Milvus is running. Error: %w", address, err)
	}

	// Get or create collection
	if err := mdb.ensureCollection(ctx); err != nil {
		_ = client.Close(ctx)
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	return mdb, nil
}

// healthCheck verifies that Milvus is accessible by checking if it's connected.
func (m *MilvusDB) healthCheck(ctx context.Context) error {
	// Try to list collections as a health check
	_, err := m.client.ListCollections(ctx, milvusclient.NewListCollectionOption())
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}

// ensureCollection ensures the collection exists, creating it if necessary.
func (m *MilvusDB) ensureCollection(ctx context.Context) error {
	// Check if collection exists
	exists, err := m.client.HasCollection(ctx, milvusclient.NewHasCollectionOption(m.collectionName))
	if err != nil {
		return fmt.Errorf("failed to check if collection exists: %w", err)
	}

	if exists {
		return nil
	}

	// Create collection schema using builder pattern
	schema := entity.NewSchema().
		WithName(m.collectionName).
		WithDescription("Collection for agent knowledge base").
		WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeVarChar).WithIsPrimaryKey(true).WithMaxLength(100)).
		WithField(entity.NewField().WithName("text").WithDataType(entity.FieldTypeVarChar).WithMaxLength(65535)).
		WithField(entity.NewField().WithName("embedding").WithDataType(entity.FieldTypeFloatVector).WithDim(int64(m.vectorDim))).
		WithField(entity.NewField().WithName("metadata").WithDataType(entity.FieldTypeJSON))

	// Create index on embedding field for efficient search
	indexOptions := []milvusclient.CreateIndexOption{
		milvusclient.NewCreateIndexOption(m.collectionName, "embedding", index.NewHNSWIndex(entity.L2, 16, 200)),
	}

	// Create collection with index
	err = m.client.CreateCollection(ctx, milvusclient.NewCreateCollectionOption(m.collectionName, schema).
		WithIndexOptions(indexOptions...))
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

// ensureCollectionLoaded ensures the collection is loaded into memory before querying/searching.
// Collections must be loaded before they can be queried or searched in Milvus.
func (m *MilvusDB) ensureCollectionLoaded(ctx context.Context) error {
	// Check if collection is already loaded
	loadStateResp, err := m.client.GetLoadState(ctx, milvusclient.NewGetLoadStateOption(m.collectionName))
	if err != nil {
		// If we can't check the state, try loading anyway
		_, loadErr := m.client.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(m.collectionName))
		if loadErr != nil {
			return fmt.Errorf("failed to load collection: %w", loadErr)
		}
		return nil
	}

	// Check if already loaded - LoadStateCode enum values: 0=NotExist, 1=NotLoaded, 2=Loading, 3=Loaded
	// We need to check loadStateResp.State
	if loadStateResp.State != 3 {
		// Load the collection
		_, err = m.client.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(m.collectionName))
		if err != nil {
			return fmt.Errorf("failed to load collection: %w", err)
		}
	}

	return nil
}

// Index stores a document with its embedding, text, metadata, and embedding model.
// Implements core.VectorDB interface.
func (m *MilvusDB) Index(embedding []float32, text string, metadata map[string]any, embeddingModel string) (string, error) {
	ctx := context.Background()

	// Generate a document ID using UUID
	documentID := uuid.New().String()

	// Ensure embedding model is in metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadata["_embedding_model"] = embeddingModel

	// Serialize metadata to JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Insert data using column-based insert option
	_, err = m.client.Insert(ctx, milvusclient.NewColumnBasedInsertOption(m.collectionName).
		WithVarcharColumn("id", []string{documentID}).
		WithVarcharColumn("text", []string{text}).
		WithFloatVectorColumn("embedding", m.vectorDim, [][]float32{embedding}).
		WithColumns(column.NewColumnJSONBytes("metadata", [][]byte{metadataJSON})))
	if err != nil {
		return "", fmt.Errorf("failed to insert document to Milvus: %w", err)
	}

	return documentID, nil
}

// Search performs semantic search with optional metadata filters.
// Implements core.VectorDB interface.
func (m *MilvusDB) Search(queryEmbedding []float32, topK int, filters map[string]any) ([]core.SearchResult, error) {
	ctx := context.Background()

	// Ensure collection is loaded before searching
	if err := m.ensureCollectionLoaded(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure collection is loaded: %w", err)
	}

	if topK <= 0 {
		topK = m.config.DefaultTopK
	}

	// Build search vector
	searchVectors := []entity.Vector{entity.FloatVector(queryEmbedding)}

	// Build filter expression if filters are provided
	var filterExpr string
	// TODO: Implement proper filter expression building for JSON metadata
	// For JSON field, we need to use JSON_CONTAINS or similar
	// For now, we'll search without filters
	_ = len(filters) // Suppress unused variable warning until filter implementation

	// Perform search
	searchOption := milvusclient.NewSearchOption(m.collectionName, topK, searchVectors).
		WithOutputFields("id", "text", "metadata")
	if filterExpr != "" {
		searchOption = searchOption.WithFilter(filterExpr)
	}

	searchResults, err := m.client.Search(ctx, searchOption)
	if err != nil {
		return nil, fmt.Errorf("failed to search Milvus: %w", err)
	}

	// Convert search results to SearchResult slice
	results := make([]core.SearchResult, 0)

	if len(searchResults) == 0 {
		return results, nil
	}

	// Process first query result (we only have one query vector)
	resultSet := searchResults[0]

	// Get columns
	idColumn := resultSet.GetColumn("id")
	textColumn := resultSet.GetColumn("text")
	metadataColumn := resultSet.GetColumn("metadata")

	// Extract IDs
	var ids []string
	if idColumn != nil {
		if idVals, ok := idColumn.(*column.ColumnVarChar); ok {
			ids = idVals.Data()
		}
	}

	// Extract scores
	scores := resultSet.Scores

	// Process each result
	for i := 0; i < len(ids) && i < len(scores); i++ {
		documentID := ids[i]
		score := scores[i]

		// Get text
		var text string
		if textColumn != nil {
			if textVals, ok := textColumn.(*column.ColumnVarChar); ok && i < len(textVals.Data()) {
				text = textVals.Data()[i]
			}
		}

		// Get metadata
		var metadata map[string]any
		if metadataColumn != nil {
			if metaVals, ok := metadataColumn.(*column.ColumnJSONBytes); ok && i < len(metaVals.Data()) {
				metaBytes := metaVals.Data()[i]
				if len(metaBytes) > 0 {
					if err := json.Unmarshal(metaBytes, &metadata); err != nil {
						metadata = make(map[string]any)
					}
				} else {
					metadata = make(map[string]any)
				}
			}
		} else {
			metadata = make(map[string]any)
		}

		// Convert distance to similarity (higher is better)
		similarity := -score

		result := core.SearchResult{
			DocumentID: documentID,
			Text:       text,
			Metadata:   metadata,
			Score:      similarity,
		}
		results = append(results, result)
	}

	return results, nil
}

// ListDocuments retrieves documents with pagination and optional filters.
// Implements core.VectorDB interface.
func (m *MilvusDB) ListDocuments(opts core.ListOptions) ([]core.DocumentSummary, int, error) {
	ctx := context.Background()

	// Ensure collection is loaded before querying
	if err := m.ensureCollectionLoaded(ctx); err != nil {
		return nil, 0, fmt.Errorf("failed to ensure collection is loaded: %w", err)
	}

	// Build filter expression from filters if provided
	var filterExpr string
	if len(opts.Filters) > 0 {
		// Build filter expression for JSON metadata field
		// Milvus supports JSON path expressions like JSON_EXTRACT(metadata, '$.key')
		filterParts := make([]string, 0, len(opts.Filters))
		for k, v := range opts.Filters {
			// Use JSON path extraction for metadata field filtering
			// Format: JSON_EXTRACT(metadata, '$.key') == 'value'
			valueStr := fmt.Sprintf("%v", v)
			// Escape single quotes in value
			escapedValue := strings.ReplaceAll(valueStr, "'", "''")
			filterParts = append(filterParts, fmt.Sprintf("JSON_EXTRACT(metadata, '$.%s') == '%s'", k, escapedValue))
		}
		if len(filterParts) > 0 {
			filterExpr = filterParts[0]
			for i := 1; i < len(filterParts); i++ {
				filterExpr = fmt.Sprintf("%s && %s", filterExpr, filterParts[i])
			}
		}
	}

	// If no filter, use a simple expression that matches all documents
	if filterExpr == "" {
		filterExpr = "id != \"\""
	}

	// Set default limit if not specified
	limit := opts.Limit
	if limit <= 0 {
		limit = 100 // Default limit
	}

	// Build query option
	// Note: Milvus Query API doesn't directly support offset
	// We'll fetch more results and skip offset manually if needed
	// For better performance with large offsets, consider using ID-based pagination
	queryLimit := limit
	if opts.Offset > 0 {
		// Fetch offset + limit results, then skip the first offset items
		queryLimit = opts.Offset + limit
	}

	queryOption := milvusclient.NewQueryOption(m.collectionName).
		WithFilter(filterExpr).
		WithOutputFields("id", "text", "metadata").
		WithLimit(queryLimit)

	// Execute query
	queryResults, err := m.client.Query(ctx, queryOption)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query documents from Milvus: %w", err)
	}

	// Get columns
	idColumn := queryResults.GetColumn("id")
	textColumn := queryResults.GetColumn("text")
	metadataColumn := queryResults.GetColumn("metadata")

	// Extract IDs
	var ids []string
	if idColumn != nil {
		if idVals, ok := idColumn.(*column.ColumnVarChar); ok {
			ids = idVals.Data()
		}
	}

	// Get total count (before applying offset)
	totalCount := len(ids)

	// Apply offset if needed
	startIdx := 0
	if opts.Offset > 0 && opts.Offset < totalCount {
		startIdx = opts.Offset
	}

	// Calculate end index
	endIdx := startIdx + limit
	if endIdx > totalCount {
		endIdx = totalCount
	}

	// Build DocumentSummary slice
	summaries := make([]core.DocumentSummary, 0, endIdx-startIdx)

	for i := startIdx; i < endIdx; i++ {
		documentID := ids[i]

		// Get text
		var text string
		if textColumn != nil {
			if textVals, ok := textColumn.(*column.ColumnVarChar); ok && i < len(textVals.Data()) {
				text = textVals.Data()[i]
			}
		}

		// Get metadata
		var metadata map[string]any
		if metadataColumn != nil {
			if metaVals, ok := metadataColumn.(*column.ColumnJSONBytes); ok && i < len(metaVals.Data()) {
				metaBytes := metaVals.Data()[i]
				if len(metaBytes) > 0 {
					if err := json.Unmarshal(metaBytes, &metadata); err != nil {
						metadata = make(map[string]any)
					}
				} else {
					metadata = make(map[string]any)
				}
			}
		} else {
			metadata = make(map[string]any)
		}

		summary := core.DocumentSummary{
			DocumentID: documentID,
			Text:       text,
			Metadata:   metadata,
		}
		summaries = append(summaries, summary)
	}

	// For total count, we need to query without limit to get accurate count
	// This is expensive for large collections, but required for accurate pagination
	// In production, consider caching or using approximate counts
	if len(opts.Filters) > 0 || opts.Offset == 0 {
		// Only get accurate count if we have filters or are at the start
		// Otherwise, return the count we got from the limited query
		countQueryOption := milvusclient.NewQueryOption(m.collectionName).
			WithFilter(filterExpr).
			WithOutputFields("id").
			WithLimit(10000) // Large limit to get count, but not unlimited

		countResults, err := m.client.Query(ctx, countQueryOption)
		if err == nil {
			if countIDColumn := countResults.GetColumn("id"); countIDColumn != nil {
				if countIDVals, ok := countIDColumn.(*column.ColumnVarChar); ok {
					totalCount = len(countIDVals.Data())
				}
			}
		}
		// If count query fails, use the count from the main query
	}

	return summaries, totalCount, nil
}

// Delete removes a document by ID.
// Implements core.VectorDB interface.
func (m *MilvusDB) Delete(documentID string) error {
	ctx := context.Background()

	// Build delete expression
	expr := fmt.Sprintf("id == \"%s\"", documentID)

	// Delete document
	_, err := m.client.Delete(ctx, milvusclient.NewDeleteOption(m.collectionName).WithExpr(expr))
	if err != nil {
		return fmt.Errorf("failed to delete document from Milvus: %w", err)
	}

	return nil
}

// Close closes the Milvus client connection.
func (m *MilvusDB) Close() error {
	if m.client != nil {
		return m.client.Close(context.Background())
	}
	return nil
}
