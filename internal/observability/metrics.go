// Package observability provides metrics collection using Prometheus
package observability

import (
	"fmt"
	"net/http"
	"time"

	"github.com/aRustyDev/pcf-mcp/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for the application
type Metrics struct {
	// RequestsTotal counts total HTTP requests
	RequestsTotal *prometheus.CounterVec

	// RequestDuration tracks HTTP request duration
	RequestDuration *prometheus.HistogramVec

	// ActiveConnections tracks current active connections
	ActiveConnections prometheus.Gauge

	// ToolExecutions counts tool executions
	ToolExecutions *prometheus.CounterVec

	// ToolErrors counts tool execution errors
	ToolErrors *prometheus.CounterVec

	// ToolDuration tracks tool execution duration
	ToolDuration *prometheus.HistogramVec

	// registry is the Prometheus registry
	registry *prometheus.Registry

	// enabled indicates if metrics collection is active
	enabled bool
}

// InitMetrics initializes the Prometheus metrics
func InitMetrics(cfg config.MetricsConfig) (*Metrics, error) {
	// Create custom registry
	registry := prometheus.NewRegistry()

	// Create metrics
	m := &Metrics{
		enabled:  cfg.Enabled,
		registry: registry,
	}

	if !cfg.Enabled {
		// Return no-op implementation
		return m, nil
	}

	// HTTP request metrics
	m.RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pcf_mcp_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	m.RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pcf_mcp_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// Connection metrics
	m.ActiveConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "pcf_mcp_active_connections",
			Help: "Current number of active connections",
		},
	)

	// Tool metrics
	m.ToolExecutions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pcf_mcp_tool_executions_total",
			Help: "Total number of tool executions",
		},
		[]string{"tool", "status"},
	)

	m.ToolErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pcf_mcp_tool_errors_total",
			Help: "Total number of tool execution errors",
		},
		[]string{"tool"},
	)

	m.ToolDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pcf_mcp_tool_duration_seconds",
			Help:    "Tool execution duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"tool"},
	)

	// Register all metrics
	registry.MustRegister(
		m.RequestsTotal,
		m.RequestDuration,
		m.ActiveConnections,
		m.ToolExecutions,
		m.ToolErrors,
		m.ToolDuration,
		// Also register standard Go metrics
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)

	return m, nil
}

// RecordRequest records an HTTP request metric
func (m *Metrics) RecordRequest(method, path string, status int, duration time.Duration) {
	if !m.enabled || m.RequestsTotal == nil {
		return
	}

	statusStr := fmt.Sprintf("%d", status)

	m.RequestsTotal.WithLabelValues(method, path, statusStr).Inc()
	m.RequestDuration.WithLabelValues(method, path, statusStr).Observe(duration.Seconds())
}

// RecordToolExecution records a tool execution metric
func (m *Metrics) RecordToolExecution(toolName string, success bool, duration time.Duration) {
	if !m.enabled || m.ToolExecutions == nil {
		return
	}

	status := "success"
	if !success {
		status = "error"
		m.ToolErrors.WithLabelValues(toolName).Inc()
	}

	m.ToolExecutions.WithLabelValues(toolName, status).Inc()
	m.ToolDuration.WithLabelValues(toolName).Observe(duration.Seconds())
}

// ConnectionOpened increments the active connections gauge
func (m *Metrics) ConnectionOpened() {
	if !m.enabled || m.ActiveConnections == nil {
		return
	}

	m.ActiveConnections.Inc()
}

// ConnectionClosed decrements the active connections gauge
func (m *Metrics) ConnectionClosed() {
	if !m.enabled || m.ActiveConnections == nil {
		return
	}

	m.ActiveConnections.Dec()
}

// Handler returns the Prometheus HTTP handler
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		Registry: m.registry,
	})
}

// StartServer starts the metrics HTTP server
func (m *Metrics) StartServer(cfg config.MetricsConfig) error {
	if !cfg.Enabled {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle(cfg.Path, m.Handler())

	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	return server.ListenAndServe()
}

// HTTPMiddleware is a middleware that records HTTP metrics
func (m *Metrics) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Serve request
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start)
		m.RecordRequest(r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.ResponseWriter.WriteHeader(code)
		rw.written = true
	}
}

// Write ensures WriteHeader is called
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}
