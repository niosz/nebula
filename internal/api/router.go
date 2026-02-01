package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nebula/nebula/internal/auth"
	"github.com/nebula/nebula/internal/config"
	"github.com/nebula/nebula/internal/files"
	"github.com/nebula/nebula/internal/metrics"
	"github.com/nebula/nebula/internal/packages"
	"github.com/nebula/nebula/internal/process"
	"github.com/nebula/nebula/internal/service"
	"github.com/nebula/nebula/internal/terminal"
	"github.com/nebula/nebula/internal/updater"
	"github.com/nebula/nebula/internal/websocket"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Router holds all route handlers and dependencies
type Router struct {
	engine           *gin.Engine
	config           *config.Manager
	metricsHandler   *MetricsHandler
	processHandler   *ProcessHandler
	serviceHandler   *ServiceHandler
	filesHandler     *FilesHandler
	packagesHandler  *PackagesHandler
	terminalHandler  *TerminalHandler
	systemHandler    *SystemHandler
	authHandler      *AuthHandler
	hub              *websocket.Hub
	terminalHub      *websocket.TerminalHub
	metricsCollector *metrics.Collector
	privilegeManager *auth.PrivilegeManager
}

// NewRouter creates a new router with all dependencies
func NewRouter(
	cfg *config.Manager,
	store interface{},
	metricsCollector *metrics.Collector,
	processManager *process.Manager,
	serviceManager service.Manager,
	filesManager *files.Manager,
	packagesManager packages.Manager,
	terminalManager *terminal.Manager,
	upd *updater.Updater,
	privilegeManager *auth.PrivilegeManager,
) *Router {
	// Set Gin mode based on config
	if cfg.Get().Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(corsMiddleware())
	engine.Use(loggerMiddleware())

	hub := websocket.NewHub()
	terminalHub := websocket.NewTerminalHub()

	r := &Router{
		engine:           engine,
		config:           cfg,
		hub:              hub,
		terminalHub:      terminalHub,
		metricsCollector: metricsCollector,
		privilegeManager: privilegeManager,
		metricsHandler:   NewMetricsHandler(metricsCollector),
		processHandler:   NewProcessHandler(processManager),
		serviceHandler:   NewServiceHandler(serviceManager),
		filesHandler:     NewFilesHandler(filesManager),
		packagesHandler:  NewPackagesHandler(packagesManager),
		terminalHandler:  NewTerminalHandler(terminalManager, terminalHub),
		systemHandler:    NewSystemHandler(cfg, metricsCollector, upd),
		authHandler:      NewAuthHandler(privilegeManager),
	}

	r.setupRoutes()
	return r
}

