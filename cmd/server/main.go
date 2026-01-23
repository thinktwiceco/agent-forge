package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/thinktwice/agentForge/src/apis"
	"github.com/thinktwice/agentForge/src/builder"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/integrations"
	"github.com/thinktwice/agentForge/src/llms"
)

func main() {
	// Parse command-line flags
	port := flag.String("port", "8080", "Port to listen on")
	flag.Parse()

	fmt.Println("Starting ThinkTwice Agent API Server...")

	// Initialize vector components (optional - can fail silently)
	vectorDB, embeddingGenerator, _ := initializeVectorComponents()

	// Create server
	server := apis.NewServer()
	if vectorDB != nil && embeddingGenerator != nil {
		server.SetVectorComponents(vectorDB, embeddingGenerator)
	}

	// Initialize and register agents
	if err := initializeAgents(server); err != nil {
		fmt.Printf("Error initializing agents: %v\n", err)
		os.Exit(1)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		fmt.Printf("Server listening on port %s\n", *port)
		fmt.Printf("API endpoint: http://localhost:%s/api/server/{agentname}/chat\n", *port)
		if err := server.Start(*port); err != nil {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	fmt.Println("\nShutting down server...")
	if err := server.Shutdown(); err != nil {
		fmt.Printf("Error shutting down server: %v\n", err)
	}
	fmt.Println("Server stopped")
}

// initializeVectorComponents initializes Milvus and embedding generator for vector operations.
// Returns the vectorDB, embeddingGenerator, and an error if initialization fails.
// Errors are ignored to allow the server to run without vector components.
func initializeVectorComponents() (core.VectorDB, core.EmbeddingGenerator, error) {
	// Initialize Milvus for vector database
	// Note: VectorDim should match your embedding model dimension (e.g., 1536 for text-embedding-3-small)
	milvusConfig := integrations.MilvusConfig{
		Host:           "localhost",
		Port:           19530, // Default Milvus port
		CollectionName: "agent_knowledge",
		DefaultTopK:    10,
		VectorDim:      1536, // text-embedding-3-small dimension
		// Username and Password are optional if Milvus doesn't require authentication
	}
	milvusDB, err := integrations.NewMilvusDB(milvusConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize Milvus: %w", err)
	}

	// Initialize OpenAI embedding generator for vector tool
	embeddingGenerator, err := integrations.NewOpenAIEmbeddingGenerator("", "text-embedding-3-small")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize embedding generator: %w", err)
	}

	return milvusDB, embeddingGenerator, nil
}

// initializeAgents creates and registers agents with the server.
// This is similar to cmd/chat/main.go's initializeAgent() function.
func initializeAgents(server *apis.Server) error {
	// Initialize a default agent similar to cmd/chat/main.go
	agentName := "test-agent"
	internalName := "Test Agent"
	model := fmt.Sprintf("openai::%s", llms.OPENAI_GPT5_2)
	workingDir := "/home/verte/Desktop/sandbox"

	tools := []builder.Tool{
		builder.FILE_SYSTEM_TOOL,
		builder.GIT_TOOL,
	}

	subagents := map[builder.Subagent]string{
		builder.REASONING_AGENT: fmt.Sprintf("deepseek::%s", llms.DEEPSEEK_REASONING),
		builder.VECTOR_DB_AGENT: fmt.Sprintf("togetherai::%s", llms.TOGETHERAI_ZaiGLM47),
		builder.WEB_AGENT:       fmt.Sprintf("deepseek::%s", llms.DEEPSEEK_CHAT),
	}

	plugins := []builder.Plugin{
		builder.LOGGER_PLUGIN,
		builder.TODO_PLUGIN,
	}

	if err := server.InitializeAgent(
		agentName,
		internalName,
		model,
		workingDir,
		tools,
		subagents,
		plugins,
	); err != nil {
		return fmt.Errorf("failed to initialize agent '%s': %w", agentName, err)
	}

	fmt.Printf("Registered agent: %s\n", agentName)
	return nil
}
