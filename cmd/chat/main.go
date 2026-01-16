package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/agents"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/integrations"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/plugins/logger"
	"github.com/thinktwice/agentForge/src/plugins/todo"
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
	provider := flag.String("provider", "togetherai", "LLM provider to use: togetherai, openai, or deepseek")
	flag.Parse()

	printBanner()

	// Display provider information
	providerName := "TogetherAI and Llama"
	if *provider == "openai" {
		providerName = "OpenAI"
	} else if *provider == "deepseek" {
		providerName = "DeepSeek"
	}
	fmt.Printf("Chat with a reasoning agent powered by %s\n", providerName)
	fmt.Printf("%sType 'exit' or 'quit' to end the conversation%s\n\n", ColorDim, ColorReset)

	// Initialize the agent
	agent, err := initializeAgent(*provider)
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

func onTodoUpdate(todos []*todo.TodoItem) {
	fmt.Println("======== TODO =========")
	for _, item := range todos {
		completed := "✅"
		if !item.Completed {
			completed = "⬜"
		}
		fmt.Printf("%s %s: %s\n", completed, item.Title, item.Description)
	}
	fmt.Println("======== END TODO =========")
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
func initializeAgent(provider string) (*agents.Agent, error) {

	var llmEngine llms.LLMEngine
	var err error

	// Create LLM engine based on provider
	switch strings.ToLower(provider) {
	case "openai":
		llmEngine, err = llms.NewOpenAILLMBuilder("openai").
			SetModel(llms.OPENAI_GPT5_1).
			Build()
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenAI LLM: %w", err)
		}
	case "togetherai":
		llmEngine, err = llms.NewOpenAILLMBuilder("togetherai").
			SetModel(llms.TOGETHERAI_Llama3170BInstructTurbo).
			Build()
		if err != nil {
			return nil, fmt.Errorf("failed to create TogetherAI LLM: %w", err)
		}
	case "deepseek":
		llmEngine, err = llms.NewOpenAILLMBuilder("deepseek").
			SetModel(llms.DEEPSEEK_CHAT).
			Build()
		if err != nil {
			return nil, fmt.Errorf("failed to create DeepSeek LLM: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported provider: %s (supported: togetherai, openai, deepseek)", provider)
	}

	codebasePath := "/home/verte/Desktop/thinktwice-agent"

	// Create logger plugin with default rules and stdout output
	loggerPlugin := logger.NewPlugin(logger.DefaultColorRules(), logger.DefaultLabelRules(), os.Stdout)
	todoPlugin := todo.NewTodoPlugin(onTodoUpdate)

	tools := []llms.Tool{}
	// Initialize vector database components
	var vectorDB core.VectorDB
	var embeddingGenerator core.EmbeddingGenerator
	vectorDB, embeddingGenerator, err = initializeVectorComponents()
	if err != nil {
		agentforge.Warn("Failed to initialize vector components: %v", err)
		agentforge.Info("Application will continue without vector database capabilities. " +
			"Vector operations will not be available.")
	}

	// Create agent configuration
	config := agents.AgentConfig{
		LLMEngine:   llmEngine,
		AgentName:   "Assistant",
		Description: "A helpful assistant with reasoning capabilities",
		Tone:        "keep-it-short",
		Trace:       agents.TraceResponse,
		CanExpand:   true,
		SystemPrompt: `You are a testing agent
		You will receive requests in order to test your sub agents and tool usage.
		Report at the best of your ability.
		`,
		MainAgent:   true,
		Persistence: "json",
		Plugins:     []core.Plugin{loggerPlugin, todoPlugin},
		Tools:       tools,
	}

	// Create the agent
	agent := agents.NewAgent(&config)

	codingLLMEngine, err := llms.NewOpenAILLMBuilder("togetherai").
		SetModel(llms.TOGETHERAI_Qwen3Coder480B).
		Build()

	if err != nil {
		return nil, fmt.Errorf("failed to create coding LLM: %w", err)
	}

	// Add system agents
	agent.AddSystemAgent(agents.GitAgent(llmEngine, codebasePath))
	agent.AddSystemAgent(agents.ReasoningAgent(llmEngine))
	agent.AddSystemAgent(agents.OsAgent(llmEngine, codebasePath))
	agent.AddSystemAgent(agents.CodingAgent(codingLLMEngine, codebasePath))

	// Add vector agent if vector components were initialized successfully
	if vectorDB != nil && embeddingGenerator != nil {
		agent.AddSystemAgent(agents.VectorAgent(llmEngine, vectorDB, embeddingGenerator))
		agentforge.Info("Vector agent initialized successfully")
	}

	return agent, nil
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
