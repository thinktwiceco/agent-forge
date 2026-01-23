package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/thinktwice/agentForge/src/agents"
	"github.com/thinktwice/agentForge/src/builder"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/integrations"
	"github.com/thinktwice/agentForge/src/llms"
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
	agent, err := initializeAgent()
	if err != nil {
		fmt.Printf("%sError initializing agent: %v%s\n", ColorRed, err, ColorReset)
		os.Exit(1)
	}

	// Start chat loop
	scanner := bufio.NewScanner(os.Stdin)
	for {
		// Get user input
		fmt.Printf("%s%sYou: %s", ColorGreen, ColorBold, ColorReset)
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
		if err := processResponse(agent, userInput); err != nil {
			fmt.Printf("%sError: %v%s\n", ColorRed, err, ColorReset)
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

// initializeVectorComponents initializes Milvus and embedding generator for vector operations.
// Returns the vectorDB, embeddingGenerator, and an error if initialization fails.
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

// initializeAgent creates and configures the agent with the specified provider
func initializeAgent() (*agents.Agent, error) {

	// Initialize vector database components
	var vectorDB core.VectorDB
	var embeddingGenerator core.EmbeddingGenerator
	vectorDB, embeddingGenerator, _ = initializeVectorComponents()

	b := builder.NewAgentBuilder("Test Agent", "json")
	b.AddTools(
		builder.FILE_SYSTEM_TOOL,
		builder.GIT_TOOL,
	)
	b.SetModel(fmt.Sprintf("openai::%s", llms.OPENAI_GPT5_2))
	b.SetWorkingDir("/home/verte/Desktop/sandbox")
	b.SetEmbeddingGenerator(embeddingGenerator)
	b.SetVectorDB(vectorDB)
	b.AddSubagent(builder.REASONING_AGENT, fmt.Sprintf("deepseek::%s", llms.DEEPSEEK_REASONING))
	b.AddSubagent(builder.VECTOR_DB_AGENT, fmt.Sprintf("togetherai::%s", llms.TOGETHERAI_ZaiGLM47))
	b.AddSubagent(builder.WEB_AGENT, fmt.Sprintf("deepseek::%s", llms.DEEPSEEK_CHAT))
	b.AddPlugin(builder.LOGGER_PLUGIN)
	b.AddPlugin(builder.TODO_PLUGIN)

	return b.Build()
}

// processResponse sends a message to the agent and processes the response
// All chunks are formatted by the logger plugin hook when read from the channel
func processResponse(agent *agents.Agent, message string) error {
	// Get response channel
	responseCh := agent.ChatStream(message)

	// Process streaming response
	for chunk := range responseCh.Start() {
		// Check for errors
		if chunk.Status == llms.StatusError {
			return fmt.Errorf("agent error: %s", chunk.Content)
		}

		// Hook is triggered automatically when chunks are read from the channel
		// All formatting is handled by the logger plugin hook
	}

	fmt.Println() // Final newline
	return nil
}
