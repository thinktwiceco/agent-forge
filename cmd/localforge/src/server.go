package main

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

//go:embed static
var embeddedStatic embed.FS

type Server struct {
	engine       *gin.Engine
	agentMgr     *AgentManager
	configMgr    *ConfigManager
	todoMgr      *TodoManager
	httpSrv      *http.Server
	convRegistry *ConversationRegistry
	pushRegistry *PushRegistry
	devMode      bool
	appDir       string
}

func NewServer(agentMgr *AgentManager, configMgr *ConfigManager, todoMgr *TodoManager, devMode bool, appDir string) *Server {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery(), corsMiddleware())

	server := &Server{
		engine:       engine,
		agentMgr:     agentMgr,
		configMgr:    configMgr,
		todoMgr:      todoMgr,
		convRegistry: NewConversationRegistry(),
		pushRegistry: NewPushRegistry(),
		devMode:      devMode,
		appDir:       appDir,
	}
	server.setupRoutes()
	return server
}

func (s *Server) staticFileSystem() (fs.FS, error) {
	if s.devMode {
		return os.DirFS(filepath.Join(s.appDir, "src", "static")), nil
	}
	return fs.Sub(embeddedStatic, "static")
}

func (s *Server) setupRoutes() {
	staticFS, err := s.staticFileSystem()
	if err != nil {
		panic(err)
	}

	s.engine.StaticFS("/static", http.FS(staticFS))
	s.engine.GET("/", func(c *gin.Context) {
		data, err := fs.ReadFile(staticFS, "index.html")
		if err != nil {
			c.String(http.StatusNotFound, "index.html not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	api := s.engine.Group("/api")
	api.POST("/chat", s.handleChat)
	api.POST("/chat/stop", s.handleStopChat)
	api.GET("/chat/push", s.handlePush)
	api.GET("/conversations", s.handleListConversations)
	api.GET("/conversations/:id", s.handleGetConversation)
	api.DELETE("/conversations/:id", s.handleDeleteConversation)
	api.GET("/config", s.handleGetConfig)
	api.PUT("/config/tools/:toolName", s.handleUpdateToolConfig)
	api.POST("/agent/reload", s.handleReload)
	api.GET("/todos", s.handleGetTodos)
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
	if s.httpSrv == nil {
		return nil
	}
	return s.httpSrv.Shutdown(ctx)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
