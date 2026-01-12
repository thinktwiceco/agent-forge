package knowledge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/qdrant/go-client/qdrant"
	agentforge "github.com/thinktwice/agentForge/src"
)

// vectorDB handles Qdrant database operations for vector storage
type vectorDB struct {
	client     *qdrant.Client
	collection string
	dimensions uint64
}

// documentChunk represents a chunk of a document with its embedding
type documentChunk struct {
	id           uint64
	documentPath string
	content      string
	embedding    []float32
	chunkIndex   int
}

// newVectorDB creates a new vector database connection to Qdrant
func newVectorDB(dbPath string) (*vectorDB, error) {
	// Extract Qdrant connection info from dbPath or use defaults
	// Format: "qdrant://host:port/collection" or just use defaults
	host := "localhost"
	port := 6334
	collection := "knowledge_base"

	// If dbPath is provided and not empty, try to parse it
	if dbPath != "" && dbPath != "./knowledge.db" {
		// Simple parsing: if it contains "://", parse it
		// Otherwise use as collection name
		if filepath.Ext(dbPath) == "" {
			collection = filepath.Base(dbPath)
		}
	}

	// Create Qdrant client config
	config := &qdrant.Config{
		Host: host,
		Port: port,
	}

	// Connect to Qdrant
	client, err := qdrant.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	// Use 384 dimensions to match embedding service
	dimensions := uint64(384)

	vdb := &vectorDB{
		client:     client,
		collection: collection,
		dimensions: dimensions,
	}

	if err := vdb.initCollection(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize collection: %w", err)
	}

	return vdb, nil
}

// initCollection creates the collection if it doesn't exist
func (vdb *vectorDB) initCollection() error {
	ctx := context.Background()

	// Check if collection exists
	exists, err := vdb.client.CollectionExists(ctx, vdb.collection)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	// Create collection if it doesn't exist
	if !exists {
		err := vdb.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: vdb.collection,
			VectorsConfig: &qdrant.VectorsConfig{
				Config: &qdrant.VectorsConfig_Params{
					Params: &qdrant.VectorParams{
						Size:     vdb.dimensions,
						Distance: qdrant.Distance_Cosine,
					},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
		agentforge.Info("Created Qdrant collection: %s", vdb.collection)
	} else {
		agentforge.Debug("Using existing Qdrant collection: %s", vdb.collection)
	}

	return nil
}

// chunkToInsert represents a chunk ready for batch insertion
type chunkToInsert struct {
	docPath    string
	content    string
	embedding  []float32
	chunkIndex int
}

// insertChunk inserts a document chunk with its embedding
func (vdb *vectorDB) insertChunk(ctx context.Context, docPath string, content string, embedding []float32, chunkIndex int) error {
	return vdb.insertChunksBatch(ctx, []chunkToInsert{{
		docPath:    docPath,
		content:    content,
		embedding:  embedding,
		chunkIndex: chunkIndex,
	}})
}

// insertChunksBatch inserts multiple chunks in a single batch for better performance
func (vdb *vectorDB) insertChunksBatch(ctx context.Context, chunks []chunkToInsert) error {
	if len(chunks) == 0 {
		return nil
	}

	points := make([]*qdrant.PointStruct, 0, len(chunks))

	for _, chunk := range chunks {
		// Generate a unique ID for this chunk
		// Using a hash of docPath + chunkIndex for uniqueness
		pointID := hashString(fmt.Sprintf("%s:%d", chunk.docPath, chunk.chunkIndex))

		point := &qdrant.PointStruct{
			Id: &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Num{
					Num: pointID,
				},
			},
			Vectors: &qdrant.Vectors{
				VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{
						Data: chunk.embedding, // Qdrant accepts []float32 directly
					},
				},
			},
			Payload: map[string]*qdrant.Value{
				"document_path": {
					Kind: &qdrant.Value_StringValue{
						StringValue: chunk.docPath,
					},
				},
				"content": {
					Kind: &qdrant.Value_StringValue{
						StringValue: chunk.content,
					},
				},
				"chunk_index": {
					Kind: &qdrant.Value_IntegerValue{
						IntegerValue: int64(chunk.chunkIndex),
					},
				},
			},
		}
		points = append(points, point)
	}

	// Upsert points in batch
	_, err := vdb.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: vdb.collection,
		Points:         points,
		Wait:           qdrant.PtrOf(true),
	})
	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}

	return nil
}

// searchSimilar finds the most similar chunks to the query embedding using Qdrant's efficient HNSW indexing
func (vdb *vectorDB) searchSimilar(ctx context.Context, queryEmbedding []float32, limit int) ([]documentChunk, error) {
	if limit <= 0 {
		limit = 10
	}

	// Perform vector search using the PointsClient directly
	searchResult, err := vdb.client.GetPointsClient().Search(ctx, &qdrant.SearchPoints{
		CollectionName: vdb.collection,
		Vector:         queryEmbedding,
		Limit:          uint64(limit),
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Convert results to documentChunk format
	chunks := make([]documentChunk, 0, len(searchResult.Result))
	for _, result := range searchResult.Result {
		chunk := documentChunk{
			id: result.Id.GetNum(),
		}

		// Extract payload
		if payload := result.Payload; payload != nil {
			if docPath, ok := payload["document_path"]; ok {
				chunk.documentPath = docPath.GetStringValue()
			}
			if content, ok := payload["content"]; ok {
				chunk.content = content.GetStringValue()
			}
			if chunkIdx, ok := payload["chunk_index"]; ok {
				chunk.chunkIndex = int(chunkIdx.GetIntegerValue())
			}
		}

		// Note: Qdrant doesn't return the full vector by default, but we have the score
		// If needed, we can request it with WithVectors option
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// close closes the database connection
func (vdb *vectorDB) close() error {
	if vdb.client != nil {
		return vdb.client.Close()
	}
	return nil
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(dir string) error {
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}

// hashString creates a simple hash from a string for use as a point ID
func hashString(s string) uint64 {
	var h uint64 = 5381
	for i := 0; i < len(s); i++ {
		h = ((h << 5) + h) + uint64(s[i])
	}
	return h
}
