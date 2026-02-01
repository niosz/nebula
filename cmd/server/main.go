package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/nebula/nebula/internal/api"
	"github.com/nebula/nebula/internal/auth"
	"github.com/nebula/nebula/internal/config"
	"github.com/nebula/nebula/internal/files"
	"github.com/nebula/nebula/internal/metrics"
	"github.com/nebula/nebula/internal/packages"
	"github.com/nebula/nebula/internal/process"
	"github.com/nebula/nebula/internal/service"
	"github.com/nebula/nebula/internal/storage"
	"github.com/nebula/nebula/internal/terminal"
	"github.com/nebula/nebula/internal/updater"
	"github.com/nebula/nebula/web"
)

// @title Nebula API
// @version 1.0
// @description System Administration Panel API
// @host localhost:8080
// @BasePath /api/v1
func main() {
	log.Println("Starting Nebula...")

	// Check for root/admin privileges (skip with NEBULA_NO_ROOT=1 for development)
	if os.Getenv("NEBULA_NO_ROOT") != "1" {
		if err := auth.RequireRoot(); err != nil {
			log.Fatalf("ERRORE: %v", err)
		}
		log.Println("Running with elevated privileges")
	} else {
		log.Println("WARNING: Running without root check (development mode)")
	}

	// Load configuration
	configPath := "config.yaml"
	if envPath := os.Getenv("NEBULA_CONFIG"); envPath != "" {
		configPath = envPath
	}

	// Initialize storage first (needed for config)
	store, err := storage.New("nebula.db")
	if err != nil {
		log.Printf("Warning: Failed to initialize storage: %v", err)
		// Continue without storage
	}
	if store != nil {
		defer store.Close()
	}

	// Initialize privilege manager
	privilegeManager := auth.NewPrivilegeManager(store)
	if privilegeManager.HasCredentials() {
		log.Println("Stored credentials found")
	}

	// Load configuration
	cfg, err := config.NewManager(configPath, store)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	appConfig := cfg.Get()
	log.Printf("Configuration loaded from %s", configPath)

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector(
		store,
		appConfig.Metrics.Interval,
		appConfig.Metrics.HistorySize,
	)

	// Initialize process manager
	processManager := process.NewManager()

	// Initialize service manager
	serviceManager, err := service.NewManager()
	if err != nil {
		log.Printf("Warning: Service manager not available: %v", err)
		// Continue with nil service manager
	}

	// Initialize file manager
	filesManager := files.NewManager(
		appConfig.Files.RootPath,
		appConfig.Files.MaxUploadSize,
		appConfig.Files.AllowedExtensions,
	)

	// Initialize package manager
	packagesManager, err := packages.DetectManager()
	if err != nil {
		log.Printf("Warning: Package manager not available: %v", err)
	}

	// Initialize terminal manager
	terminalManager := terminal.NewManager(
		appConfig.Terminal.MaxSessions,
		appConfig.Terminal.AllowedShells,
		appConfig.Terminal.DefaultShell,
	)

	// Initialize updater
	upd := updater.NewUpdater(
		appConfig.Updater.Enabled,
		appConfig.Updater.CheckInterval,
	)

	// Create router
	router := api.NewRouter(
		cfg,
		store,
		metricsCollector,
		processManager,
		serviceManager,
		filesManager,
		packagesManager,
		terminalManager,
		upd,
		privilegeManager,
	)

	// Register static files
	web.RegisterStaticRoutes(router.Engine())

	// Start WebSocket hub
	router.StartWebSocketHub()

	// Start metrics collector in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go metricsCollector.Start(ctx)

	// Broadcast metrics to WebSocket clients
	go func() {
		sub := metricsCollector.Subscribe()
		defer metricsCollector.Unsubscribe(sub)
		for m := range sub {
			router.BroadcastMetrics(m)
		}
	}()

	// Create HTTP server
	server := &http.Server{
		Addr:         appConfig.Address(),
		Handler:      router.Engine(),
		ReadTimeout:  appConfig.Server.ReadTimeout,
		WriteTimeout: appConfig.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on http://%s", appConfig.Address())
		log.Printf("Swagger UI: http://%s/swagger/index.html", appConfig.Address())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	sig := <-quit
	log.Printf("Received signal: %v", sig)

	// Handle SIGHUP for config reload
	if sig == syscall.SIGHUP {
		log.Println("Reloading configuration...")
		if err := cfg.Reload(); err != nil {
			log.Printf("Failed to reload config: %v", err)
		}
		// Continue running
		<-quit
	}

	// Graceful shutdown
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), appConfig.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Close terminal sessions
	terminalManager.Close()

	// Shutdown server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Cancel metrics collection
	cancel()

	log.Println("Server stopped")
}
