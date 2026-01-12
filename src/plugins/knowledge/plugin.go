package knowledge

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/agents"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// Plugin represents a knowledge plugin that stores and searches documents using Qdrant and vector embeddings
type KnowledgePlugin struct {
	knowledgeIdentifier string // Something that the agent can use to identify the knowledge plugin
	documentPaths       []string
	dbPath              string
	vectorDB            *vectorDB
	embeddingService    *embeddingService
	qdrantDocker        *qdrantDocker // Docker container manager for Qdrant
	chunkSize           int
	chunkOverlap        int
	loadCtx             context.Context
	loadCancel          context.CancelFunc
	maxWorkers          int // Maximum number of concurrent workers
	batchSize           int // Number of chunks to batch insert
}

// DocumentChunk represents a chunk of a document with its embedding
// This is exported for use by consumers of the plugin
type DocumentChunk struct {
	ID           int64
	DocumentPath string
	Content      string
	Embedding    []float32
	ChunkIndex   int
}

// NewPlugin creates a new knowledge plugin
func NewPlugin(documentPaths []string, dbPath string, knowledgeIdentifier string) (*KnowledgePlugin, error) {
	if dbPath == "" {
		dbPath = "./knowledge.db"
	}

	// Create a context for initialization
	ctx := context.Background()

	// Ensure Qdrant Docker container is running
	qdrantDocker, err := ensureQdrantRunning(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure Qdrant is running: %w", err)
	}

	// Initialize vector database
	vectorDB, err := newVectorDB(dbPath)
	if err != nil {
		// Clean up Docker container if we started it
		if qdrantDocker.containerStarted {
			qdrantDocker.stopContainer()
		}
		return nil, fmt.Errorf("failed to initialize vector database: %w", err)
	}

	// Initialize embedding service (open source, local)
	embeddingService, err := newEmbeddingService()

	if err != nil {
		vectorDB.close()
		return nil, fmt.Errorf("failed to initialize embedding service: %w", err)
	}

	// Create a context that cancels on SIGTERM or SIGINT
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		agentforge.Info("Knowledge Plugin: Received termination signal, cancelling document loading")
		cancel()
	}()

	return &KnowledgePlugin{
		documentPaths:       documentPaths,
		knowledgeIdentifier: knowledgeIdentifier,
		dbPath:              dbPath,
		vectorDB:            vectorDB,
		embeddingService:    embeddingService,
		qdrantDocker:        qdrantDocker,
		chunkSize:           1000, // default chunk size
		chunkOverlap:        200,  // default overlap
		loadCtx:             ctx,
		loadCancel:          cancel,
		maxWorkers:          1,  // Process sequentially to prevent system overload
		batchSize:           20, // Smaller batch size for better memory management
	}, nil
}

/// Plugin Interface Implementations ///

// Name implements the core.Plugin interface
func (p *KnowledgePlugin) Name() string {
	return "knowledge"
}

// On implements the core.Plugin interface
func (p *KnowledgePlugin) On(event core.Event) core.AgentHookFn {
	switch event {
	case core.EventAgentInitialization:
		// Cast the function
		fn := agents.OnAgentInitializationHook(p.handleAgentInitialization)
		return fn
	case core.EventAgentInitialized:
		fn := agents.OnAgentInitializedHook(p.handleAgentInitialized)
		return fn
	}
	return nil
}

func (p *KnowledgePlugin) Tools() []llms.Tool {
	return []llms.Tool{NewSearchTool(p)}
}

/// ========================================///

// Hooks returned by the plugin
func (p *KnowledgePlugin) handleAgentInitialization(a *agents.Agent, config *agents.AgentConfig) error {
	// Load documents synchronously, one at a time
	// Add panic recovery to prevent crashes
	defer func() {
		if r := recover(); r != nil {
			agentforge.Error("Knowledge Plugin: Panic during document loading: %v", r)
		}
	}()

	if err := p.loadDocuments(); err != nil {
		if err == context.Canceled {
			agentforge.Info("Knowledge Plugin: Document loading cancelled due to termination signal")
		} else {
			agentforge.Error("Knowledge Plugin: Failed to load documents: %v", err)
		}
		return err
	}

	agentforge.Info("Knowledge Plugin: Documents loaded successfully")
	return nil
}

func (p *KnowledgePlugin) handleAgentInitialized(a *agents.Agent) error {
	agentforge.Info("Knowledge Plugin: Agent %s initialized", a.Name())
	return nil
}

// maxFileSize limits the size of files we'll process (50MB)
const maxFileSize = 50 * 1024 * 1024

