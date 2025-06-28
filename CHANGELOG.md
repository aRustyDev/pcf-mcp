# Changelog

All notable changes to PCF-MCP will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive documentation suite
  - API documentation with full tool reference
  - Architecture documentation with diagrams
  - Deployment guide for Docker and Kubernetes
  - Configuration reference with all options
  - Troubleshooting guide with common issues

## [0.8.0] - 2024-01-03

### Added
- Comprehensive integration and end-to-end tests
- Mock PCF server for testing
- Concurrent request stress testing
- Authentication testing for HTTP transport
- Integration test commands in Justfile

### Fixed
- List credentials nil slice issue (always initialize as empty slice)
- Race condition in concurrent tests with proper WaitGroup
- Metrics endpoint expectations in e2e tests

## [0.7.0] - 2024-01-03

### Added
- HTTP transport implementation with REST API
- CORS support for web clients
- Bearer token authentication
- Dedicated metrics registry per server instance
- HTTP-specific middleware chain (logging → tracing → metrics → auth → CORS)
- Graceful shutdown support
- HTTP run commands in Justfile

### Changed
- Server now supports both stdio and HTTP transports
- Configuration extended with HTTP-specific settings

## [0.6.0] - 2024-01-02

### Added
- Complete MCP tool set implementation (9 tools total)
  - list_hosts: List hosts with filtering
  - add_host: Add new hosts to projects
  - list_issues: List issues with severity filtering
  - create_issue: Create security findings
  - list_credentials: List stored credentials (redacted)
  - add_credential: Store credentials securely
  - generate_report: Generate reports in multiple formats
- Comprehensive test coverage for all tools
- Tool-specific client interfaces for better testing

### Security
- Credential values always redacted in responses
- Secure credential storage implementation

## [0.5.0] - 2024-01-02

### Added
- Docker support with multi-stage build
- Distroless base image for security
- Docker Compose configuration
- Prometheus scraping configuration
- GitHub Actions CI/CD pipeline
- Docker-specific documentation

### Changed
- Optimized binary size with build flags
- Non-root user execution in container

## [0.4.0] - 2024-01-02

### Added
- OpenTelemetry distributed tracing support
- Multiple exporter support (Jaeger, Zipkin, OTLP)
- Trace context propagation
- Configurable sampling rates
- Service name customization
- Comprehensive tracing tests

## [0.3.0] - 2024-01-02

### Added
- Prometheus metrics collection
- Custom metrics registry
- MCP tool execution metrics
- HTTP request metrics
- Active connection tracking
- Metrics HTTP endpoint
- Comprehensive metrics tests

## [0.2.0] - 2024-01-02

### Added
- First MCP tool: list_projects
- Tool registration system
- JSON Schema validation
- PCF client implementation
- Tool execution framework
- Comprehensive tool tests

## [0.1.0] - 2024-01-01

### Added
- Initial project structure
- MCP server foundation
- Multi-source configuration (CLI, env, file)
- Structured logging with slog
- Basic project layout
- Comprehensive test framework
- Development automation with Just

### Project Structure
- Clean architecture design
- Dependency injection patterns
- Interface-based design
- Test-driven development setup

[Unreleased]: https://github.com/analyst/pcf-mcp/compare/v0.8.0...HEAD
[0.8.0]: https://github.com/analyst/pcf-mcp/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/analyst/pcf-mcp/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/analyst/pcf-mcp/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/analyst/pcf-mcp/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/analyst/pcf-mcp/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/analyst/pcf-mcp/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/analyst/pcf-mcp/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/analyst/pcf-mcp/releases/tag/v0.1.0