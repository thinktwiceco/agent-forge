package agents

import (
	"encoding/json"
	"fmt"

	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/llms"
)

// ===== Core Chat Execution =====

// executeChatWithTools executes the chat loop with automatic tool execution.
// It handles streaming responses, tool call detection, execution, and iteration.
func (a *Agent) executeChatWithTools() error {
	iteration := 0
	var cleanupFuncs []func() // Accumulate cleanup functions from all tool executions

	for iteration < a.config.MaxToolIterations {
		iteration++
		messages := a.history.History()
		agentforge.Debug("Starting iteration %d with %d messages in history", iteration, len(messages))
		// Call LLM with current history and tools
		llmResponseCh := a.llmEngine.ChatStream(messages, a.tools)

		var fullContent string
		var toolCalls []llms.ToolCall
		var hasToolCalls bool
		var completedChunkBytes []byte // Store completed chunk to forward later if needed
		var promptTokens, completionTokens, totalTokens int

		// Process streaming response
		agentforge.Debug("Waiting for LLM response chunks...")
		for {
			select {
			case chunkBytes, ok := <-llmResponseCh.Response:
				if !ok {
					// LLM response channel closed, streaming complete
					agentforge.Debug("LLM response channel closed, processing tool calls")
					goto processToolCalls
				}

				agentforge.Debug("Received chunk from LLM (size: %d bytes)", len(chunkBytes))
				// Deserialize chunk
				var chunk llms.ChunkResponse
				if err := json.Unmarshal(chunkBytes, &chunk); err != nil {
					agentforge.Debug("Error deserializing chunk: %v", err)
					return fmt.Errorf("failed to deserialize chunk: %w", err)
				}
				agentforge.Debug("Chunk deserialized: Status=%s, Type=%s", chunk.Status, chunk.Type)

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
					agentforge.Debug("Tool calls detected: %d tool calls", len(toolCalls))
					// Forward tool-call chunk to consumer
					if !a.responseCh.TrySend(chunkBytes) {
						// Channel closed, stop processing
						return nil
					}
					// Don't continue here - let the loop continue naturally
					// The completed chunk should arrive next, or the channel will close
				}

				// If this is a completed chunk, store it but don't forward yet
				// We need to check if there are tool calls to execute first
				if chunk.Status == llms.StatusCompleted {
					completedChunkBytes = chunkBytes
					// Extract token usage and full content from completed chunk
					promptTokens = chunk.PromptTokens
					completionTokens = chunk.CompletionTokens
					totalTokens = chunk.TotalTokens
					// Use FullContent from the completed chunk if available, otherwise use accumulated content
					if chunk.FullContent != "" {
						fullContent = chunk.FullContent
					}
					agentforge.Debug("Received completed chunk, going to processToolCalls")
					goto processToolCalls
				}

				// Forward all other chunks to consumer
				// Hook will be called when chunks are read from the channel
				if !a.responseCh.TrySend(chunkBytes) {
					// Channel closed, stop processing
					return nil
				}

			case err := <-llmResponseCh.Error:
				if err != nil {
					agentforge.Debug("LLM stream error received: %v", err)
					return fmt.Errorf("llm stream error: %w", err)
				}
				// Error channel closed without error, continue processing
				agentforge.Debug("LLM stream error channel closed (no error), going to processToolCalls")
				goto processToolCalls
			}
		}

	processToolCalls:
		agentforge.Debug("Processing tool calls: hasToolCalls=%v, toolCalls count=%d", hasToolCalls, len(toolCalls))
		// If no tool calls, forward the completed chunk (if any) and we're done
		if !hasToolCalls {
			// Ensure we have a completion chunk to forward
			if completedChunkBytes == nil {
				// Stream ended without StatusCompleted chunk - this should never happen
				// Create a completion chunk with accumulated content
				if fullContent == "" {
					return fmt.Errorf("LLM stream ended without content and without StatusCompleted chunk")
				}
				agentforge.Debug("WARNING: Stream ended without StatusCompleted chunk, creating one")
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
				var err error
				completedChunkBytes, err = json.Marshal(completionChunk)
				if err != nil {
					return fmt.Errorf("failed to marshal completion chunk: %v", err)
				}
			}

			// Forward completed chunk to consumer
			// Hook will be called when chunks are read from the channel
			if !a.responseCh.TrySend(completedChunkBytes) {
				// Channel closed, stop processing
				return nil
			}

			// Save the message to history with token usage
			if fullContent == "" {
				return fmt.Errorf("attempting to save empty message to history")
			}

			a.history.addAssistantMessage(fullContent, promptTokens, completionTokens, totalTokens)
			a.hooks.newAssistantMessageEvent(a, fullContent, promptTokens, completionTokens, totalTokens)
			return nil
		}

		// Extract FullContent from completed chunk if available
		if completedChunkBytes != nil {
			var completedChunk llms.ChunkResponse
			if err := json.Unmarshal(completedChunkBytes, &completedChunk); err == nil && completedChunk.FullContent != "" {
				fullContent = completedChunk.FullContent
			}
		}
		a.hooks.newAssistantMessageWithToolCallsEvent(a, fullContent, toolCalls, promptTokens, completionTokens, totalTokens)

		a.history.addAssistantMessageWithToolCalls(fullContent, toolCalls, promptTokens, completionTokens, totalTokens)

		agentforge.Debug("Detected %d tool calls, executing tools", len(toolCalls))
		// Execute each tool
		for _, toolCall := range toolCalls {
			agentforge.Debug("Executing tool: %s (ID: %s)", toolCall.Name, toolCall.ID)
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
			// Forward tool-executing chunk to consumer
			// Hook will be called when chunks are read from the channel
			if !a.responseCh.TrySend(executingBytes) {
				// Channel closed, stop processing
				return nil
			}

			errs := a.hooks.beforeToolExecutionEvent(a, &toolCall)
			logHookErrors(errs)

			// Find and execute the tool
			toolResult := a.executeTool(toolCall)

			// Accumulate cleanup function if present
			if toolResult.Cleanup != nil {
				cleanupFuncs = append(cleanupFuncs, toolResult.Cleanup)
			}

			errs = a.hooks.toolExecutionEvent(a, &toolResult)
			logHookErrors(errs)

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
			// Forward tool-result chunk to consumer
			// Hook will be called when chunks are read from the channel
			if !a.responseCh.TrySend(resultBytes) {
				// Channel closed, stop processing
				agentforge.Debug("Channel closed when trying to send tool-result chunk for tool %s", toolCall.Name)
				return nil
			}

			// Add tool result to history
			a.history.addToolMessage(toolCall.ID, toolResult.Result, toolResult.Ephemeral)
		}

		// Continue to next iteration (will call LLM again with tool results)
		agentforge.Debug("Completed iteration %d, continuing to next iteration with tool results in history", iteration)
	}

	// Execute all accumulated cleanup functions in reverse order (LIFO)
	if len(cleanupFuncs) > 0 {
		agentforge.Debug("Executing %d cleanup functions", len(cleanupFuncs))
		for i := len(cleanupFuncs) - 1; i >= 0; i-- {
			if cleanupFuncs[i] != nil {
				cleanupFuncs[i]()
			}
		}
	}

	// If we reached max iterations, return error
	agentforge.Debug("Reached maximum tool iterations (%d)", a.config.MaxToolIterations)
	return fmt.Errorf("reached maximum tool iterations (%d)", a.config.MaxToolIterations)
}

// executeTool finds and executes a tool by name.
func (a *Agent) executeTool(toolCall llms.ToolCall) llms.ToolResult {
	// Call contextBuildEvent hook to allow plugins to modify context before building
	errs := a.hooks.contextBuildEvent(a, a.agentContext)
	logHookErrors(errs)

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

	// Execute the tool
	result := tool.Call(agentContext, toolCall.Arguments)

	// Sync changes from context map back to struct to ensure persistence
	// This ensures mutable fields (e.g., LastSubagentMessage, PluginFields) persist across tool calls
	if err := a.agentContext.SyncFromMap(agentContext); err != nil {
		agentforge.Debug("Warning: failed to sync context after tool execution: %v", err)
	}

	// Convert to ToolResult
	return llms.ToolResult{
		ToolCallID: toolCall.ID,
		ToolName:   toolCall.Name,
		Success:    result.Success(),
		Result:     result.Data(),
		Error:      result.Error(),
		Ephemeral:  result.Ephemeral(),
		Cleanup:    result.Cleanup(),
	}
}
