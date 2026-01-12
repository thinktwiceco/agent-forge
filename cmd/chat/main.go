package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/thinktwice/agentForge/src/agents"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/plugins/logger"
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

	codebasePath := "/home/verte/Desktop/thinktwice-agent/cmd/chat"

	// Create logger plugin with default rules and stdout output
	loggerPlugin := logger.NewPlugin(logger.DefaultColorRules(), logger.DefaultLabelRules(), os.Stdout)

	// Create agent configuration with reasoning enabled
	config := agents.AgentConfig{
		LLMEngine:   llmEngine,
		AgentName:   "Assistant",
		Description: "A helpful assistant with reasoning capabilities",
		Trace:       agents.TraceResponse,
		Reasoning:   true, // Enable reasoning agent
		CanExpand:   true,
		SystemPrompt: `You are a testing agent
		You will receive requests in order to test your sub agents and tool usage.
		Report at the best of your ability.
		`,
		MainAgent:   true,
		Persistence: "json",
		Plugins:     []core.Plugin{loggerPlugin},
	}

	// Create the agent
	agent := agents.NewAgent(&config)

	agent.AddSystemAgent(agents.OsAgent(llmEngine, codebasePath))

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
