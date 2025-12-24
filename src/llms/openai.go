package llms

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

// openAILLM implements an OpenAI llm with channel-based streaming.
//
// This llm is self-contained and uses channels for streaming responses
// instead of callback functions. It is a pure API communication layer.
type openAILLM struct {
	ctx     context.Context
	baseURL string
	model   string
	apiKey  string
	client  openai.Client
}

// newOpenAILLM creates a new openAILLM instance.
//
// Parameters:
//   - ctx: Context for cancellation
//   - baseURL: API base URL (empty string uses default OpenAI URL)
//   - model: Model name (e.g., "gpt-4", "gpt-3.5-turbo")
//   - apiKey: OpenAI API key
func newOpenAILLM(ctx context.Context, baseURL, model, apiKey string) *openAILLM {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := openai.NewClient(opts...)

	return &openAILLM{
		ctx:     ctx,
		baseURL: baseURL,
		model:   model,
		apiKey:  apiKey,
		client:  client,
	}
}

// ChatStream sends messages with optional tools and returns a ResponseCh for streaming responses.
//
// This method creates a ResponseCh with channels for receiving streaming chunks
// and errors. The actual streaming happens in a goroutine.
//
// Parameters:
//   - messages: The messages to send
//   - tools: Optional tools available for this request (can be nil or empty)
//
// Returns:
//   - *responseCh: responseCh instance with channels for streaming
func (a *openAILLM) ChatStream(messages []UnifiedMessage, tools []Tool) *responseCh {
	responseCh := newResponseCh()

	// Start streaming in a goroutine
	go a.streamResponse(messages, tools, responseCh)

	return responseCh
}

func toOpenAIMessages(messages []UnifiedMessage) ([]openai.ChatCompletionMessageParamUnion, error) {
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, len(messages))
	for i, message := range messages {
		if message.Role() == "system" {
			openaiMessages[i] = openai.SystemMessage(message.Content())
		} else if message.Role() == "user" {
			openaiMessages[i] = openai.UserMessage(message.Content())
		} else if message.Role() == "assistant" {
			// Check if assistant message has tool calls
			if len(message.ToolCalls()) > 0 {
				// Convert our ToolCall format to OpenAI format
				toolCalls := make([]openai.ChatCompletionMessageToolCallUnionParam, len(message.ToolCalls()))
				for j, tc := range message.ToolCalls() {
					// Serialize arguments to JSON string
					argsJSON, err := json.Marshal(tc.Arguments)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal tool call arguments: %w", err)
					}
					toolCalls[j] = openai.ChatCompletionMessageToolCallUnionParam{
						OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
							ID: tc.ID,
							Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
								Name:      tc.Name,
								Arguments: string(argsJSON),
							},
						},
					}
				}
				openaiMessages[i] = openai.ChatCompletionMessageParamUnion{
					OfAssistant: &openai.ChatCompletionAssistantMessageParam{
						Content: openai.ChatCompletionAssistantMessageParamContentUnion{
							OfString: openai.String(message.Content()),
						},
						ToolCalls: toolCalls,
					},
				}
			} else {
				openaiMessages[i] = openai.AssistantMessage(message.Content())
			}
		} else if message.Role() == "tool" {
			// Construct tool message manually to ensure correct parameter mapping
			// Note: openai.ToolMessage() helper has incorrect parameter order, so we construct manually
			openaiMessages[i] = openai.ChatCompletionMessageParamUnion{
				OfTool: &openai.ChatCompletionToolMessageParam{
					Role:       "tool",
					Content:    openai.ChatCompletionToolMessageParamContentUnion{OfString: openai.String(message.Content())},
					ToolCallID: message.ToolCallID(),
				},
			}
		} else {
			return nil, fmt.Errorf("invalid message role: %s", message.Role())
		}
	}
	return openaiMessages, nil
}

