package agents

import (
	"fmt"
	"testing"

	"github.com/thinktwice/agentForge/src/llms"
)

// TestAgent_Reasoning_TwoTraces tests that when reasoning is enabled,
// the agent produces chunks with two different traces:
// - The main agent's trace
// - The reasoning agent's "reasoning" trace
//
// This test will fail until the delegate tool is implemented, because
// the main agent won't be able to delegate to the reasoning agent.
func TestAgent_Reasoning_TwoTraces(t *testing.T) {

	llm, err := llms.NewOpenAILLMBuilder("togetherai").
		SetModel(llms.TOGETHERAI_Llama3170BInstructTurbo).
		Build()
	if err != nil {
		t.Skipf("Skipping reasoning test: %v", err)
	}

	// Create agent with reasoning enabled
	agent := NewAgent(AgentConfig{
		LLMEngine: llm,
		AgentName: "main agent",
		Trace:     "main-trace",
		Reasoning: true,
	})

	// Send a message that should trigger reasoning
	responseCh := agent.ChatStream("Can you tell me what is the surface of a trapezoid with the following sides: 10, 20, 10, 20 and height 10?")

	// Track which traces we observe
	observedTraces := make(map[string]bool)

	fmt.Println("\n=== Streaming Response ===")
	// No casting needed - Start() returns the concrete channel type
	for chunk := range responseCh.Start() {
		// Record the trace
		if chunk.Trace != "" {
			observedTraces[chunk.Trace] = true
		}

		// Print chunk info
		if chunk.Content != "" {
			fmt.Print(chunk.Content)
		}

		// Print tool calls if any
		if chunk.Status == llms.StatusToolCall && len(chunk.ToolCalls) > 0 {
			fmt.Printf("  ✓ Tool call: name=%s, args=%+v\n",
				chunk.ToolCalls[0].Name,
				chunk.ToolCalls[0].Arguments)
		}
	}

	// Print summary of observed traces
	fmt.Println("\n=== Observed Traces ===")
	for trace := range observedTraces {
		fmt.Printf("✓ %s\n", trace)
	}

	// Verify we saw both traces
	if !observedTraces["main-trace"] {
		t.Error("Expected to see main agent trace 'main-trace'")
	}

	if !observedTraces["reasoning"] {
		t.Error("Expected to see reasoning agent trace 'reasoning'")
		fmt.Println("\nℹ️  This is expected to fail until the delegate tool is implemented.")
		fmt.Println("   The main agent needs a delegate tool to forward tasks to the reasoning agent.")
	}

	// Log summary
	fmt.Println("\n=== Test Summary ===")
	fmt.Printf("✓ Saw main-trace: %v\n", observedTraces["main-trace"])
	fmt.Printf("✓ Saw reasoning trace: %v\n", observedTraces["reasoning"])
	fmt.Printf("Total unique traces observed: %d\n", len(observedTraces))
}
