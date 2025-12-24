package agents

import (
	"encoding/json"
	"fmt"

	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/persistence"
	"github.com/thinktwice/agentForge/src/tools"
)

// Agent represents an advanced agent with an LLM engine.
//
// This struct wraps an LLM engine (OpenAI-compatible) and provides
// a higher-level agent interface.
type Agent struct {
	// llmEngine is the underlying LLM engine that handles streaming responses.
	// It implements the llms.Agent interface which provides ChatStream method.
	llmEngine            llms.LLMEngine
	agentName            string
	description          string
	advanceDescription   string
	troubleshooting      string
	trace                string
	systemPrompt         string
	tools                []llms.Tool
	history              *History
	maxToolIterations    int
	toolExecutionContext map[string]any
	subAgents            []*Agent
	mainAgent            bool
	extraEngines         map[string]llms.LLMEngine
	persistence          string
}

// NewAgent creates a new Agent instance with the provided configuration.
//
// Tools can be set on the LLM engine before or after creating the Agent.
// Use GetTools() and SetTools() methods to manage tools via the Agent interface.
//
// This function validates that all required fields are set before creating the Agent.
// It panics if validation fails to ensure invalid agents are never created.
//
// Parameters:
//   - config: AgentConfig struct containing all agent configuration parameters
//
// Returns:
//   - *Agent: A new Agent instance
//
// Panics:
//   - If required fields (LLMEngine or AgentName) are missing
func NewAgent(config AgentConfig) *Agent {
	if err := config.validate(); err != nil {
		panic(fmt.Errorf("invalid AgentConfig: %w", err))
	}

	// Set default max tool iterations if not specified
	maxIterations := config.MaxToolIterations
	if maxIterations <= 0 {
		maxIterations = 10
	}

	// Initialize tool execution context if not provided
	toolContext := config.ToolExecutionContext
	if toolContext == nil {
		toolContext = make(map[string]any)
	}

	subAgents := []*Agent{}

	if config.Reasoning {
		// Create reasoning agent from template
		// Check if a specific engine is configured for this sub agent
		var engineForReasoning llms.LLMEngine
		if config.ExtraEngines != nil {
			if engine, ok := config.ExtraEngines["system-reasoning"]; ok && engine != nil {
				engineForReasoning = engine
			} else {
				engineForReasoning = config.LLMEngine
			}
		} else {
			engineForReasoning = config.LLMEngine
		}
		raConfig := ReasoningAgentTemplate.ToAgentConfig(engineForReasoning)
		ra := NewAgent(raConfig)
		subAgents = append(subAgents, ra)
	}

	// Initialize tools slice if not provided
	agentTools := config.Tools
	if agentTools == nil {
		agentTools = []llms.Tool{}
	}

	// Add delegate tool if there are sub agents
	if len(subAgents) > 0 {
		// Convert sub agents to SubAgent interface using adapters
		var subAgentInterfaces []tools.SubAgent
		for _, sa := range subAgents {
			subAgentInterfaces = append(subAgentInterfaces, newAgentSubAgentAdapter(sa))
		}
		delegateTool := tools.NewDelegateTool(subAgentInterfaces)
		agentTools = append(agentTools, delegateTool)
	}

	return &Agent{
		llmEngine:            config.LLMEngine,
		agentName:            config.AgentName,
		description:          config.Description,
		advanceDescription:   config.AdvanceDescription,
		troubleshooting:      config.Troubleshooting,
		trace:                config.Trace,
		systemPrompt:         config.SystemPrompt,
		tools:                agentTools,
		maxToolIterations:    maxIterations,
		toolExecutionContext: toolContext,
		subAgents:            subAgents,
		mainAgent:            config.MainAgent,
		extraEngines:         config.ExtraEngines,
		persistence:          config.Persistence,
	}
}

////// PUBLIC METHODS //////

