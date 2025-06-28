package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/analyst/pcf-mcp/internal/config"
)

// TestHTTPTransport tests the HTTP transport functionality
func TestHTTPTransport(t *testing.T) {
	// Create a test server
	cfg := config.ServerConfig{
		Transport:    "http",
		Host:         "localhost",
		Port:         0, // Use any available port
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	
	// Register a test tool
	testTool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"properties": map[string]interface{}{
				"message": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []string{"message"},
		},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			message, _ := params["message"].(string)
			return map[string]interface{}{
				"response": "Echo: " + message,
			}, nil
		},
	}
	
	err = server.RegisterTool(testTool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}
	
	// Create HTTP handler
	handler := server.HTTPHandler()
	
	// Create test server
	ts := httptest.NewServer(handler)
	defer ts.Close()
	
	// Test cases
	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name:           "GET /health",
			method:         "GET",
			path:           "/health",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp["status"] != "healthy" {
					t.Errorf("Expected status 'healthy', got %v", resp["status"])
				}
			},
		},
		{
			name:           "GET /info",
			method:         "GET",
			path:           "/info",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp["name"] != "pcf-mcp" {
					t.Errorf("Expected name 'pcf-mcp', got %v", resp["name"])
				}
				if resp["version"] != Version {
					t.Errorf("Expected version '%s', got %v", Version, resp["version"])
				}
			},
		},
		{
			name:           "GET /tools",
			method:         "GET",
			path:           "/tools",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var resp struct {
					Tools []map[string]interface{} `json:"tools"`
				}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(resp.Tools) != 1 {
					t.Fatalf("Expected 1 tool, got %d", len(resp.Tools))
				}
				if resp.Tools[0]["name"] != "test_tool" {
					t.Errorf("Expected tool name 'test_tool', got %v", resp.Tools[0]["name"])
				}
			},
		},
		{
			name:   "POST /tools/test_tool",
			method: "POST",
			path:   "/tools/test_tool",
			body: map[string]interface{}{
				"message": "Hello, MCP!",
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				result, ok := resp["result"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected result to be a map, got %T", resp["result"])
				}
				if result["response"] != "Echo: Hello, MCP!" {
					t.Errorf("Expected response 'Echo: Hello, MCP!', got %v", result["response"])
				}
			},
		},
		{
			name:           "POST /tools/nonexistent",
			method:         "POST",
			path:           "/tools/nonexistent",
			body:           map[string]interface{}{},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp["error"] == nil {
					t.Error("Expected error in response")
				}
			},
		},
		{
			name:           "Invalid method",
			method:         "PUT",
			path:           "/tools/test_tool",
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Not found path",
			method:         "GET",
			path:           "/unknown",
			expectedStatus: http.StatusNotFound,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader *bytes.Reader
			if tt.body != nil {
				bodyBytes, err := json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("Failed to marshal body: %v", err)
				}
				bodyReader = bytes.NewReader(bodyBytes)
			} else {
				bodyReader = bytes.NewReader([]byte{})
			}
			
			req, err := http.NewRequest(tt.method, ts.URL+tt.path, bodyReader)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
			
			if tt.validateBody != nil {
				var body bytes.Buffer
				_, err := body.ReadFrom(resp.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}
				tt.validateBody(t, body.Bytes())
			}
		})
	}
}

// TestHTTPTransportCORS tests CORS headers
func TestHTTPTransportCORS(t *testing.T) {
	cfg := config.ServerConfig{
		Transport:    "http",
		Host:         "localhost",
		Port:         0,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	
	handler := server.HTTPHandler()
	ts := httptest.NewServer(handler)
	defer ts.Close()
	
	// Test preflight request
	req, err := http.NewRequest("OPTIONS", ts.URL+"/tools", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	// Check CORS headers
	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}
	
	for header, expected := range expectedHeaders {
		actual := resp.Header.Get(header)
		if actual != expected {
			t.Errorf("Expected header %s to be '%s', got '%s'", header, expected, actual)
		}
	}
}

// TestHTTPTransportAuthentication tests authentication if enabled
func TestHTTPTransportAuthentication(t *testing.T) {
	cfg := config.ServerConfig{
		Transport:    "http",
		Host:         "localhost",
		Port:         0,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		AuthRequired: true,
		AuthToken:    "test-token-123",
	}
	
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	
	handler := server.HTTPHandler()
	ts := httptest.NewServer(handler)
	defer ts.Close()
	
	tests := []struct {
		name           string
		path           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "Valid token",
			path:           "/info",
			authHeader:     "Bearer test-token-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid token",
			path:           "/info",
			authHeader:     "Bearer wrong-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing token",
			path:           "/info",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid format",
			path:           "/info",
			authHeader:     "test-token-123",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Health endpoint without auth",
			path:           "/health",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Metrics endpoint without auth",
			path:           "/metrics",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", ts.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// TestHTTPTransportMetrics tests that metrics are properly recorded
func TestHTTPTransportMetrics(t *testing.T) {
	cfg := config.ServerConfig{
		Transport:    "http",
		Host:         "localhost",
		Port:         0,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	
	// Register a test tool
	testTool := Tool{
		Name:        "metrics_test",
		Description: "Tool for testing metrics",
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{"status": "ok"}, nil
		},
	}
	
	err = server.RegisterTool(testTool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}
	
	handler := server.HTTPHandler()
	ts := httptest.NewServer(handler)
	defer ts.Close()
	
	// Make several requests
	client := &http.Client{Timeout: 5 * time.Second}
	for i := 0; i < 5; i++ {
		req, err := http.NewRequest("POST", ts.URL+"/tools/metrics_test", bytes.NewReader([]byte("{}")))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		resp.Body.Close()
	}
	
	// Check metrics endpoint
	req, err := http.NewRequest("GET", ts.URL+"/metrics", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	// Verify that metrics are present
	var body bytes.Buffer
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	
	bodyStr := body.String()
	expectedMetrics := []string{
		"http_requests_total",
		"http_request_duration_seconds",
	}
	
	for _, metric := range expectedMetrics {
		if !bytes.Contains([]byte(bodyStr), []byte(metric)) {
			t.Errorf("Expected metric '%s' not found in response", metric)
		}
	}
}