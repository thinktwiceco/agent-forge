package main

type ChatRequest struct {
	Message string `json:"message" binding:"required"`
}

type UpdateToolConfigRequest struct {
	PostgresURL    *string   `json:"postgresURL,omitempty"`
	Mode           *string   `json:"mode,omitempty"`
	AllowedTables  *[]string `json:"allowedTables,omitempty"`
	AllowedSchemas *[]string `json:"allowedSchemas,omitempty"`
}

type ConversationSummary struct {
	ID        string `json:"id"`
	UpdatedAt string `json:"updatedAt"`
}

type AgentConfigResponse struct {
	Name         string               `json:"name"`
	Model        string               `json:"model"`
	SystemPrompt string               `json:"systemPrompt"`
	WorkingDir   string               `json:"workingDir"`
	Persistence  string               `json:"persistence"`
	Tools        []ToolConfigResponse `json:"tools"`
}

type ToolConfigResponse struct {
	Name           string   `json:"name"`
	PostgresURL    string   `json:"postgresURL,omitempty"`
	Mode           string   `json:"mode,omitempty"`
	AllowedTables  []string `json:"allowedTables,omitempty"`
	AllowedSchemas []string `json:"allowedSchemas,omitempty"`
}

type ProviderContext struct {
	Provider    string
	RecipientID string
}

const (
	SSEEventContent       = "content"
	SSEEventToolCall      = "tool_call"
	SSEEventToolExecuting = "tool_executing"
	SSEEventToolResult    = "tool_result"
	SSEEventCompleted     = "completed"
	SSEEventError         = "error"
	SSEEventThinking      = "thinking"
)