// ChatStream sends a message to the underlying LLM engine and returns a ResponseCh
// for streaming responses with agent-specific context.
//
// This method creates a ResponseCh with channels for receiving streaming chunks
// and errors. The chunks are forwarded from the underlying LLM's ResponseCh and enriched
// with agent name and trace information.
//
// Parameters:
//   - message: The user message to send
//
// Returns:
//   - *ResponseCh: ResponseCh instance with channels for streaming
func (a *Agent) ChatStream(message string) *responseCh {
	// Retrieve history
	a.ensureHistory()
	a.history.get()

	// Get the underlying LLM's response channel
	var messages = a.handleSystemPromptInjection()
	messages = a.handleNewUserMessage(message)

	agentforge.Debug("messages-> %+v", messages)

	// Create agent-specific response channel
	agentResponseCh := newResponseCh(a.agentName, a.trace)

	// Start the tool execution loop in a goroutine
	go func() {
		defer agentResponseCh.Close()

		if err := a.executeChatWithTools(agentResponseCh); err != nil {
			agentResponseCh.Error <- err
		}
	}()

	return agentResponseCh
}

// GetTools returns the list of tools currently configured for this agent.
//
// Returns:
//   - []llms.Tool: Slice of tools (empty slice if no tools configured, never nil)
func (a *Agent) GetTools() []llms.Tool {
	if a.tools == nil {
		return []llms.Tool{}
	}
	return a.tools
}

// SetTools sets the tools available to this agent.
//
// Tools can be set at any time and will be used in subsequent ChatStream calls.
//
// Parameters:
//   - tools: Slice of tools to configure (can be nil or empty)
func (a *Agent) SetTools(tools []llms.Tool) {
	a.tools = tools
}

func (a *Agent) Name() string {
	return a.agentName
}

func (a *Agent) Description() string {
	return a.description
}

func (a *Agent) Trace() string {
	return a.trace
}

func (a *Agent) SystemPrompt() string {
	return a.systemPrompt
}

// BasicDescription returns a short one-line description of the agent.
// This implements the agentforge.Discoverable interface.
func (a *Agent) BasicDescription() string {
	return a.description
}

// AdvanceDescription returns detailed information about the agent's
// capabilities, tools, sub-agents, and usage patterns.
// This implements the agentforge.Discoverable interface.
func (a *Agent) AdvanceDescription() string {
	return a.advanceDescription
}

// Troubleshooting returns information about common issues, debugging tips,
// and configuration guidance for this agent.
// This implements the agentforge.Discoverable interface.
func (a *Agent) Troubleshooting() string {
	return a.troubleshooting
}

// //// PRIVATE METHODS //////

// executeChatWithTools executes the chat loop with automatic tool execution.
// It handles streaming responses, tool call detection, execution, and iteration.
func (a *Agent) executeChatWithTools(agentResponseCh *responseCh) error {
	iteration := 0

	for iteration < a.maxToolIterations {
		iteration++

		// Get current history
		a.ensureHistory()
		// Load history from persistence
		a.history.get()
		messages := a.history.History()

		// Call LLM with current history and tools
		llmResponseCh := a.llmEngine.ChatStream(messages, a.tools)

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

				// Accumulate content
				if chunk.Content != "" {
					fullContent += chunk.Content
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
				agentResponseCh.Response <- chunkBytes

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
				agentResponseCh.Response <- completedChunkBytes
				// Save the message to history with token usage
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
			agentResponseCh.Response <- executingBytes

			// Find and execute the tool
			toolResult := a.executeTool(toolCall, agentResponseCh)

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
			agentResponseCh.Response <- resultBytes

			// Add tool result to history
			a.history.addToolMessage(toolCall.ID, toolResult.Result)
			a.history.save()
		}

		// Continue to next iteration (will call LLM again with tool results)
	}

	// If we reached max iterations, return error
	return fmt.Errorf("reached maximum tool iterations (%d)", a.maxToolIterations)
}

