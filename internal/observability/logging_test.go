package observability

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/analyst/pcf-mcp/internal/config"
)

// TestNewLogger tests the creation of a new logger instance
func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		config config.LoggingConfig
	}{
		{
			name: "JSON format logger",
			config: config.LoggingConfig{
				Level:     "info",
				Format:    "json",
				AddSource: false,
			},
		},
		{
			name: "Text format logger",
			config: config.LoggingConfig{
				Level:     "debug",
				Format:    "text",
				AddSource: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)
			if err != nil {
				t.Fatalf("Failed to create logger: %v", err)
			}

			if logger == nil {
				t.Fatal("NewLogger returned nil")
			}
		})
	}
}

// TestLoggerLevels tests that the logger respects configured log levels
func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		name           string
		level          string
		logDebug       bool
		logInfo        bool
		logWarn        bool
		logError       bool
		expectedLogs   int
	}{
		{
			name:         "Debug level shows all",
			level:        "debug",
			logDebug:     true,
			logInfo:      true,
			logWarn:      true,
			logError:     true,
			expectedLogs: 4,
		},
		{
			name:         "Info level hides debug",
			level:        "info",
			logDebug:     true,
			logInfo:      true,
			logWarn:      true,
			logError:     true,
			expectedLogs: 3,
		},
		{
			name:         "Warn level shows warn and error",
			level:        "warn",
			logDebug:     true,
			logInfo:      true,
			logWarn:      true,
			logError:     true,
			expectedLogs: 2,
		},
		{
			name:         "Error level shows only error",
			level:        "error",
			logDebug:     true,
			logInfo:      true,
			logWarn:      true,
			logError:     true,
			expectedLogs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture log output
			var buf bytes.Buffer

			// Create logger with custom writer
			cfg := config.LoggingConfig{
				Level:  tt.level,
				Format: "json",
			}

			logger, err := NewLoggerWithWriter(cfg, &buf)
			if err != nil {
				t.Fatalf("Failed to create logger: %v", err)
			}

			// Log at different levels
			if tt.logDebug {
				logger.Debug("debug message", "key", "value")
			}
			if tt.logInfo {
				logger.Info("info message", "key", "value")
			}
			if tt.logWarn {
				logger.Warn("warn message", "key", "value")
			}
			if tt.logError {
				logger.Error("error message", "key", "value")
			}

			// Count logged lines
			logLines := strings.Split(strings.TrimSpace(buf.String()), "\n")
			actualLogs := 0
			for _, line := range logLines {
				if line != "" {
					actualLogs++
				}
			}

			if actualLogs != tt.expectedLogs {
				t.Errorf("Expected %d log entries, got %d", tt.expectedLogs, actualLogs)
				t.Logf("Log output:\n%s", buf.String())
			}
		})
	}
}

// TestLoggerFormats tests different log output formats
func TestLoggerFormats(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		validate func(t *testing.T, output string)
	}{
		{
			name:   "JSON format",
			format: "json",
			validate: func(t *testing.T, output string) {
				var logEntry map[string]interface{}
				if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
					t.Errorf("Failed to parse JSON log: %v", err)
				}

				// Check required fields
				if _, ok := logEntry["time"]; !ok {
					t.Error("JSON log missing 'time' field")
				}
				if _, ok := logEntry["level"]; !ok {
					t.Error("JSON log missing 'level' field")
				}
				if _, ok := logEntry["msg"]; !ok {
					t.Error("JSON log missing 'msg' field")
				}
			},
		},
		{
			name:   "Text format",
			format: "text",
			validate: func(t *testing.T, output string) {
				// Text format should contain level and message
				if !strings.Contains(output, "INFO") {
					t.Error("Text log doesn't contain level")
				}
				if !strings.Contains(output, "test message") {
					t.Error("Text log doesn't contain message")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			cfg := config.LoggingConfig{
				Level:  "info",
				Format: tt.format,
			}

			logger, err := NewLoggerWithWriter(cfg, &buf)
			if err != nil {
				t.Fatalf("Failed to create logger: %v", err)
			}

			// Log a test message
			logger.Info("test message", "key", "value")

			// Validate the output
			output := strings.TrimSpace(buf.String())
			tt.validate(t, output)
		})
	}
}

