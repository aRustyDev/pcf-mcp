# Production Monitoring Guide

This guide covers monitoring PCF-MCP in production environments.

## Overview

PCF-MCP provides comprehensive monitoring through:
- Prometheus metrics
- Distributed tracing with OpenTelemetry
- Structured logging
- Health checks
- Custom Grafana dashboards
- Alerting rules

## Quick Start

Deploy the complete monitoring stack:

```bash
./scripts/setup-monitoring.sh
```

This sets up:
- Prometheus for metrics collection
- Grafana for visualization
- Alertmanager for alert routing

## Metrics

### Application Metrics

PCF-MCP exports the following custom metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `pcf_mcp_tool_executions_total` | Counter | Total tool executions |
| `pcf_mcp_tool_errors_total` | Counter | Total tool execution errors |
| `pcf_mcp_tool_duration_seconds` | Histogram | Tool execution duration |
| `pcf_mcp_active_tools` | Gauge | Currently executing tools |
| `pcf_mcp_tool_queue_size` | Gauge | Pending tools in queue |

### HTTP Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `http_requests_total` | Counter | Total HTTP requests |
| `http_request_duration_seconds` | Histogram | HTTP request duration |
| `http_request_size_bytes` | Histogram | HTTP request size |
| `http_response_size_bytes` | Histogram | HTTP response size |

### System Metrics

Standard Go runtime metrics are also exported:
- Memory usage (`go_memstats_*`)
- Goroutines (`go_goroutines`)
- GC statistics (`go_gc_*`)

## Grafana Dashboards

### PCF-MCP Dashboard

The main dashboard includes:
- Request rate and error rate
- P50/P95/P99 latencies
- Tool execution metrics
- Resource usage
- Active connections

Import the dashboard:
```bash
kubectl create configmap grafana-dashboards \
  --from-file=pcf-mcp-dashboard.json=docs/grafana-dashboard.json \
  -n monitoring
```

### Custom Dashboards

Create custom dashboards for specific use cases:

```json
{
  "dashboard": {
    "title": "PCF Tool Performance",
    "panels": [
      {
        "title": "Tool Execution Time by Tool",
        "targets": [{
          "expr": "histogram_quantile(0.95, sum(rate(pcf_mcp_tool_duration_seconds_bucket[5m])) by (le, tool))"
        }]
      }
    ]
  }
}
```

## Alerts

### Critical Alerts

Alerts that require immediate attention:

1. **NoActiveInstances**
   - All PCF-MCP instances are down
   - Check pod status and logs
   - Verify PCF connectivity

2. **VeryHighErrorRate**
   - Error rate > 5%
   - Check recent deployments
   - Review error logs

3. **CertificateExpiringSoon**
   - SSL certificate expiring in < 7 days
   - Renew certificates immediately

### Warning Alerts

Alerts that require investigation:

1. **HighErrorRate**
   - Error rate > 1%
   - Monitor for escalation
   - Check specific tool failures

2. **HighLatency**
   - P99 latency > 5s
   - Check PCF API performance
   - Review resource usage

3. **HighMemoryUsage**
   - Memory > 80% of limit
   - Consider scaling up
   - Check for memory leaks

## Log Aggregation

### Structured Logging

PCF-MCP uses structured JSON logging:

```json
{
  "time": "2024-01-20T10:30:45Z",
  "level": "INFO",
  "msg": "Tool executed successfully",
  "tool": "list_projects",
  "duration": 0.125,
  "trace_id": "abc123",
  "span_id": "def456"
}
```

### Log Queries

Useful LogQL queries for Loki:

```logql
# All errors
{app="pcf-mcp"} |= "ERROR"

# Slow tool executions
{app="pcf-mcp"} | json | duration > 1

# Specific tool failures
{app="pcf-mcp"} | json | tool="create_issue" | level="ERROR"

# Rate of errors
sum(rate({app="pcf-mcp"} |= "ERROR" [5m]))
```

## Distributed Tracing

### Trace Collection

