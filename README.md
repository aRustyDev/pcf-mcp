# PCF-MCP Server

A Golang-based Model Context Protocol (MCP) server that integrates with the Pentest Collaboration Framework (PCF), enabling AI assistants to interact with pentesting projects, manage security findings, and collaborate on security assessments.

## Features

- **MCP Protocol Support**: Full implementation of the Model Context Protocol for AI assistant integration
- **PCF Integration**: Complete API client for Pentest Collaboration Framework operations
- **Multi-Transport**: Supports both stdio (full-featured) and HTTP (stateless) transports
- **Cloud-Native**: Built with Kubernetes, observability, and containerization in mind
- **Comprehensive Configuration**: Supports CLI args, environment variables, config files, and Kubernetes ConfigMaps
- **Production-Ready Observability**:
  - OpenTelemetry (OTEL) compatible distributed tracing
  - Prometheus-compatible metrics
  - Structured logging with configurable levels

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Docker (optional, for containerized deployment)
- Access to a PCF instance

### Installation

```bash
# Clone the repository
git clone https://github.com/analyst/pcf-mcp
cd pcf-mcp

# Install dependencies
just deps

# Run tests
just test

# Build the binary
just build
```

### Running

```bash
# Run with default configuration
./bin/pcf-mcp

# Run with custom PCF endpoint
./bin/pcf-mcp --pcf-url http://localhost:5000 --pcf-api-key your-api-key

# Run with environment variables
export PCF_MCP_PCF_URL=http://localhost:5000
export PCF_MCP_PCF_API_KEY=your-api-key
./bin/pcf-mcp
```

### Docker

```bash
# Build Docker image
just docker

# Run container
docker run -p 8080:8080 pcf-mcp:latest
```

## Configuration

The server supports hierarchical configuration (in order of precedence):

1. Command-line arguments
2. Environment variables (prefixed with `PCF_MCP_`)
3. Configuration file (YAML/JSON/TOML)
4. Kubernetes ConfigMaps
5. Default values

### Example Configuration File

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  transport: "stdio"  # or "http"

pcf:
  url: "http://localhost:5000"
  api_key: "your-api-key"
  timeout: 30s

logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # or "text"

metrics:
  enabled: true
  port: 9090

tracing:
  enabled: true
  exporter: "jaeger"  # or "zipkin", "otlp"
  endpoint: "http://localhost:14268/api/traces"
```

## MCP Tools Available

- **Project Management**
  - `list_projects`: List all pentest projects
  - `create_project`: Create a new project
  - `update_project`: Update project details

- **Host Management**
  - `list_hosts`: List hosts in a project
  - `add_host`: Add a new host
  - `update_host`: Update host information

- **Issue Tracking**
  - `list_issues`: List security issues
  - `create_issue`: Create a new security finding
  - `update_issue`: Update issue details

- **Credential Storage**
  - `list_credentials`: List stored credentials
  - `add_credential`: Store new credentials
  - `get_credential`: Retrieve specific credentials

- **Report Generation**
  - `generate_report`: Generate reports in various formats

## Development

### Project Structure

```
pcf-mcp/
├── cmd/pcf-mcp/          # Main application entry point
├── internal/
│   ├── config/           # Configuration management
│   ├── pcf/              # PCF client implementation
│   ├── mcp/              # MCP server implementation
│   ├── observability/    # Tracing, metrics, logging
│   └── transport/        # Stdio and HTTP transports
├── pkg/                  # Public packages
├── tests/                # Test files
├── docs/                 # Documentation
├── Dockerfile            # Multi-stage build
├── justfile              # Task automation
├── go.mod & go.sum       # Dependencies
└── README.md             # This file
```

### Testing

```bash
# Run all tests
just test

# Run quick tests (no coverage)
just test-quick

# Run integration tests
just test-integration

# View coverage report
just cover
```

### Building

```bash
# Build for current platform
just build

# Build Docker image
just docker

# Clean build artifacts
just clean
```

## Documentation

- [API Reference](docs/api.md) - Complete API documentation and tool reference
- [Architecture](docs/architecture.md) - System design and component architecture
- [Configuration](docs/configuration.md) - All configuration options explained
- [Deployment Guide](docs/deployment.md) - Docker, Kubernetes, and production deployment
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions

## Deployment

See the [Deployment Guide](docs/deployment.md) for detailed instructions on:
- Local development setup
- Docker deployment
- Docker Compose configuration
- Kubernetes deployment with examples
- Production best practices

## Observability

### Metrics

Prometheus metrics are exposed on `/metrics` endpoint (default port 9090):

- `pcf_mcp_requests_total`: Total number of MCP requests
- `pcf_mcp_request_duration_seconds`: Request duration histogram
- `pcf_mcp_active_connections`: Current active connections

### Tracing

OpenTelemetry tracing supports multiple exporters:
- Jaeger
- Zipkin
- OTLP

### Logging

Structured logging with slog, supporting:
- JSON and text formats
- Configurable log levels
- Kubernetes-friendly output

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests first (TDD)
4. Implement your feature
5. Ensure all tests pass
6. Submit a pull request

## License

[MIT License](LICENSE)

## Support

- GitHub Issues: [github.com/analyst/pcf-mcp/issues](https://github.com/analyst/pcf-mcp/issues)
- PCF Documentation: [PCF Wiki](https://gitlab.com/invuls/pentest-projects/pcf/-/wikis/home)
- MCP Documentation: [MCP Specification](https://modelcontextprotocol.io)

## Version History

See [CHANGELOG.md](CHANGELOG.md) for version history and release notes.