# PCF-MCP Troubleshooting Guide

This guide helps diagnose and resolve common issues with PCF-MCP deployments.

## Table of Contents

- [Common Issues](#common-issues)
- [Diagnostic Tools](#diagnostic-tools)
- [Error Messages](#error-messages)
- [Performance Issues](#performance-issues)
- [Connection Problems](#connection-problems)
- [Authentication Issues](#authentication-issues)
- [Tool Execution Failures](#tool-execution-failures)
- [Deployment Issues](#deployment-issues)
- [Debug Mode](#debug-mode)

## Common Issues

### Server Won't Start

**Symptoms:**
- Process exits immediately
- No output or error messages

**Possible Causes:**
1. Invalid configuration
2. Port already in use
3. Missing required environment variables

**Solutions:**

```bash
# Check configuration validity
./pcf-mcp --config config.yaml --log-level debug

# Check if port is in use
lsof -i :8080
netstat -an | grep 8080

# Verify environment variables
env | grep PCF_MCP

# Run with minimal config to isolate issues
./pcf-mcp --server-transport stdio --log-level debug
```

### Cannot Connect to PCF

**Symptoms:**
- "Connection refused" errors
- Timeout errors
- 401/403 authentication errors

**Solutions:**

```bash
# Test PCF connectivity
curl -v https://pcf.example.com/api/projects \
  -H "Authorization: Bearer your-api-key"

# Check DNS resolution
nslookup pcf.example.com
dig pcf.example.com

# Test from within container/pod
docker exec pcf-mcp curl https://pcf.example.com
kubectl exec deployment/pcf-mcp -- curl https://pcf.example.com

# Verify API key
echo $PCF_MCP_PCF_API_KEY
```

### High Memory Usage

**Symptoms:**
- OOMKilled pods
- Slow response times
- Memory alerts

**Solutions:**

```bash
# Check current memory usage
docker stats pcf-mcp
kubectl top pod -l app=pcf-mcp

# Limit concurrent tool executions
export PCF_MCP_SERVER_MAX_CONCURRENT_TOOLS=5

# Enable memory profiling
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

## Diagnostic Tools

### Health Check

```bash
# Local
curl http://localhost:8080/health

# Docker
docker exec pcf-mcp curl http://localhost:8080/health

# Kubernetes
kubectl exec deployment/pcf-mcp -- curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00Z",
  "version": "0.1.0"
}
```

### Metrics Analysis

```bash
# Get current metrics
curl http://localhost:8080/metrics | grep pcf_mcp

# Key metrics to check
curl -s http://localhost:8080/metrics | grep -E "(pcf_mcp_tool_errors_total|pcf_mcp_tool_duration_seconds)"

# Check error rates
curl -s http://localhost:8080/metrics | grep pcf_mcp_tool_errors_total
```

### Log Analysis

```bash
# View logs with debug level
./pcf-mcp --log-level debug 2>&1 | jq '.'

# Filter error logs
docker logs pcf-mcp 2>&1 | jq 'select(.level == "ERROR")'

# Search for specific tool errors
kubectl logs deployment/pcf-mcp | jq 'select(.tool != null and .error != null)'

# Get logs with timestamps
kubectl logs deployment/pcf-mcp --timestamps=true
```

### Trace Analysis

Enable tracing for detailed request flow:

```yaml
tracing:
  enabled: true
  exporter: "jaeger"
  endpoint: "http://jaeger:14268/api/traces"
  sampling_rate: 1.0  # 100% for debugging
```

Query traces:
```bash
# Find slow requests
curl "http://jaeger:16686/api/traces?service=pcf-mcp&minDuration=5s"

# Find failed requests
curl "http://jaeger:16686/api/traces?service=pcf-mcp&tags=error%3Dtrue"
```

## Error Messages

### Configuration Errors

**Error:** `invalid configuration: server.port must be between 1 and 65535`
```bash
# Fix: Use valid port number
export PCF_MCP_SERVER_PORT=8080
```

**Error:** `invalid transport type: websocket (must be 'stdio' or 'http')`
```bash
# Fix: Use supported transport
./pcf-mcp --server-transport http
```

**Error:** `failed to unmarshal config: yaml: unmarshal errors`
```bash
# Fix: Validate YAML syntax
yamllint config.yaml
# Or use JSON
./pcf-mcp --config config.json
```

### Connection Errors

**Error:** `failed to connect to PCF: dial tcp: i/o timeout`
```bash
# Check network connectivity
ping pcf.example.com
traceroute pcf.example.com

# Increase timeout
export PCF_MCP_PCF_TIMEOUT=60s

# Check firewall rules
sudo iptables -L -n
```

**Error:** `x509: certificate signed by unknown authority`
```bash
# For development only:
export PCF_MCP_PCF_INSECURE_SKIP_VERIFY=true

# For production: Add CA certificate
export SSL_CERT_FILE=/path/to/ca-bundle.crt
```

### Authentication Errors

**Error:** `401 Unauthorized: Invalid API key`
```bash
# Verify API key
echo $PCF_MCP_PCF_API_KEY | base64 -d  # If encoded

# Test directly
curl -H "Authorization: Bearer $PCF_MCP_PCF_API_KEY" https://pcf.example.com/api/projects
```

**Error:** `Authorization header required`
```bash
# Enable auth and set token
export PCF_MCP_SERVER_AUTH_REQUIRED=true
export PCF_MCP_SERVER_AUTH_TOKEN="secret-token"

# Include in requests
curl -H "Authorization: Bearer secret-token" http://localhost:8080/tools
```

## Performance Issues

### Slow Tool Execution

**Diagnosis:**
```bash
# Check tool execution metrics
curl -s http://localhost:8080/metrics | grep pcf_mcp_tool_duration_seconds

# Enable debug logging
export PCF_MCP_LOGGING_LEVEL=debug

# Check PCF API latency
time curl https://pcf.example.com/api/projects
```

**Solutions:**
1. Increase timeouts:
   ```yaml
   server:
     tool_timeout: 120s
   pcf:
     timeout: 60s
   ```

2. Optimize concurrent executions:
   ```yaml
   server:
     max_concurrent_tools: 20
   ```

3. Enable connection pooling (default)

### High CPU Usage

**Diagnosis:**
```bash
# Profile CPU usage
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Check goroutine count
curl http://localhost:8080/debug/pprof/goroutine?debug=1
```

**Solutions:**
1. Reduce concurrent operations
2. Check for infinite loops in logs
3. Review trace sampling rate

## Connection Problems

### Intermittent Failures

**Symptoms:**
- Random connection timeouts
- "Connection reset by peer"
- Inconsistent response times

**Diagnosis:**
```bash
# Monitor connection pool
curl http://localhost:8080/debug/pprof/goroutine | grep http

# Check keep-alive settings
sysctl net.ipv4.tcp_keepalive_time

# Test network stability
mtr pcf.example.com
```

**Solutions:**
1. Enable retries:
   ```yaml
   pcf:
     max_retries: 5
   ```

2. Adjust timeouts:
   ```yaml
   pcf:
     timeout: 45s
   ```

3. Use connection pooling (enabled by default)

### DNS Issues

**Symptoms:**
- "No such host" errors
- Slow initial connections

**Solutions:**
```bash
# Check DNS resolution
nslookup pcf.example.com
dig pcf.example.com @8.8.8.8

# Use IP address temporarily
export PCF_MCP_PCF_URL=https://192.168.1.100:5000

# Configure custom DNS
echo "nameserver 8.8.8.8" > /etc/resolv.conf
```

## Authentication Issues

### Token Validation Failures

**Symptoms:**
- 401 errors despite correct token
- "Invalid authorization format"

**Diagnosis:**
```bash
# Check token format
echo $PCF_MCP_SERVER_AUTH_TOKEN | wc -c

# Verify header format
curl -v -H "Authorization: Bearer $TOKEN" http://localhost:8080/info

# Test without auth
curl http://localhost:8080/health  # Should work
```

**Solutions:**
1. Ensure token doesn't contain newlines:
   ```bash
   export PCF_MCP_SERVER_AUTH_TOKEN=$(echo -n "your-token")
   ```

2. Check for special characters:
   ```bash
   export PCF_MCP_SERVER_AUTH_TOKEN='complex!token@with#special$chars'
   ```

## Tool Execution Failures

### Tool Not Found

**Error:** `tool not found: list_project` (typo)

**Solution:**
```bash
# List available tools
curl http://localhost:8080/tools | jq '.tools[].name'

# Correct tool name
curl -X POST http://localhost:8080/tools/list_projects
```

### Parameter Validation Errors

**Error:** `project_id parameter must be a string`

**Solution:**
```bash
# Check tool schema
curl http://localhost:8080/tools | jq '.tools[] | select(.name=="list_hosts")'

# Provide correct parameters
curl -X POST http://localhost:8080/tools/list_hosts \
  -H "Content-Type: application/json" \
  -d '{"project_id": "proj-123"}'
```

### Timeout Errors

**Error:** `tool execution timeout after 60s`

**Solutions:**
1. Increase tool timeout:
   ```bash
   export PCF_MCP_SERVER_TOOL_TIMEOUT=120s
   ```

2. Check PCF API performance:
   ```bash
   time curl https://pcf.example.com/api/projects
   ```

3. Enable tracing to identify bottlenecks

## Deployment Issues

### Kubernetes Pod Crashes

**Symptoms:**
- CrashLoopBackOff
- OOMKilled
- Liveness probe failures

**Diagnosis:**
```bash
# Check pod events
kubectl describe pod -l app=pcf-mcp

# View previous logs
kubectl logs -p deployment/pcf-mcp

# Check resource usage
kubectl top pod -l app=pcf-mcp
```

**Solutions:**

1. Increase resource limits:
   ```yaml
   resources:
     requests:
       memory: "256Mi"
       cpu: "200m"
     limits:
       memory: "1Gi"
       cpu: "1000m"
   ```

2. Adjust probe settings:
   ```yaml
   livenessProbe:
     initialDelaySeconds: 30
     periodSeconds: 60
     timeoutSeconds: 10
   ```

### Docker Container Exits

**Diagnosis:**
```bash
# Check exit code
docker ps -a | grep pcf-mcp

# View detailed logs
docker logs --details pcf-mcp

# Inspect container
docker inspect pcf-mcp | jq '.[0].State'
```

**Common Solutions:**
1. Missing environment variables
2. Port binding conflicts
3. Incorrect file permissions

## Debug Mode

### Enable Comprehensive Debugging

```bash
# Maximum verbosity
export PCF_MCP_LOGGING_LEVEL=debug
export PCF_MCP_LOGGING_FORMAT=text
export PCF_MCP_LOGGING_ADD_SOURCE=true

# Enable all observability
export PCF_MCP_METRICS_ENABLED=true
export PCF_MCP_TRACING_ENABLED=true
export PCF_MCP_TRACING_SAMPLING_RATE=1.0

# Run with debug output
./pcf-mcp 2>&1 | tee debug.log
```

### Debug Endpoints (Development Only)

Add pprof endpoints for profiling:

```go
import _ "net/http/pprof"

// Access at:
// http://localhost:8080/debug/pprof/
// http://localhost:8080/debug/pprof/heap
// http://localhost:8080/debug/pprof/goroutine
// http://localhost:8080/debug/pprof/trace?seconds=5
```

### Trace Specific Requests

```bash
# Add trace header
curl -H "X-Trace-ID: debug-123" http://localhost:8080/tools/list_projects

# Search logs for trace
docker logs pcf-mcp | grep "debug-123"
```

## Getting Help

If you're still experiencing issues:

1. **Check the logs** with debug level enabled
2. **Review the metrics** for anomalies
3. **Enable tracing** for detailed flow analysis
4. **Search existing issues** on GitHub
5. **Create a new issue** with:
   - PCF-MCP version
   - Configuration (sanitized)
   - Error messages
   - Steps to reproduce
   - Debug logs

### Useful Commands Summary

```bash
# Quick health check
curl -s http://localhost:8080/health | jq .

# Check error rate
curl -s http://localhost:8080/metrics | grep -E "pcf_mcp_tool_errors_total|pcf_mcp_requests_total" | grep -v "^#"

# Recent errors in logs
kubectl logs deployment/pcf-mcp --since=1h | jq 'select(.level == "ERROR")'

# Test tool execution
curl -X POST http://localhost:8080/tools/list_projects \
  -H "Content-Type: application/json" \
  -d '{}' | jq .

# Full debug mode
PCF_MCP_LOGGING_LEVEL=debug PCF_MCP_LOGGING_FORMAT=text ./pcf-mcp
```