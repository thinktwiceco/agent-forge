package apis

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/thinktwiceco/agent-forge/src/builder"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// Server represents an HTTP server that exposes agent chat functionality via REST API.
type Server struct {
	agents     map[string]core.SubAgent
	mu         sync.RWMutex
	httpServer *http.Server
}

// ChatRequest represents the JSON request body for chat endpoint.
type ChatRequest struct {
	Message string `json:"message"`
}

// AgentsResponse represents the JSON response for listing agents.
type AgentsResponse struct {
	Agents []string `json:"agents"`
}

// NewServer creates a new Server instance.
func NewServer() *Server {
	return &Server{
		agents: make(map[string]core.SubAgent),
	}
}

// InitializeAgentFromConfig builds and registers an agent using a configuration file.
func (s *Server) InitializeAgentFromConfig(name string, configPath string) error {

	agentBuilder, err := builder.NewAgentBuilderFromConfig(configPath)

	if err != nil {
		return fmt.Errorf("failed to create agent builder from config: %w", err)
	}

	// Set name for registration: use provided name if not empty, otherwise use name from config
	agentName := name

	if agentName == "" {
		agentName = agentBuilder.GetName()
	}

	if agentName == "" {
		return fmt.Errorf("registration name is required (none provided and none found in config)")
	}

	// Initialize Vector Builder from configs
	vectorBuilder, err := builder.NewVectorBuilderFromConfig(configPath)

	if err != nil {
		return fmt.Errorf("failed to create vector builder from config: %w", err)
	}

	err = vectorBuilder.Build()

	if err != nil {
		return fmt.Errorf("failed to build vector components: %w", err)
	}

	// Extract the vector components if present and add them to the agent
	vectorDB := vectorBuilder.GetVectorDB()
	embeddingGenerator := vectorBuilder.GetEmbeddingGenerator()

	agentBuilder.SetVectorDB(vectorDB)
	agentBuilder.SetEmbeddingGenerator(embeddingGenerator)

	agent, err := agentBuilder.Build()

	if err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.agents[agentName] = agent
	return nil
}

// RegisterAgent registers a pre-built agent with the given name.
func (s *Server) RegisterAgent(name string, agent core.SubAgent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agents[name] = agent
}

// GetAgent retrieves an agent by name. Returns nil if not found.
func (s *Server) GetAgent(name string) core.SubAgent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agents[name]
}

// handleHealth handles requests to /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		http.Error(w, "Failed to encode health status", http.StatusInternalServerError)
	}
}

// handleListAgents handles GET requests to /api/server/agents
func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	// Only allow GET
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all agent names
	s.mu.RLock()
	agentNames := make([]string, 0, len(s.agents))
	for name := range s.agents {
		agentNames = append(agentNames, name)
	}
	s.mu.RUnlock()

	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Encode and send response
	response := AgentsResponse{Agents: agentNames}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleChat handles POST requests to /api/server/{agentname}/chat
// handleChat handles POST requests to /api/server/{agentname}/chat
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	// Only allow POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent name from URL path
	// Expected path: /api/server/{agentname}/chat
	path := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(path, "/")
	if len(pathParts) != 4 || pathParts[0] != "api" || pathParts[1] != "server" || pathParts[3] != "chat" {
		http.Error(w, "Invalid path. Expected: /api/server/{agentname}/chat", http.StatusBadRequest)
		return
	}
	agentName := pathParts[2]

	// Get agent
	agent := s.GetAgent(agentName)
	if agent == nil {
		http.Error(w, fmt.Sprintf("Agent '%s' not found", agentName), http.StatusNotFound)
		return
	}

	// Parse request body
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Message field is required", http.StatusBadRequest)
		return
	}

	// Extract optional conversationId query parameter
	conversationId := r.URL.Query().Get("conversationId")

	// Set headers for streaming JSON (NDJSON format)
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Get response channel from agent with conversationId
	// Use request context for cancellation support
	responseCh := agent.ChatStream(r.Context(), req.Message, conversationId)

	// Stream chunks as NDJSON (one JSON object per line)
	encoder := json.NewEncoder(w)
	for chunk := range responseCh.Start() {
		// Check for errors
		if chunk.Status == llms.StatusError {
			// Send error as chunk
			errorChunk := core.ExtendedChunkResponse{
				Content:   chunk.Content,
				Status:    llms.StatusError,
				Type:      llms.TypeContent,
				AgentName: chunk.AgentName,
				Trace:     chunk.Trace,
				ChatId:    chunk.ChatId,
			}
			if err := encoder.Encode(errorChunk); err != nil {
				// If encoding fails, we can't send error response
				return
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			return
		}

		// Encode chunk as JSON and write with newline
		if err := encoder.Encode(chunk); err != nil {
			// If encoding fails, stop streaming
			return
		}

		// Flush to send chunk immediately
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

// Start starts the HTTP server on the specified port.
// The server will listen on all interfaces (0.0.0.0).
func (s *Server) Start(port string) error {
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/server/agents", s.handleListAgents)
	mux.HandleFunc("/api/server/", s.handleChat)
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown() error {
	if s.httpServer != nil {
		// Use a context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
