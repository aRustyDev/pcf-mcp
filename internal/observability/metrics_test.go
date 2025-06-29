package observability

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aRustyDev/pcf-mcp/internal/config"
)

// TestInitMetrics tests the initialization of metrics
func TestInitMetrics(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: true,
		Port:    9090,
		Path:    "/metrics",
	}

	metrics, err := InitMetrics(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize metrics: %v", err)
	}

	if metrics == nil {
		t.Fatal("InitMetrics returned nil")
	}

	// Verify metrics are registered
	if metrics.RequestsTotal == nil {
		t.Error("RequestsTotal metric not initialized")
	}

	if metrics.RequestDuration == nil {
		t.Error("RequestDuration metric not initialized")
	}

	if metrics.ActiveConnections == nil {
		t.Error("ActiveConnections gauge not initialized")
	}

	if metrics.ToolExecutions == nil {
		t.Error("ToolExecutions counter not initialized")
	}

	if metrics.ToolErrors == nil {
		t.Error("ToolErrors counter not initialized")
	}
}

// TestMetricsDisabled tests that metrics can be disabled
func TestMetricsDisabled(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: false,
	}

	metrics, err := InitMetrics(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize metrics: %v", err)
	}

	// With disabled metrics, we should still get a valid object
	// but it should be a no-op implementation
	if metrics == nil {
		t.Fatal("InitMetrics should return a no-op implementation when disabled")
	}
}

// TestRecordRequest tests recording HTTP request metrics
func TestRecordRequest(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: true,
		Port:    9090,
		Path:    "/metrics",
	}

	metrics, err := InitMetrics(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize metrics: %v", err)
	}

	// Record some requests
	metrics.RecordRequest("GET", "/api/projects", 200, 100*time.Millisecond)
	metrics.RecordRequest("POST", "/api/projects", 201, 150*time.Millisecond)
	metrics.RecordRequest("GET", "/api/projects", 500, 50*time.Millisecond)

	// Start metrics server
	server := httptest.NewServer(metrics.Handler())
	defer server.Close()

	// Fetch metrics
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read metrics: %v", err)
	}

	metricsOutput := string(body)

	// Verify metrics are present
	if !strings.Contains(metricsOutput, "pcf_mcp_requests_total") {
		t.Error("Metrics output missing pcf_mcp_requests_total")
	}

	if !strings.Contains(metricsOutput, "pcf_mcp_request_duration_seconds") {
		t.Error("Metrics output missing pcf_mcp_request_duration_seconds")
	}

	// Check for specific labels
	if !strings.Contains(metricsOutput, `method="GET"`) {
		t.Error("Metrics output missing GET method label")
	}

	if !strings.Contains(metricsOutput, `status="200"`) {
		t.Error("Metrics output missing 200 status label")
	}
}

// TestRecordToolExecution tests recording tool execution metrics
func TestRecordToolExecution(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: true,
		Port:    9090,
		Path:    "/metrics",
	}

	metrics, err := InitMetrics(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize metrics: %v", err)
	}

	// Record tool executions
	metrics.RecordToolExecution("list_projects", true, 50*time.Millisecond)
	metrics.RecordToolExecution("list_projects", true, 75*time.Millisecond)
	metrics.RecordToolExecution("create_project", false, 25*time.Millisecond)

	// Start metrics server
	server := httptest.NewServer(metrics.Handler())
	defer server.Close()

	// Fetch metrics
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read metrics: %v", err)
	}

	metricsOutput := string(body)

	// Verify tool metrics are present
	if !strings.Contains(metricsOutput, "pcf_mcp_tool_executions_total") {
		t.Error("Metrics output missing pcf_mcp_tool_executions_total")
	}

	if !strings.Contains(metricsOutput, "pcf_mcp_tool_errors_total") {
		t.Error("Metrics output missing pcf_mcp_tool_errors_total")
	}

	if !strings.Contains(metricsOutput, "pcf_mcp_tool_duration_seconds") {
		t.Error("Metrics output missing pcf_mcp_tool_duration_seconds")
	}

	// Check for tool name label
	if !strings.Contains(metricsOutput, `tool="list_projects"`) {
		t.Error("Metrics output missing list_projects tool label")
	}
}

// TestActiveConnections tests the active connections gauge
func TestActiveConnections(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: true,
		Port:    9090,
		Path:    "/metrics",
	}

	metrics, err := InitMetrics(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize metrics: %v", err)
	}

	// Simulate connection lifecycle
	metrics.ConnectionOpened()
	metrics.ConnectionOpened()
	metrics.ConnectionOpened()
	metrics.ConnectionClosed()

	// Start metrics server
	server := httptest.NewServer(metrics.Handler())
	defer server.Close()

	// Fetch metrics
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read metrics: %v", err)
	}

	metricsOutput := string(body)

	// Verify active connections metric
	if !strings.Contains(metricsOutput, "pcf_mcp_active_connections") {
		t.Error("Metrics output missing pcf_mcp_active_connections")
	}

	// Should show 2 active connections (3 opened - 1 closed)
	if !strings.Contains(metricsOutput, "pcf_mcp_active_connections 2") {
		t.Error("Active connections count should be 2")
	}
}

// TestMetricsServer tests the metrics HTTP server
func TestMetricsServer(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: true,
		Port:    9999, // Use a different port to avoid conflicts
		Path:    "/metrics",
	}

	metrics, err := InitMetrics(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize metrics: %v", err)
	}

	// Start metrics server
	go func() {
		if err := metrics.StartServer(cfg); err != nil && err != http.ErrServerClosed {
			t.Errorf("Metrics server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test fetching metrics
	resp, err := http.Get("http://localhost:9999/metrics")
	if err != nil {
		t.Fatalf("Failed to fetch metrics from server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test non-metrics endpoint
	resp2, err := http.Get("http://localhost:9999/health")
	if err != nil {
		t.Fatalf("Failed to fetch health endpoint: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-metrics path, got %d", resp2.StatusCode)
	}
}

// TestHTTPMiddleware tests the HTTP metrics middleware
func TestHTTPMiddleware(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: true,
		Port:    9090,
		Path:    "/metrics",
	}

	metrics, err := InitMetrics(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize metrics: %v", err)
	}

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate some work
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with middleware
	wrapped := metrics.HTTPMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Execute request
	wrapped.ServeHTTP(rr, req)

	// Verify response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Start metrics server to check recorded metrics
	server := httptest.NewServer(metrics.Handler())
	defer server.Close()

	// Fetch metrics
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read metrics: %v", err)
	}

	metricsOutput := string(body)

	// Verify request was recorded
	if !strings.Contains(metricsOutput, `path="/test"`) {
		t.Error("Metrics output missing /test path label")
	}
}
