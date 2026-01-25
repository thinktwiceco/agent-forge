package apis

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/thinktwice/agentForge/src/agents"
	"github.com/thinktwice/agentForge/src/builder"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// Server represents an HTTP server that exposes agent chat functionality via REST API.
type Server struct {
	agents             map[string]*agents.Agent
	mu                 sync.RWMutex
	vectorDB           core.VectorDB
	embeddingGenerator core.EmbeddingGenerator
	httpServer         *http.Server
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
		agents: make(map[string]*agents.Agent),
	}
}

// SetVectorComponents sets the vector database and embedding generator for agent initialization.
// These components are optional and will be used when initializing agents that require them.
func (s *Server) SetVectorComponents(vectorDB core.VectorDB, embeddingGenerator core.EmbeddingGenerator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vectorDB = vectorDB
	s.embeddingGenerator = embeddingGenerator
}

// InitializeAgent builds and registers an agent using the agent builder.
// This method is similar to cmd/chat/main.go's initializeAgent() function.
//
// Parameters:
//   - name: The name to register the agent under
//   - agentName: The agent's internal name (used in builder)
//   - model: The LLM model string (e.g., "openai::gpt-4")
//   - workingDir: Working directory for the agent
//   - tools: List of tools to add to the agent
//   - subagents: Map of subagent types to their model strings
//   - plugins: List of plugins to add
//
// Returns:
//   - error: Any error that occurred during agent initialization
func (s *Server) InitializeAgent(
	name string,
	agentName string,
	model string,
	workingDir string,
	tools []builder.Tool,
	subagents map[builder.Subagent]string,
	plugins []builder.Plugin,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b := builder.NewAgentBuilder(agentName, "json")

	// Add tools
	if len(tools) > 0 {
		b.AddTools(tools...)
	}

	// Set model
	if model != "" {
		b.SetModel(model)
	}

	// Set working directory
	if workingDir != "" {
		b.SetWorkingDir(workingDir)
	}

	// Set vector components if available
	if s.vectorDB != nil {
		b.SetVectorDB(s.vectorDB)
	}
	if s.embeddingGenerator != nil {
		b.SetEmbeddingGenerator(s.embeddingGenerator)
	}

	// Add subagents
	for subagent, subagentModel := range subagents {
		b.AddSubagent(subagent, subagentModel)
	}

	// Add plugins
	for _, plugin := range plugins {
		b.AddPlugin(plugin)
	}

	agent, err := b.Build()
	if err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
	}

	s.agents[name] = agent
	return nil
}

// InitializeAgentFromConfig builds and registers an agent using a configuration file.
func (s *Server) InitializeAgentFromConfig(name string, configPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := builder.NewAgentBuilderFromConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to create builder from config: %w", err)
	}

	// Set name for registration if provided, otherwise use from config
	agentName := name
	if agentName == "" {
		// Use default name from config if registration name is empty
		// We can't access private fields of b easily here, but we can assume
		// the user wants to use for registration what they put in the config if name is empty.
		// However, s.InitializeAgent takes 'name' as registration key.
		// Let's just say 'name' is required for registration key in Server.
		return fmt.Errorf("registration name is required")
	}

	// Set vector components if available
	if s.vectorDB != nil {
		b.SetVectorDB(s.vectorDB)
	}
	if s.embeddingGenerator != nil {
		b.SetEmbeddingGenerator(s.embeddingGenerator)
	}

	agent, err := b.Build()
	if err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
	}

	s.agents[agentName] = agent
	return nil
}

// RegisterAgent registers a pre-built agent with the given name.
func (s *Server) RegisterAgent(name string, agent *agents.Agent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agents[name] = agent
}

// GetAgent retrieves an agent by name. Returns nil if not found.
func (s *Server) GetAgent(name string) *agents.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agents[name]
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
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	// Only allow POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent name from URL path
	// Expected path: /api/server/{agentname}/chat
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
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

	// Set headers for streaming JSON (NDJSON format)
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Get response channel from agent
	responseCh := agent.ChatStream(req.Message)

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

	s.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown() error {
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}
