package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/history"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/telemetry"
)

// Ensure Executor implements agents.ExecutionEngine interface at compile time
// This is a forward declaration - the actual interface check is done
// in the agents package to avoid circular dependencies

// Config holds execution configuration.
type Config struct {
	MaxToolIterations int
	AgentName         string
	Tracer            telemetry.Tracer
}

// Executor handles tool and chat execution.
type Executor struct {
	llmEngine    llms.LLMEngine
	tools        []llms.Tool
	agentContext *core.AgentContext
	config       Config
	hooks        *HooksRunner
}

// UpdateTools updates the tools slice. Call when tools are added or modified (e.g. AddTools, loadDelegateTool).
func (e *Executor) UpdateTools(tools []llms.Tool) {
	e.tools = tools
}

// UpdateAgentContext updates the agent context. Call when context is rebuilt (e.g. initAgentContext).
func (e *Executor) UpdateAgentContext(agentContext *core.AgentContext) {
	e.agentContext = agentContext
}

// NewExecutor creates a new Executor.
func NewExecutor(
	llmEngine llms.LLMEngine,
	tools []llms.Tool,
	agentContext *core.AgentContext,
	config Config,
	hooks *HooksRunner,
) *Executor {
	return &Executor{
		llmEngine:    llmEngine,
		tools:        tools,
		agentContext: agentContext,
		config:       config,
		hooks:        hooks,
	}
}

