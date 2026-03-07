package main

// ─── HTTP Request / Response Types ───────────────────────────────────────────

type ChatRequest struct {
	Message string `json:"message" binding:"required"`
}

// ─── Tool config ──────────────────────────────────────────────────────────────

type UpdateToolConfigRequest struct {
	PostgresURL    *string   `json:"postgresURL,omitempty"`
	Mode           *string   `json:"mode,omitempty"`
	AllowedTables  *[]string `json:"allowedTables,omitempty"`
	AllowedSchemas *[]string `json:"allowedSchemas,omitempty"`
}

// ─── Conversation ─────────────────────────────────────────────────────────────

type ConversationSummary struct {
	ID        string `json:"id"`
	Title     string `json:"title,omitempty"`
	UpdatedAt string `json:"updatedAt"`
}

type RenameChatRequest struct {
	Title string `json:"title" binding:"required"`
}

// ─── Agent config – read ──────────────────────────────────────────────────────

type AgentConfigResponse struct {
	Name         string               `json:"name"`
	Model        string               `json:"model"`
	SystemPrompt string               `json:"systemPrompt"`
	WorkingDir   string               `json:"workingDir"`
	Persistence  string               `json:"persistence"`
	Tools        []ToolConfigResponse `json:"tools"`
	Subagents    map[string]string    `json:"subagents"`
	Plugins      []string             `json:"plugins"`
}

type ToolConfigResponse struct {
	Name           string   `json:"name"`
	PostgresURL    string   `json:"postgresURL,omitempty"`
	Mode           string   `json:"mode,omitempty"`
	AllowedTables  []string `json:"allowedTables,omitempty"`
	AllowedSchemas []string `json:"allowedSchemas,omitempty"`
}

// ─── Agent config – write ─────────────────────────────────────────────────────

// UpdateAgentRequest updates top-level agent identity fields.
// All fields are optional; only provided (non-nil) fields are patched.
type UpdateAgentRequest struct {
	Name         *string `json:"name,omitempty"`
	Model        *string `json:"model,omitempty"`
	SystemPrompt *string `json:"systemPrompt,omitempty"`
	WorkingDir   *string `json:"workingDir,omitempty"`
	Persistence  *string `json:"persistence,omitempty"`
}

// UpdatePluginsRequest replaces the full plugin list.
type UpdatePluginsRequest struct {
	Plugins []string `json:"plugins" binding:"required"`
}

// UpdateSubagentsRequest replaces the full subagents map (role → model string).
type UpdateSubagentsRequest struct {
	Subagents map[string]string `json:"subagents" binding:"required"`
}

// ─── Provider / API key config ────────────────────────────────────────────────

// ProviderConfig describes a single provider's token state for the UI.
type ProviderConfig struct {
	// EnvKey is the environment variable name (e.g. "AF_OPENAI_API_KEY")
	EnvKey string `json:"envKey"`
	// Label is the human-readable name shown in the UI
	Label string `json:"label"`
	// Group is "llm" or "messaging" for grouping in the UI
	Group string `json:"group"`
	// IsSet is true when a non-empty value exists for the key
	IsSet bool `json:"isSet"`
	// MaskedValue is the last 4 chars of the token, or empty string
	MaskedValue string `json:"maskedValue"`
}

// UpdateProvidersRequest maps env key → new token value.
// An empty string value clears the key.
type UpdateProvidersRequest struct {
	Providers map[string]string `json:"providers" binding:"required"`
}

// ─── Authentication ───────────────────────────────────────────────────────────

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthStatusResponse struct {
	Enabled       bool   `json:"enabled"`
	Authenticated bool   `json:"authenticated"`
	Username      string `json:"username,omitempty"`
	Next          string `json:"next,omitempty"`
}

// ─── SSE event names ──────────────────────────────────────────────────────────

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
