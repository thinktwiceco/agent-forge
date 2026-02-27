package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to agent config")
	port := flag.String("port", "8080", "server port")
	devMode := flag.Bool("dev", false, "serve static files from disk instead of embedded FS")
	flag.Parse()

	// appDir is the directory containing the config — used to resolve .env and static assets.
	appDir := filepath.Dir(*configPath)

	// Load .env from the app directory, then fall back to cwd.
	_ = godotenv.Load(filepath.Join(appDir, ".env"))
	_ = godotenv.Load()

	cwd, _ := os.Getwd()
	log.Printf("Working directory : %s", cwd)
	log.Printf("App directory     : %s", appDir)
	log.Printf("Config            : %s", *configPath)
	log.Printf("Dev mode          : %v", *devMode)
	log.Printf("Port              : %s", *port)

	configMgr, err := NewConfigManager(*configPath)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	cfg := configMgr.GetConfig()
	log.Printf("Agent name        : %s", cfg.Agent.Name)
	log.Printf("Agent model       : %s", cfg.Agent.Model)

	// Create TodoManager and set its callback globally before building the agent
	todoMgr := NewTodoManager()

	agentMgr, err := NewAgentManager(configMgr)
	if err != nil {
		log.Fatalf("agent error: %v", err)
	}

	server := NewServer(agentMgr, configMgr, todoMgr, *devMode, appDir)

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
