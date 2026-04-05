package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/heartbeatack"
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
	// HeartbeatAckMaxChars mirrors heartbeat plugin ack_max_chars (0 = default 300).
	HeartbeatAckMaxChars int
	// TruncateHistory is called before each LLM call to keep accumulated history
	// within the model's context window. Nil means no mid-turn truncation.
	TruncateHistory func([]*llms.UnifiedMessage) []*llms.UnifiedMessage
}

// ExecuteResult reports execution outcomes for one ChatStream turn.
type ExecuteResult struct {
	HeartbeatAckSuppressed bool
}

// Executor handles tool and chat execution.
type Executor struct {
	llmEngine    llms.LLMEngine
	tools        []llms.Tool
	agentContext *core.AgentContext
	config       Config
	hooks        *HooksRunner
}

// UpdateTools updates the tools slice. Call when tools are added or modified (e.g. AddTools).
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
func (e *Executor) ExecuteChatWithTools(ctx context.Context, hm history.Manager, responseCh *core.ResponseCh) (result ExecuteResult, err error) {
	start := time.Now()
	if e.config.Tracer != nil {
		e.config.Tracer.TraceAgentStart(ctx, e.config.AgentName)
	}
	defer func() {
		if e.config.Tracer != nil {
			e.config.Tracer.TraceAgentComplete(ctx, e.config.AgentName, time.Since(start), err)
		}
	}()

	// Per-call snapshot: mutable fields (SessionStorage, PluginFields)
	// are isolated to this call so concurrent ChatStream invocations cannot bleed state.
	callCtx := e.agentContext.Snapshot()

	iteration := 0
	var cleanupFuncs []func()

	for iteration < e.config.MaxToolIterations {
		select {
		case <-ctx.Done():
			agentforge.Debug("Context cancelled, stopping execution")
			return result, fmt.Errorf("execution cancelled: %w", ctx.Err())
		default:
		}

		iteration++
		rawMessages := hm.Messages()
		heartbeatTurn := heartbeatack.IsHeartbeatTickUserContent(lastUserContent(rawMessages))
		messages := rawMessages
		if e.config.TruncateHistory != nil {
			messages = e.config.TruncateHistory(messages)
		}
		agentforge.Debug("Starting iteration %d with %d messages in history", iteration, len(messages))

		llmResponseCh := e.llmEngine.ChatStream(messages, e.tools)

		res, err := e.streamLLMResponse(ctx, iteration, heartbeatTurn, llmResponseCh, responseCh)
		if err != nil {
			if err.Error() == "channel closed" {
				return result, nil
			}
			return result, err
		}

		agentforge.Debug("Processing tool calls: hasToolCalls=%v, toolCalls count=%d", res.HasToolCalls, len(res.ToolCalls))

		ackMax := e.config.HeartbeatAckMaxChars
		if ackMax == 0 {
			ackMax = 300
		}

		if !res.HasToolCalls {
			if res.CompletedChunkBytes == nil {
				if res.FullContent == "" {
					return result, fmt.Errorf("LLM stream ended without content and without StatusCompleted chunk")
				}
				agentforge.Debug("WARNING: Stream ended without StatusCompleted chunk, creating one")
				completionChunk := llms.ChunkResponse{
					Content:          "",
					Delta:            "",
					FullContent:      res.FullContent,
					Status:           llms.StatusCompleted,
					Type:             llms.TypeCompletion,
					PromptTokens:     res.PromptTokens,
					CompletionTokens: res.CompletionTokens,
					TotalTokens:      res.TotalTokens,
					Iteration:        iteration,
				}
				var marshalErr error
				res.CompletedChunkBytes, marshalErr = json.Marshal(completionChunk)
				if marshalErr != nil {
					return result, fmt.Errorf("failed to marshal completion chunk: %w", marshalErr)
				}
			}

			if heartbeatTurn && res.FullContent != "" && heartbeatack.ShouldSuppressAckReply(res.FullContent, ackMax) {
				agentforge.Debug("[heartbeat] HEARTBEAT_OK suppressed (no stream, no persistence)")
				result.HeartbeatAckSuppressed = true
				return result, nil
			}

			if heartbeatTurn {
				for _, b := range res.StreamBuf {
					if !responseCh.TrySend(b) {
						return result, nil
					}
				}
				res.StreamBuf = nil
			}

			if !responseCh.TrySend(res.CompletedChunkBytes) {
				return result, nil
			}

			if res.FullContent == "" {
				agentforge.Debug("LLM returned empty completion; skipping history save")
				return result, nil
			}

			hm.AddAssistantMessage(res.FullContent, history.TokenUsage{
				PromptTokens:     res.PromptTokens,
				CompletionTokens: res.CompletionTokens,
				TotalTokens:      res.TotalTokens,
			})
			if e.config.Tracer != nil {
				e.config.Tracer.TraceTokenUsage(ctx, telemetry.TokenUsageEvent{
					AgentName:        e.config.AgentName,
					PromptTokens:     res.PromptTokens,
					CompletionTokens: res.CompletionTokens,
					TotalTokens:      res.TotalTokens,
					Iteration:        iteration,
					HadToolCalls:     false,
				})
			}
			if e.hooks != nil && e.hooks.OnNewAssistantMessage != nil {
				errs := e.hooks.OnNewAssistantMessage(res.FullContent, res.PromptTokens, res.CompletionTokens, res.TotalTokens)
				if e.hooks.LogHookErrors != nil {
					e.hooks.LogHookErrors(errs)
				}
			}
			return result, nil
		}

		if res.CompletedChunkBytes != nil {
			var completedChunk llms.ChunkResponse
			if err := json.Unmarshal(res.CompletedChunkBytes, &completedChunk); err == nil && completedChunk.FullContent != "" {
				res.FullContent = completedChunk.FullContent
			}
		}
		for i := range res.ToolCalls {
			res.ToolCalls[i].Name = llms.ResolveToolNameForTools(res.ToolCalls[i].Name, e.tools)
		}
		if e.config.Tracer != nil {
			e.config.Tracer.TraceTokenUsage(ctx, telemetry.TokenUsageEvent{
				AgentName:        e.config.AgentName,
				PromptTokens:     res.PromptTokens,
				CompletionTokens: res.CompletionTokens,
				TotalTokens:      res.TotalTokens,
				Iteration:        iteration,
				HadToolCalls:     true,
			})
		}
		if e.hooks != nil && e.hooks.OnNewAssistantMessageWithTools != nil {
			errs := e.hooks.OnNewAssistantMessageWithTools(res.FullContent, res.ToolCalls, res.PromptTokens, res.CompletionTokens, res.TotalTokens)
			if e.hooks.LogHookErrors != nil {
				e.hooks.LogHookErrors(errs)
			}
		}

		hm.AddAssistantMessageWithToolCalls(res.FullContent, res.FullReasoningContent, res.ToolCalls, history.TokenUsage{
			PromptTokens:     res.PromptTokens,
			CompletionTokens: res.CompletionTokens,
			TotalTokens:      res.TotalTokens,
		})

		agentforge.Debug("Detected %d tool calls, executing tools in parallel", len(res.ToolCalls))

		type iterResult struct {
			result   llms.ToolResult
			duration time.Duration
			localCtx *core.AgentContext
		}
		iterResults := make([]iterResult, len(res.ToolCalls))
		hookErrsList := make([][]error, len(res.ToolCalls))

		// Pre-phase (serial): deep-copy args, send StatusToolExecuting, run OnBeforeToolExecution.
		// Hooks that mutate arguments (e.g. vault secret decryption) must run before goroutines start.
		for i := range res.ToolCalls {
			tc := &res.ToolCalls[i]
			if tc.Arguments != nil {
				argsCopy := make(map[string]any, len(tc.Arguments))
				for k, v := range tc.Arguments {
					argsCopy[k] = v
				}
				tc.Arguments = argsCopy
			}

			agentforge.Debug("Queuing tool for parallel execution: %s (ID: %s)", tc.Name, tc.ID)
			executingChunk := llms.ChunkResponse{
				Status:        llms.StatusToolExecuting,
				Type:          llms.TypeToolExecuting,
				ToolExecuting: tc,
				Iteration:     iteration,
			}
			executingBytes, err := json.Marshal(executingChunk)
			if err != nil {
				return result, fmt.Errorf("failed to serialize tool-executing chunk: %w", err)
			}
			if !responseCh.TrySend(executingBytes) {
				return result, nil
			}

			if e.hooks != nil && e.hooks.OnBeforeToolExecution != nil {
				hookErrsList[i] = e.hooks.OnBeforeToolExecution(tc)
				if e.hooks.LogHookErrors != nil {
					e.hooks.LogHookErrors(hookErrsList[i])
				}
			}
		}

		// Parallel phase: each tool runs in its own goroutine with an isolated context snapshot.
		// Writing to iterResults[i] is safe without a mutex since each goroutine owns a unique index.
		var wg sync.WaitGroup
		for i, toolCall := range res.ToolCalls {
			wg.Add(1)
			go func(idx int, tc llms.ToolCall, hookErrs []error) {
				defer wg.Done()
				localCtx := callCtx.Snapshot()
				start := time.Now()
				var result llms.ToolResult
				if len(hookErrs) > 0 {
					msgs := make([]string, len(hookErrs))
					for j, herr := range hookErrs {
						msgs[j] = herr.Error()
					}
					result = llms.ToolResult{
						ToolCallID: tc.ID,
						ToolName:   tc.Name,
						Success:    false,
						Error:      strings.Join(msgs, "; "),
					}
				} else {
					result = e.executeToolWithContext(tc, responseCh, localCtx)
				}
				iterResults[idx] = iterResult{result, time.Since(start), localCtx}
			}(i, toolCall, hookErrsList[i])
		}
		wg.Wait()

		// Post-phase (serial): merge context snapshots, run post-execution hooks,
		// send result chunks, and write to history — all in original tool call order.
		const maxToolResultLen = 50_000
		for i, toolCall := range res.ToolCalls {
			r := iterResults[i]

			// Merge mutable state from the goroutine's isolated snapshot back into callCtx.
			// Last-writer-wins for keys written by multiple tools in the same batch.
			if syncErr := callCtx.SyncFromMap(r.localCtx.BuildContext(responseCh)); syncErr != nil {
				agentforge.Debug("Warning: failed to sync context after parallel tool execution: %v", syncErr)
			}

			if e.config.Tracer != nil {
				e.config.Tracer.TraceToolExecution(ctx, telemetry.ToolExecutionEvent{
					AgentName: e.config.AgentName,
					ToolName:  toolCall.Name,
					Duration:  r.duration,
					Success:   r.result.Success,
					Error:     r.result.Error,
					Iteration: iteration,
				})
			}

			if r.result.Cleanup != nil {
				cleanupFuncs = append(cleanupFuncs, r.result.Cleanup)
			}

			if e.hooks != nil && e.hooks.OnToolExecution != nil {
				errs := e.hooks.OnToolExecution(&r.result)
				if e.hooks.LogHookErrors != nil {
					e.hooks.LogHookErrors(errs)
				}
			}

			resultChunk := llms.ChunkResponse{
				Status:      llms.StatusToolResult,
				Type:        llms.TypeToolResult,
				ToolResults: []llms.ToolResult{r.result},
				Iteration:   iteration,
			}
			resultBytes, err := json.Marshal(resultChunk)
			if err != nil {
				return result, fmt.Errorf("failed to serialize tool-result chunk: %w", err)
			}
			if !responseCh.TrySend(resultBytes) {
				agentforge.Debug("Channel closed when trying to send tool-result chunk for tool %s", toolCall.Name)
				return result, nil
			}

			content := r.result.Result
			if !r.result.Success && r.result.Error != "" {
				if content != "" {
					content = content + "\nError: " + r.result.Error
				} else {
					content = "Error: " + r.result.Error
				}
			}
			if len(content) > maxToolResultLen {
				content = content[:maxToolResultLen] +
					"\n\n[...truncated — use head_limit/offset to paginate]"
			}
			if r.result.Success && strings.HasPrefix(content, "data:") {
				hm.AddToolMessage(toolCall.ID, "[image loaded]", r.result.Ephemeral)
				hm.AddUserMessageWithImages("", content)
			} else {
				hm.AddToolMessage(toolCall.ID, content, r.result.Ephemeral)
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
	return result, fmt.Errorf("reached maximum tool iterations (%d)", e.config.MaxToolIterations)
}

func lastUserContent(messages []*llms.UnifiedMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role() == llms.MessageRoleUser {
			return messages[i].Content()
		}
	}
	return ""
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

	resolved := llms.ResolveToolNameForTools(toolCall.Name, e.tools)
	var tool llms.Tool
	for _, t := range e.tools {
		if t.GetName() == resolved {
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
		ToolName:   resolved,
		Success:    result.Success(),
		Result:     result.Data(),
		Error:      result.Error(),
		Ephemeral:  result.Ephemeral(),
		Cleanup:    result.Cleanup(),
	}
}

// streamResult holds the result of streaming LLM response chunks.
type streamResult struct {
	FullContent          string
	FullReasoningContent string
	ToolCalls            []llms.ToolCall
	HasToolCalls         bool
	CompletedChunkBytes  []byte
	PromptTokens         int
	CompletionTokens     int
	TotalTokens          int
	StreamBuf            [][]byte
}

func (e *Executor) streamLLMResponse(
	ctx context.Context,
	iteration int,
	heartbeatTurn bool,
	llmResponseCh *llms.ResponseCh,
	responseCh *core.ResponseCh,
) (streamResult, error) {
	var res streamResult
	llmErrorCh := llmResponseCh.Error

	flushStreamBuf := func() bool {
		for _, b := range res.StreamBuf {
			if !responseCh.TrySend(b) {
				return false
			}
		}
		res.StreamBuf = nil
		return true
	}

	sendOrBuffer := func(chunkBytes []byte) bool {
		if heartbeatTurn {
			res.StreamBuf = append(res.StreamBuf, chunkBytes)
			return true
		}
		return responseCh.TrySend(chunkBytes)
	}

	agentforge.Debug("Waiting for LLM response chunks...")
	for {
		select {
		case <-ctx.Done():
			agentforge.Debug("Context cancelled during LLM response streaming")
			return res, fmt.Errorf("execution cancelled: %w", ctx.Err())
		case chunkBytes, ok := <-llmResponseCh.Response:
			if !ok {
				agentforge.Debug("LLM response channel closed, processing tool calls")
				return res, nil
			}

			agentforge.Debug("Received chunk from LLM (size: %d bytes)", len(chunkBytes))
			var chunk llms.ChunkResponse
			if err := json.Unmarshal(chunkBytes, &chunk); err != nil {
				agentforge.Debug("Error deserializing chunk: %v", err)
				return res, fmt.Errorf("failed to deserialize chunk: %w", err)
			}

			if chunk.Content != "" {
				res.FullContent += chunk.Content
			} else if chunk.Delta != "" {
				res.FullContent += chunk.Delta
			}

			chunk.Iteration = iteration

			chunkBytes, err := json.Marshal(chunk)
			if err != nil {
				agentforge.Debug("Error re-serializing chunk with iteration: %v", err)
				return res, fmt.Errorf("failed to re-serialize chunk: %w", err)
			}

			if chunk.Status == llms.StatusToolCall && len(chunk.ToolCalls) > 0 {
				res.ToolCalls = chunk.ToolCalls
				res.HasToolCalls = true
				res.FullReasoningContent = chunk.ReasoningContent
				agentforge.Debug("Tool calls detected: %d tool calls", len(res.ToolCalls))
				if heartbeatTurn {
					if !flushStreamBuf() {
						return res, fmt.Errorf("channel closed")
					}
				}
				if !responseCh.TrySend(chunkBytes) {
					return res, fmt.Errorf("channel closed")
				}
				continue
			}

			if chunk.Status == llms.StatusCompleted {
				res.CompletedChunkBytes = chunkBytes
				res.PromptTokens = chunk.PromptTokens
				res.CompletionTokens = chunk.CompletionTokens
				res.TotalTokens = chunk.TotalTokens
				if chunk.FullContent != "" {
					res.FullContent = chunk.FullContent
				}
				agentforge.Debug("Received completed chunk, returning to processToolCalls")
				return res, nil
			}

			if !sendOrBuffer(chunkBytes) {
				return res, fmt.Errorf("channel closed")
			}

		case err, ok := <-llmErrorCh:
			if !ok {
				llmErrorCh = nil
				continue
			}
			if err != nil {
				agentforge.Debug("LLM stream error received: %v", err)
				return res, fmt.Errorf("llm stream error: %w", err)
			}
		}
	}
}