// TestLoggerWithSource tests that source code location is included when configured
func TestLoggerWithSource(t *testing.T) {
	var buf bytes.Buffer

	cfg := config.LoggingConfig{
		Level:     "info",
		Format:    "json",
		AddSource: true,
	}

	logger, err := NewLoggerWithWriter(cfg, &buf)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log a message
	logger.Info("test with source")

	// Parse the output
	var logEntry map[string]interface{}
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log: %v", err)
	}

	// Check for source information
	if source, ok := logEntry["source"]; !ok {
		t.Error("Log entry missing source information")
	} else {
		sourceMap, ok := source.(map[string]interface{})
		if !ok {
			t.Error("Source is not a map")
		} else {
			if _, ok := sourceMap["function"]; !ok {
				t.Error("Source missing function name")
			}
			if _, ok := sourceMap["file"]; !ok {
				t.Error("Source missing file name")
			}
			if _, ok := sourceMap["line"]; !ok {
				t.Error("Source missing line number")
			}
		}
	}
}

// TestLoggerContext tests that the logger can be stored and retrieved from context
func TestLoggerContext(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "json",
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create context with logger
	ctx := WithLogger(nil, logger)

	// Retrieve logger from context
	retrievedLogger := FromContext(ctx)
	if retrievedLogger == nil {
		t.Fatal("Failed to retrieve logger from context")
	}

	// Verify it's the same logger (or at least behaves the same)
	// We can't directly compare pointers, but we can verify it works
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	testLogger := slog.New(handler)
	
	ctx2 := WithLogger(nil, testLogger)
	retrievedLogger2 := FromContext(ctx2)
	
	retrievedLogger2.Info("test message")
	if !strings.Contains(buf.String(), "test message") {
		t.Error("Retrieved logger doesn't log properly")
	}
}

// TestSetGlobalLogger tests setting and using the global logger
func TestSetGlobalLogger(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "json",
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Set as global logger
	SetGlobalLogger(logger)

	// The global logger should now be used by slog.Default()
	// We can't easily test this without capturing stdout,
	// but we can at least verify the function doesn't panic
}

// TestLoggerAttributes tests that structured attributes are properly logged
func TestLoggerAttributes(t *testing.T) {
	var buf bytes.Buffer

	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "json",
	}

	logger, err := NewLoggerWithWriter(cfg, &buf)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log with various attribute types
	logger.Info("test attributes",
		"string", "value",
		"int", 42,
		"float", 3.14,
		"bool", true,
		slog.Group("nested",
			"key1", "value1",
			"key2", "value2",
		),
	)

	// Parse the output
	var logEntry map[string]interface{}
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log: %v", err)
	}

	// Verify attributes
	if logEntry["string"] != "value" {
		t.Error("String attribute not logged correctly")
	}

	if logEntry["int"] != float64(42) { // JSON unmarshals numbers as float64
		t.Error("Int attribute not logged correctly")
	}

	if logEntry["float"] != 3.14 {
		t.Error("Float attribute not logged correctly")
	}

	if logEntry["bool"] != true {
		t.Error("Bool attribute not logged correctly")
	}

	// Check nested group
	if nested, ok := logEntry["nested"].(map[string]interface{}); ok {
		if nested["key1"] != "value1" {
			t.Error("Nested attribute key1 not logged correctly")
		}
		if nested["key2"] != "value2" {
			t.Error("Nested attribute key2 not logged correctly")
		}
	} else {
		t.Error("Nested group not logged correctly")
	}
}

// TestInvalidLogLevel tests that invalid log levels are rejected
func TestInvalidLogLevel(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:  "invalid",
		Format: "json",
	}

	_, err := NewLogger(cfg)
	if err == nil {
		t.Error("Expected error for invalid log level, got nil")
	}
}

// TestInvalidLogFormat tests that invalid log formats are rejected
func TestInvalidLogFormat(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "invalid",
	}

	_, err := NewLogger(cfg)
	if err == nil {
		t.Error("Expected error for invalid log format, got nil")
	}
}