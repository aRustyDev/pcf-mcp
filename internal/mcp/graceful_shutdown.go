package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// GracefulServer provides graceful shutdown capabilities
type GracefulServer struct {
	server         *Server
	httpServer     *http.Server
	shutdownChan   chan struct{}
	wg             sync.WaitGroup
	activeRequests sync.WaitGroup
}

// NewGracefulServer creates a server with graceful shutdown support
func NewGracefulServer(server *Server) *GracefulServer {
	return &GracefulServer{
		server:       server,
		shutdownChan: make(chan struct{}),
	}
}

// Run starts the server and handles graceful shutdown
func (gs *GracefulServer) Run(ctx context.Context) error {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create cancellable context
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle transport-specific startup
	switch gs.server.config.Transport {
	case "http":
		return gs.runHTTP(runCtx, sigChan)
	case "stdio":
		return gs.runStdio(runCtx, sigChan)
	default:
		return fmt.Errorf("unsupported transport: %s", gs.server.config.Transport)
	}
}

// runHTTP runs the HTTP server with graceful shutdown
func (gs *GracefulServer) runHTTP(ctx context.Context, sigChan chan os.Signal) error {
	addr := fmt.Sprintf("%s:%d", gs.server.config.Host, gs.server.config.Port)

	gs.httpServer = &http.Server{
		Addr:         addr,
		Handler:      gs.wrapHandler(gs.server.HTTPHandler()),
		ReadTimeout:  gs.server.config.ReadTimeout,
		WriteTimeout: gs.server.config.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	gs.wg.Add(1)
	go func() {
		defer gs.wg.Done()
		slog.Info("Starting HTTP server",
			"address", addr,
			"transport", "http",
		)
		if err := gs.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-ctx.Done():
		slog.Info("Context cancelled, initiating shutdown")
	case sig := <-sigChan:
		slog.Info("Received signal, initiating shutdown", "signal", sig)
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	}

	// Initiate graceful shutdown
	return gs.shutdown()
}

// runStdio runs the stdio server with graceful shutdown
func (gs *GracefulServer) runStdio(ctx context.Context, sigChan chan os.Signal) error {
	// Start stdio server in goroutine
	serverErr := make(chan error, 1)
	gs.wg.Add(1)
	go func() {
		defer gs.wg.Done()
		slog.Info("Starting stdio server", "transport", "stdio")
		if err := gs.server.Start(ctx); err != nil {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-ctx.Done():
		slog.Info("Context cancelled, initiating shutdown")
	case sig := <-sigChan:
		slog.Info("Received signal, initiating shutdown", "signal", sig)
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	}

	// Signal shutdown
	close(gs.shutdownChan)

	// Wait for server to finish
	gs.wg.Wait()

	return nil
}

// shutdown performs graceful shutdown
func (gs *GracefulServer) shutdown() error {
	slog.Info("Starting graceful shutdown")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Signal shutdown
	close(gs.shutdownChan)

	// Shutdown HTTP server if running
	if gs.httpServer != nil {
		// Wait for active requests to complete
		done := make(chan struct{})
		go func() {
			gs.activeRequests.Wait()
			close(done)
		}()

		select {
		case <-done:
			slog.Info("All active requests completed")
		case <-time.After(20 * time.Second):
			slog.Warn("Timeout waiting for active requests")
		}

		// Shutdown HTTP server
		if err := gs.httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("Error during HTTP server shutdown", "error", err)
			return err
		}
	}

	// Wait for all goroutines
	doneChan := make(chan struct{})
	go func() {
		gs.wg.Wait()
		close(doneChan)
	}()

	select {
	case <-doneChan:
		slog.Info("Graceful shutdown completed")
		return nil
	case <-shutdownCtx.Done():
		slog.Error("Shutdown timeout exceeded")
		return shutdownCtx.Err()
	}
}

// wrapHandler wraps the HTTP handler to track active requests
func (gs *GracefulServer) wrapHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if shutdown is in progress
		select {
		case <-gs.shutdownChan:
			http.Error(w, "Server is shutting down", http.StatusServiceUnavailable)
			return
		default:
		}

		// Track active request
		gs.activeRequests.Add(1)
		defer gs.activeRequests.Done()

		// Create request context that respects shutdown
		ctx := r.Context()
		reqCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Monitor for shutdown
		go func() {
			select {
			case <-gs.shutdownChan:
				cancel()
			case <-reqCtx.Done():
			}
		}()

		// Serve request with wrapped context
		handler.ServeHTTP(w, r.WithContext(reqCtx))
	})
}

// Shutdown initiates graceful shutdown
func (gs *GracefulServer) Shutdown(ctx context.Context) error {
	return gs.shutdown()
}

// ShutdownManager provides centralized shutdown coordination
type ShutdownManager struct {
	hooks    []func(context.Context) error
	mu       sync.Mutex
	shutdown bool
}

// NewShutdownManager creates a new shutdown manager
func NewShutdownManager() *ShutdownManager {
	return &ShutdownManager{
		hooks: make([]func(context.Context) error, 0),
	}
}

// RegisterHook registers a shutdown hook
func (sm *ShutdownManager) RegisterHook(hook func(context.Context) error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.shutdown {
		slog.Warn("Cannot register hook during shutdown")
		return
	}

	sm.hooks = append(sm.hooks, hook)
}

// Shutdown executes all shutdown hooks
func (sm *ShutdownManager) Shutdown(ctx context.Context) error {
	sm.mu.Lock()
	if sm.shutdown {
		sm.mu.Unlock()
		return nil
	}
	sm.shutdown = true
	hooks := make([]func(context.Context) error, len(sm.hooks))
	copy(hooks, sm.hooks)
	sm.mu.Unlock()

	slog.Info("Executing shutdown hooks", "count", len(hooks))

	var firstErr error
	for i, hook := range hooks {
		if err := hook(ctx); err != nil {
			slog.Error("Shutdown hook failed", "index", i, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}
