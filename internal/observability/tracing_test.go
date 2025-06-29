package observability

import (
	"context"
	"errors"
	"testing"

	"github.com/aRustyDev/pcf-mcp/internal/config"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// TestInitTracing tests the initialization of OpenTelemetry tracing
func TestInitTracing(t *testing.T) {
	tests := []struct {
		name      string
		config    config.TracingConfig
		expectErr bool
	}{
		{
			name: "Disabled tracing",
			config: config.TracingConfig{
				Enabled: false,
			},
			expectErr: false,
		},
		{
			name: "OTLP exporter",
			config: config.TracingConfig{
				Enabled:      true,
				Exporter:     "otlp",
				Endpoint:     "http://localhost:4317",
				SamplingRate: 1.0,
				ServiceName:  "test-service",
			},
			expectErr: false,
		},
		{
			name: "Jaeger exporter",
			config: config.TracingConfig{
				Enabled:      true,
				Exporter:     "jaeger",
				Endpoint:     "http://localhost:14268/api/traces",
				SamplingRate: 0.5,
				ServiceName:  "test-service",
			},
			expectErr: false,
		},
		{
			name: "Zipkin exporter",
			config: config.TracingConfig{
				Enabled:      true,
				Exporter:     "zipkin",
				Endpoint:     "http://localhost:9411/api/v2/spans",
				SamplingRate: 0.1,
				ServiceName:  "test-service",
			},
			expectErr: false,
		},
		{
			name: "Invalid exporter",
			config: config.TracingConfig{
				Enabled:  true,
				Exporter: "invalid",
			},
			expectErr: true,
		},
		{
			name: "Invalid sampling rate",
			config: config.TracingConfig{
				Enabled:      true,
				Exporter:     "otlp",
				Endpoint:     "http://localhost:4317",
				SamplingRate: 2.0, // Invalid: > 1.0
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shutdown, err := InitTracing(tt.config)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Cleanup
			if shutdown != nil {
				shutdown(context.Background())
			}
		})
	}
}

// TestTraceProvider tests that the global trace provider is set correctly
func TestTraceProvider(t *testing.T) {
	cfg := config.TracingConfig{
		Enabled:      true,
		Exporter:     "otlp",
		Endpoint:     "http://localhost:4317",
		SamplingRate: 1.0,
		ServiceName:  "test-provider",
	}

	shutdown, err := InitTracing(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer shutdown(context.Background())

	// Get global tracer
	tracer := otel.Tracer("test")
	if tracer == nil {
		t.Error("Global tracer should not be nil")
	}

	// Create a span
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	if span == nil {
		t.Error("Failed to create span")
	}
	span.End()
}

// TestStartSpan tests the helper function for starting spans
func TestStartSpan(t *testing.T) {
	cfg := config.TracingConfig{
		Enabled:      true,
		Exporter:     "otlp",
		Endpoint:     "http://localhost:4317",
		SamplingRate: 1.0,
		ServiceName:  "test-span-service",
	}

	shutdown, err := InitTracing(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer shutdown(context.Background())

	// Test starting a span
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-operation")

	if span == nil {
		t.Fatal("StartSpan returned nil span")
	}

	// Verify span is recording
	if !span.IsRecording() {
		t.Error("Span should be recording")
	}

	// Test span attributes
	span.SetAttributes(
		StringAttribute("key1", "value1"),
		IntAttribute("key2", 42),
		BoolAttribute("key3", true),
	)

	// End span
	span.End()
}

// TestSpanFromContext tests extracting spans from context
func TestSpanFromContext(t *testing.T) {
	cfg := config.TracingConfig{
		Enabled:      true,
		Exporter:     "otlp",
		Endpoint:     "http://localhost:4317",
		SamplingRate: 1.0,
		ServiceName:  "test-context-service",
	}

	shutdown, err := InitTracing(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer shutdown(context.Background())

	// Create context with span
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "parent-span")
	defer span.End()

	// Extract span from context
	extractedSpan := SpanFromContext(ctx)
	if extractedSpan == nil {
		t.Fatal("Failed to extract span from context")
	}

	// Verify it's the same span context by checking trace ID
	if span.SpanContext().TraceID() != extractedSpan.SpanContext().TraceID() {
		t.Error("Extracted span trace ID doesn't match original")
	}
}

// TestRecordError tests error recording in spans
func TestRecordError(t *testing.T) {
	cfg := config.TracingConfig{
		Enabled:      true,
		Exporter:     "otlp",
		Endpoint:     "http://localhost:4317",
		SamplingRate: 1.0,
		ServiceName:  "test-error-service",
	}

	shutdown, err := InitTracing(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer shutdown(context.Background())

	// Create span
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "error-operation")

	// Record an error
	testErr := errors.New("test error")
	RecordError(span, testErr)

	// End span
	span.End()
}

// TestHTTPCarrier tests HTTP header propagation
func TestHTTPCarrier(t *testing.T) {
	cfg := config.TracingConfig{
		Enabled:      true,
		Exporter:     "otlp",
		Endpoint:     "http://localhost:4317",
		SamplingRate: 1.0,
		ServiceName:  "test-propagation-service",
	}

	shutdown, err := InitTracing(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer shutdown(context.Background())

	// Create span
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "http-operation")
	defer span.End()

	// Test injecting into HTTP headers
	headers := make(map[string]string)
	InjectHTTPHeaders(ctx, headers)

	// Verify headers were added
	if len(headers) == 0 {
		t.Error("No headers were injected")
	}

	// Test extracting from HTTP headers
	newCtx := ExtractHTTPHeaders(context.Background(), headers)
	newSpan := SpanFromContext(newCtx)

	if newSpan == nil {
		t.Error("Failed to extract span from headers")
	}
}

// MockExporter is a mock trace exporter for testing
type MockExporter struct {
	spans []sdktrace.ReadOnlySpan
}

func (m *MockExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	m.spans = append(m.spans, spans...)
	return nil
}

func (m *MockExporter) Shutdown(ctx context.Context) error {
	return nil
}

// TestCustomExporter tests using a custom exporter
func TestCustomExporter(t *testing.T) {
	mockExporter := &MockExporter{}

	cfg := config.TracingConfig{
		Enabled:      true,
		Exporter:     "custom",
		SamplingRate: 1.0,
		ServiceName:  "test-custom-service",
	}

	// Initialize with custom exporter
	shutdown, err := InitTracingWithExporter(cfg, mockExporter)
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer shutdown(context.Background())

	// Create and end a span
	ctx := context.Background()
	_, span := StartSpan(ctx, "custom-operation")
	span.End()

	// Force flush to ensure export
	if tp, ok := otel.GetTracerProvider().(interface{ ForceFlush(context.Context) error }); ok {
		tp.ForceFlush(context.Background())
	}

	// Verify span was exported
	if len(mockExporter.spans) == 0 {
		t.Error("No spans were exported")
	}
}
