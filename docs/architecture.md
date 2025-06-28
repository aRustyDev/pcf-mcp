# PCF-MCP Architecture

This document describes the architecture of the PCF-MCP server, including its design principles, component structure, and integration patterns.

## Table of Contents

- [Overview](#overview)
- [Design Principles](#design-principles)
- [Component Architecture](#component-architecture)
- [Data Flow](#data-flow)
- [Transport Layers](#transport-layers)
- [Observability Architecture](#observability-architecture)
- [Security Architecture](#security-architecture)

## Overview

PCF-MCP is a Model Context Protocol (MCP) server that bridges AI assistants with the Pentest Collaboration Framework (PCF). It follows a clean architecture pattern with clear separation of concerns.

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   AI Assistant  │────▶│   PCF-MCP Server │────▶│   PCF Instance  │
│  (Claude, etc)  │◀────│   (This Project) │◀────│   (API Server)  │
└─────────────────┘     └──────────────────┘     └─────────────────┘
         MCP                                            REST API
```

## Design Principles

1. **Clean Architecture**: Business logic is independent of frameworks and external services
2. **Dependency Injection**: Components receive dependencies through interfaces
3. **Test-Driven Development**: Tests drive the design and implementation
4. **Cloud-Native**: Built for containerization and Kubernetes deployment
5. **Observable by Default**: Comprehensive metrics, logging, and tracing
6. **Configuration Flexibility**: Multiple configuration sources with clear precedence
7. **Fail-Safe Defaults**: Secure and sensible defaults for all settings

## Component Architecture

### Layer Structure

```
┌────────────────────────────────────────────────┐
│                 Presentation                    │
│          (HTTP/stdio transports)               │
├────────────────────────────────────────────────┤
│                 Application                     │
│           (MCP server & tools)                 │
├────────────────────────────────────────────────┤
│                   Domain                        │
│        (Business logic & interfaces)           │
├────────────────────────────────────────────────┤
│                Infrastructure                   │
│    (PCF client, config, observability)         │
└────────────────────────────────────────────────┘
```

### Core Components

#### 1. MCP Server (`internal/mcp/server.go`)

The central component that:
- Manages tool registration and execution
- Handles protocol negotiation
- Enforces rate limiting and concurrency controls
- Records metrics for all operations

```go
type Server struct {
    config     ServerConfig
    mcpServer  *server.MCPServer
    tools      map[string]Tool
    toolsMutex sync.RWMutex
    metrics    MetricsRecorder
}
```

#### 2. PCF Client (`internal/pcf/client.go`)

REST API client for PCF communication:
- Automatic retry with exponential backoff
- Connection pooling for performance
- Request/response logging
- Error categorization

```go
type Client struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
    logger     *slog.Logger
}
```

#### 3. Configuration (`internal/config/`)

Hierarchical configuration system:
- CLI arguments (highest priority)
- Environment variables
- Configuration files
- Kubernetes ConfigMaps
- Default values (lowest priority)

#### 4. Observability (`internal/observability/`)

Three pillars of observability:
- **Metrics**: Prometheus-compatible metrics
- **Logging**: Structured logging with slog
- **Tracing**: OpenTelemetry distributed tracing

### Tool Architecture

Each MCP tool follows a consistent pattern:

```go
type Tool struct {
    Name        string
    Description string
    InputSchema map[string]interface{}
    Handler     ToolHandler
}

type ToolHandler func(ctx context.Context, params map[string]interface{}) (interface{}, error)
```

Tools are organized by domain:
- Project management tools
- Host management tools
- Issue tracking tools
- Credential management tools
- Report generation tools

## Data Flow

### Request Flow (HTTP Transport)

```
1. HTTP Request → Logging Middleware
2. → Tracing Middleware
3. → Metrics Middleware
4. → Authentication Middleware
5. → CORS Middleware
6. → Router
7. → Tool Execution
8. → PCF Client
9. → PCF API
10. ← Response transformation
11. ← JSON serialization
12. ← HTTP Response
```

### Request Flow (stdio Transport)

```
1. JSON-RPC Request → stdio reader
2. → MCP protocol handler
3. → Tool resolver
4. → Tool execution
5. → PCF Client
6. → PCF API
7. ← Response transformation
8. ← JSON-RPC response
9. ← stdio writer
```

## Transport Layers

### stdio Transport (Full MCP)

- Bidirectional communication
- Stateful connections
- Full MCP protocol support
- Used by desktop AI assistants

### HTTP Transport (Stateless)

- RESTful API design
- Stateless operations
- CORS support for web clients
- Bearer token authentication
- Prometheus metrics endpoint

## Observability Architecture

### Metrics Collection

```
┌─────────────┐     ┌──────────────┐     ┌────────────┐
│   Request   │────▶│   Metrics    │────▶│ Prometheus │
│             │     │  Middleware  │     │  Registry  │
└─────────────┘     └──────────────┘     └────────────┘
                            │
                            ▼
                    ┌──────────────┐
                    │   Counters   │
                    │  Histograms  │
                    │    Gauges    │
                    └──────────────┘
```

Key metrics:
- Request rate and latency
- Tool execution counts and duration
- Active connections
- Error rates by category

### Distributed Tracing

```
┌─────────────┐     ┌──────────────┐     ┌────────────┐
│  HTTP Call  │────▶│ Create Span  │────▶│   Export   │
│             │     │ Add Context  │     │  to OTLP   │
└─────────────┘     └──────────────┘     └────────────┘
                            │
                            ▼
                    ┌──────────────┐
                    │ Propagate to │
                    │  PCF Client  │
                    └──────────────┘
```

### Structured Logging

```
┌─────────────┐     ┌──────────────┐     ┌────────────┐
│  Operation  │────▶│ Add Context  │────▶│   Output   │
│             │     │    Fields    │     │  JSON/Text │
└─────────────┘     └──────────────┘     └────────────┘
                            │
                            ▼
                    ┌──────────────┐
                    │ Correlation  │
                    │   IDs, etc   │
                    └──────────────┘
```

## Security Architecture

### Authentication Flow

```
┌─────────────┐     ┌──────────────┐     ┌────────────┐
│   Request   │────▶│ Auth Check   │────▶│  Proceed   │
│             │     │ Bearer Token │     │    or      │
└─────────────┘     └──────────────┘     │   Reject   │
                                         └────────────┘
```

### Security Measures

1. **Transport Security**
   - TLS for all external communication
   - Certificate validation
   - Secure defaults

2. **Authentication**
   - Bearer token for HTTP transport
   - Configurable auth requirements
   - Token rotation support

3. **Authorization**
   - Tool-level access control (planned)
   - Project-based isolation
   - Audit logging

4. **Data Protection**
   - Credential encryption at rest
   - Redacted values in responses
   - No sensitive data in logs

5. **Input Validation**
   - JSON schema validation
   - Parameter sanitization
   - Rate limiting

### Threat Model

```
┌──────────────────┐
│ Threat Vectors   │
├──────────────────┤
│ • Unauthorized   │
│   access         │
│ • Data exposure  │
│ • Injection      │
│ • DoS attacks    │
│ • MITM attacks   │
└──────────────────┘
         │
         ▼
┌──────────────────┐
│  Mitigations     │
├──────────────────┤
│ • Authentication │
│ • Encryption     │
│ • Validation     │
│ • Rate limiting  │
│ • TLS/mTLS       │
└──────────────────┘
```

## Deployment Architecture

### Container Structure

```
┌─────────────────────────────────┐
│         Container Image          │
├─────────────────────────────────┤
│ • Distroless base image         │
│ • Single static binary          │
│ • Non-root user                 │
│ • Read-only filesystem          │
│ • No shell or package manager   │
└─────────────────────────────────┘
```

### Kubernetes Deployment

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Deployment  │────▶│   Service   │────▶│   Ingress   │
│  (Pods)     │     │ (ClusterIP) │     │   (HTTPS)   │
└─────────────┘     └─────────────┘     └─────────────┘
       │
       ▼
┌─────────────┐     ┌─────────────┐
│  ConfigMap  │     │   Secret    │
│   (Config)  │     │ (API Keys)  │
└─────────────┘     └─────────────┘
```

## Performance Considerations

### Optimization Strategies

1. **Connection Pooling**: Reuse HTTP connections to PCF
2. **Caching**: In-memory caching for frequently accessed data
3. **Concurrency Control**: Limit parallel tool executions
4. **Request Batching**: Batch multiple operations when possible
5. **Resource Limits**: Memory and CPU limits in Kubernetes

### Scalability

- Horizontal scaling via Kubernetes replicas
- Stateless design enables easy scaling
- Load balancing at ingress level
- Prometheus metrics for autoscaling decisions

## Future Architecture Considerations

### Planned Enhancements

1. **WebSocket Support**: Real-time updates for long-running operations
2. **Event Streaming**: Kafka/NATS integration for event-driven architecture
3. **Multi-Tenancy**: Project-level isolation and quotas
4. **Plugin System**: Extensible tool architecture
5. **GraphQL API**: Alternative to REST for complex queries