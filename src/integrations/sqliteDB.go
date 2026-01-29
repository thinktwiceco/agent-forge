package integrations

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thinktwiceco/agent-forge/src/core"
)

// SQLiteDB implements the core.VectorDB interface using SQLite as the backend.
type SQLiteDB struct {
	db        *sql.DB
	tableName string
	vectorDim int
	config    SQLiteConfig
}

// SQLiteConfig holds configuration parameters for SQLite vector database.
type SQLiteConfig struct {
	// DBPath is the path to the SQLite database file (e.g., "./vectors.db")
	DBPath string

	// TableName is the name of the table to use for documents (default: "documents")
	TableName string

	// DefaultTopK is the default number of results to return for search (default: 10)
	DefaultTopK int

	// VectorDim is the dimension of the vector embeddings (required)
	VectorDim int
}

// Validate checks that all required fields are set and sets defaults.
func (c *SQLiteConfig) Validate() error {
	if c.DBPath == "" {
		return fmt.Errorf("DBPath is required")
	}
	if c.TableName == "" {
		c.TableName = "documents"
	}
	if c.DefaultTopK == 0 {
		c.DefaultTopK = 10
	}
	if c.VectorDim <= 0 {
		return fmt.Errorf("VectorDim must be greater than 0")
	}
	return nil
}

// NewSQLiteDB creates a new SQLiteDB instance that implements the VectorDB interface.
//
// Parameters:
//   - config: SQLiteConfig with database path and table settings
//
// Returns:
//   - *SQLiteDB: A new SQLiteDB instance
//   - error: Any error that occurred during initialization
func NewSQLiteDB(config SQLiteConfig) (*SQLiteDB, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Open SQLite database
	db, err := sql.Open("sqlite3", config.DBPath+"?_journal_mode=WAL&_foreign_keys=1")
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Test connection
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to SQLite database: %w", err)
	}

	sdb := &SQLiteDB{
		db:        db,
		tableName: config.TableName,
		vectorDim: config.VectorDim,
		config:    config,
	}

	// Ensure table exists
	if err := sdb.ensureTable(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ensure table: %w", err)
	}

	return sdb, nil
}

// ensureTable ensures the table exists, creating it if necessary.
func (s *SQLiteDB) ensureTable(ctx context.Context) error {
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			text TEXT NOT NULL,
			embedding BLOB NOT NULL,
			metadata TEXT NOT NULL,
			embedding_model TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`, s.tableName)

	_, err := s.db.ExecContext(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create index on embedding_model for filtering
	indexSQL := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_embedding_model ON %s(embedding_model)
	`, s.tableName, s.tableName)
	_, err = s.db.ExecContext(ctx, indexSQL)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

// serializeEmbedding converts []float32 to BLOB for storage.
func serializeEmbedding(embedding []float32) ([]byte, error) {
	// Store as little-endian float32 array
	data := make([]byte, len(embedding)*4)
	for i, v := range embedding {
		binary.LittleEndian.PutUint32(data[i*4:(i+1)*4], math.Float32bits(v))
	}
	return data, nil
}

// deserializeEmbedding converts BLOB to []float32.
func deserializeEmbedding(data []byte) ([]float32, error) {
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("invalid embedding data length: %d", len(data))
	}
	embedding := make([]float32, len(data)/4)
	for i := 0; i < len(embedding); i++ {
		bits := binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
		embedding[i] = math.Float32frombits(bits)
	}
	return embedding, nil
}

// cosineSimilarity calculates the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have the same dimension: %d != %d", len(a), len(b))
	}

	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0, nil
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB)))), nil
}

// Index stores a document with its embedding, text, metadata, and embedding model.
// Implements core.VectorDB interface.
func (s *SQLiteDB) Index(embedding []float32, text string, metadata map[string]any, embeddingModel string) (string, error) {
	ctx := context.Background()

	// Validate embedding dimension
	if len(embedding) != s.vectorDim {
		return "", fmt.Errorf("embedding dimension mismatch: expected %d, got %d", s.vectorDim, len(embedding))
	}

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

	// Serialize embedding to BLOB
	embeddingBlob, err := serializeEmbedding(embedding)
	if err != nil {
		return "", fmt.Errorf("failed to serialize embedding: %w", err)
	}

	// Insert document
	insertSQL := fmt.Sprintf(`
		INSERT INTO %s (id, text, embedding, metadata, embedding_model)
		VALUES (?, ?, ?, ?, ?)
	`, s.tableName)

	_, err = s.db.ExecContext(ctx, insertSQL, documentID, text, embeddingBlob, string(metadataJSON), embeddingModel)
	if err != nil {
		return "", fmt.Errorf("failed to insert document: %w", err)
	}

	return documentID, nil
}

