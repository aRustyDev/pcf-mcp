# PCF-MCP Deployment Guide

This guide covers various deployment options for the PCF-MCP server, from local development to production Kubernetes deployments.

## Table of Contents

- [Local Development](#local-development)
- [Docker Deployment](#docker-deployment)
- [Docker Compose](#docker-compose)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Production Considerations](#production-considerations)
- [Monitoring Setup](#monitoring-setup)

## Local Development

### Prerequisites

- Go 1.21 or higher
- Access to a PCF instance
- Just (command runner) - optional but recommended

### Quick Start

```bash
# Clone repository
git clone https://github.com/analyst/pcf-mcp
cd pcf-mcp

# Install dependencies
go mod download

# Run with default settings (stdio transport)
go run cmd/pcf-mcp/main.go

# Run with HTTP transport
go run cmd/pcf-mcp/main.go \
  --server-transport http \
  --server-port 8080 \
  --pcf-url http://localhost:5000 \
  --pcf-api-key your-api-key
```

### Using Just

```bash
# Install dependencies
just deps

# Run tests
just test

# Build binary
just build

# Run the server
just run

# Run with HTTP transport
just run-http
```

### Development Configuration

Create a local configuration file `config.yaml`:

```yaml
server:
  transport: http
  host: localhost
  port: 8080
  
pcf:
  url: http://localhost:5000
  api_key: dev-api-key
  
logging:
  level: debug
  format: text
  
metrics:
  enabled: true
  port: 9090
```

Run with configuration file:
```bash
./bin/pcf-mcp --config config.yaml
```

## Docker Deployment

### Building the Image

```bash
# Build using Dockerfile
docker build -t pcf-mcp:latest .

# Or using Just
just docker
```

### Running the Container

#### stdio Transport (Interactive)

```bash
docker run -it --rm \
  -e PCF_MCP_PCF_URL=http://pcf.example.com \
  -e PCF_MCP_PCF_API_KEY=your-api-key \
  pcf-mcp:latest
```

#### HTTP Transport

```bash
docker run -d \
  --name pcf-mcp \
  -p 8080:8080 \
  -p 9090:9090 \
  -e PCF_MCP_SERVER_TRANSPORT=http \
  -e PCF_MCP_PCF_URL=http://pcf.example.com \
  -e PCF_MCP_PCF_API_KEY=your-api-key \
  pcf-mcp:latest
```

### Docker Configuration

All configuration can be passed via environment variables:

```bash
docker run -d \
  --name pcf-mcp \
  -p 8080:8080 \
  -e PCF_MCP_SERVER_TRANSPORT=http \
  -e PCF_MCP_SERVER_HOST=0.0.0.0 \
  -e PCF_MCP_SERVER_PORT=8080 \
  -e PCF_MCP_SERVER_AUTH_REQUIRED=true \
  -e PCF_MCP_SERVER_AUTH_TOKEN=secret-token \
  -e PCF_MCP_PCF_URL=http://pcf.example.com \
  -e PCF_MCP_PCF_API_KEY=your-api-key \
  -e PCF_MCP_LOGGING_LEVEL=info \
  -e PCF_MCP_LOGGING_FORMAT=json \
  -e PCF_MCP_METRICS_ENABLED=true \
  -e PCF_MCP_METRICS_PORT=9090 \
  -e PCF_MCP_TRACING_ENABLED=true \
  -e PCF_MCP_TRACING_EXPORTER=otlp \
  -e PCF_MCP_TRACING_ENDPOINT=http://jaeger:4317 \
  pcf-mcp:latest
```

## Docker Compose

### Basic Setup

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  pcf-mcp:
    image: pcf-mcp:latest
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      PCF_MCP_SERVER_TRANSPORT: http
      PCF_MCP_PCF_URL: ${PCF_URL}
      PCF_MCP_PCF_API_KEY: ${PCF_API_KEY}
      PCF_MCP_LOGGING_LEVEL: info
      PCF_MCP_METRICS_ENABLED: "true"
    restart: unless-stopped

  # Optional: Prometheus for metrics
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'

  # Optional: Jaeger for tracing
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"
      - "4317:4317"
    environment:
      COLLECTOR_OTLP_ENABLED: "true"

volumes:
  prometheus_data:
```

### Running with Docker Compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f pcf-mcp

# Stop all services
docker-compose down
```

## Kubernetes Deployment

### Basic Deployment

Create `kubernetes/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pcf-mcp
  labels:
    app: pcf-mcp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: pcf-mcp
  template:
    metadata:
      labels:
        app: pcf-mcp
    spec:
      containers:
      - name: pcf-mcp
        image: pcf-mcp:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: PCF_MCP_SERVER_TRANSPORT
          value: "http"
        - name: PCF_MCP_PCF_URL
          valueFrom:
            configMapKeyRef:
              name: pcf-mcp-config
              key: pcf.url
        - name: PCF_MCP_PCF_API_KEY
          valueFrom:
            secretKeyRef:
              name: pcf-mcp-secrets
              key: pcf.api.key
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 5
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: pcf-mcp
  labels:
    app: pcf-mcp
spec:
  selector:
    app: pcf-mcp
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  - name: metrics
    port: 9090
    targetPort: 9090
  type: ClusterIP
```

### ConfigMap and Secrets

Create `kubernetes/configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: pcf-mcp-config
data:
  pcf.url: "http://pcf.example.com"
  server.config: |
    server:
      transport: http
      host: 0.0.0.0
      port: 8080
      max_concurrent_tools: 20
      tool_timeout: 60s
    logging:
      level: info
      format: json
    metrics:
      enabled: true
      port: 9090
    tracing:
      enabled: true
      exporter: otlp
      endpoint: http://jaeger-collector:4317
```

Create `kubernetes/secret.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: pcf-mcp-secrets
type: Opaque
stringData:
  pcf.api.key: "your-api-key"
  auth.token: "your-bearer-token"
```

### Ingress Configuration

Create `kubernetes/ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: pcf-mcp
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - pcf-mcp.example.com
    secretName: pcf-mcp-tls
  rules:
  - host: pcf-mcp.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: pcf-mcp
            port:
              number: 8080
```

### Deployment Commands

```bash
# Create namespace
kubectl create namespace pcf-mcp

# Apply configurations
kubectl apply -f kubernetes/configmap.yaml -n pcf-mcp
kubectl apply -f kubernetes/secret.yaml -n pcf-mcp
kubectl apply -f kubernetes/deployment.yaml -n pcf-mcp
kubectl apply -f kubernetes/ingress.yaml -n pcf-mcp

# Check status
kubectl get pods -n pcf-mcp
kubectl get svc -n pcf-mcp
kubectl get ingress -n pcf-mcp

# View logs
kubectl logs -f deployment/pcf-mcp -n pcf-mcp

# Scale deployment
kubectl scale deployment pcf-mcp --replicas=5 -n pcf-mcp
```

### Horizontal Pod Autoscaler

Create `kubernetes/hpa.yaml`:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: pcf-mcp
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: pcf-mcp
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Production Considerations

### Security

1. **Network Policies**
   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: pcf-mcp-network-policy
   spec:
     podSelector:
       matchLabels:
         app: pcf-mcp
     policyTypes:
     - Ingress
     - Egress
     ingress:
     - from:
       - namespaceSelector:
           matchLabels:
             name: ingress-nginx
       ports:
       - protocol: TCP
         port: 8080
     egress:
     - to:
       - namespaceSelector: {}
       ports:
       - protocol: TCP
         port: 443
   ```

2. **Pod Security Policy**
   ```yaml
   apiVersion: policy/v1beta1
   kind: PodSecurityPolicy
   metadata:
     name: pcf-mcp-psp
   spec:
     privileged: false
     runAsUser:
       rule: MustRunAsNonRoot
     seLinux:
       rule: RunAsAny
     fsGroup:
       rule: RunAsAny
     volumes:
     - 'configMap'
     - 'secret'
     - 'emptyDir'
   ```

3. **RBAC Configuration**
   ```yaml
   apiVersion: rbac.authorization.k8s.io/v1
   kind: Role
   metadata:
     name: pcf-mcp-role
   rules:
   - apiGroups: [""]
     resources: ["configmaps", "secrets"]
     verbs: ["get", "list", "watch"]
   ```

### High Availability

1. **Multiple Replicas**: Run at least 3 replicas across availability zones
2. **Pod Disruption Budget**:
   ```yaml
   apiVersion: policy/v1
   kind: PodDisruptionBudget
   metadata:
     name: pcf-mcp-pdb
   spec:
     minAvailable: 2
     selector:
       matchLabels:
         app: pcf-mcp
   ```

3. **Anti-Affinity Rules**:
   ```yaml
   affinity:
     podAntiAffinity:
       requiredDuringSchedulingIgnoredDuringExecution:
       - labelSelector:
           matchExpressions:
           - key: app
             operator: In
             values:
             - pcf-mcp
         topologyKey: kubernetes.io/hostname
   ```

### Resource Management

1. **Resource Quotas**:
   ```yaml
   apiVersion: v1
   kind: ResourceQuota
   metadata:
     name: pcf-mcp-quota
   spec:
     hard:
       requests.cpu: "10"
       requests.memory: 10Gi
       limits.cpu: "20"
       limits.memory: 20Gi
       persistentvolumeclaims: "0"
   ```

2. **Vertical Pod Autoscaler**:
   ```yaml
   apiVersion: autoscaling.k8s.io/v1
   kind: VerticalPodAutoscaler
   metadata:
     name: pcf-mcp-vpa
   spec:
     targetRef:
       apiVersion: apps/v1
       kind: Deployment
       name: pcf-mcp
     updatePolicy:
       updateMode: "Auto"
   ```

## Monitoring Setup

### Prometheus Configuration

Add to `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'pcf-mcp'
    kubernetes_sd_configs:
    - role: pod
      namespaces:
        names:
        - pcf-mcp
    relabel_configs:
    - source_labels: [__meta_kubernetes_pod_label_app]
      action: keep
      regex: pcf-mcp
    - source_labels: [__meta_kubernetes_pod_name]
      target_label: instance
    - target_label: __address__
      replacement: 'pcf-mcp:9090'
```

### Grafana Dashboard

Import the PCF-MCP dashboard (JSON available in `docs/grafana-dashboard.json`) which includes:

- Request rate and latency
- Tool execution metrics
- Error rates
- Resource utilization
- Active connections

### Alerts

Example Prometheus alerting rules:

```yaml
groups:
- name: pcf-mcp
  rules:
  - alert: HighErrorRate
    expr: rate(pcf_mcp_tool_errors_total[5m]) > 0.1
    for: 5m
    annotations:
      summary: "High error rate detected"
      
  - alert: HighLatency
    expr: histogram_quantile(0.99, rate(pcf_mcp_tool_duration_seconds_bucket[5m])) > 5
    for: 5m
    annotations:
      summary: "High tool execution latency"
      
  - alert: PodCrashLooping
    expr: rate(kube_pod_container_status_restarts_total{namespace="pcf-mcp"}[5m]) > 0
    for: 5m
    annotations:
      summary: "Pod is crash looping"
```

## Troubleshooting Deployment Issues

### Common Issues

1. **Connection Refused**
   - Check PCF URL is accessible from the pod
   - Verify network policies allow egress
   - Check service discovery is working

2. **Authentication Failures**
   - Verify API key is correctly set in secret
   - Check secret is mounted in pod
   - Ensure PCF instance accepts the API key

3. **High Memory Usage**
   - Review concurrent tool execution limits
   - Check for memory leaks with pprof
   - Adjust resource limits

4. **Slow Performance**
   - Check PCF API latency
   - Review Prometheus metrics
   - Enable distributed tracing
   - Check for network issues

### Debug Commands

```bash
# Get pod logs
kubectl logs -f deployment/pcf-mcp -n pcf-mcp

# Exec into pod
kubectl exec -it deployment/pcf-mcp -n pcf-mcp -- /bin/sh

# Port forward for local debugging
kubectl port-forward deployment/pcf-mcp 8080:8080 9090:9090 -n pcf-mcp

# Describe pod for events
kubectl describe pod -l app=pcf-mcp -n pcf-mcp

# Check environment variables
kubectl exec deployment/pcf-mcp -n pcf-mcp -- env | grep PCF_MCP
```