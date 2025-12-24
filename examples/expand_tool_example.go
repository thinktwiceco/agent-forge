package main

import (
	"context"
	"fmt"

	"github.com/thinktwice/agentForge/src/agents"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/tools"
)

// This example demonstrates the Expand tool, which allows agents to
// progressively discover detailed information about tools and sub-agents.

func main() {
	fmt.Println("=== Expand Tool Example ===\n")

	// Note: This example requires a valid LLM API key
	// Set TOGETHERAI_API_KEY environment variable to run

	ctx := context.Background()
	llm, err := llms.GetTogetherAILLM(ctx, llms.Qwen257BInstructTurbo)
	if err != nil {
		fmt.Printf("⚠️  Skipping example - LLM API not available: %v\n", err)
		fmt.Println("\nTo run this example, set TOGETHERAI_API_KEY environment variable")
		demonstrateExpandToolDirectly()
		return
	}

	// Create an agent with multiple tools including the expand tool
	agent := agents.NewAgent(agents.AgentConfig{
		LLMEngine:   llm,
		AgentName:   "explorer-agent",
		Description: "An agent that can discover information about its tools",
		SystemPrompt: `You are an explorer agent. You have access to several tools.
When asked about a tool's capabilities, use the expand tool to get detailed information.`,
		Tools: []llms.Tool{
			tools.NewFooTool(),
			tools.NewExpandTool(),
		},
		MainAgent: true,
	})

	// Ask the agent to expand information about the foo tool
	fmt.Println("Asking agent to expand information about the 'foo' tool...\n")

	responseCh := agent.ChatStream("Use the expand tool to get detailed information about the 'foo' tool, including troubleshooting.")

	fmt.Println("=== Agent Response ===")
	for chunk := range responseCh.Start() {
		if chunk.Content != "" {
			fmt.Print(chunk.Content)
		}

		if chunk.Status == llms.StatusToolCall && len(chunk.ToolCalls) > 0 {
			fmt.Printf("\n[Tool Call: %s]\n", chunk.ToolCalls[0].Name)
		}

		if chunk.Status == llms.StatusToolResult && len(chunk.ToolResults) > 0 {
			fmt.Printf("\n[Tool Result]:\n%s\n", chunk.ToolResults[0].Result)
		}
	}

	fmt.Println("\n\n=== Example Complete ===")
}

// demonstrateExpandToolDirectly shows how the expand tool works without an LLM
func demonstrateExpandToolDirectly() {
	fmt.Println("\n=== Direct Expand Tool Demonstration ===\n")

	// Create tools
	fooTool := tools.NewFooTool()
	expandTool := tools.NewExpandTool()

	// Simulate agent context
	agentContext := map[string]any{
		"tools": []llms.Tool{fooTool, expandTool},
	}

	fmt.Println("1. Expanding 'foo' tool without troubleshooting:")
	fmt.Println("   expand(subject_type='tool', subject_name='foo', troubleshoot=false)\n")

	result1 := expandTool.Call(agentContext, map[string]any{
		"subject_type": "tool",
		"subject_name": "foo",
		"troubleshoot": false,
	})

	if result1.Success() {
		fmt.Println(result1.Data())
	} else {
		fmt.Printf("Error: %s\n", result1.Error())
	}

	fmt.Println("\n" + repeat("=", 60) + "\n")

	fmt.Println("2. Expanding 'foo' tool WITH troubleshooting:")
	fmt.Println("   expand(subject_type='tool', subject_name='foo', troubleshoot=true)\n")

	result2 := expandTool.Call(agentContext, map[string]any{
		"subject_type": "tool",
		"subject_name": "foo",
		"troubleshoot": true,
	})

	if result2.Success() {
		fmt.Println(result2.Data())
	} else {
		fmt.Printf("Error: %s\n", result2.Error())
	}

	fmt.Println("\n" + repeat("=", 60) + "\n")

	fmt.Println("3. Expanding 'expand' tool itself (meta!):")
	fmt.Println("   expand(subject_type='tool', subject_name='expand', troubleshoot=true)\n")

	result3 := expandTool.Call(agentContext, map[string]any{
		"subject_type": "tool",
		"subject_name": "expand",
		"troubleshoot": true,
	})

	if result3.Success() {
		fmt.Println(result3.Data())
	} else {
		fmt.Printf("Error: %s\n", result3.Error())
	}

	fmt.Println("\n=== Key Features ===")
	fmt.Println("✓ Progressive discovery: Start with basic info, expand when needed")
	fmt.Println("✓ Troubleshooting on demand: Only include when necessary")
	fmt.Println("✓ Works with both tools and agents")
	fmt.Println("✓ Self-documenting: Can expand itself!")
}

// Helper to repeat strings (Go doesn't have strings.Repeat in older versions)
func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