// executeTool finds and executes a tool by name.
func (a *Agent) executeTool(toolCall llms.ToolCall, agentResponseCh *responseCh) llms.ToolResult {
	// Prepare agent context
	agentContext := make(map[string]any)
	agentContext["agentName"] = a.agentName
	agentContext["trace"] = a.trace
	agentContext["responseCh"] = agentResponseCh
	// Add tools and subAgents for discovery tools (like expand)
	agentContext["tools"] = a.tools
	agentContext["subAgents"] = a.convertSubAgentsToInterfaces()
	// Add custom context
	for k, v := range a.toolExecutionContext {
		agentContext[k] = v
	}

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

	// Convert to ToolResult
	return llms.ToolResult{
		ToolCallID: toolCall.ID,
		ToolName:   toolCall.Name,
		Success:    result.Success(),
		Result:     result.Data(),
		Error:      result.Error(),
	}
}

func (a *Agent) ensureHistory() {
	if a.history == nil {
		a.history = &History{}

		// Set up persistence if configured using the factory
		if a.persistence != "" {
			a.history.persistence = persistence.NewPersistence(a.agentName, a.persistence)
			if a.history.persistence != nil {
				agentforge.Debug("Initialized %s persistence for agent '%s'", a.persistence, a.agentName)
			}
		}
	}
}

func (a *Agent) ensureSystemPrompt() {
	if a.mainAgent {
		a.systemPrompt += `
[SYSTEM] This are information in addition to any system prompt that the user provided.
You are part of a multi-agent system designed to solve complex problems.
You are the MAIN agent of the team.
You coordinate the team, asking very precise questions to the sub agents
and read and understand the responses.

IMPORTANT - TOOL CALLS ARE OPTIONAL:
- Tool calls (function calls) are ONLY used when delegating tasks to sub-agents
- Most interactions DO NOT require any tool calls
- Greetings, casual conversation, simple Q&A, and direct answers should NEVER trigger tool calls
- Respond naturally and directly without tool calls unless you specifically need to delegate to a sub-agent
- You are NOT required to make a tool call for every message

You can ask questions to sub agents in order to keep your context clean and focused.
You might have some default sub agents that you can rely on. 
If the user asks you "what are the agents of your team?", or 
"what are your sub agents?", it very likely refers to your [SUB AGENTS].
DO NOT REPORT any other sub agents that might be part of your llm implementation.
DO NOT REPORT any agent that is not part of your [SUB AGENTS].

AT ANY MOMENT KEEP IN MIND WHAT IS YOUR GOAL AND WHAT IS THE QUESTION THAT THE USER ASKED YOU.

WHEN TO DELEGATE (USE TOOL CALLS):
Tool calls are ONLY used for delegating to sub-agents. Before making a tool call to delegate, analyze the question carefully:
- Is this a COMPLEX problem that requires breaking down into multiple steps?
- Does the problem require systematic logical reasoning and analysis?
- Is the information NOT already available in your system prompt or context?
- Would the problem benefit significantly from specialized analysis?

If the answer to ALL these questions is YES, then make a tool call to delegate to the appropriate sub agent.

WHEN NOT TO DELEGATE (NO TOOL CALLS NEEDED):
DO NOT make tool calls in these cases - just respond directly:
- Greetings and casual conversation (e.g., "Hi!", "How are you?", "Thanks!")
- Simple informational questions (e.g., "How many sub agents do you have?")
- Questions about your own capabilities or configuration (the answers are in your system prompt)
- Straightforward tasks that don't require step-by-step breakdown
- Questions where you already have the answer in your context
- Simple Q&A, calculations, or explanations you can provide directly

BE MINDFUL
- Respond naturally without tool calls for most interactions
- Only use the "delegate" tool when truly delegating a complex task to a sub-agent
- Read carefully the task and your sub agents descriptions
- Only delegate COMPLEX tasks that truly benefit from specialized analysis
- Answer simple questions and have normal conversations directly yourself
`
	}

	if a.systemPrompt == "" {
		a.systemPrompt = `You are an helpful assistant`
	}
	a.addSubAgents()
	a.ensureHistory()
	a.history.addSystemMessage(a.systemPrompt)
}

