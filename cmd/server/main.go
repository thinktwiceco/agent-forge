package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/thinktwiceco/agent-forge/src/apis"
)

func main() {
	// Parse command-line flags
	port := flag.String("port", "8080", "Port to listen on")
	flag.Parse()

	fmt.Println("Starting ThinkTwice Agent API Server...")

	// Create server
	server := apis.NewServer()

	// Initialize and register agents
	if err := initializeAgents(server); err != nil {
		fmt.Printf("Error initializing agents: %v\n", err)
		os.Exit(1)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		fmt.Printf("Server listening on port %s\n", *port)
		fmt.Printf("API endpoint: http://localhost:%s/api/server/{agentname}/chat\n", *port)
		if err := server.Start(*port); err != nil {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	fmt.Println("\nShutting down server...")
	if err := server.Shutdown(); err != nil {
		fmt.Printf("Error shutting down server: %v\n", err)
	}
	fmt.Println("Server stopped")
}

// initializeAgents creates and registers agents with the server.
func initializeAgents(server *apis.Server) error {
	// Initialize agent from config file
	configPath := "agent_config.yaml"
	agentName := "test-agent"

	if err := server.InitializeAgentFromConfig(agentName, configPath); err != nil {
		return fmt.Errorf("failed to initialize agent from config: %w", err)
	}

	fmt.Printf("Registered agent from config: %s\n", agentName)
	return nil
}