// isTextFile checks if a file should be processed based on its extension
func isTextFile(path string, dbPath string) bool {
	// Skip the database file itself
	if path == dbPath {
		return false
	}

	ext := filepath.Ext(path)
	ext = strings.ToLower(ext)

	// Exclude database files
	dbExtensions := []string{".db", ".sqlite", ".sqlite3", ".db-shm", ".db-wal", ".db-journal"}
	for _, dbExt := range dbExtensions {
		if ext == dbExt {
			return false
		}
	}

	// Include common text file extensions
	textExtensions := []string{
		".txt", ".md", ".markdown", ".rst",
		".py", ".go", ".js", ".ts", ".jsx", ".tsx", ".java", ".c", ".cpp", ".h", ".hpp",
		".cs", ".rb", ".php", ".swift", ".kt", ".scala", ".rs", ".r", ".m", ".mm",
		".sh", ".bash", ".zsh", ".fish", ".ps1", ".bat", ".cmd",
		".html", ".htm", ".xml", ".json", ".yaml", ".yml", ".toml", ".ini", ".cfg", ".conf",
		".css", ".scss", ".sass", ".less",
		".sql", ".pl", ".pm", ".lua", ".vim", ".el",
		".log", ".csv", ".tsv",
		".tex", ".latex", ".bib",
		".org", ".wiki",
		".dockerfile", ".makefile",
		".cmake", ".gradle", ".maven",
		".proto", ".thrift", ".avdl",
		".graphql", ".gql",
		".vue", ".svelte",
		".dart", ".elm",
		".clj", ".cljs", ".cljc", ".edn",
		".ex", ".exs", ".heex",
		".fs", ".fsx", ".fsi", ".fsproj",
		".ml", ".mli", ".mll", ".mly",
		".hs", ".lhs",
		".nim", ".cr", ".d",
		".zig", ".v", ".odin",
		".pas", ".pp", ".lpr",
		".ada", ".adb", ".ads",
		".f90", ".f95", ".f03", ".f08",
		".cob", ".cbl",
		".asm", ".s", ".S",
		".ll", ".bc",
		".rkt", ".scm", ".ss",
		".jl",
		".ex", ".exs",
		".erl", ".hrl",
		".pro", ".pl",
		".tcl", ".tk",
		".lisp", ".lsp",
		".cl", ".lisp",
		".moon", ".lua",
		".pike", ".pmod",
		".io",
		".nu",
		".raku", ".rakumod", ".rakutest", ".rakudoc",
		".red", ".reds",
		".rexx", ".rex",
		".ring",
		".sas",
		".st",
		".tcl",
		".xtend",
		".zep",
		".zig",
		".zsh",
	}

	for _, textExt := range textExtensions {
		if ext == textExt {
			return true
		}
	}

	// If no extension or unknown extension, check if it's likely a text file
	// by checking if it's not a binary file (we'll do a simple heuristic)
	if ext == "" {
		// Files without extensions might be text files (like README, LICENSE, etc.)
		// We'll include them but they'll be checked during reading
		return true
	}

	// Exclude binary files by default
	return false
}

