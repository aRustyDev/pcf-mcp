package mcp

import (
	"context"
	"fmt"
	"testing"

	"github.com/analyst/pcf-mcp/internal/config"
)

// BenchmarkToolExecution benchmarks tool execution
func BenchmarkToolExecution(b *testing.B) {
	cfg := config.ServerConfig{
		Transport: "stdio",
		MaxConcurrentTools: 10,
	}

	server, err := NewServer(cfg)
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}

	// Register a simple test tool
	testTool := Tool{
		Name:        "bench_tool",
		Description: "Benchmark test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"value": map[string]interface{}{
					"type": "string",
				},
			},
		},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			// Simulate some work
			result := make(map[string]interface{})
			if val, ok := params["value"].(string); ok {
				result["processed"] = val + "_processed"
			}
			return result, nil
		},
	}

	if err := server.RegisterTool(testTool); err != nil {
		b.Fatalf("Failed to register tool: %v", err)
	}

	ctx := context.Background()
	params := map[string]interface{}{
		"value": "test_input",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := server.ExecuteTool(ctx, "bench_tool", params)
			if err != nil {
				b.Errorf("Tool execution failed: %v", err)
			}
		}
	})
}

// BenchmarkConcurrentToolExecution benchmarks concurrent tool execution
func BenchmarkConcurrentToolExecution(b *testing.B) {
	concurrencyLevels := []int{1, 5, 10, 20}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrency-%d", concurrency), func(b *testing.B) {
			cfg := config.ServerConfig{
				Transport:          "stdio",
				MaxConcurrentTools: concurrency,
			}

			server, err := NewServer(cfg)
			if err != nil {
				b.Fatalf("Failed to create server: %v", err)
			}

			// Register test tool
			testTool := Tool{
				Name:        "concurrent_bench_tool",
				Description: "Concurrent benchmark test tool",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
				Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
					// Simulate variable work
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					default:
						return map[string]interface{}{"status": "ok"}, nil
					}
				},
			}

			if err := server.RegisterTool(testTool); err != nil {
				b.Fatalf("Failed to register tool: %v", err)
			}

			ctx := context.Background()
			params := map[string]interface{}{}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, _ = server.ExecuteTool(ctx, "concurrent_bench_tool", params)
				}
			})
		})
	}
}

// BenchmarkToolRegistration benchmarks tool registration
func BenchmarkToolRegistration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cfg := config.ServerConfig{
			Transport: "stdio",
		}

		server, err := NewServer(cfg)
		if err != nil {
			b.Fatalf("Failed to create server: %v", err)
		}

		tool := Tool{
			Name:        fmt.Sprintf("tool_%d", i),
			Description: "Test tool",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				return nil, nil
			},
		}

		if err := server.RegisterTool(tool); err != nil {
			b.Errorf("Failed to register tool: %v", err)
		}
	}
}

// BenchmarkToolValidation benchmarks input validation
func BenchmarkToolValidation(b *testing.B) {
	cfg := config.ServerConfig{
		Transport: "stdio",
	}

	server, err := NewServer(cfg)
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}

	// Register tool with complex schema
	tool := Tool{
		Name:        "validation_tool",
		Description: "Tool with complex validation",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":      "string",
					"minLength": 3,
					"maxLength": 50,
				},
				"age": map[string]interface{}{
					"type":    "integer",
					"minimum": 0,
					"maximum": 150,
				},
				"email": map[string]interface{}{
					"type":    "string",
					"format":  "email",
					"pattern": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
				},
				"tags": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
					"minItems": 1,
					"maxItems": 10,
				},
			},
			"required": []string{"name", "email"},
		},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return params, nil
		},
	}

	if err := server.RegisterTool(tool); err != nil {
		b.Fatalf("Failed to register tool: %v", err)
	}

	validParams := map[string]interface{}{
		"name":  "John Doe",
		"age":   30,
		"email": "john.doe@example.com",
		"tags":  []string{"tag1", "tag2"},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = server.ExecuteTool(ctx, "validation_tool", validParams)
	}
}