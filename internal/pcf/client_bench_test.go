package pcf

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aRustyDev/pcf-mcp/internal/config"
)

// BenchmarkAPIClient benchmarks various API client operations
func BenchmarkAPIClient(b *testing.B) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/projects":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"1","name":"Test Project"}]`))
		case "/api/projects/1/hosts":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"1","ip":"192.168.1.1","hostname":"test.local"}]`))
		case "/api/projects/1/issues":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"1","title":"Test Issue","severity":"high"}]`))
		case "/api/projects/1/credentials":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"1","username":"admin","service":"ssh"}]`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	defer server.Close()

	cfg := config.PCFConfig{
		URL:     server.URL,
		APIKey:  "test-token",
		Timeout: 30 * time.Second,
	}
	client, err := NewClient(cfg)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()

	b.Run("ListProjects", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := client.ListProjects(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ListHosts", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := client.ListHosts(ctx, "1")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ListIssues", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := client.ListIssues(ctx, "1")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ListCredentials", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := client.ListCredentials(ctx, "1")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConcurrentRequests benchmarks concurrent API requests
func BenchmarkConcurrentRequests(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return appropriate response based on endpoint
		if r.URL.Path == "/api/projects" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"1","name":"Test Project"}]`))
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	defer server.Close()

	cfg := config.PCFConfig{
		URL:     server.URL,
		APIKey:  "test-token",
		Timeout: 30 * time.Second,
	}
	client, err := NewClient(cfg)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()

	// Use more reasonable concurrency levels to avoid port exhaustion
	concurrencyLevels := []int{1, 5, 10, 20}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrency-%d", concurrency), func(b *testing.B) {
			b.SetParallelism(concurrency)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, err := client.ListProjects(ctx)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

// BenchmarkJSONParsing benchmarks JSON parsing performance
func BenchmarkJSONParsing(b *testing.B) {
	testCases := []struct {
		name string
		size int
	}{
		{"small", 10},
		{"medium", 100},
		{"large", 1000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Generate JSON response
			projects := make([]string, tc.size)
			for i := 0; i < tc.size; i++ {
				projects[i] = fmt.Sprintf(`{"id":"%d","name":"Project %d"}`, i, i)
			}
			response := fmt.Sprintf(`[%s]`, joinStrings(projects, ","))

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(response))
			}))
			defer server.Close()

			cfg := config.PCFConfig{
				URL:     server.URL,
				APIKey:  "test-token",
				Timeout: 30 * time.Second,
			}
			client, err := NewClient(cfg)
			if err != nil {
				b.Fatal(err)
			}
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := client.ListProjects(ctx)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
