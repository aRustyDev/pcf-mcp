# Kubernetes Deployment for PCF-MCP

This directory contains Kubernetes manifests and Helm charts for deploying PCF-MCP.

## Directory Structure

```
kubernetes/
├── base/                    # Base Kubernetes manifests
│   ├── namespace.yaml      # Namespace definition
│   ├── configmap.yaml      # Configuration
│   ├── secret.yaml         # Secrets (API keys)
│   ├── deployment.yaml     # Main deployment
│   ├── service.yaml        # Services
│   ├── serviceaccount.yaml # Service account
│   ├── networkpolicy.yaml  # Network policies
│   └── kustomization.yaml  # Kustomize config
├── overlays/               # Environment-specific overlays
│   ├── development/        # Dev environment
│   └── production/         # Prod environment
└── examples/               # Example configurations
```

## Quick Start

### Using kubectl

Deploy to the default namespace:

```bash
kubectl apply -f kubernetes/base/
```

### Using Kustomize

Deploy development environment:

```bash
kubectl apply -k kubernetes/overlays/development/
```

Deploy production environment:

```bash
kubectl apply -k kubernetes/overlays/production/
```

### Using Helm

Install with default values:

```bash
helm install pcf-mcp charts/pcf-mcp/
```

Install with custom values:

```bash
helm install pcf-mcp charts/pcf-mcp/ \
  --set config.pcf.url=https://pcf.example.com \
  --set config.pcf.apiKey=your-api-key \
  --set config.server.authToken=your-auth-token
```

## Configuration

### Required Configuration

Before deploying, you must set:

1. **PCF URL**: The URL of your PCF instance
2. **PCF API Key**: Your PCF API authentication key

### Setting Secrets

#### Using kubectl

Edit the secret before applying:

```bash
# Edit kubernetes/base/secret.yaml
vim kubernetes/base/secret.yaml

# Or create from command line
kubectl create secret generic pcf-mcp-secrets \
  --from-literal=pcf-api-key=your-api-key \
  --from-literal=auth-token=your-auth-token \
  -n pcf-mcp
```

#### Using Helm

```bash
helm install pcf-mcp charts/pcf-mcp/ \
  --set config.pcf.apiKey=your-api-key \
  --set config.server.authToken=your-auth-token
```

#### Using External Secrets

For production, use external secret managers:

```yaml
# Example with Kubernetes External Secrets Operator
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: pcf-mcp-secrets
spec:
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore
  target:
    name: pcf-mcp-secrets
  data:
  - secretKey: pcf-api-key
    remoteRef:
      key: pcf-mcp/api-key
  - secretKey: auth-token
    remoteRef:
      key: pcf-mcp/auth-token
```

## Environment Overlays

### Development

- Single replica for resource efficiency
- Debug logging enabled
- Relaxed security policies
- No authentication required
- Smaller resource requests/limits

```bash
kubectl apply -k kubernetes/overlays/development/
```

### Production

- High availability with 5 replicas
- Production logging (JSON format)
- Strict security policies
- Authentication required
- Horizontal pod autoscaling
- Pod disruption budgets
- Network policies
- Ingress with TLS

```bash
kubectl apply -k kubernetes/overlays/production/
```

## Monitoring

### Prometheus Integration

The deployment includes Prometheus annotations for automatic scraping:

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "9090"
  prometheus.io/path: "/metrics"
```

### Grafana Dashboard

Import the dashboard from `docs/grafana-dashboard.json`:

```bash
kubectl create configmap pcf-mcp-dashboard \
  --from-file=docs/grafana-dashboard.json \
  -n monitoring
```

## Scaling

### Manual Scaling

```bash
kubectl scale deployment pcf-mcp --replicas=10 -n pcf-mcp
```

### Horizontal Pod Autoscaler

The production overlay includes HPA configuration:

```bash
# View HPA status
kubectl get hpa -n pcf-mcp-prod

# Describe HPA
kubectl describe hpa pcf-mcp -n pcf-mcp-prod
```

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n pcf-mcp
kubectl describe pod <pod-name> -n pcf-mcp
```

### View Logs

```bash
kubectl logs -f deployment/pcf-mcp -n pcf-mcp
kubectl logs -f <pod-name> -n pcf-mcp --previous
```

### Test Connectivity

```bash
# Port forward for local testing
kubectl port-forward deployment/pcf-mcp 8080:8080 -n pcf-mcp

# Test health endpoint
curl http://localhost:8080/health
```

### Common Issues

1. **ImagePullBackOff**: Check image name and registry credentials
2. **CrashLoopBackOff**: Check logs for startup errors
3. **Pending Pods**: Check resource availability and node selectors
4. **Connection Refused**: Verify PCF URL and network policies

## Security Considerations

1. **Never commit secrets** to version control
2. **Use RBAC** to limit access to secrets
3. **Enable network policies** in production
4. **Use TLS** for all external communication
5. **Rotate credentials** regularly
6. **Audit access** to the namespace

## Best Practices

1. **Use namespaces** to isolate environments
2. **Set resource limits** to prevent resource exhaustion
3. **Configure pod disruption budgets** for high availability
4. **Use health checks** for proper pod lifecycle
5. **Enable autoscaling** for production workloads
6. **Monitor metrics** and set up alerts
7. **Use external secrets** for sensitive data
8. **Regular backups** of configuration

## Helm Chart

See [charts/pcf-mcp/README.md](../charts/pcf-mcp/README.md) for detailed Helm documentation.

## Kustomize

Kustomize allows you to customize raw YAML files for multiple environments:

```bash
# Show generated manifests
kubectl kustomize kubernetes/overlays/production/

# Apply directly
kubectl apply -k kubernetes/overlays/production/

# Build to file
kubectl kustomize kubernetes/overlays/production/ > production.yaml
```

## GitOps Integration

For GitOps workflows with ArgoCD or Flux:

```yaml
# ArgoCD Application
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: pcf-mcp
spec:
  source:
    repoURL: https://github.com/analyst/pcf-mcp
    targetRevision: main
    path: kubernetes/overlays/production
  destination:
    server: https://kubernetes.default.svc
    namespace: pcf-mcp-prod
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```