// setupRoutes configures all routes
func (r *Router) setupRoutes() {
	// Auth middleware (optional)
	authMiddleware := r.authMiddleware()

	// API v1 group
	v1 := r.engine.Group("/api/v1")
	if r.config.Get().Auth.Enabled {
		v1.Use(authMiddleware)
	}

	// Metrics routes
	metricsGroup := v1.Group("/metrics")
	{
		metricsGroup.GET("/cpu", r.metricsHandler.GetCPU)
		metricsGroup.GET("/memory", r.metricsHandler.GetMemory)
		metricsGroup.GET("/disk", r.metricsHandler.GetDisk)
		metricsGroup.GET("/network", r.metricsHandler.GetNetwork)
		metricsGroup.GET("/all", r.metricsHandler.GetAll)
		metricsGroup.GET("/history", r.metricsHandler.GetHistory)
	}

	// Process routes
	processGroup := v1.Group("/processes")
	{
		processGroup.GET("", r.processHandler.List)
		processGroup.GET("/search", r.processHandler.Search)
		processGroup.GET("/:pid", r.processHandler.Get)
		processGroup.POST("/:pid/kill", r.processHandler.Kill)
		processGroup.GET("/:pid/tree", r.processHandler.Tree)
	}

	// Service routes
	serviceGroup := v1.Group("/services")
	{
		serviceGroup.GET("", r.serviceHandler.List)
		serviceGroup.GET("/:name", r.serviceHandler.Get)
		serviceGroup.POST("/:name/start", r.serviceHandler.Start)
		serviceGroup.POST("/:name/stop", r.serviceHandler.Stop)
		serviceGroup.POST("/:name/restart", r.serviceHandler.Restart)
		serviceGroup.POST("/:name/enable", r.serviceHandler.Enable)
		serviceGroup.POST("/:name/disable", r.serviceHandler.Disable)
		serviceGroup.GET("/:name/logs", r.serviceHandler.Logs)
	}

	// Files routes
	filesGroup := v1.Group("/files")
	{
		filesGroup.GET("/list", r.filesHandler.List)
		filesGroup.GET("/info", r.filesHandler.Info)
		filesGroup.GET("/download", r.filesHandler.Download)
		filesGroup.POST("/upload", r.filesHandler.Upload)
		filesGroup.POST("/mkdir", r.filesHandler.Mkdir)
		filesGroup.DELETE("/delete", r.filesHandler.Delete)
		filesGroup.PUT("/rename", r.filesHandler.Rename)
		filesGroup.GET("/read", r.filesHandler.Read)
		filesGroup.PUT("/write", r.filesHandler.Write)
	}

	// Packages routes
	packagesGroup := v1.Group("/packages")
	{
		packagesGroup.GET("", r.packagesHandler.List)
		packagesGroup.GET("/search", r.packagesHandler.Search)
		packagesGroup.GET("/info", r.packagesHandler.Info)
		packagesGroup.GET("/type", r.packagesHandler.GetType)
		packagesGroup.POST("/install", r.packagesHandler.Install)
		packagesGroup.DELETE("/remove", r.packagesHandler.Remove)
		packagesGroup.POST("/update", r.packagesHandler.Update)
		packagesGroup.POST("/upgrade-all", r.packagesHandler.UpgradeAll)
	}

	// Terminal routes
	terminalGroup := v1.Group("/terminal")
	{
		terminalGroup.GET("/shells", r.terminalHandler.GetShells)
		terminalGroup.GET("/sessions", r.terminalHandler.GetSessions)
	}

	// System routes
	v1.GET("/system/info", r.systemHandler.GetSystemInfo)
	v1.GET("/config", r.systemHandler.GetConfig)
	v1.POST("/config/reload", r.systemHandler.ReloadConfig)
	v1.GET("/update/check", r.systemHandler.CheckUpdate)
	v1.POST("/update/apply", r.systemHandler.ApplyUpdate)
	v1.GET("/version", r.systemHandler.GetVersion)

	// Auth routes
	authGroup := v1.Group("/auth")
	{
		authGroup.GET("/status", r.authHandler.GetPrivilegeStatus)
		authGroup.POST("/credentials", r.authHandler.SetCredentials)
		authGroup.DELETE("/credentials", r.authHandler.ClearCredentials)
		authGroup.POST("/validate", r.authHandler.ValidateCredentials)
	}

	// WebSocket routes
	r.engine.GET("/ws/metrics", r.handleMetricsWebSocket)
	r.engine.GET("/ws/terminal", r.terminalHandler.HandleWebSocket)

	// Swagger
	r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

// handleMetricsWebSocket handles metrics WebSocket connections
func (r *Router) handleMetricsWebSocket(c *gin.Context) {
	clientID := c.Query("client")
	if clientID == "" {
		clientID = "anonymous"
	}
	r.hub.HandleWebSocket(c.Writer, c.Request, clientID)
}

// authMiddleware returns the authentication middleware
func (r *Router) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := r.config.Get()
		if !cfg.Auth.Enabled {
			c.Next()
			return
		}

		username, password, ok := c.Request.BasicAuth()
		if !ok || username != cfg.Auth.Username || password != cfg.Auth.Password {
			c.Header("WWW-Authenticate", `Basic realm="Nebula"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Next()
	}
}

// corsMiddleware returns CORS middleware
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// loggerMiddleware returns logging middleware
func loggerMiddleware() gin.HandlerFunc {
	return gin.Logger()
}

// Engine returns the Gin engine
func (r *Router) Engine() *gin.Engine {
	return r.engine
}

// Hub returns the WebSocket hub
func (r *Router) Hub() *websocket.Hub {
	return r.hub
}

// StartWebSocketHub starts the WebSocket hub
func (r *Router) StartWebSocketHub() {
	go r.hub.Run()
}

// BroadcastMetrics broadcasts metrics to all connected clients
func (r *Router) BroadcastMetrics(metrics interface{}) {
	r.hub.BroadcastJSON("metrics", metrics)
}
