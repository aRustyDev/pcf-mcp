// Package observability provides logging, metrics, and tracing infrastructure
// for the PCF-MCP server. This package implements structured logging using
// slog with support for JSON and text formats, configurable log levels,
// and Kubernetes-friendly output.
package observability

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/aRustyDev/pcf-mcp/internal/config"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// loggerKey is the context key for storing the logger
const loggerKey contextKey = "logger"

// NewLogger creates a new structured logger based on the provided configuration.
// It supports JSON and text output formats, configurable log levels, and
// optional source code location tracking.
func NewLogger(cfg config.LoggingConfig) (*slog.Logger, error) {
	return NewLoggerWithWriter(cfg, os.Stdout)
}

// NewLoggerWithWriter creates a new logger with a custom writer.
// This is useful for testing or directing logs to specific outputs.
func NewLoggerWithWriter(cfg config.LoggingConfig, w io.Writer) (*slog.Logger, error) {
	// Parse and validate log level
	level, err := parseLogLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	// Configure handler options
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	// Create handler based on format
	var handler slog.Handler
	switch strings.ToLower(cfg.Format) {
	case "json":
		handler = slog.NewJSONHandler(w, opts)
	case "text":
		handler = slog.NewTextHandler(w, opts)
	default:
		return nil, fmt.Errorf("invalid log format: %s (must be 'json' or 'text')", cfg.Format)
	}

	// Create and return logger
	logger := slog.New(handler)
	return logger, nil
}

// parseLogLevel converts a string log level to slog.Level
func parseLogLevel(level string) (slog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid log level: %s", level)
	}
}

// WithLogger stores a logger in the context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves a logger from the context.
// If no logger is found, it returns the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}

	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok && logger != nil {
		return logger
	}

	return slog.Default()
}

// SetGlobalLogger sets the global default logger.
// This affects all code that uses slog.Default().
func SetGlobalLogger(logger *slog.Logger) {
	slog.SetDefault(logger)
}

// LoggerMiddleware is a helper type for adding consistent fields to all logs
type LoggerMiddleware struct {
	logger *slog.Logger
	fields []any
}

// NewLoggerMiddleware creates a new logger middleware with common fields
func NewLoggerMiddleware(logger *slog.Logger, fields ...any) *LoggerMiddleware {
	return &LoggerMiddleware{
		logger: logger,
		fields: fields,
	}
}

// With returns a new LoggerMiddleware with additional fields
func (lm *LoggerMiddleware) With(fields ...any) *LoggerMiddleware {
	newFields := make([]any, len(lm.fields)+len(fields))
	copy(newFields, lm.fields)
	copy(newFields[len(lm.fields):], fields)

	return &LoggerMiddleware{
		logger: lm.logger,
		fields: newFields,
	}
}

// Debug logs at debug level with middleware fields
func (lm *LoggerMiddleware) Debug(msg string, fields ...any) {
	lm.logger.Debug(msg, append(lm.fields, fields...)...)
}

// Info logs at info level with middleware fields
func (lm *LoggerMiddleware) Info(msg string, fields ...any) {
	lm.logger.Info(msg, append(lm.fields, fields...)...)
}

// Warn logs at warn level with middleware fields
func (lm *LoggerMiddleware) Warn(msg string, fields ...any) {
	lm.logger.Warn(msg, append(lm.fields, fields...)...)
}

// Error logs at error level with middleware fields
func (lm *LoggerMiddleware) Error(msg string, fields ...any) {
	lm.logger.Error(msg, append(lm.fields, fields...)...)
}

// Logger returns the underlying slog.Logger
func (lm *LoggerMiddleware) Logger() *slog.Logger {
	return lm.logger.With(lm.fields...)
}

// Common log field keys for consistency across the application
const (
	// FieldRequestID is the key for request ID in logs
	FieldRequestID = "request_id"

	// FieldUserID is the key for user ID in logs
	FieldUserID = "user_id"

	// FieldMethod is the key for HTTP method in logs
	FieldMethod = "method"

	// FieldPath is the key for request path in logs
	FieldPath = "path"

	// FieldStatus is the key for response status in logs
	FieldStatus = "status"

	// FieldDuration is the key for request duration in logs
	FieldDuration = "duration_ms"

	// FieldError is the key for error details in logs
	FieldError = "error"

	// FieldTool is the key for MCP tool name in logs
	FieldTool = "tool"

	// FieldProject is the key for PCF project ID in logs
	FieldProject = "project_id"

	// FieldHost is the key for target host in logs
	FieldHost = "host"

	// FieldComponent is the key for component name in logs
	FieldComponent = "component"
)

// LogError is a helper function to log errors with consistent formatting
func LogError(logger *slog.Logger, msg string, err error, fields ...any) {
	allFields := append([]any{FieldError, err.Error()}, fields...)
	logger.Error(msg, allFields...)
}

// LogRequest logs HTTP request details
func LogRequest(logger *slog.Logger, method, path string, fields ...any) {
	allFields := append([]any{
		FieldMethod, method,
		FieldPath, path,
	}, fields...)
	logger.Info("request received", allFields...)
}

// LogResponse logs HTTP response details
func LogResponse(logger *slog.Logger, method, path string, status int, duration int64, fields ...any) {
	allFields := append([]any{
		FieldMethod, method,
		FieldPath, path,
		FieldStatus, status,
		FieldDuration, duration,
	}, fields...)

	// Choose log level based on status code
	switch {
	case status >= 500:
		logger.Error("request completed", allFields...)
	case status >= 400:
		logger.Warn("request completed", allFields...)
	default:
		logger.Info("request completed", allFields...)
	}
}