// Search performs semantic search with optional metadata filters.
// Implements core.VectorDB interface.
func (s *SQLiteDB) Search(queryEmbedding []float32, topK int, filters map[string]any) ([]core.SearchResult, error) {
	ctx := context.Background()

	// Validate embedding dimension
	if len(queryEmbedding) != s.vectorDim {
		return nil, fmt.Errorf("query embedding dimension mismatch: expected %d, got %d", s.vectorDim, len(queryEmbedding))
	}

	if topK <= 0 {
		topK = s.config.DefaultTopK
	}

	// Build WHERE clause for filters
	whereClause := "1=1"
	args := []interface{}{}
	if len(filters) > 0 {
		conditions := []string{}
		for k, v := range filters {
			// Check if filter key exists in metadata JSON
			// SQLite JSON functions: json_extract(metadata, '$.key')
			conditions = append(conditions, fmt.Sprintf("json_extract(metadata, '$.%s') = ?", k))
			args = append(args, v)
		}
		if len(conditions) > 0 {
			whereClause = strings.Join(conditions, " AND ")
		}
	}

	// Fetch all matching documents (with filters applied)
	querySQL := fmt.Sprintf(`
		SELECT id, text, embedding, metadata
		FROM %s
		WHERE %s
	`, s.tableName, whereClause)

	rows, err := s.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	// Load all documents and calculate similarities
	type docWithEmbedding struct {
		id        string
		text      string
		embedding []float32
		metadata  map[string]any
		score     float32
	}

	documents := []docWithEmbedding{}
	for rows.Next() {
		var id, text string
		var embeddingBlob []byte
		var metadataJSON string

		if err := rows.Scan(&id, &text, &embeddingBlob, &metadataJSON); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		// Deserialize embedding
		embedding, err := deserializeEmbedding(embeddingBlob)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize embedding: %w", err)
		}

		// Deserialize metadata
		var metadata map[string]any
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			metadata = make(map[string]any)
		}

		// Calculate cosine similarity
		similarity, err := cosineSimilarity(queryEmbedding, embedding)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate similarity: %w", err)
		}

		documents = append(documents, docWithEmbedding{
			id:        id,
			text:      text,
			embedding: embedding,
			metadata:  metadata,
			score:     similarity,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating documents: %w", err)
	}

	// Sort by similarity (descending) and take top K
	sort.Slice(documents, func(i, j int) bool {
		return documents[i].score > documents[j].score
	})

	// Take top K results
	if topK > len(documents) {
		topK = len(documents)
	}

	results := make([]core.SearchResult, 0, topK)
	for i := 0; i < topK; i++ {
		results = append(results, core.SearchResult{
			DocumentID: documents[i].id,
			Text:       documents[i].text,
			Metadata:   documents[i].metadata,
			Score:      documents[i].score,
		})
	}

	return results, nil
}

// ListDocuments retrieves documents with pagination and optional filters.
// Implements core.VectorDB interface.
func (s *SQLiteDB) ListDocuments(opts core.ListOptions) ([]core.DocumentSummary, int, error) {
	ctx := context.Background()

	// Build WHERE clause for filters
	whereClause := "1=1"
	args := []interface{}{}
	if len(opts.Filters) > 0 {
		conditions := []string{}
		for k, v := range opts.Filters {
			conditions = append(conditions, fmt.Sprintf("json_extract(metadata, '$.%s') = ?", k))
			args = append(args, v)
		}
		if len(conditions) > 0 {
			whereClause = strings.Join(conditions, " AND ")
		}
	}

	// Get total count
	countSQL := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s WHERE %s
	`, s.tableName, whereClause)

	var totalCount int
	err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count documents: %w", err)
	}

	// Set default limit if not specified
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}

	// Build query with pagination
	querySQL := fmt.Sprintf(`
		SELECT id, text, metadata
		FROM %s
		WHERE %s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, s.tableName, whereClause)

	args = append(args, limit, opts.Offset)
	rows, err := s.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query documents: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	summaries := make([]core.DocumentSummary, 0)
	for rows.Next() {
		var id, text, metadataJSON string

		if err := rows.Scan(&id, &text, &metadataJSON); err != nil {
			return nil, 0, fmt.Errorf("failed to scan document: %w", err)
		}

		// Deserialize metadata
		var metadata map[string]any
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			metadata = make(map[string]any)
		}

		summaries = append(summaries, core.DocumentSummary{
			DocumentID: id,
			Text:       text,
			Metadata:   metadata,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating documents: %w", err)
	}

	return summaries, totalCount, nil
}

// Delete removes a document by ID.
// Implements core.VectorDB interface.
func (s *SQLiteDB) Delete(documentID string) error {
	ctx := context.Background()

	deleteSQL := fmt.Sprintf(`
		DELETE FROM %s WHERE id = ?
	`, s.tableName)

	result, err := s.db.ExecContext(ctx, deleteSQL, documentID)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("document with ID %s not found", documentID)
	}

	return nil
}

// Close closes the SQLite database connection.
func (s *SQLiteDB) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
