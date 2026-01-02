package agents

import (
	"encoding/json"
	"fmt"

	"github.com/thinktwice/agentForge/src/llms"
)

// ===== Core Chat Execution =====

// executeChatWithTools executes the chat loop with automatic tool execution.
// It handles streaming responses, tool call detection, execution, and iteration.
func (a *Agent) executeChatWithTools() error {
	iteration := 0

	for iteration < a.config.MaxToolIterations {
		iteration++
		messages := a.history.History()
		// Call LLM with current history and tools
		llmResponseCh := (*a.llmEngine).ChatStream(messages, a.tools)

		var fullContent string
		var toolCalls []llms.ToolCall
		var hasToolCalls bool
		var completedChunkBytes []byte // Store completed chunk to forward later if needed
		var promptTokens, completionTokens, totalTokens int

		// Process streaming response
		for {
			select {
			case chunkBytes, ok := <-llmResponseCh.Response:
				if !ok {
					// LLM response channel closed, streaming complete
					goto processToolCalls
				}

				// Deserialize chunk
				var chunk llms.ChunkResponse
				if err := json.Unmarshal(chunkBytes, &chunk); err != nil {
					return fmt.Errorf("failed to deserialize chunk: %w", err)
				}

				// Accumulate content (check both Content and Delta)
				if chunk.Content != "" {
					fullContent += chunk.Content
				} else if chunk.Delta != "" {
					fullContent += chunk.Delta
				}

				// Check for tool calls
				if chunk.Status == llms.StatusToolCall && len(chunk.ToolCalls) > 0 {
					toolCalls = chunk.ToolCalls
					hasToolCalls = true
				}

				// If this is a completed chunk, store it but don't forward yet
				// We need to check if there are tool calls to execute first
				if chunk.Status == llms.StatusCompleted {
					completedChunkBytes = chunkBytes
					// Extract token usage from completed chunk
					promptTokens = chunk.PromptTokens
					completionTokens = chunk.CompletionTokens
					totalTokens = chunk.TotalTokens
					goto processToolCalls
				}

				// Forward all other chunks to consumer
				a.responseCh.Response <- chunkBytes

			case err := <-llmResponseCh.Error:
				if err != nil {
					return fmt.Errorf("llm stream error: %w", err)
				}
				goto processToolCalls
			}
		}

	processToolCalls:
		// If no tool calls, forward the completed chunk (if any) and we're done
		if !hasToolCalls {
			if completedChunkBytes != nil {
				a.responseCh.Response <- completedChunkBytes
			} else if fullContent != "" {
				// Stream ended without StatusCompleted chunk, but we have content
				// Send a completion chunk with accumulated content
				completionChunk := llms.ChunkResponse{
					Content:          "",
					Delta:            "",
					FullContent:      fullContent,
					Status:           llms.StatusCompleted,
					Type:             llms.TypeCompletion,
					PromptTokens:     promptTokens,
					CompletionTokens: completionTokens,
					TotalTokens:      totalTokens,
				}
				completionBytes, err := json.Marshal(completionChunk)
				if err == nil {
					a.responseCh.Response <- completionBytes
				}
			}
			// Save the message to history with token usage
			if fullContent != "" {
				a.history.addAssistantMessage(fullContent, promptTokens, completionTokens, totalTokens)
				a.history.save()
			}
			return nil
		}

		// Store assistant message with tool calls in history with token usage
		a.history.addAssistantMessageWithToolCalls(fullContent, toolCalls, promptTokens, completionTokens, totalTokens)
		a.history.save()

		// Execute each tool
		for _, toolCall := range toolCalls {
			// Emit tool-executing chunk
			executingChunk := llms.ChunkResponse{
				Status:        llms.StatusToolExecuting,
				Type:          llms.TypeToolExecuting,
				ToolExecuting: &toolCall,
			}
			executingBytes, err := json.Marshal(executingChunk)
			if err != nil {
				return fmt.Errorf("failed to serialize tool-executing chunk: %w", err)
			}
			a.responseCh.Response <- executingBytes

			// Find and execute the tool
			toolResult := a.executeTool(toolCall)

			// Emit tool-result chunk
			resultChunk := llms.ChunkResponse{
				Status:      llms.StatusToolResult,
				Type:        llms.TypeToolResult,
				ToolResults: []llms.ToolResult{toolResult},
			}
			resultBytes, err := json.Marshal(resultChunk)
			if err != nil {
				return fmt.Errorf("failed to serialize tool-result chunk: %w", err)
			}
			a.responseCh.Response <- resultBytes

			// Add tool result to history
			a.history.addToolMessage(toolCall.ID, toolResult.Result, toolResult.Ephemeral)
			a.history.save()
		}

		// Continue to next iteration (will call LLM again with tool results)
	}

	// If we reached max iterations, return error
	return fmt.Errorf("reached maximum tool iterations (%d)", a.config.MaxToolIterations)
}

// executeTool finds and executes a tool by name.
func (a *Agent) executeTool(toolCall llms.ToolCall) llms.ToolResult {
	// Build agent context from pre-built context struct
	agentContext := a.agentContext.BuildContext(a.responseCh)

	// Find the tool
	var tool llms.Tool
	for _, t := range a.tools {
		if t.GetName() == toolCall.Name {
			tool = t
			break
		}
	}

	if tool == nil {
		return llms.ToolResult{
			ToolCallID: toolCall.ID,
			ToolName:   toolCall.Name,
			Success:    false,
			Result:     "",
			Error:      fmt.Sprintf("tool not found: %s", toolCall.Name),
		}
	}

	errs := a.hooks.beforeToolExecutionEvent(a, &toolCall)
	logHookErrors(errs)

	// Execute the tool
	result := tool.Call(agentContext, toolCall.Arguments)

	errs = a.hooks.toolExecutionEvent(a, &result)
	logHookErrors(errs)

	// Convert to ToolResult
	return llms.ToolResult{
		ToolCallID: toolCall.ID,
		ToolName:   toolCall.Name,
		Success:    result.Success(),
		Result:     result.Data(),
		Error:      result.Error(),
		Ephemeral:  result.Ephemeral(),
	}
}
