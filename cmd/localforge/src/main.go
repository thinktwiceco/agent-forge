package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/builder"
	"github.com/thinktwiceco/agent-forge/src/core"
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

	// Initialize agentforge logger so AF_LOG_LEVEL from .env is respected
	if config, err := agentforge.NewConfig(); err == nil {
		agentforge.InitLogger(config)
	}

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

	builder.SetConfigWriter(newConfigWriterAdapter(configMgr, agentMgr))

	server := NewServer(agentMgr, configMgr, todoMgr, *devMode, appDir)

	// Route background-drain chunks to the push SSE endpoint.
	// Heartbeat turns go to the fixed "heartbeat-live" push channel so the web
	// client can maintain a permanent subscription for them.
	agentMgr.SetChunkRouter(func(chatId string, chunk core.ExtendedChunkResponse) {
		if strings.HasPrefix(chatId, "heartbeat-") {
			server.pushRegistry.Push("heartbeat-live", chunk)
			return
		}
		server.pushRegistry.Push(chatId, chunk)
	})

	// After each heartbeat turn, send the full response to all known Telegram recipients.
	// Recipients are discovered from conversation files and the telegram thread store —
	// no TELEGRAM_ALLOWED_USER_IDS env var required.
	agentMgr.SetTurnCompleteRouter(func(chatId, fullContent string) {
		if !strings.HasPrefix(chatId, "heartbeat-") {
			return
		}
		log.Printf("[heartbeat] turn complete for %s (content len=%d)", chatId, len(fullContent))
		tp := server.providerRegistry.Get("telegram")
		if tp == nil {
			log.Printf("[heartbeat] no telegram provider registered — skipping broadcast")
			return
		}
		recipients := server.knownTelegramChatIDs()
		log.Printf("[heartbeat] broadcasting to %d telegram recipient(s): %v", len(recipients), recipients)
		ctx := context.Background()
		for _, recipientID := range recipients {
			if err := tp.SendMessage(ctx, recipientID, fullContent); err != nil {
				log.Printf("[heartbeat] telegram send to %s failed: %v", recipientID, err)
			} else {
				log.Printf("[heartbeat] telegram send to %s OK", recipientID)
			}
		}
	})

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