// streamResponse handles the actual streaming from OpenAI API.
func (a *openAILLM) streamResponse(messages []UnifiedMessage, tools []Tool, responseCh *responseCh) {
	defer responseCh.Close()

	// Build messages
	openaiMessages, err := toOpenAIMessages(messages)
	if err != nil {
		responseCh.Error <- fmt.Errorf("failed to convert messages to OpenAI messages: %w", err)
		return
	}

	// Build parameters
	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(a.model),
		Messages: openaiMessages,
		StreamOptions: openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		},
	}

	// Add tools if available
	if len(tools) > 0 {
		openaiTools := make([]openai.ChatCompletionToolUnionParam, len(tools))
		for i, tool := range tools {
			openaiTool := ToOpenAITool(tool)
			// Convert local FunctionDefinition to OpenAI format
			openaiTools[i] = openaiTool
		}
		params.Tools = openaiTools
	}

	// Create streaming request
	stream := a.client.Chat.Completions.NewStreaming(a.ctx, params)
	defer stream.Close()

	var fullContent string
	var promptTokens, completionTokens, totalTokens int
	// Track tool calls - map of tool call index to accumulated data
	toolCallsMap := make(map[int]*struct {
		ID        string
		Name      string
		Arguments string
	})

	// Process stream chunks
	for stream.Next() {
		chunk := stream.Current()

		// Capture usage information if available
		if chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0 {
			promptTokens = int(chunk.Usage.PromptTokens)
			completionTokens = int(chunk.Usage.CompletionTokens)
			totalTokens = int(chunk.Usage.TotalTokens)
		}

		for _, choice := range chunk.Choices {
			delta := choice.Delta

			// Handle content streaming
			if delta.Content != "" {
				fullContent += delta.Content

				// Create chunk response
				chunkResp := ChunkResponse{
					Content:     delta.Content,
					Delta:       delta.Content,
					FullContent: fullContent,
					Status:      StatusStreaming,
					Type:        TypeContent,
				}

				// Serialize and send
				jsonBytes, err := serializeChunk(chunkResp)
				if err != nil {
					responseCh.Error <- fmt.Errorf("failed to serialize chunk: %w", err)
					return
				}

				select {
				case responseCh.Response <- jsonBytes:
				case <-a.ctx.Done():
					return
				}
			}

			// Handle tool calls - accumulate deltas
			if len(delta.ToolCalls) > 0 {
				for _, toolCallDelta := range delta.ToolCalls {
					idx := int(toolCallDelta.Index)

					// Initialize tool call entry if not exists
					if toolCallsMap[idx] == nil {
						toolCallsMap[idx] = &struct {
							ID        string
							Name      string
							Arguments string
						}{}
					}

					// Accumulate tool call data
					if toolCallDelta.ID != "" {
						toolCallsMap[idx].ID = toolCallDelta.ID
					}
					if toolCallDelta.Function.Name != "" {
						toolCallsMap[idx].Name = toolCallDelta.Function.Name
					}
					if toolCallDelta.Function.Arguments != "" {
						toolCallsMap[idx].Arguments += toolCallDelta.Function.Arguments
					}
				}
			}
		}
	}

	// Check for stream errors
	if err := stream.Err(); err != nil {
		responseCh.Error <- fmt.Errorf("openai stream error: %w", err)
		return
	}

	// If we have tool calls, parse and send them
	if len(toolCallsMap) > 0 {
		toolCalls := make([]ToolCall, 0, len(toolCallsMap))

		// Convert map to sorted slice
		for i := 0; i < len(toolCallsMap); i++ {
			if toolData, exists := toolCallsMap[i]; exists {
				// Parse JSON arguments
				var args map[string]any
				if toolData.Arguments != "" {
					if err := json.Unmarshal([]byte(toolData.Arguments), &args); err != nil {
						responseCh.Error <- fmt.Errorf("failed to parse tool call arguments for %s: %w", toolData.Name, err)
						return
					}
				} else {
					args = make(map[string]any)
				}

				toolCalls = append(toolCalls, ToolCall{
					ID:        toolData.ID,
					Name:      toolData.Name,
					Arguments: args,
				})
			}
		}

		// Send tool call chunk
		toolCallChunk := ChunkResponse{
			Content:     "",
			Delta:       "",
			FullContent: fullContent,
			Status:      StatusToolCall,
			Type:        TypeToolCall,
			ToolCalls:   toolCalls,
		}

		jsonBytes, err := serializeChunk(toolCallChunk)
		if err != nil {
			responseCh.Error <- fmt.Errorf("failed to serialize tool call chunk: %w", err)
			return
		}

		select {
		case responseCh.Response <- jsonBytes:
		case <-a.ctx.Done():
			return
		}
	}

	// Send final completed chunk with token usage
	finalChunk := ChunkResponse{
		Content:          "",
		Delta:            "",
		FullContent:      fullContent,
		Status:           StatusCompleted,
		Type:             TypeCompletion,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
	}

	jsonBytes, err := serializeChunk(finalChunk)
	if err != nil {
		responseCh.Error <- fmt.Errorf("failed to serialize final chunk: %w", err)
		return
	}

	select {
	case responseCh.Response <- jsonBytes:
	case <-a.ctx.Done():
		return
	}
}
