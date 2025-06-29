package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	// HTTP header names
	headerContentType   = "Content-Type"
	headerAuthorization = "Authorization"

	// Content types
	contentTypeJSON = "application/json"

	// Bearer token prefix
	bearerPrefix = "Bearer "
)

// httpMetrics holds HTTP-specific Prometheus metrics
type httpMetrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	registry        *prometheus.Registry
}

// newHTTPMetrics creates HTTP metrics with a dedicated registry
func newHTTPMetrics() *httpMetrics {
	registry := prometheus.NewRegistry()

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	registry.MustRegister(requestsTotal)
	registry.MustRegister(requestDuration)

	return &httpMetrics{
		requestsTotal:   requestsTotal,
		requestDuration: requestDuration,
		registry:        registry,
	}
}

// HTTPHandler returns an HTTP handler for the MCP server
func (s *Server) HTTPHandler() http.Handler {
	mux := http.NewServeMux()

	// Initialize HTTP metrics
	httpMetrics := newHTTPMetrics()

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// Server info endpoint
	mux.HandleFunc("/info", s.handleInfo)

	// List tools endpoint
	mux.HandleFunc("/tools", s.handleTools)

	// Tool execution endpoint (pattern matches /tools/{toolName})
	mux.HandleFunc("/tools/", s.handleToolExecution)

	// Metrics endpoint with custom registry
	mux.Handle("/metrics", promhttp.HandlerFor(httpMetrics.registry, promhttp.HandlerOpts{}))

	// Wrap with middleware
	handler := s.corsMiddleware(mux)
	handler = s.authMiddleware(handler)
	handler = s.metricsMiddleware(handler, httpMetrics)
	handler = s.tracingMiddleware(handler)
	handler = s.loggingMiddleware(handler)

	return handler
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   Version,
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleInfo handles server info requests
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	caps := s.Capabilities()
	response := map[string]interface{}{
		"name":    s.Name(),
		"version": s.Version(),
		"capabilities": map[string]bool{
			"tools":     caps.Tools,
			"resources": caps.Resources,
			"prompts":   caps.Prompts,
		},
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleTools handles tool listing requests
func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tools := s.ListTools()
	toolList := make([]map[string]interface{}, 0, len(tools))

	for _, tool := range tools {
		toolInfo := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
		}
		if tool.InputSchema != nil {
			toolInfo["inputSchema"] = tool.InputSchema
		}
		toolList = append(toolList, toolInfo)
	}

	response := map[string]interface{}{
		"tools": toolList,
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleToolExecution handles tool execution requests
func (s *Server) handleToolExecution(w http.ResponseWriter, r *http.Request) {
	// Only allow POST and OPTIONS
	if r.Method == http.MethodOptions {
		// CORS preflight handled by middleware
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract tool name from path
	path := strings.TrimPrefix(r.URL.Path, "/tools/")
	if path == "" || strings.Contains(path, "/") {
		s.writeError(w, http.StatusNotFound, "Tool not found")
		return
	}

	// Parse request body
	var params map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Execute tool
	result, err := s.ExecuteTool(r.Context(), path, params)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	response := map[string]interface{}{
		"result": result,
	}

	s.writeJSON(w, http.StatusOK, response)
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// authMiddleware handles authentication if enabled
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if not required
		if !s.config.AuthRequired {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for health and metrics endpoints
		if r.URL.Path == "/health" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		// Check Authorization header
		authHeader := r.Header.Get(headerAuthorization)
		if authHeader == "" {
			s.writeError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		// Validate Bearer token
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			s.writeError(w, http.StatusUnauthorized, "Invalid authorization format")
			return
		}

		token := strings.TrimPrefix(authHeader, bearerPrefix)
		if token != s.config.AuthToken {
			s.writeError(w, http.StatusUnauthorized, "Invalid authorization token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// metricsMiddleware records HTTP metrics
func (s *Server) metricsMiddleware(next http.Handler, metrics *httpMetrics) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Handle request
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		status := fmt.Sprintf("%d", wrapped.statusCode)

		metrics.requestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
		metrics.requestDuration.WithLabelValues(r.Method, r.URL.Path, status).Observe(duration)
	})
}

// tracingMiddleware adds distributed tracing
func (s *Server) tracingMiddleware(next http.Handler) http.Handler {
	tracer := otel.Tracer("pcf-mcp-http")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path),
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.user_agent", r.UserAgent()),
			),
		)
		defer span.End()

		// Pass context with span
		r = r.WithContext(ctx)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Handle request
		next.ServeHTTP(wrapped, r)

		// Add response attributes
		span.SetAttributes(
			attribute.Int("http.status_code", wrapped.statusCode),
		)
	})
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Handle request
		next.ServeHTTP(wrapped, r)

		// Log request
		duration := time.Since(start)
		slog.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration", duration,
			"remote_addr", r.RemoteAddr,
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// writeJSON writes a JSON response
func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
	}
}

// writeError writes an error response
func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	response := map[string]interface{}{
		"error": message,
	}
	s.writeJSON(w, status, response)
}

// StartHTTP starts the HTTP server
func (s *Server) StartHTTP(ctx context.Context) error {
	// Build address from host and port
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      s.HTTPHandler(),
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		slog.Info("Starting HTTP server", "address", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		// Graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		slog.Info("Shutting down HTTP server")
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("HTTP server shutdown error: %w", err)
		}
		return nil

	case err := <-errCh:
		return err
	}
}
