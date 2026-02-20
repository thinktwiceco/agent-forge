package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	configPath := flag.String("config", "agent_config.yaml", "path to agent config")
	port := flag.String("port", "8080", "server port")
	flag.Parse()

	// Load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	// Register API hooks before creating agent
	RegisterApiHooks()

	configMgr, err := NewConfigManager(*configPath)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// Create TodoManager and set its callback globally before building the agent
	todoMgr := NewTodoManager()

	agentMgr, err := NewAgentManager(configMgr)
	if err != nil {
		log.Fatalf("agent error: %v", err)
	}

	server := NewServer(agentMgr, configMgr, todoMgr)

	// Route background-drain chunks (sub-agent responses) to the push SSE endpoint.
	agentMgr.SetChunkRouter(server.pushRegistry.Push)

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Run(*port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-shutdownCh
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
