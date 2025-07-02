package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/health"
	"github.com/dmmcquay/katago-mcp/internal/katago"
	"github.com/dmmcquay/katago-mcp/internal/logging"
	mcptools "github.com/dmmcquay/katago-mcp/internal/mcp"
	"github.com/dmmcquay/katago-mcp/internal/metrics"
	"github.com/dmmcquay/katago-mcp/internal/ratelimit"
	httpserver "github.com/dmmcquay/katago-mcp/internal/server"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	// Version information injected at build time.
	GitCommit string = "unknown"
	BuildTime string = "unknown"
)

func main() {
	// Parse command line flags
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.Parse()

	// Handle version flag
	if showVersion {
		fmt.Printf("katago-mcp version 0.1.0\n")
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Build time: %s\n", BuildTime)
		os.Exit(0)
	}

	// Load configuration
	configPath := config.GetConfigPath()
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Create logger using new factory
	logConfig := &logging.Config{
		Level:   cfg.Logging.Level,
		Format:  logging.LogFormat(os.Getenv("KATAGO_LOG_FORMAT")), // Will default to JSON if not set
		Service: cfg.Server.Name,
		Version: cfg.Server.Version,
		Prefix:  cfg.Logging.Prefix,
	}
	logger := logging.NewLoggerFromConfig(logConfig)
	logger.Info("Starting KataGo MCP Server version %s (commit: %s, built: %s)",
		cfg.Server.Version, GitCommit, BuildTime)

	// Detect KataGo installation
	logger.Info("Detecting KataGo installation...")
	detection, err := katago.DetectKataGo()
	if err != nil {
		logger.Error("KataGo detection failed: %v", err)
		logger.Info("\n%s", katago.GetInstallationInstructions())
		os.Exit(1)
	}

	// Log detection results
	if detection.BinaryPath != "" {
		logger.Info("Found KataGo binary: %s", detection.BinaryPath)
		if detection.Version != "" {
			logger.Info("KataGo version: %s", detection.Version)
		}
	}
	if detection.ModelPath != "" {
		logger.Info("Found model: %s", detection.ModelPath)
	}
	if detection.ConfigPath != "" {
		logger.Info("Found config: %s", detection.ConfigPath)
	}

	// Report any non-critical errors
	if len(detection.Errors) > 0 {
		logger.Warn("Detection warnings:")
		for _, err := range detection.Errors {
			logger.Warn("  %s", err)
		}
	}

	// Override with config values if specified
	if cfg.KataGo.BinaryPath != "" && cfg.KataGo.BinaryPath != "katago" {
		detection.BinaryPath = cfg.KataGo.BinaryPath
	}
	if cfg.KataGo.ModelPath != "" {
		detection.ModelPath = cfg.KataGo.ModelPath
	}
	if cfg.KataGo.ConfigPath != "" {
		detection.ConfigPath = cfg.KataGo.ConfigPath
	}

	// Update config with detected values
	if cfg.KataGo.BinaryPath == "katago" {
		cfg.KataGo.BinaryPath = detection.BinaryPath
	}
	if cfg.KataGo.ModelPath == "" {
		cfg.KataGo.ModelPath = detection.ModelPath
	}
	if cfg.KataGo.ConfigPath == "" {
		cfg.KataGo.ConfigPath = detection.ConfigPath
	}

	// Create KataGo engine
	engine := katago.NewEngine(&cfg.KataGo, logger)

	// Create metrics collector
	metricsCollector := metrics.NewCollector()

	// Create rate limiter
	rateLimiter := ratelimit.NewLimiter(&cfg.RateLimit, logger)

	// Set up health checker
	healthChecker := health.NewChecker(logger, cfg.Server.Version, GitCommit)

	// Register KataGo health check
	healthChecker.RegisterCheck("katago", func(ctx context.Context) error {
		return engine.Ping(ctx)
	})

	// Start HTTP health check server
	healthAddr := os.Getenv("KATAGO_HEALTH_ADDR")
	if healthAddr == "" {
		healthAddr = cfg.Server.HealthAddr
	}
	if healthAddr == "" {
		healthAddr = ":8080" // Default health check port
	}
	httpServer := httpserver.NewHTTPServer(healthAddr, logger, healthChecker)
	if err := httpServer.Start(); err != nil {
		logger.Error("Failed to start health check server", "error", err)
		os.Exit(1)
	}
	logger.Info("Health check server started", "addr", healthAddr)

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("Shutting down...")
		cancel()

		// Stop health check server
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := httpServer.Stop(shutdownCtx); err != nil {
			logger.Error("Failed to stop health check server", "error", err)
		}

		_ = engine.Stop()
	}()

	// Create MCP server
	mcpServer := server.NewMCPServer(
		cfg.Server.Name,
		cfg.Server.Version,
		server.WithLogging(),
	)

	// Create middleware
	middleware := mcptools.NewMiddleware(logger, metricsCollector, rateLimiter)

	// Create and register tools
	toolsHandler := mcptools.NewToolsHandler(engine, logger)
	toolsHandler.SetMiddleware(middleware)
	toolsHandler.RegisterTools(mcpServer)

	// Register health check tool
	healthTool := mcp.NewTool("health",
		mcp.WithDescription("Check server and KataGo health status"),
	)
	mcpServer.AddTool(healthTool, func(checkCtx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		status := "KataGo MCP Server Health Status\n"
		status += "==============================\n"
		status += fmt.Sprintf("Server Version: %s\n", cfg.Server.Version)
		status += fmt.Sprintf("Git Commit: %s\n", GitCommit)
		status += fmt.Sprintf("Build Time: %s\n", BuildTime)
		status += "\nKataGo Status:\n"
		status += fmt.Sprintf("  Binary: %s\n", cfg.KataGo.BinaryPath)
		if detection.Version != "" {
			status += fmt.Sprintf("  Version: %s\n", detection.Version)
		}
		status += fmt.Sprintf("  Model: %s\n", cfg.KataGo.ModelPath)
		status += fmt.Sprintf("  Config: %s\n", cfg.KataGo.ConfigPath)
		status += "\nEngine Status: "
		if engine.IsRunning() {
			status += "running\n"
		} else {
			status += "stopped\n"
		}

		// Add rate limit status
		if rateLimiter != nil {
			rlStatus := rateLimiter.GetStatus()
			status += "\nRate Limiting:\n"
			status += fmt.Sprintf("  Enabled: %v\n", rlStatus["enabled"])
			if enabled, ok := rlStatus["enabled"].(bool); ok && enabled {
				status += fmt.Sprintf("  Requests/min: %d\n", rlStatus["requestsPerMin"])
				status += fmt.Sprintf("  Burst size: %d\n", rlStatus["burstSize"])
				status += fmt.Sprintf("  Active clients: %d\n", rlStatus["activeClients"])
			}
		}

		return mcp.NewToolResultText(status), nil
	})

	// Start server
	logger.Info("KataGo MCP Server ready")

	// Serve with context for cancellation support
	done := make(chan error, 1)
	go func() {
		done <- server.ServeStdio(mcpServer)
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.Error("Server error", "error", err)
		}
	case <-ctx.Done():
		logger.Info("Server stopped by context cancellation")
	}

	_ = engine.Stop()
}
