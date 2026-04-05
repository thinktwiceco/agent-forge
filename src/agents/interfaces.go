package agents

import (
	"context"

	"github.com/thinktwiceco/agent-forge/src/agents/execution"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/history"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// ===== Core Agent Interfaces =====
//
// These interfaces define contracts between major internal components.
// They enable:
// - Loose coupling between modules
// - Easy unit testing with mock implementations
// - Flexibility to swap implementations
// - Clear API boundaries

// HistoryManager is an alias for history.Manager to maintain consistency
// with the naming conventions in this package while using the external interface.
//
// HistoryManager handles conversation history lifecycle including:
// - Adding messages (user, assistant, system, tool)
// - Managing chat sessions
// - Persisting and loading history
// - Token tracking for completions
type HistoryManager interface {
	history.Manager
}

// PromptBuilder constructs system prompts from configuration.
//
// The prompt builder is responsible for:
// - Building the base system prompt
// - Adding concise tool-use guidance when tools are configured
// - Applying tone and style configurations
// - Normalizing prompt formatting
type PromptBuilder interface {
	// Build returns the complete system prompt including all configured components
	Build() string

	// UpdateConfig updates the builder's configuration
	// Call when tools change to rebuild the prompt
	UpdateConfig(config interface{})
}

// ExecutionEngine handles chat execution and tool invocation.
//
// The execution engine is responsible for:
// - Managing the chat loop with automatic tool execution
// - Streaming responses from the LLM
// - Executing tool calls
// - Managing iteration limits
// - Triggering execution hooks
type ExecutionEngine interface {
	// ExecuteChatWithTools executes the chat loop with automatic tool execution
	// It manages multiple iterations of LLM responses and tool calls until:
	// - The LLM provides a final response (no tool calls)
	// - Maximum iterations are reached
	// - An error occurs
	ExecuteChatWithTools(ctx context.Context, hm history.Manager, responseCh *core.ResponseCh) (execution.ExecuteResult, error)

	// ExecuteTool finds and executes a specific tool by its call details
	// Returns a ToolResult containing the execution outcome
	ExecuteTool(toolCall llms.ToolCall, responseCh *core.ResponseCh) llms.ToolResult

	// UpdateTools updates the tools available to the executor
	// Call when tools are added or modified
	UpdateTools(tools []llms.Tool)

	// UpdateAgentContext updates the agent context used during tool execution
	// Call when context is rebuilt
	UpdateAgentContext(agentContext *core.AgentContext)
}

// HookRegistry manages agent lifecycle hooks.
//
// Hooks provide extension points throughout the agent lifecycle:
// - Agent initialization and configuration
// - Context building
// - Tool execution (before and after)
// - Message events (user, assistant, chunks)
//
// The registry allows plugins and external code to observe and modify
// agent behavior without directly coupling to implementation details.
type HookRegistry interface {
	// Event registration methods
	// Each method registers a hook for a specific lifecycle event
	on(event core.Event, hook any)

	// Event trigger methods
	// These are called by the agent at appropriate lifecycle points
	contextBuildEvent(a *Agent, agentContext *core.AgentContext) []error
	beforeToolExecutionEvent(a *Agent, toolCall *llms.ToolCall) []error
	toolExecutionEvent(a *Agent, toolResult *llms.ToolResult) []error
	newUserMessageEvent(a *Agent, message string) []error
	newAssistantMessageEvent(a *Agent, message string, promptTokens, completionTokens, totalTokens int) []error
	newAssistantMessageWithToolCallsEvent(a *Agent, message string, toolCalls []llms.ToolCall, promptTokens, completionTokens, totalTokens int) []error
	addedToolsEvent(a *Agent, tools []llms.Tool) []error
	agentInitializationEvent(a *Agent, config *AgentConfig) []error
	agentInitializedEvent(a *Agent) []error
	newChunkEvent(a *Agent, chunk *core.ExtendedChunkResponse) []error
	chatStartEvent(a *Agent, chatId string) []error
}

// ContextManager manages agent context lifecycle and operations.
//
// The context manager is responsible for:
// - Creating and maintaining the AgentContext
// - Building context maps for tool execution
// - Syncing changes back after tool execution
// - Updating context when tools change
// - Managing session storage across requests
// - Managing plugin fields
// - Truncating message history (if configured)
// - Preserving context state during updates
type ContextManager interface {
	// Context returns the current AgentContext
	Context() *core.AgentContext

	// BuildContext converts the AgentContext to a map for tool execution
	// Merges the static context with the session-specific responseCh
	BuildContext(responseCh *core.ResponseCh) map[string]any

	// SyncFromMap syncs mutable fields from the context map back to AgentContext
	// Ensures changes made by tools persist across tool calls
	SyncFromMap(contextMap map[string]any) error

	// TruncateHistory applies truncation to message history if configured
	// Returns the truncated history, or the original if truncation is not configured
	TruncateHistory(messages []*llms.UnifiedMessage) []*llms.UnifiedMessage

	// UpdateConfig updates the manager configuration and rebuilds context
	// Call when tools change
	UpdateConfig(config interface{})

	// UpdateTools updates just the tools without rebuilding entire config
	UpdateTools(tools []llms.Tool)
}

// ===== Compile-time Interface Assertions =====
//
// These assertions ensure that concrete types implement their respective interfaces.
// If a type doesn't properly implement its interface, compilation will fail,
// catching interface mismatches early.

// Ensure ConversationHistory from history package implements HistoryManager
// (This is implicit since HistoryManager is an alias for history.Manager)
var _ HistoryManager = (*history.ConversationHistory)(nil)

// Note: The following interface assertions cannot be placed here due to circular dependencies.
// They are documented here for reference, but the actual compile-time checks are implicit
// through usage in the Agent struct:
//
// - prompts.Builder should implement PromptBuilder
// - execution.Executor should implement ExecutionEngine
// - context.Manager should implement ContextManager
// - AgentHooks should implement HookRegistry
//
// These assertions are enforced by the Agent struct field types and method signatures.
