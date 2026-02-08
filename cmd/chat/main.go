package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/builder"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/integrations"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/tools/api"
)

const (
	// ANSI color codes
	ColorReset   = "\033[0m"
	ColorCyan    = "\033[36m" // Main agent color
	ColorYellow  = "\033[33m" // Reasoning agent color
	ColorRed     = "\033[31m" // Error color
	ColorGreen   = "\033[32m" // User input color
	ColorBlue    = "\033[34m" // Info color
	ColorMagenta = "\033[35m" // Tool execution color
	ColorBold    = "\033[1m"
	ColorDim     = "\033[2m"
)

func main() {
	// Parse command-line flags
	flag.Parse()

	printBanner()

	// Display provider information
	fmt.Printf("%sType 'exit' or 'quit' to end the conversation%s\n\n", ColorDim, ColorReset)

	// Initialize the agent
	fmt.Printf("%s%sInitializing agent...%s\n", ColorDim, ColorBlue, ColorReset)
	agent, err := initializeAgent()
	if err != nil {
		fmt.Printf("%sError initializing agent: %v%s\n", ColorRed, err, ColorReset)
		os.Exit(1)
	}

	// Start chat loop
	scanner := bufio.NewScanner(os.Stdin)
	var chatId string // Store chatId to retain conversation history
	for {
		// Get user input
		fmt.Printf("%s%sYou: %s", ColorGreen, ColorBold, ColorReset)
		_ = os.Stdout.Sync() // Ensure prompt is displayed immediately
		if !scanner.Scan() {
			break
		}

		userInput := strings.TrimSpace(scanner.Text())
		if userInput == "" {
			continue
		}

		// Check for exit commands
		if strings.ToLower(userInput) == "exit" || strings.ToLower(userInput) == "quit" {
			fmt.Printf("\n%sGoodbye!%s\n", ColorBold, ColorReset)
			break
		}

		// Send message and process response
		fmt.Println() // Add newline for better formatting
		fmt.Printf("%s%sAgent is thinking... (chatId: %s)%s\n", ColorDim, ColorBlue, chatId, ColorReset)
		if newChatId, err := processResponse(agent, userInput, chatId); err != nil {
			fmt.Printf("%sError: %v%s\n", ColorRed, err, ColorReset)
		} else {
			// Update chatId for next iteration to retain history
			if newChatId != chatId && newChatId != "" {
				fmt.Printf("%s%s  → ChatId updated: %s%s\n", ColorDim, ColorBlue, newChatId, ColorReset)
			}
			chatId = newChatId
		}
		fmt.Println() // Add newline after response
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("%sError reading input: %v%s\n", ColorRed, err, ColorReset)
	}
}

// printBanner displays the CLI banner
func printBanner() {
	banner := `
╔════════════════════════════════════════════╗
║     🤖 ThinkTwice Agent CLI 🤖             ║
╚════════════════════════════════════════════╝
`
	fmt.Printf("%s%s%s\n", ColorBold, ColorCyan, banner)
	fmt.Print(ColorReset)
}

// initializeVectorComponents initializes SQLite vector DB and embedding generator for vector operations.
// Returns the vectorDB, embeddingGenerator, and an error if initialization fails.
func initializeVectorComponents() (core.VectorDB, core.EmbeddingGenerator, error) {
	// Initialize SQLite for vector database
	// Note: VectorDim should match your embedding model dimension (e.g., 1536 for text-embedding-3-small)
	sqliteConfig := integrations.SQLiteConfig{
		DBPath:    "./vector.db", // Store in current directory
		VectorDim: 1536,          // text-embedding-3-small dimension
	}
	sqliteDB, err := integrations.NewSQLiteDB(sqliteConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize SQLite vector DB: %w", err)
	}

	// Initialize OpenAI embedding generator for vector tool
	embeddingGenerator, err := integrations.NewOpenAIEmbeddingGenerator("", "text-embedding-3-small")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize embedding generator: %w", err)
	}

	return sqliteDB, embeddingGenerator, nil
}