// ExecuteChatWithTools executes the chat loop with automatic tool execution.
func (e *Executor) ExecuteChatWithTools(ctx context.Context, hm history.Manager, responseCh *core.ResponseCh) (err error) {
	start := time.Now()
	if e.config.Tracer != nil {
		e.config.Tracer.TraceAgentStart(ctx, e.config.AgentName)
	}
	defer func() {
		if e.config.Tracer != nil {
			e.config.Tracer.TraceAgentComplete(ctx, e.config.AgentName, time.Since(start), err)
		}
	}()

	// Per-call snapshot: mutable fields (SessionStorage, PluginFields, LastSubagentMessage)
	// are isolated to this call so concurrent ChatStream invocations cannot bleed state.
	callCtx := e.agentContext.Snapshot()

	iteration := 0
	var cleanupFuncs []func()

	for iteration < e.config.MaxToolIterations {
		select {
		case <-ctx.Done():
			agentforge.Debug("Context cancelled, stopping execution")
			return fmt.Errorf("execution cancelled: %w", ctx.Err())
		default:
		}

		iteration++
		messages := hm.Messages()
		agentforge.Debug("Starting iteration %d with %d messages in history", iteration, len(messages))

		llmResponseCh := e.llmEngine.ChatStream(messages, e.tools)

		var fullContent string
		var fullReasoningContent string
		var toolCalls []llms.ToolCall
		var hasToolCalls bool
		var completedChunkBytes []byte
		var promptTokens, completionTokens, totalTokens int

		llmErrorCh := llmResponseCh.Error

		agentforge.Debug("Waiting for LLM response chunks...")
		for {
			select {
			case <-ctx.Done():
				agentforge.Debug("Context cancelled during LLM response streaming")
				return fmt.Errorf("execution cancelled: %w", ctx.Err())
			case chunkBytes, ok := <-llmResponseCh.Response:
				if !ok {
					agentforge.Debug("LLM response channel closed, processing tool calls")
					goto processToolCalls
				}

				agentforge.Debug("Received chunk from LLM (size: %d bytes)", len(chunkBytes))
				var chunk llms.ChunkResponse
				if err := json.Unmarshal(chunkBytes, &chunk); err != nil {
					agentforge.Debug("Error deserializing chunk: %v", err)
					return fmt.Errorf("failed to deserialize chunk: %w", err)
				}
				agentforge.Debug("Chunk deserialized: Status=%s, Type=%s", chunk.Status, chunk.Type)

				if chunk.Content != "" {
					fullContent += chunk.Content
				} else if chunk.Delta != "" {
					fullContent += chunk.Delta
				}

				chunk.Iteration = iteration

				chunkBytes, err := json.Marshal(chunk)
				if err != nil {
					agentforge.Debug("Error re-serializing chunk with iteration: %v", err)
					return fmt.Errorf("failed to re-serialize chunk: %w", err)
				}

				if chunk.Status == llms.StatusToolCall && len(chunk.ToolCalls) > 0 {
					toolCalls = chunk.ToolCalls
					hasToolCalls = true
					fullReasoningContent = chunk.ReasoningContent
					agentforge.Debug("Tool calls detected: %d tool calls", len(toolCalls))
					if !responseCh.TrySend(chunkBytes) {
						return nil
					}
				}

				if chunk.Status == llms.StatusCompleted {
					completedChunkBytes = chunkBytes
					promptTokens = chunk.PromptTokens
					completionTokens = chunk.CompletionTokens
					totalTokens = chunk.TotalTokens
					if chunk.FullContent != "" {
						fullContent = chunk.FullContent
					}
					agentforge.Debug("Received completed chunk, going to processToolCalls")
					goto processToolCalls
				}

				if !responseCh.TrySend(chunkBytes) {
					return nil
				}

			case err, ok := <-llmErrorCh:
				if !ok {
					llmErrorCh = nil
					continue
				}
				if err != nil {
					agentforge.Debug("LLM stream error received: %v", err)
					return fmt.Errorf("llm stream error: %w", err)
				}
			}
		}

	processToolCalls:
		agentforge.Debug("Processing tool calls: hasToolCalls=%v, toolCalls count=%d", hasToolCalls, len(toolCalls))

		if !hasToolCalls {
			if completedChunkBytes == nil {
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
					Iteration:        iteration,
				}
				var err error
				completedChunkBytes, err = json.Marshal(completionChunk)
				if err != nil {
					return fmt.Errorf("failed to marshal completion chunk: %w", err)
				}
			}

			if !responseCh.TrySend(completedChunkBytes) {
				return nil
			}

			if fullContent == "" {
				agentforge.Debug("LLM returned empty completion; skipping history save")
				return nil
			}

			hm.AddAssistantMessage(fullContent, history.TokenUsage{
				PromptTokens:     promptTokens,
				CompletionTokens: completionTokens,
				TotalTokens:      totalTokens,
			})
			if e.config.Tracer != nil {
				e.config.Tracer.TraceTokenUsage(ctx, telemetry.TokenUsageEvent{
					AgentName:        e.config.AgentName,
					PromptTokens:     promptTokens,
					CompletionTokens: completionTokens,
					TotalTokens:      totalTokens,
					Iteration:        iteration,
					HadToolCalls:     false,
				})
			}
			if e.hooks != nil && e.hooks.OnNewAssistantMessage != nil {
				errs := e.hooks.OnNewAssistantMessage(fullContent, promptTokens, completionTokens, totalTokens)
				if e.hooks.LogHookErrors != nil {
					e.hooks.LogHookErrors(errs)
				}
			}
			return nil
		}

		if completedChunkBytes != nil {
			var completedChunk llms.ChunkResponse
			if err := json.Unmarshal(completedChunkBytes, &completedChunk); err == nil && completedChunk.FullContent != "" {
				fullContent = completedChunk.FullContent
			}
		}
		if e.config.Tracer != nil {
			e.config.Tracer.TraceTokenUsage(ctx, telemetry.TokenUsageEvent{
				AgentName:        e.config.AgentName,
				PromptTokens:     promptTokens,
				CompletionTokens: completionTokens,
				TotalTokens:      totalTokens,
				Iteration:        iteration,
				HadToolCalls:     true,
			})
		}
		if e.hooks != nil && e.hooks.OnNewAssistantMessageWithTools != nil {
			errs := e.hooks.OnNewAssistantMessageWithTools(fullContent, toolCalls, promptTokens, completionTokens, totalTokens)
			if e.hooks.LogHookErrors != nil {
				e.hooks.LogHookErrors(errs)
			}
		}

		hm.AddAssistantMessageWithToolCalls(fullContent, fullReasoningContent, toolCalls, history.TokenUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
		})

		agentforge.Debug("Detected %d tool calls, executing tools", len(toolCalls))
		for _, toolCall := range toolCalls {
			// Deep-copy Arguments so hook mutations (e.g. vault secret decryption)
			// do not corrupt the shared map that the history entry still references.
			if toolCall.Arguments != nil {
				argsCopy := make(map[string]any, len(toolCall.Arguments))
				for k, v := range toolCall.Arguments {
					argsCopy[k] = v
				}
				toolCall.Arguments = argsCopy
			}

			agentforge.Debug("Executing tool: %s (ID: %s)", toolCall.Name, toolCall.ID)
			executingChunk := llms.ChunkResponse{
				Status:        llms.StatusToolExecuting,
				Type:          llms.TypeToolExecuting,
				ToolExecuting: &toolCall,
				Iteration:     iteration,
			}
			executingBytes, err := json.Marshal(executingChunk)
			if err != nil {
				return fmt.Errorf("failed to serialize tool-executing chunk: %w", err)
			}
			if !responseCh.TrySend(executingBytes) {
				return nil
			}

			var hookErrs []error
			if e.hooks != nil && e.hooks.OnBeforeToolExecution != nil {
				hookErrs = e.hooks.OnBeforeToolExecution(&toolCall)
				if e.hooks.LogHookErrors != nil {
					e.hooks.LogHookErrors(hookErrs)
				}
			}

			toolStart := time.Now()
			var toolResult llms.ToolResult
			if len(hookErrs) > 0 {
				msgs := make([]string, len(hookErrs))
				for i, err := range hookErrs {
					msgs[i] = err.Error()
				}
				toolResult = llms.ToolResult{
					ToolCallID: toolCall.ID,
					ToolName:   toolCall.Name,
					Success:    false,
					Error:      strings.Join(msgs, "; "),
				}
			} else {
				toolResult = e.executeToolWithContext(toolCall, responseCh, callCtx)
			}
			if e.config.Tracer != nil {
				e.config.Tracer.TraceToolExecution(ctx, telemetry.ToolExecutionEvent{
					AgentName: e.config.AgentName,
					ToolName:  toolCall.Name,
					Duration:  time.Since(toolStart),
					Success:   toolResult.Success,
					Error:     toolResult.Error,
					Iteration: iteration,
				})
			}

			if toolResult.Cleanup != nil {
				cleanupFuncs = append(cleanupFuncs, toolResult.Cleanup)
			}

			if e.hooks != nil && e.hooks.OnToolExecution != nil {
				errs := e.hooks.OnToolExecution(&toolResult)
				if e.hooks.LogHookErrors != nil {
					e.hooks.LogHookErrors(errs)
				}
			}

			resultChunk := llms.ChunkResponse{
				Status:      llms.StatusToolResult,
				Type:        llms.TypeToolResult,
				ToolResults: []llms.ToolResult{toolResult},
				Iteration:   iteration,
			}
			resultBytes, err := json.Marshal(resultChunk)
			if err != nil {
				return fmt.Errorf("failed to serialize tool-result chunk: %w", err)
			}
			if !responseCh.TrySend(resultBytes) {
				agentforge.Debug("Channel closed when trying to send tool-result chunk for tool %s", toolCall.Name)
				return nil
			}

			content := toolResult.Result
			if !toolResult.Success && toolResult.Error != "" {
				if content != "" {
					content = content + "\nError: " + toolResult.Error
				} else {
					content = "Error: " + toolResult.Error
				}
			}
			if toolResult.Success && strings.HasPrefix(content, "data:") {
				hm.AddToolMessage(toolCall.ID, "[image loaded]", toolResult.Ephemeral)
				hm.AddUserMessageWithImages("", content)
			} else {
				hm.AddToolMessage(toolCall.ID, content, toolResult.Ephemeral)
			}
		}

		agentforge.Debug("Completed iteration %d, continuing to next iteration with tool results in history", iteration)
	}

	if len(cleanupFuncs) > 0 {
		agentforge.Debug("Executing %d cleanup functions", len(cleanupFuncs))
		for i := len(cleanupFuncs) - 1; i >= 0; i-- {
			if cleanupFuncs[i] != nil {
				cleanupFuncs[i]()
			}
		}
	}

	agentforge.Debug("Reached maximum tool iterations (%d)", e.config.MaxToolIterations)
	return fmt.Errorf("reached maximum tool iterations (%d)", e.config.MaxToolIterations)
}

