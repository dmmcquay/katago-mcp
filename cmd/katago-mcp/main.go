package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/katago"
	"github.com/dmmcquay/katago-mcp/internal/logging"
	"github.com/dmmcquay/katago-mcp/internal/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	// Version information injected at build time
	GitCommit string = "unknown"
	BuildTime string = "unknown"
)

func main() {
	// Load configuration
	configPath := config.GetConfigPath()
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Create logger
	logger := logging.NewLogger(cfg.Logging.Prefix, cfg.Logging.Level)
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

	// Create MCP server
	s := server.NewMCPServer(
		cfg.Server.Name,
		cfg.Server.Version,
		server.WithDescription(cfg.Server.Description),
	)

	// Register health check tool
	healthTool := server.NewTool("health",
		server.WithDescription("Check server and KataGo health status"),
	)
	s.AddTool(healthTool, func(ctx context.Context, req server.CallToolRequest) (*server.CallToolResult, error) {
		status := fmt.Sprintf("KataGo MCP Server Health Status\n")
		status += fmt.Sprintf("==============================\n")
		status += fmt.Sprintf("Server Version: %s\n", cfg.Server.Version)
		status += fmt.Sprintf("Git Commit: %s\n", GitCommit)
		status += fmt.Sprintf("Build Time: %s\n", BuildTime)
		status += fmt.Sprintf("\nKataGo Status:\n")
		status += fmt.Sprintf("  Binary: %s\n", detection.BinaryPath)
		if detection.Version != "" {
			status += fmt.Sprintf("  Version: %s\n", detection.Version)
		}
		status += fmt.Sprintf("  Model: %s\n", detection.ModelPath)
		status += fmt.Sprintf("  Config: %s\n", detection.ConfigPath)
		
		return server.NewToolResultText(status), nil
	})

	// TODO: Initialize KataGo engine when we implement it
	// TODO: Register analysis tools when we implement handlers

	// Start server
	logger.Info("KataGo MCP Server ready")
	if err := s.Serve(context.Background()); err != nil {
		logger.Fatal("Server error: %v", err)
	}
}