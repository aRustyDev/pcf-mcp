# PCF-MCP Configuration Reference

This document provides a comprehensive reference for all configuration options available in PCF-MCP.

## Table of Contents

- [Configuration Sources](#configuration-sources)
- [Configuration Precedence](#configuration-precedence)
- [Server Configuration](#server-configuration)
- [PCF Configuration](#pcf-configuration)
- [Logging Configuration](#logging-configuration)
- [Metrics Configuration](#metrics-configuration)
- [Tracing Configuration](#tracing-configuration)
- [Complete Example](#complete-example)
- [Environment Variables](#environment-variables)
- [Command Line Arguments](#command-line-arguments)

## Configuration Sources

PCF-MCP supports multiple configuration sources:

1. **Command-line arguments** - Highest precedence
2. **Environment variables** - Override file and defaults
3. **Configuration file** - YAML, JSON, or TOML format
4. **Kubernetes ConfigMap** - For K8s deployments
5. **Default values** - Built-in defaults

## Configuration Precedence

Configuration values are resolved in the following order (highest to lowest precedence):

```
CLI Args > Environment Variables > Config File > ConfigMap > Defaults
```

Example:
```bash
# Default port is 8080
# Config file sets port to 8081
# Environment variable sets port to 8082
# CLI argument sets port to 8083

./pcf-mcp --server-port 8083  # Port will be 8083
```

## Server Configuration

Server configuration controls the MCP server behavior.

### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `server.host` | string | `0.0.0.0` | Server bind address |
| `server.port` | int | `8080` | Server listen port |
| `server.transport` | string | `stdio` | Transport type (`stdio` or `http`) |
| `server.read_timeout` | duration | `30s` | Maximum duration for reading requests |
| `server.write_timeout` | duration | `30s` | Maximum duration for writing responses |
| `server.max_concurrent_tools` | int | `10` | Maximum concurrent tool executions |
| `server.tool_timeout` | duration | `60s` | Maximum duration for tool execution |
| `server.auth_required` | bool | `false` | Enable authentication for HTTP transport |
| `server.auth_token` | string | `""` | Bearer token for authentication |

### Examples

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  transport: "http"
  read_timeout: 30s
  write_timeout: 30s
  max_concurrent_tools: 20
  tool_timeout: 120s
  auth_required: true
  auth_token: "secret-bearer-token"
```

### Transport-Specific Behavior

#### stdio Transport
- Ignores `host` and `port` settings
- Uses standard input/output for communication
- Suitable for desktop AI assistants
- Maintains stateful connection

#### HTTP Transport
- Listens on `host:port`
- Stateless REST API
- Supports CORS for web clients
- Optional bearer token authentication

## PCF Configuration

PCF configuration controls the connection to the Pentest Collaboration Framework.

### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `pcf.url` | string | `http://localhost:5000` | PCF API base URL |
| `pcf.api_key` | string | `""` | API key for PCF authentication |
| `pcf.timeout` | duration | `30s` | HTTP client timeout |
| `pcf.max_retries` | int | `3` | Maximum retry attempts |
| `pcf.insecure_skip_verify` | bool | `false` | Skip TLS certificate verification |

### Examples

```yaml
pcf:
  url: "https://pcf.example.com"
  api_key: "your-api-key-here"
  timeout: 45s
  max_retries: 5
  insecure_skip_verify: false
```

### Security Considerations

- **Never commit API keys** to version control
- Use environment variables or secrets for API keys
- Only use `insecure_skip_verify` for development
- Consider using mTLS for production

## Logging Configuration

Logging configuration controls application logging behavior.

### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `logging.level` | string | `info` | Minimum log level (`debug`, `info`, `warn`, `error`) |
| `logging.format` | string | `json` | Log format (`json` or `text`) |
| `logging.add_source` | bool | `false` | Include source code location in logs |

### Examples

```yaml
logging:
  level: "info"
  format: "json"
  add_source: false
```

### Log Levels

- **debug**: Detailed debugging information
- **info**: General informational messages
- **warn**: Warning messages for potential issues
- **error**: Error messages for failures

### Output Formats

#### JSON Format (Recommended for production)
```json
{
  "time": "2024-01-01T00:00:00Z",
  "level": "INFO",
  "msg": "Server started",
  "transport": "http",
  "address": "0.0.0.0:8080"
}
```

#### Text Format (Better for development)
```
2024-01-01T00:00:00Z INF Server started transport=http address=0.0.0.0:8080
```

## Metrics Configuration

Metrics configuration controls Prometheus metrics collection.

### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `metrics.enabled` | bool | `true` | Enable metrics collection |
| `metrics.port` | int | `9090` | Metrics endpoint port |
| `metrics.path` | string | `/metrics` | Metrics endpoint path |

### Examples

```yaml
metrics:
  enabled: true
  port: 9090
  path: "/metrics"
```

### Available Metrics

- `pcf_mcp_requests_total` - Total HTTP requests
- `pcf_mcp_request_duration_seconds` - Request duration histogram
- `pcf_mcp_active_connections` - Active connection gauge
- `pcf_mcp_tool_executions_total` - Tool execution counter
- `pcf_mcp_tool_errors_total` - Tool error counter
- `pcf_mcp_tool_duration_seconds` - Tool execution duration

### Prometheus Scrape Configuration

```yaml
scrape_configs:
  - job_name: 'pcf-mcp'
    static_configs:
    - targets: ['localhost:9090']
```

## Tracing Configuration

Tracing configuration controls OpenTelemetry distributed tracing.

### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `tracing.enabled` | bool | `false` | Enable distributed tracing |
| `tracing.exporter` | string | `otlp` | Exporter type (`jaeger`, `zipkin`, `otlp`) |
| `tracing.endpoint` | string | `http://localhost:4317` | Collector endpoint |
| `tracing.sampling_rate` | float | `1.0` | Trace sampling rate (0.0-1.0) |
| `tracing.service_name` | string | `pcf-mcp` | Service name in traces |

### Examples

```yaml
tracing:
  enabled: true
  exporter: "otlp"
  endpoint: "http://otel-collector:4317"
  sampling_rate: 0.1  # Sample 10% of requests
  service_name: "pcf-mcp-prod"
```

### Exporter-Specific Endpoints

#### Jaeger
- HTTP: `http://jaeger-collector:14268/api/traces`
- gRPC: `jaeger-agent:6831`

#### Zipkin
- HTTP: `http://zipkin:9411/api/v2/spans`

#### OTLP
- gRPC: `http://otel-collector:4317`
- HTTP: `http://otel-collector:4318`

## Complete Example

### YAML Configuration File

```yaml
# config.yaml
server:
  host: "0.0.0.0"
  port: 8080
  transport: "http"
  read_timeout: 30s
  write_timeout: 30s
  max_concurrent_tools: 20
  tool_timeout: 60s
  auth_required: true
  auth_token: "${AUTH_TOKEN}"  # Can use env var substitution

pcf:
  url: "https://pcf.example.com"
  api_key: "${PCF_API_KEY}"
  timeout: 30s
  max_retries: 3
  insecure_skip_verify: false

logging:
  level: "info"
  format: "json"
  add_source: false

metrics:
  enabled: true
  port: 9090
  path: "/metrics"

tracing:
  enabled: true
  exporter: "otlp"
  endpoint: "http://otel-collector:4317"
  sampling_rate: 0.1
  service_name: "pcf-mcp-prod"
```

### JSON Configuration File

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "transport": "http",
    "read_timeout": "30s",
    "write_timeout": "30s",
    "max_concurrent_tools": 20,
    "tool_timeout": "60s",
    "auth_required": true,
    "auth_token": "secret-token"
  },
  "pcf": {
    "url": "https://pcf.example.com",
    "api_key": "your-api-key",
    "timeout": "30s",
    "max_retries": 3,
    "insecure_skip_verify": false
  },
  "logging": {
    "level": "info",
    "format": "json",
    "add_source": false
  },
  "metrics": {
    "enabled": true,
    "port": 9090,
    "path": "/metrics"
  },
  "tracing": {
    "enabled": true,
    "exporter": "otlp",
    "endpoint": "http://otel-collector:4317",
    "sampling_rate": 0.1,
    "service_name": "pcf-mcp-prod"
  }
}
```

## Environment Variables

All configuration options can be set via environment variables using the prefix `PCF_MCP_` and replacing dots with underscores.

### Mapping Rules

1. Add prefix: `PCF_MCP_`
2. Convert to uppercase
3. Replace dots (.) with underscores (_)

### Examples

| Configuration | Environment Variable |
|--------------|---------------------|
| `server.port` | `PCF_MCP_SERVER_PORT` |
| `server.transport` | `PCF_MCP_SERVER_TRANSPORT` |
| `pcf.url` | `PCF_MCP_PCF_URL` |
| `pcf.api_key` | `PCF_MCP_PCF_API_KEY` |
| `logging.level` | `PCF_MCP_LOGGING_LEVEL` |
| `metrics.enabled` | `PCF_MCP_METRICS_ENABLED` |
| `tracing.sampling_rate` | `PCF_MCP_TRACING_SAMPLING_RATE` |

### Shell Example

```bash
export PCF_MCP_SERVER_TRANSPORT=http
export PCF_MCP_SERVER_PORT=8080
export PCF_MCP_PCF_URL=https://pcf.example.com
export PCF_MCP_PCF_API_KEY=your-api-key
export PCF_MCP_LOGGING_LEVEL=debug
export PCF_MCP_METRICS_ENABLED=true

./pcf-mcp
```

## Command Line Arguments

Command-line arguments have the highest precedence and override all other configuration sources.

### Available Arguments

```bash
./pcf-mcp --help

Flags:
  --config string                    Config file path
  
  # Server flags
  --server-host string              Server bind address
  --server-port int                 Server listen port
  --server-transport string         Transport type (stdio or http)
  --server-auth-required            Enable authentication
  --server-auth-token string        Bearer token for auth
  
  # PCF flags
  --pcf-url string                  PCF API URL
  --pcf-api-key string              PCF API key
  
  # Logging flags
  --log-level string                Log level (debug, info, warn, error)
  --log-format string               Log format (json or text)
  
  # Feature flags
  --metrics-enabled                 Enable metrics collection
  --tracing-enabled                 Enable distributed tracing
```

### Examples

```bash
# Basic HTTP server
./pcf-mcp \
  --server-transport http \
  --server-port 8080 \
  --pcf-url https://pcf.example.com \
  --pcf-api-key your-api-key

# Debug mode with text logging
./pcf-mcp \
  --log-level debug \
  --log-format text \
  --config config.yaml

# Production with auth
./pcf-mcp \
  --server-transport http \
  --server-auth-required \
  --server-auth-token "Bearer secret-token" \
  --metrics-enabled \
  --tracing-enabled
```

## Configuration Best Practices

### Development

```yaml
server:
  transport: "http"
  host: "localhost"
  port: 8080
  
logging:
  level: "debug"
  format: "text"
  add_source: true
  
metrics:
  enabled: true
  
tracing:
  enabled: false
```

### Production

```yaml
server:
  transport: "http"
  host: "0.0.0.0"
  port: 8080
  auth_required: true
  auth_token: "${AUTH_TOKEN}"
  max_concurrent_tools: 50
  
pcf:
  url: "${PCF_URL}"
  api_key: "${PCF_API_KEY}"
  timeout: 45s
  max_retries: 5
  
logging:
  level: "info"
  format: "json"
  
metrics:
  enabled: true
  
tracing:
  enabled: true
  sampling_rate: 0.01  # 1% sampling
```

### Security Recommendations

1. **Never hardcode sensitive values** - Use environment variables or secrets
2. **Enable authentication** in production HTTP deployments
3. **Use TLS** for all external communication
4. **Rotate API keys** regularly
5. **Limit concurrent executions** to prevent resource exhaustion
6. **Set appropriate timeouts** to prevent hanging requests

### Performance Tuning

1. **Concurrent Tools**: Set based on available CPU cores
2. **Timeouts**: Balance between reliability and responsiveness
3. **Sampling Rate**: Lower rates for high-traffic production
4. **Log Level**: Use `info` or `warn` in production
5. **Metrics**: Essential for performance monitoring

## Validation

PCF-MCP validates configuration on startup and will fail fast with clear error messages:

```
Error: invalid configuration: server.port must be between 1 and 65535
Error: invalid configuration: pcf.url is required
Error: invalid configuration: logging.level must be one of: debug, info, warn, error
```

Common validation rules:
- Port numbers: 1-65535
- URLs: Must be valid URLs
- Durations: Must be valid Go duration strings (e.g., "30s", "5m")
- Enum values: Must match allowed values
- Required fields: Must be non-empty