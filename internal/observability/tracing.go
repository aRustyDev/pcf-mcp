// Package observability provides distributed tracing using OpenTelemetry
package observability

import (
	"context"
	"fmt"
	"net/url"

	"github.com/aRustyDev/pcf-mcp/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"
)

// InitTracing initializes OpenTelemetry tracing with the configured exporter
func InitTracing(cfg config.TracingConfig) (func(context.Context) error, error) {
	if !cfg.Enabled {
		// Return no-op shutdown function
		return func(ctx context.Context) error { return nil }, nil
	}

	// Validate configuration
	if cfg.SamplingRate < 0.0 || cfg.SamplingRate > 1.0 {
		return nil, fmt.Errorf("invalid sampling rate: %f (must be between 0.0 and 1.0)", cfg.SamplingRate)
	}

	// Create exporter based on configuration
	var exporter sdktrace.SpanExporter
	var err error

	switch cfg.Exporter {
	case "otlp":
		exporter, err = createOTLPExporter(cfg.Endpoint)
	case "jaeger":
		exporter, err = createJaegerExporter(cfg.Endpoint)
	case "zipkin":
		exporter, err = createZipkinExporter(cfg.Endpoint)
	default:
		return nil, fmt.Errorf("unsupported exporter: %s", cfg.Exporter)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Use custom exporter if provided
	return initTracingWithExporter(cfg, exporter)
}

// InitTracingWithExporter initializes tracing with a custom exporter
func InitTracingWithExporter(cfg config.TracingConfig, exporter sdktrace.SpanExporter) (func(context.Context) error, error) {
	return initTracingWithExporter(cfg, exporter)
}

// initTracingWithExporter is the internal implementation
func initTracingWithExporter(cfg config.TracingConfig, exporter sdktrace.SpanExporter) (func(context.Context) error, error) {
	// Create resource with service information
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "pcf-mcp"
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("0.1.0"),
			attribute.String("service.environment", "production"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider with sampling
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
	)

	// Set as global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Return shutdown function
	return tp.Shutdown, nil
}

// createOTLPExporter creates an OTLP exporter
func createOTLPExporter(endpoint string) (sdktrace.SpanExporter, error) {
	// Parse endpoint to extract host:port
	// otlptracehttp expects just host:port, not full URL
	if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
		endpoint = u.Host
	}

	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(), // TODO: Configure TLS properly for production
	)

	return otlptrace.New(context.Background(), client)
}

// createJaegerExporter creates a Jaeger exporter
func createJaegerExporter(endpoint string) (sdktrace.SpanExporter, error) {
	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
}

// createZipkinExporter creates a Zipkin exporter
func createZipkinExporter(endpoint string) (sdktrace.SpanExporter, error) {
	return zipkin.New(endpoint)
}

// StartSpan starts a new span with the given name
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer("github.com/aRustyDev/pcf-mcp")
	return tracer.Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from the context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// RecordError records an error in the span
func RecordError(span trace.Span, err error, opts ...trace.EventOption) {
	if err != nil {
		span.RecordError(err, opts...)
		span.SetStatus(codes.Error, err.Error())
	}
}

// Attribute creation helpers

// StringAttribute creates a string attribute
func StringAttribute(key, value string) attribute.KeyValue {
	return attribute.String(key, value)
}

// IntAttribute creates an int attribute
func IntAttribute(key string, value int) attribute.KeyValue {
	return attribute.Int(key, value)
}

// BoolAttribute creates a bool attribute
func BoolAttribute(key string, value bool) attribute.KeyValue {
	return attribute.Bool(key, value)
}

// HTTP propagation helpers

// InjectHTTPHeaders injects trace context into HTTP headers
func InjectHTTPHeaders(ctx context.Context, headers map[string]string) {
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.MapCarrier(headers))
}

// ExtractHTTPHeaders extracts trace context from HTTP headers
func ExtractHTTPHeaders(ctx context.Context, headers map[string]string) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(ctx, propagation.MapCarrier(headers))
}

// TracedHandler wraps an HTTP handler with tracing
func TracedHandler(name string, handler func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		ctx, span := StartSpan(ctx, name)
		defer span.End()

		err := handler(ctx)
		if err != nil {
			RecordError(span, err)
		}

		return err
	}
}

// Common trace attributes
const (
	// AttributeRequestID is the trace attribute for request ID
	AttributeRequestID = "request.id"

	// AttributeUserID is the trace attribute for user ID
	AttributeUserID = "user.id"

	// AttributeToolName is the trace attribute for MCP tool name
	AttributeToolName = "mcp.tool.name"

	// AttributeProjectID is the trace attribute for PCF project ID
	AttributeProjectID = "pcf.project.id"

	// AttributeHTTPMethod is the trace attribute for HTTP method
	AttributeHTTPMethod = "http.method"

	// AttributeHTTPPath is the trace attribute for HTTP path
	AttributeHTTPPath = "http.path"

	// AttributeHTTPStatus is the trace attribute for HTTP status code
	AttributeHTTPStatus = "http.status"

	// AttributeErrorType is the trace attribute for error type
	AttributeErrorType = "error.type"
)
