package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/analyst/pcf-mcp/internal/config"
)

// BenchmarkHTTPEndpoints benchmarks various HTTP endpoints
func BenchmarkHTTPEndpoints(b *testing.B) {
	cfg := config.ServerConfig{
		Transport: "http",
		Host:      "localhost",
		Port:      0,
	}

	server, err := NewServer(cfg)
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}

	// Register test tools
	testTool := Tool{
		Name:        "bench_tool",
		Description: "Benchmark test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"data": map[string]interface{}{
					"type": "string",
				},
			},
		},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{"status": "ok", "data": params}, nil
		},
	}
	server.RegisterTool(testTool)

	handler := server.HTTPHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	b.Run("Health", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			resp, err := http.Get(srv.URL + "/health")
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})

	b.Run("Info", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			resp, err := http.Get(srv.URL + "/info")
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})

	b.Run("ListTools", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			resp, err := http.Get(srv.URL + "/tools")
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})

	b.Run("ExecuteTool", func(b *testing.B) {
		payload := map[string]interface{}{
			"data": "test data",
		}
		body, _ := json.Marshal(payload)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := http.Post(srv.URL+"/tools/bench_tool", "application/json", bytes.NewReader(body))
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkConcurrentHTTPRequests benchmarks concurrent HTTP requests
func BenchmarkConcurrentHTTPRequests(b *testing.B) {
	cfg := config.ServerConfig{
		Transport:          "http",
		Host:               "localhost",
		Port:               0,
		MaxConcurrentTools: 10,
	}

	server, err := NewServer(cfg)
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}

	// Register multiple test tools
	for i := 0; i < 5; i++ {
		tool := Tool{
			Name:        fmt.Sprintf("tool_%d", i),
			Description: fmt.Sprintf("Test tool %d", i),
			InputSchema: map[string]interface{}{"type": "object"},
			Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				// Simulate some work
				return map[string]interface{}{"tool": i, "status": "ok"}, nil
			},
		}
		server.RegisterTool(tool)
	}

	handler := server.HTTPHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	concurrencyLevels := []int{1, 10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrency-%d", concurrency), func(b *testing.B) {
			b.SetParallelism(concurrency)
			b.RunParallel(func(pb *testing.PB) {
				client := &http.Client{}
				toolIndex := 0
				for pb.Next() {
					// Rotate through different tools
					toolName := fmt.Sprintf("tool_%d", toolIndex%5)
					toolIndex++

					resp, err := client.Post(srv.URL+"/tools/"+toolName, "application/json", bytes.NewReader([]byte("{}")))
					if err != nil {
						b.Fatal(err)
					}
					resp.Body.Close()
				}
			})
		})
	}
}

// BenchmarkHTTPPayloadSizes benchmarks different payload sizes
func BenchmarkHTTPPayloadSizes(b *testing.B) {
	cfg := config.ServerConfig{
		Transport: "http",
		Host:      "localhost",
		Port:      0,
	}

	server, err := NewServer(cfg)
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}

	// Register echo tool
	echoTool := Tool{
		Name:        "echo",
		Description: "Echo tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"data": map[string]interface{}{
					"type": "string",
				},
			},
		},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return params, nil
		},
	}
	server.RegisterTool(echoTool)

	handler := server.HTTPHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	payloadSizes := []int{
		1 * 1024,       // 1 KB
		10 * 1024,      // 10 KB
		100 * 1024,     // 100 KB
		1 * 1024 * 1024, // 1 MB
	}

	for _, size := range payloadSizes {
		b.Run(fmt.Sprintf("payload-%dKB", size/1024), func(b *testing.B) {
			// Generate payload of specified size
			data := make([]byte, size)
			for i := range data {
				data[i] = 'a'
			}

			payload := map[string]interface{}{
				"data": string(data),
			}
			body, _ := json.Marshal(payload)

			b.ResetTimer()
			b.SetBytes(int64(len(body)))

			for i := 0; i < b.N; i++ {
				resp, err := http.Post(srv.URL+"/tools/echo", "application/json", bytes.NewReader(body))
				if err != nil {
					b.Fatal(err)
				}
				resp.Body.Close()
			}
		})
	}
}

// BenchmarkHTTPMiddleware benchmarks the middleware stack
func BenchmarkHTTPMiddleware(b *testing.B) {
	testCases := []struct {
		name   string
		config config.ServerConfig
	}{
		{
			name: "NoAuth",
			config: config.ServerConfig{
				Transport:    "http",
				Host:         "localhost",
				Port:         0,
				AuthRequired: false,
			},
		},
		{
			name: "WithAuth",
			config: config.ServerConfig{
				Transport:    "http",
				Host:         "localhost",
				Port:         0,
				AuthRequired: true,
				AuthToken:    "test-token",
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			server, err := NewServer(tc.config)
			if err != nil {
				b.Fatalf("Failed to create server: %v", err)
			}

			handler := server.HTTPHandler()
			srv := httptest.NewServer(handler)
			defer srv.Close()

			req, _ := http.NewRequest("GET", srv.URL+"/health", nil)
			if tc.config.AuthRequired {
				req.Header.Set("Authorization", "Bearer "+tc.config.AuthToken)
			}

			client := &http.Client{}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				resp, err := client.Do(req)
				if err != nil {
					b.Fatal(err)
				}
				resp.Body.Close()
			}
		})
	}
}