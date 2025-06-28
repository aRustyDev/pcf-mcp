package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/analyst/pcf-mcp/internal/config"
	"github.com/analyst/pcf-mcp/internal/mcp"
	"github.com/analyst/pcf-mcp/internal/mcp/tools"
	"github.com/analyst/pcf-mcp/internal/observability"
	"github.com/analyst/pcf-mcp/internal/pcf"
)

// main is the entry point for the PCF-MCP server application
func main() {
	// Create configuration
	cfg := config.New()
	
	// Load configuration from various sources
	// 1. Load from config file if specified
	if configFile := os.Getenv("PCF_MCP_CONFIG_FILE"); configFile != "" {
		if err := cfg.LoadFromFile(configFile); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config file: %v\n", err)
			os.Exit(1)
		}
	}
	
	// 2. Load from environment variables
	if err := cfg.LoadFromEnvironment(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load environment config: %v\n", err)
		os.Exit(1)
	}
	
	// 3. Load from CLI arguments
	if err := cfg.LoadFromCLI(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse CLI arguments: %v\n", err)
		os.Exit(1)
	}
	
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		os.Exit(1)
	}
	
	// Initialize logging
	logger, err := observability.NewLogger(cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	
	// Set as global logger
	observability.SetGlobalLogger(logger)
	
	logger.Info("PCF-MCP Server starting",
		"version", mcp.Version,
		"transport", cfg.Server.Transport,
	)
	
	// Initialize metrics
	metrics, err := observability.InitMetrics(cfg.Metrics)
	if err != nil {
		logger.Error("Failed to initialize metrics", "error", err)
		os.Exit(1)
	}
	
	// Start metrics server if enabled
	if cfg.Metrics.Enabled {
		go func() {
			logger.Info("Starting metrics server",
				"port", cfg.Metrics.Port,
				"path", cfg.Metrics.Path,
			)
			if err := metrics.StartServer(cfg.Metrics); err != nil {
				logger.Error("Metrics server error", "error", err)
			}
		}()
	}
	
	// Initialize tracing
	var tracingShutdown func(context.Context) error
	if cfg.Tracing.Enabled {
		tracingShutdown, err = observability.InitTracing(cfg.Tracing)
		if err != nil {
			logger.Error("Failed to initialize tracing", "error", err)
			os.Exit(1)
		}
		logger.Info("Tracing initialized",
			"exporter", cfg.Tracing.Exporter,
			"endpoint", cfg.Tracing.Endpoint,
			"sampling_rate", cfg.Tracing.SamplingRate,
		)
	}
	
	// Create PCF client
	pcfClient, err := pcf.NewClient(cfg.PCF)
	if err != nil {
		logger.Error("Failed to create PCF client", "error", err)
		os.Exit(1)
	}
	
	// Create MCP server
	mcpServer, err := mcp.NewServer(cfg.Server)
	if err != nil {
		logger.Error("Failed to create MCP server", "error", err)
		os.Exit(1)
	}
	
	// Set metrics on server
	mcpServer.SetMetrics(metrics)
	
	// Register all tools
	if err := tools.RegisterAllTools(mcpServer, pcfClient); err != nil {
		logger.Error("Failed to register tools", "error", err)
		os.Exit(1)
	}
	
	logger.Info("Registered MCP tools", "count", len(mcpServer.ListTools()))
	
	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		sig := <-sigChan
		logger.Info("Received signal, shutting down", "signal", sig)
		cancel()
	}()
	
	// Start the server
	logger.Info("Starting MCP server", "transport", cfg.Server.Transport)
	
	if err := mcpServer.Start(ctx); err != nil && err != context.Canceled {
		logger.Error("Server error", "error", err)
		os.Exit(1)
	}
	
	// Cleanup tracing if enabled
	if tracingShutdown != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracingShutdown(shutdownCtx); err != nil {
			logger.Error("Failed to shutdown tracing", "error", err)
		}
	}
	
	logger.Info("PCF-MCP Server stopped")
}