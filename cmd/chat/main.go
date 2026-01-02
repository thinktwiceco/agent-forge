package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/thinktwice/agentForge/src/agents"
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

	// Create agent configuration with reasoning enabled
	config := agents.AgentConfig{
		LLMEngine:   llmEngine,
		AgentName:   "Assistant",
		Description: "A helpful assistant with reasoning capabilities",
		Trace:       "response",
		Reasoning:   true, // Enable reasoning agent
		CanExpand:   true,
		SystemPrompt: `You are a testing agent
		You will receive requests in order to test your sub agents and tool usage.
		Report at the best of your ability.
		`,
		MainAgent:   true,
		Persistence: "json",
	}

	// Create the agent
	agent := agents.NewAgent(&config)

	sandboxPath := "/home/verte/Desktop/thinktwice-agent/"

	agent.AddSystemAgent(agents.OsAgent(llmEngine, sandboxPath))

	return agent, nil
}

// processResponse sends a message to the agent and displays the response with colored output
func processResponse(agent *agents.Agent, message string) error {
	// Get response channel
	responseCh := agent.ChatStream(message)

	// Track which agents we've seen
	currentAgent := ""
	currentTrace := ""

	// Process streaming response
	// No casting needed - Start() returns the concrete channel type
	for chunk := range responseCh.Start() {
		// Check for errors
		if chunk.Status == llms.StatusError {
			return fmt.Errorf("agent error: %s", chunk.Content)
		}

		// Determine color based on agent name/trace
		color := getColorForAgent(chunk.AgentName, chunk.Trace)

		// Print agent header if agent changed
		if chunk.AgentName != currentAgent || chunk.Trace != currentTrace {
			currentAgent = chunk.AgentName
			currentTrace = chunk.Trace

			// Print agent name header
			agentLabel := formatAgentLabel(chunk.AgentName, chunk.Trace)
			fmt.Printf("\n%s%s%s%s\n", ColorBold, color, agentLabel, ColorReset)
		}

		// Handle different chunk types
		switch chunk.Type {
		case llms.TypeContent:
			// Stream content as it arrives
			if chunk.Content != "" || chunk.Delta != "" {
				content := chunk.Content
				if content == "" {
					content = chunk.Delta
				}
				fmt.Printf("%s%s%s", color, content, ColorReset)
			}

		case llms.TypeCompletion:
			// Final completion - display token usage if available
			if chunk.TotalTokens > 0 {
				fmt.Printf("\n%s%s📊 Tokens: %d prompt + %d completion = %d total%s\n",
					ColorBlue, ColorDim,
					chunk.PromptTokens, chunk.CompletionTokens, chunk.TotalTokens,
					ColorReset)
			}

		case llms.TypeToolExecuting:
			// Show tool execution (suppress for delegate tool when delegating to sub-agents)
			if chunk.ToolExecuting != nil && chunk.ToolExecuting.Name != "delegate" {
				fmt.Printf("\n%s%s⚙️  Executing tool: %s%s\n", ColorMagenta, ColorBold, chunk.ToolExecuting.Name, ColorReset)
			}

		case llms.TypeToolResult:
			// Show tool results (suppress for delegate tool when delegating to sub-agents)
			if len(chunk.ToolResults) > 0 {
				for _, result := range chunk.ToolResults {
					if result.ToolName == "delegate" {
						// Skip verbose delegate tool completion messages for sub-agents
						continue
					}
					if result.Success {
						fmt.Printf("%s%s✓ Tool completed: %s%s\n", ColorGreen, ColorBold, result.ToolName, ColorReset)
					} else {
						fmt.Printf("%s%s✗ Tool failed: %s - %s%s\n", ColorRed, ColorBold, result.ToolName, result.Error, ColorReset)
					}
				}
			}
		}
	}

	fmt.Println() // Final newline
	return nil
}

// getColorForAgent returns the appropriate color code based on agent name and trace
func getColorForAgent(agentName, trace string) string {
	// Check if this is the reasoning agent
	if strings.Contains(strings.ToLower(agentName), "reasoning") ||
		strings.Contains(strings.ToLower(trace), "thinking") ||
		strings.Contains(strings.ToLower(trace), "reasoning") {
		return ColorYellow
	}

	// Check if this is a sub-agent (not the main "Assistant" agent)
	if agentName != "Assistant" && !strings.Contains(strings.ToLower(agentName), "reasoning") {
		// Use dim yellow for sub-agents to make them subtle like reasoning traces
		return ColorDim + ColorYellow
	}

	// Default to cyan for main agent
	return ColorCyan
}

// formatAgentLabel creates a formatted label for the agent
func formatAgentLabel(agentName, trace string) string {
	// Check if this is a sub-agent (not the main "Assistant" agent)
	isSubAgent := agentName != "Assistant" && !strings.Contains(strings.ToLower(agentName), "reasoning")

	// Add emoji based on agent type
	emoji := "💬"
	if strings.Contains(strings.ToLower(agentName), "reasoning") ||
		strings.Contains(strings.ToLower(trace), "thinking") {
		emoji = "🧠"
	} else if isSubAgent {
		// Use a subtle emoji for sub-agents
		emoji = "→"
	}

	// For sub-agents, use a more subtle format
	if isSubAgent {
		if trace != "" {
			return fmt.Sprintf("%s %s", emoji, trace)
		}
		return fmt.Sprintf("%s %s", emoji, agentName)
	}

	// Main agent format
	if trace != "" {
		return fmt.Sprintf("%s %s - %s", emoji, agentName, trace)
	}
	return fmt.Sprintf("%s %s", emoji, agentName)
}
