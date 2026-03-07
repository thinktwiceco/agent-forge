package main

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	localauth "github.com/thinktwiceco/agent-forge/cmd/localforge/src/auth"
	"github.com/thinktwiceco/agent-forge/cmd/localforge/src/providers"
)

//go:embed static
var embeddedStatic embed.FS

type Server struct {
	engine           *gin.Engine
	agentMgr         *AgentManager
	configMgr        *ConfigManager
	todoMgr          *TodoManager
	httpSrv          *http.Server
	convRegistry     *ConversationRegistry
	pushRegistry     *PushRegistry
	providerRegistry *ProviderRegistry
	knowledgeDB      *sql.DB // opened once at startup; nil if DB not yet available
	devMode          bool
	appDir           string
	staticFS         fs.FS
	authConfig       localauth.Config
	sessionStore     *localauth.SessionStore
}

func NewServer(agentMgr *AgentManager, configMgr *ConfigManager, todoMgr *TodoManager, devMode bool, appDir string) *Server {
	authConfig := localauth.LoadConfigFromEnv()
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery(), corsMiddleware(authConfig.Enabled))

	// Initialize provider registry
	providerRegistry := NewProviderRegistry()

	// Register Instagram provider if token exists
	if token := os.Getenv("INSTAGRAM_ACCESS_TOKEN"); token != "" {
		providerRegistry.Register(providers.NewInstagramProvider(token))
	}

	// Register Telegram provider if token exists
	if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token != "" {
		providerRegistry.Register(providers.NewTelegramProvider(token))
	}

	server := &Server{
		engine:           engine,
		agentMgr:         agentMgr,
		configMgr:        configMgr,
		todoMgr:          todoMgr,
		convRegistry:     NewConversationRegistry(),
		pushRegistry:     NewPushRegistry(),
		providerRegistry: providerRegistry,
		devMode:          devMode,
		appDir:           appDir,
		authConfig:       authConfig,
		sessionStore:     localauth.NewSessionStore(),
	}

	staticFS, err := server.staticFileSystem()
	if err != nil {
		panic(err)
	}
	server.staticFS = staticFS
	if authConfig.Enabled {
		server.sessionStore.StartCleanup(time.Hour)
	}

	// Open the knowledge DB once so all handlers share a connection pool.
	// If the DB file doesn't exist yet we leave knowledgeDB nil and handlers
	// that need it will return an appropriate error.
	if db, err := openKnowledgeDB(configMgr); err != nil {
		log.Printf("knowledge DB not available at startup (will retry per-request): %v", err)
	} else {
		server.knowledgeDB = db
	}

	server.setupRoutes()
	return server
}

// openKnowledgeDB opens the SQLite knowledge database using settings from the config.
func openKnowledgeDB(configMgr *ConfigManager) (*sql.DB, error) {
	cfg := configMgr.GetConfig()
	workingDir := cfg.Agent.WorkingDir
	if workingDir == "" {
		workingDir = "."
	}
	dbPath := filepath.Join(workingDir, "knowledge", "knowledge.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=1")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func (s *Server) staticFileSystem() (fs.FS, error) {
	if s.devMode {
		return os.DirFS(filepath.Join(s.appDir, "src", "static")), nil
	}
	return fs.Sub(embeddedStatic, "static")
}

func (s *Server) setupRoutes() {
	// Serve static files. In dev mode wrap the handler to disable browser caching
	// so file edits are visible immediately without a hard refresh.
	staticHandler := http.StripPrefix("/static", http.FileServer(http.FS(s.staticFS)))
	if s.devMode {
		original := staticHandler
		staticHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			original.ServeHTTP(w, r)
		})
	}
	s.engine.GET("/static/*filepath", func(c *gin.Context) {
		staticHandler.ServeHTTP(c.Writer, c.Request)
	})
	s.engine.HEAD("/static/*filepath", func(c *gin.Context) {
		staticHandler.ServeHTTP(c.Writer, c.Request)
	})
	s.engine.GET("/login", s.handleLoginPage)

	pageRoutes := s.engine.Group("")
	pageRoutes.Use(localauth.Middleware(s.authConfig, s.sessionStore, localauth.UnauthorizedRedirect))
	pageRoutes.GET("/", s.serveStaticPage("index.html"))
	pageRoutes.GET("/knowledge", s.serveStaticPage("knowledge.html"))
	pageRoutes.GET("/settings", s.serveStaticPage("settings.html"))

	publicAPI := s.engine.Group("/api")
	publicAPI.GET("/auth/me", s.handleAuthMe)
	publicAPI.POST("/auth/login", s.handleAuthLogin)
	publicAPI.POST("/auth/logout", s.handleAuthLogout)
	publicAPI.POST("/webhooks/:provider", s.handleWebhook)
	publicAPI.POST("/webhooks/:provider/sync", s.handleWebhookSync)

	api := s.engine.Group("/api")
	api.Use(localauth.Middleware(s.authConfig, s.sessionStore, localauth.UnauthorizedJSON))
	api.POST("/chat", s.handleChat)
	api.POST("/chat/stop", s.handleStopChat)
	api.POST("/upload", s.handleUpload)
	api.GET("/chat/push", s.handlePush)
	api.GET("/conversations", s.handleListConversations)
	api.GET("/conversations/:id", s.handleGetConversation)
	api.DELETE("/conversations/:id", s.handleDeleteConversation)
	api.PUT("/conversations/:id/title", s.handleRenameConversation)
	api.GET("/config", s.handleGetConfig)
	api.PUT("/config", s.handleUpdateAgentConfig)
	api.PUT("/config/tools/:toolName", s.handleUpdateToolConfig)
	api.PUT("/config/plugins", s.handleUpdatePlugins)
	api.PUT("/config/subagents", s.handleUpdateSubagents)
	api.GET("/config/providers", s.handleGetProviders)
	api.PUT("/config/providers", s.handleUpdateProviders)
	api.POST("/agent/reload", s.handleReload)
	api.GET("/todos", s.handleGetTodos)

	// FS visualization endpoints
	api.GET("/fs/list", s.handleFSList)
	api.GET("/fs/read", s.handleFSRead)

	// Knowledge graph endpoints
	api.GET("/knowledge/graph", s.handleGetKnowledgeGraph)
	api.GET("/knowledge/stats", s.handleGetKnowledgeStats)
	api.GET("/knowledge/node/:id", s.handleGetKnowledgeNode)
}

func (s *Server) Run(port string) error {
	if port == "" {
		port = "8080"
	}
	s.httpSrv = &http.Server{
		Addr:    ":" + port,
		Handler: s.engine,
	}
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.knowledgeDB != nil {
		_ = s.knowledgeDB.Close()
	}
	if s.httpSrv == nil {
		return nil
	}
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) serveStaticPage(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := fs.ReadFile(s.staticFS, name)
		if err != nil {
			c.String(http.StatusNotFound, "%s not found", name)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	}
}

func corsMiddleware(authEnabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authEnabled {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		} else if origin := c.Request.Header.Get("Origin"); origin != "" && sameOrigin(origin, c.Request.Host) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func sameOrigin(origin string, host string) bool {
	trimmed := strings.TrimSpace(origin)
	if trimmed == "" {
		return false
	}
	return strings.HasSuffix(trimmed, "://"+host)
}