Configure trace collection with Jaeger:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: pcf-mcp-config
data:
  config.yaml: |
    tracing:
      enabled: true
      endpoint: "jaeger-collector:4317"
      sample_rate: 0.1
```

### Trace Analysis

Common trace queries:

1. **Slow requests**
   ```
   service="pcf-mcp" AND duration > 1s
   ```

2. **Failed requests**
   ```
   service="pcf-mcp" AND error=true
   ```

3. **Specific tool traces**
   ```
   service="pcf-mcp" AND operation="ExecuteTool" AND tool="create_host"
   ```

## Performance Monitoring

### Load Testing

Run load tests to establish baselines:

```bash
# Run k6 load test
k6 run scripts/load-test.js

# Run benchmarks
./scripts/benchmark.sh
```

### Performance Baselines

Expected performance metrics:

| Metric | Baseline | Warning | Critical |
|--------|----------|---------|----------|
| P99 Latency | < 1s | > 5s | > 10s |
| Error Rate | < 0.1% | > 1% | > 5% |
| CPU Usage | < 50% | > 80% | > 90% |
| Memory Usage | < 60% | > 80% | > 90% |

## Troubleshooting

### High Memory Usage

1. Check for memory leaks:
   ```bash
   kubectl exec -it pcf-mcp-xxx -- go tool pprof -http=:8080 http://localhost:6060/debug/pprof/heap
   ```

2. Review heap profile:
   ```bash
   curl http://localhost:6060/debug/pprof/heap > heap.prof
   go tool pprof heap.prof
   ```

### High CPU Usage

1. Generate CPU profile:
   ```bash
   curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
   go tool pprof cpu.prof
   ```

2. Check goroutine count:
   ```bash
   curl http://localhost:6060/debug/pprof/goroutine
   ```

### Slow Requests

1. Enable trace sampling:
   ```yaml
   tracing:
     sample_rate: 1.0  # Sample all requests temporarily
   ```

2. Analyze slow spans in Jaeger

3. Check PCF API latency

## Monitoring Checklist

### Daily Checks

- [ ] Review error rate trends
- [ ] Check for any firing alerts
- [ ] Verify backup completion
- [ ] Monitor resource usage

### Weekly Checks

- [ ] Analyze performance trends
- [ ] Review capacity metrics
- [ ] Check for unusual patterns
- [ ] Update dashboards as needed

### Monthly Checks

- [ ] Review and tune alerts
- [ ] Analyze long-term trends
- [ ] Plan capacity adjustments
- [ ] Update monitoring documentation

## Integration with External Systems

### PagerDuty Integration

Configure Alertmanager for PagerDuty:

```yaml
receivers:
  - name: 'pagerduty'
    pagerduty_configs:
      - service_key: 'YOUR_SERVICE_KEY'
        description: '{{ .GroupLabels.alertname }}'
```

### Slack Integration

Configure Slack notifications:

```yaml
receivers:
  - name: 'slack'
    slack_configs:
      - api_url: 'YOUR_WEBHOOK_URL'
        channel: '#pcf-mcp-alerts'
        title: 'PCF-MCP Alert'
```

### Custom Webhooks

Send alerts to custom endpoints:

```yaml
receivers:
  - name: 'webhook'
    webhook_configs:
      - url: 'https://your-webhook.example.com'
        http_config:
          bearer_token: 'YOUR_TOKEN'
```

## Best Practices

1. **Set Appropriate Thresholds**
   - Base thresholds on historical data
   - Start conservative and tune over time
   - Consider time of day variations

2. **Minimize Alert Fatigue**
   - Group related alerts
   - Use inhibition rules
   - Implement proper severity levels

3. **Maintain Dashboards**
   - Keep dashboards focused
   - Remove unused panels
   - Version control dashboard JSON

4. **Regular Reviews**
   - Weekly alert review
   - Monthly metric analysis
   - Quarterly capacity planning

5. **Documentation**
   - Document custom queries
   - Maintain runbooks
   - Record incident learnings