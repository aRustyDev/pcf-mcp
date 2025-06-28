# Docker Deployment Guide

This guide covers building and running the PCF-MCP server using Docker.

## Quick Start

### Build the Docker Image

```bash
docker build -t pcf-mcp:latest .
```

### Run the Container

```bash
docker run -d \
  --name pcf-mcp \
  -e PCF_MCP_PCF_URL=http://your-pcf-instance:5000 \
  -e PCF_MCP_PCF_API_KEY=your-api-key \
  -p 8080:8080 \
  -p 9090:9090 \
  pcf-mcp:latest
```

## Docker Compose

For a complete development environment with monitoring:

```bash
# Start basic services
docker-compose up -d

# Start with tracing enabled
docker-compose --profile tracing up -d

# Start with metrics and visualization
docker-compose --profile metrics up -d

# Start everything
docker-compose --profile tracing --profile metrics up -d
```

### Access Services

- **PCF-MCP Server**: http://localhost:8080
- **Metrics**: http://localhost:9090/metrics
- **Prometheus UI**: http://localhost:9091 (when metrics profile is enabled)
- **Grafana**: http://localhost:3000 (when metrics profile is enabled)
- **Jaeger UI**: http://localhost:16686 (when tracing profile is enabled)

## Configuration

### Environment Variables

The Docker image supports all PCF-MCP environment variables:

#### Server Configuration
- `PCF_MCP_SERVER_HOST`: Server bind address (default: "0.0.0.0")
- `PCF_MCP_SERVER_PORT`: Server port (default: 8080)
- `PCF_MCP_SERVER_TRANSPORT`: Transport type - "stdio" or "http" (default: "stdio")

#### PCF Configuration
- `PCF_MCP_PCF_URL`: PCF instance URL (required)
- `PCF_MCP_PCF_API_KEY`: PCF API key (required)
- `PCF_MCP_PCF_TIMEOUT`: Request timeout (default: "30s")

#### Logging
- `PCF_MCP_LOGGING_LEVEL`: Log level - debug/info/warn/error (default: "info")
- `PCF_MCP_LOGGING_FORMAT`: Log format - json/text (default: "json")

#### Metrics
- `PCF_MCP_METRICS_ENABLED`: Enable metrics collection (default: "true")
- `PCF_MCP_METRICS_PORT`: Metrics port (default: 9090)

#### Tracing
- `PCF_MCP_TRACING_ENABLED`: Enable distributed tracing (default: "false")
- `PCF_MCP_TRACING_EXPORTER`: Exporter type - otlp/jaeger/zipkin (default: "otlp")
- `PCF_MCP_TRACING_ENDPOINT`: Trace collector endpoint
- `PCF_MCP_TRACING_SAMPLING_RATE`: Sampling rate 0.0-1.0 (default: "1.0")

### Using a Configuration File

Mount a configuration file into the container:

```bash
docker run -d \
  --name pcf-mcp \
  -v $(pwd)/config.yml:/config/config.yml \
  -e PCF_MCP_CONFIG_FILE=/config/config.yml \
  -p 8080:8080 \
  pcf-mcp:latest
```

## Production Deployment

### Security Considerations

1. **Run as Non-Root**: The container runs as a non-root user by default
2. **Minimal Image**: Uses scratch base image to minimize attack surface
3. **No Shell**: No shell or package manager in the final image
4. **Static Binary**: Fully static Go binary with no external dependencies

### Resource Limits

Set appropriate resource limits:

```bash
docker run -d \
  --name pcf-mcp \
  --memory="256m" \
  --cpus="0.5" \
  -e PCF_MCP_PCF_URL=http://pcf:5000 \
  -e PCF_MCP_PCF_API_KEY=your-api-key \
  pcf-mcp:latest
```

### Health Checks

The container includes a health check that runs every 30 seconds:

```bash
# Check container health
docker inspect pcf-mcp --format='{{.State.Health.Status}}'

# View health check logs
docker inspect pcf-mcp --format='{{range .State.Health.Log}}{{.Output}}{{end}}'
```

### Logging

View container logs:

```bash
# Follow logs
docker logs -f pcf-mcp

# Last 100 lines
docker logs --tail 100 pcf-mcp

# With timestamps
docker logs -t pcf-mcp
```

## Kubernetes Deployment

For Kubernetes deployment, use the provided manifests:

```bash
# Create ConfigMap
kubectl create configmap pcf-mcp-config --from-file=config.yml

# Deploy
kubectl apply -f k8s/

# Check status
kubectl get pods -l app=pcf-mcp
kubectl logs -f deployment/pcf-mcp
```

## Troubleshooting

### Container Won't Start

1. Check logs: `docker logs pcf-mcp`
2. Verify environment variables are set correctly
3. Ensure PCF instance is reachable from container

### Connection Issues

1. Verify port mapping: `docker port pcf-mcp`
2. Check firewall rules
3. Test connectivity: `docker exec pcf-mcp wget -O- http://localhost:8080/health`

### Performance Issues

1. Check resource usage: `docker stats pcf-mcp`
2. Increase memory/CPU limits if needed
3. Enable metrics and analyze with Prometheus/Grafana

## Building Custom Images

### Multi-Architecture Build

Build for multiple platforms:

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t pcf-mcp:latest \
  --push .
```

### Build Arguments

Customize build with arguments:

```bash
docker build \
  --build-arg VERSION=1.0.0 \
  --build-arg COMMIT=$(git rev-parse HEAD) \
  -t pcf-mcp:latest .
```

## Development Workflow

### Local Development with Docker Compose

1. Make code changes
2. Rebuild: `docker-compose build pcf-mcp`
3. Restart: `docker-compose restart pcf-mcp`
4. Check logs: `docker-compose logs -f pcf-mcp`

### Running Tests in Docker

```bash
# Run tests during build
docker build --target builder -t pcf-mcp:test .
docker run --rm pcf-mcp:test go test ./...

# Run specific tests
docker run --rm -v $(pwd):/app -w /app golang:1.23-alpine go test ./tests/...
```