func (a *Agent) addSubAgents() {
	if len(a.subAgents) == 0 || a.subAgents == nil {
		return
	}

	var saPrompt = `
=== SUB AGENTS ===
You have sub agents that have specific responsibilities.
You can delegate COMPLEX tasks to them by using the "delegate" tool (this is the ONLY time you use tool calls).
Only delegate when the task matches the sub agent's specialization and truly requires it.
Make sure to provide all the important information and details to the sub agent
necessary to perform the task.

Remember: Tool calls are OPTIONAL and ONLY for delegation. Most conversations don't need any tool calls.

[SUB AGENTS]:
	`

	for _, sa := range a.subAgents {
		sa.mainAgent = false
		// Use BasicDescription() to ensure only basic info is injected into system prompt
		saPrompt += fmt.Sprintf("📌 %s: %s\n\n", sa.agentName, sa.BasicDescription())
	}

	a.systemPrompt += saPrompt
}

func (a *Agent) handleNewUserMessage(message string) []llms.UnifiedMessage {
	a.ensureHistory()
	a.history.addUserMessage(message)
	a.history.save()
	return a.history.History()
}

func (a *Agent) handleNewAssistantMessage(message string) {
	a.ensureHistory()
	a.history.addAssistantMessage(message, 0, 0, 0)
	a.history.save()
}

func (a *Agent) handleSystemPromptInjection() []llms.UnifiedMessage {
	a.ensureHistory()
	a.ensureSystemPrompt()
	a.history.addSystemMessage(a.systemPrompt)
	a.history.save()
	return a.history.History()
}

// convertSubAgentsToInterfaces converts internal sub-agents to SubAgent interfaces
// for use in tool execution context (e.g., for the expand tool)
func (a *Agent) convertSubAgentsToInterfaces() []tools.SubAgent {
	var subAgentInterfaces []tools.SubAgent
	for _, sa := range a.subAgents {
		subAgentInterfaces = append(subAgentInterfaces, newAgentSubAgentAdapter(sa))
	}
	return subAgentInterfaces
}

// agentSubAgentAdapter adapts an Agent to implement the SubAgent interface
// required by the delegate tool. This allows agents to be used as sub agents
// without circular dependencies between packages.
type agentSubAgentAdapter struct {
	agent *Agent
}

// newAgentSubAgentAdapter creates a new adapter that wraps an Agent
func newAgentSubAgentAdapter(agent *Agent) *agentSubAgentAdapter {
	return &agentSubAgentAdapter{agent: agent}
}

// ChatStream delegates to the agent's ChatStream method and returns an adapter
// that satisfies the IResponseChStarter interface
func (a *agentSubAgentAdapter) ChatStream(message string) tools.IResponseChStarter {
	responseCh := a.agent.ChatStream(message)
	return responseCh.AsGeneric()
}

// Name returns the agent's name
func (a *agentSubAgentAdapter) Name() string {
	return a.agent.Name()
}

// BasicDescription returns the agent's basic description
// This allows the adapter to implement the Discoverable interface
func (a *agentSubAgentAdapter) BasicDescription() string {
	return a.agent.BasicDescription()
}

// AdvanceDescription returns the agent's advanced description
// This allows the adapter to implement the Discoverable interface
func (a *agentSubAgentAdapter) AdvanceDescription() string {
	return a.agent.AdvanceDescription()
}

// Troubleshooting returns the agent's troubleshooting information
// This allows the adapter to implement the Discoverable interface
func (a *agentSubAgentAdapter) Troubleshooting() string {
	return a.agent.Troubleshooting()
}