// ExecuteTool finds and executes a tool by name using the shared agentContext.
// Used by test helpers (agentChat.go); production code uses executeToolWithContext.
func (e *Executor) ExecuteTool(toolCall llms.ToolCall, responseCh *core.ResponseCh) llms.ToolResult {
	return e.executeToolWithContext(toolCall, responseCh, e.agentContext)
}

// executeToolWithContext finds and executes a tool using the provided per-call context.
// This keeps mutable state mutations isolated to the callCtx snapshot, preventing
// cross-chat bleed when multiple ChatStream calls run concurrently.
func (e *Executor) executeToolWithContext(toolCall llms.ToolCall, responseCh *core.ResponseCh, callCtx *core.AgentContext) llms.ToolResult {
	if e.hooks != nil && e.hooks.OnContextBuild != nil {
		errs := e.hooks.OnContextBuild(callCtx)
		if e.hooks.LogHookErrors != nil {
			e.hooks.LogHookErrors(errs)
		}
	}

	agentContext := callCtx.BuildContext(responseCh)

	var tool llms.Tool
	for _, t := range e.tools {
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

	result := tool.Call(agentContext, toolCall.Arguments)

	if err := callCtx.SyncFromMap(agentContext); err != nil {
		agentforge.Debug("Warning: failed to sync context after tool execution: %v", err)
	}

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