// LoadDocuments loads documents, chunks them, generates embeddings, and stores them in the database
func (p *KnowledgePlugin) loadDocuments() error {
	ctx := p.loadCtx

	// Collect all file paths (handling directories)
	var filePaths []string
	for _, path := range p.documentPaths {
		// Check for cancellation before processing each path
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fileInfo, err := os.Stat(path)
		if err != nil {
			agentforge.Error("Failed to stat path %s: %v", path, err)
			continue
		}

		if fileInfo.IsDir() {
			// Recursively walk directory to find all files
			err := filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				// Check for cancellation during directory walk
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				if !info.IsDir() {
					// Skip non-text files (e.g., database files)
					if !isTextFile(walkPath, p.dbPath) {
						agentforge.Debug("Skipping non-text file: %s", walkPath)
						return nil
					}
					// Check file size before adding
					if info.Size() > maxFileSize {
						agentforge.Warn("Skipping large file %s (%d bytes, max %d)", walkPath, info.Size(), maxFileSize)
						return nil
					}
					filePaths = append(filePaths, walkPath)
				}
				return nil
			})
			if err != nil {
				if err == context.Canceled {
					return err
				}
				agentforge.Error("Failed to walk directory %s: %v", path, err)
				continue
			}
		} else {
			// Skip non-text files (e.g., database files)
			if !isTextFile(path, p.dbPath) {
				agentforge.Debug("Skipping non-text file: %s", path)
				continue
			}
			// Check file size before adding
			if fileInfo.Size() > maxFileSize {
				agentforge.Warn("Skipping large file %s (%d bytes, max %d)", path, fileInfo.Size(), maxFileSize)
				continue
			}
			// It's a file, add it directly
			filePaths = append(filePaths, path)
		}
	}

	if len(filePaths) == 0 {
		agentforge.Info("Knowledge Plugin: No files found to process")
		return nil
	}

	agentforge.Info("Knowledge Plugin: Found %d files to process", len(filePaths))

	// Process files sequentially to prevent system overload
	processedCount := 0

	for i, path := range filePaths {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Process file with panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					agentforge.Error("Knowledge Plugin: Panic processing file %s: %v", path, r)
				}
			}()

			agentforge.Info("Knowledge Plugin: Processing document %d/%d: %s", i+1, len(filePaths), path)

			// Read document with size check
			doc, err := readFileWithLimit(path, maxFileSize)
			if err != nil {
				agentforge.Error("Failed to read document %s: %v", path, err)
				return
			}

			// Chunk the document
			chunks := chunkText(string(doc), p.chunkSize, p.chunkOverlap)
			// Clear doc from memory immediately after chunking
			doc = nil

			if len(chunks) == 0 {
				return
			}

			// Process chunks in small batches and insert immediately to avoid memory accumulation
			const embeddingBatchSize = 5 // Smaller batches for embeddings
			const insertBatchSize = 10   // Insert every 10 chunks to reduce memory

			var batch []chunkToInsert
			batch = make([]chunkToInsert, 0, insertBatchSize) // Pre-allocate with capacity

			for batchStart := 0; batchStart < len(chunks); batchStart += embeddingBatchSize {
				// Check for cancellation
				select {
				case <-ctx.Done():
					// Flush batch before exiting
					if len(batch) > 0 {
						if err := p.vectorDB.insertChunksBatch(ctx, batch); err != nil {
							agentforge.Error("Failed to insert batch: %v", err)
						}
					}
					return
				default:
				}

				batchEnd := batchStart + embeddingBatchSize
				if batchEnd > len(chunks) {
					batchEnd = len(chunks)
				}

				chunkBatch := chunks[batchStart:batchEnd]

				// Generate embeddings for this batch
				embeddings, err := p.embeddingService.generateEmbeddings(ctx, chunkBatch)
				if err != nil {
					if err == context.Canceled {
						return
					}
					agentforge.Error("Failed to generate embeddings for %s (batch %d-%d): %v", path, batchStart, batchEnd, err)
					continue
				}

				// Add chunks to batch
				for j, chunk := range chunkBatch {
					if j < len(embeddings) {
						batch = append(batch, chunkToInsert{
							docPath:    path,
							content:    chunk,
							embedding:  embeddings[j],
							chunkIndex: batchStart + j,
						})
					}
				}

				// Insert batch immediately when it reaches the size limit
				if len(batch) >= insertBatchSize {
					if err := p.vectorDB.insertChunksBatch(ctx, batch); err != nil {
						agentforge.Error("Failed to insert batch: %v", err)
					} else {
						agentforge.Debug("Inserted batch of %d chunks", len(batch))
					}
					// Clear batch and reset capacity to help GC
					batch = batch[:0]
					batch = make([]chunkToInsert, 0, insertBatchSize)
				}

				// Clear embeddings from memory
				embeddings = nil
				chunkBatch = nil

				// Delay to prevent CPU overload
				time.Sleep(50 * time.Millisecond)
			}

			// Flush remaining batch for this file
			if len(batch) > 0 {
				if err := p.vectorDB.insertChunksBatch(ctx, batch); err != nil {
					agentforge.Error("Failed to insert final batch for %s: %v", path, err)
				} else {
					agentforge.Debug("Inserted final batch of %d chunks for %s", len(batch), path)
				}
				batch = nil // Clear batch
			}

			// Clear chunks array to free memory immediately
			chunks = nil

			processedCount++
			agentforge.Info("Knowledge Plugin: Completed document %d/%d: %s", processedCount, len(filePaths), path)

			// Small delay between files
			time.Sleep(100 * time.Millisecond)
		}()
	}

	agentforge.Info("Knowledge Plugin: Finished processing %d documents", processedCount)
	return nil
}

// Search searches for similar content in the knowledge base
func (p *KnowledgePlugin) search(query string, limit int) ([]DocumentChunk, error) {
	ctx := context.Background()

	// Generate embedding for query
	queryEmbedding, err := p.embeddingService.generateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search for similar chunks
	chunks, err := p.vectorDB.searchSimilar(ctx, queryEmbedding, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search database: %w", err)
	}

	// Convert internal chunks to exported chunks
	result := make([]DocumentChunk, len(chunks))
	for i, chunk := range chunks {
		result[i] = DocumentChunk{
			ID:           int64(chunk.id),
			DocumentPath: chunk.documentPath,
			Content:      chunk.content,
			Embedding:    chunk.embedding,
			ChunkIndex:   chunk.chunkIndex,
		}
	}

	return result, nil
}

// Close closes the database connection and embedding service
// Note: The Qdrant Docker container is left running by default to preserve data.
// To stop it manually, use: docker stop thinktwice-qdrant
func (p *KnowledgePlugin) close() error {
	// Cancel the document loading context if it's still running
	if p.loadCancel != nil {
		p.loadCancel()
	}

	var errs []error
	if p.embeddingService != nil {
		if err := p.embeddingService.close(); err != nil {
			errs = append(errs, err)
		}
	}
	if p.vectorDB != nil {
		if err := p.vectorDB.close(); err != nil {
			errs = append(errs, err)
		}
	}
	// Note: We don't stop the Docker container here to preserve data
	// The container will persist between plugin restarts
	if len(errs) > 0 {
		return fmt.Errorf("errors closing resources: %v", errs)
	}
	return nil
}
