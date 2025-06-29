package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/analyst/pcf-mcp/internal/config"
	"golang.org/x/time/rate"
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     int           // requests per second
	burst    int           // burst size
	ttl      time.Duration // time to live for visitor entries
}

// visitor tracks rate limiting for a specific client
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rps int, burst int, ttl time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rps,
		burst:    burst,
		ttl:      ttl,
	}

	// Start cleanup goroutine
	go rl.cleanupVisitors()

	return rl
}

// getVisitor returns the rate limiter for a given IP
func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rate.Limit(rl.rate), rl.burst)
		rl.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupVisitors removes old entries from the visitors map
func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > rl.ttl {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns an HTTP middleware for rate limiting
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health and metrics endpoints
		if r.URL.Path == "/health" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		// Get client IP
		ip := getClientIP(r)
		limiter := rl.getVisitor(ip)

		if !limiter.Allow() {
			slog.Warn("Rate limit exceeded",
				"ip", ip,
				"path", r.URL.Path,
				"method", r.Method,
			)

			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ToolRateLimiter provides rate limiting for tool execution
type ToolRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     int // executions per minute
	burst    int
}

// NewToolRateLimiter creates a new tool rate limiter
func NewToolRateLimiter(rateLimit int, burst int) *ToolRateLimiter {
	return &ToolRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rateLimit,
		burst:    burst,
	}
}

// Allow checks if a tool execution is allowed
func (trl *ToolRateLimiter) Allow(toolName string) bool {
	trl.mu.Lock()
	defer trl.mu.Unlock()

	limiter, exists := trl.limiters[toolName]
	if !exists {
		// Convert rate per minute to rate per second
		ratePerSecond := float64(trl.rate) / 60.0
		limiter = rate.NewLimiter(rate.Limit(ratePerSecond), trl.burst)
		trl.limiters[toolName] = limiter
	}

	return limiter.Allow()
}

// Wait waits until a tool execution is allowed
func (trl *ToolRateLimiter) Wait(ctx context.Context, toolName string) error {
	trl.mu.Lock()
	limiter, exists := trl.limiters[toolName]
	if !exists {
		ratePerSecond := float64(trl.rate) / 60.0
		limiter = rate.NewLimiter(rate.Limit(ratePerSecond), trl.burst)
		trl.limiters[toolName] = limiter
	}
	trl.mu.Unlock()

	return limiter.Wait(ctx)
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}

	return r.RemoteAddr
}

// RateLimitedServer extends Server with rate limiting
type RateLimitedServer struct {
	*Server
	httpLimiter *RateLimiter
	toolLimiter *ToolRateLimiter
}

// NewRateLimitedServer creates a server with rate limiting
func NewRateLimitedServer(cfg config.ServerConfig, httpRPS int, toolRPM int) (*RateLimitedServer, error) {
	server, err := NewServer(cfg)
	if err != nil {
		return nil, err
	}

	return &RateLimitedServer{
		Server:      server,
		httpLimiter: NewRateLimiter(httpRPS, httpRPS*2, 10*time.Minute),
		toolLimiter: NewToolRateLimiter(toolRPM, 5),
	}, nil
}

// ExecuteTool executes a tool with rate limiting
func (rls *RateLimitedServer) ExecuteTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	// Apply tool rate limiting
	if err := rls.toolLimiter.Wait(ctx, name); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	return rls.Server.ExecuteTool(ctx, name, params)
}

// HTTPHandler returns an HTTP handler with rate limiting
func (rls *RateLimitedServer) HTTPHandler() http.Handler {
	handler := rls.Server.HTTPHandler()
	return rls.httpLimiter.Middleware(handler)
}