// initializeAgent creates and configures the agent with the specified provider
func initializeAgent() (*agents.Agent, error) {

	// Initialize vector database components (SQLite-based, fast initialization)
	fmt.Printf("%s%s  → Initializing vector components (SQLite)...%s\n", ColorDim, ColorBlue, ColorReset)
	var vectorDB core.VectorDB
	var embeddingGenerator core.EmbeddingGenerator
	var err error

	vectorDB, embeddingGenerator, err = initializeVectorComponents()
	if err != nil {
		fmt.Printf("%s%s  ⚠ Vector components initialization failed: %v%s\n", ColorYellow, ColorBold, err, ColorReset)
		fmt.Printf("%s%s  → Continuing without vector DB subagent%s\n", ColorDim, ColorBlue, ColorReset)
	} else {
		fmt.Printf("%s%s  ✓ Vector components initialized%s\n", ColorDim, ColorGreen, ColorReset)
	}

	fmt.Printf("%s%s  → Building agent...%s\n", ColorDim, ColorBlue, ColorReset)
	b := builder.NewAgentBuilder("Test Agent", "json")
	b.SetWorkingDir("/home/verte/Desktop/sandbox")
	b.AddTools(
		builder.Tool{Name: builder.FILE_SYSTEM_TOOL, Root: "/home/verte/Desktop/sandbox"},
		builder.Tool{Name: builder.GIT_TOOL, Root: "/home/verte/Desktop/sandbox"},
		builder.Tool{
			Name:           builder.POSTGRES_TOOL,
			PostgresURL:    "postgresql://testuser:testpass@localhost:5432/testdb?sslmode=disable",
			Mode:           "write",
			AllowedTables:  []string{"users", "products", "orders"},
			AllowedSchemas: []string{"public"},
		},
	)
	b.SetModel(fmt.Sprintf("deepseek::%s", llms.DEEPSEEK_CHAT))

	// Only add vector components if initialized
	if embeddingGenerator != nil && vectorDB != nil {
		b.SetEmbeddingGenerator(embeddingGenerator)
		b.SetVectorDB(vectorDB)
	}

	b.AddSubagent(builder.REASONING_AGENT, fmt.Sprintf("deepseek::%s", llms.DEEPSEEK_REASONING))

	// Only add vector DB subagent if vector components are available
	if vectorDB != nil && embeddingGenerator != nil {
		b.AddSubagent(builder.VECTOR_DB_AGENT, fmt.Sprintf("togetherai::%s", llms.TOGETHERAI_ZaiGLM47))
	}

	b.AddSubagent(builder.WEB_AGENT, fmt.Sprintf("deepseek::%s", llms.DEEPSEEK_CHAT))
	b.AddPlugin(builder.LOGGER_PLUGIN)
	b.AddPlugin(builder.TODO_PLUGIN)

	fmt.Printf("%s%s  → Building agent (this may take a moment)...%s\n", ColorDim, ColorBlue, ColorReset)
	agent, err := b.Build()
	if err != nil {
		return nil, err
	}

	// Configure and add Pokemon API tool
	fmt.Printf("%s%s  → Adding Pokemon API tool...%s\n", ColorDim, ColorBlue, ColorReset)
	pokemonEndpoints := []api.Endpoint{
		{
			Name:          "get_pokemon",
			URL:           "https://pokeapi.co/api/v2/pokemon/{name}",
			Method:        "GET",
			Description:   "Get detailed information about a specific Pokemon by name or ID",
			URLParameters: `- name: string - The name or ID of the Pokemon (e.g., "pikachu", "25")`,
		},
		{
			Name:        "list_pokemon",
			URL:         "https://pokeapi.co/api/v2/pokemon",
			Method:      "GET",
			Description: "List all Pokemon with pagination support",
			QueryParams: `- limit: int - Number of results to return (default: 20, max: 100)
- offset: int - Number of results to skip for pagination (default: 0)`,
		},
	}

	// No authentication needed for PokeAPI, so we pass nil for the hook
	pokemonTool := api.NewApiTool("pokemon_api", pokemonEndpoints, nil)
	agent.AddTools([]llms.Tool{pokemonTool})

	fmt.Printf("%s%s  ✓ Agent initialized successfully with Pokemon API%s\n", ColorDim, ColorGreen, ColorReset)
	return agent, nil
}

// processResponse sends a message to the agent and processes the response
// All chunks are formatted by the logger plugin hook when read from the channel
// Returns the chatId (which may be newly generated) and any error
func processResponse(agent *agents.Agent, message string, chatId string) (string, error) {
	// Get response channel with chatId to maintain conversation history
	responseCh := agent.ChatStream(message, chatId)

	// Process streaming response with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	chunkChan := responseCh.Start()
	receivedAnyChunk := false

	for {
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				// Channel closed, streaming complete
				if !receivedAnyChunk {
					return chatId, fmt.Errorf("no response received from agent - channel closed without any chunks")
				}
				fmt.Println() // Final newline
				// Get the final chatId from the response channel (may have been generated during save)
				finalChatId := responseCh.GetChatId()
				return finalChatId, nil
			}

			receivedAnyChunk = true

			// Check for errors
			if chunk.Status == llms.StatusError {
				return chatId, fmt.Errorf("agent error: %s", chunk.Content)
			}

			// Hook is triggered automatically when chunks are read from the channel
			// All formatting is handled by the logger plugin hook

		case <-ctx.Done():
			// Timeout reached
			responseCh.Close() // Close the response channel to stop the goroutine
			return chatId, fmt.Errorf("timeout waiting for agent response after 5 minutes")
		}
	}
}
