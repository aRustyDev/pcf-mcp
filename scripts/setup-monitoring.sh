#!/bin/bash
set -euo pipefail

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Setting up PCF-MCP Monitoring Stack...${NC}"

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}kubectl not found. Please install kubectl first.${NC}"
    exit 1
fi

# Check if cluster is accessible
if ! kubectl cluster-info &> /dev/null; then
    echo -e "${RED}Cannot connect to Kubernetes cluster. Please configure kubectl.${NC}"
    exit 1
fi

# Create monitoring namespace
echo -e "\n${GREEN}Creating monitoring namespace...${NC}"
kubectl apply -f deployments/monitoring/namespace.yaml

# Generate Grafana admin password if not exists
if ! kubectl get secret grafana-admin -n monitoring &> /dev/null; then
    echo -e "\n${GREEN}Generating Grafana admin password...${NC}"
    GRAFANA_PASSWORD=$(openssl rand -base64 32)
    kubectl create secret generic grafana-admin \
        --from-literal=password="$GRAFANA_PASSWORD" \
        -n monitoring
    echo -e "${YELLOW}Grafana admin password: $GRAFANA_PASSWORD${NC}"
    echo -e "${YELLOW}Please save this password securely!${NC}"
fi

# Apply monitoring stack with kustomize
echo -e "\n${GREEN}Deploying monitoring stack...${NC}"
kubectl apply -k deployments/monitoring/

# Wait for deployments to be ready
echo -e "\n${GREEN}Waiting for deployments to be ready...${NC}"
kubectl wait --for=condition=available --timeout=300s \
    deployment/grafana deployment/alertmanager \
    -n monitoring

kubectl wait --for=condition=ready --timeout=300s \
    statefulset/prometheus \
    -n monitoring

# Create port forwards for local access
echo -e "\n${GREEN}Creating port forwards for local access...${NC}"
echo -e "${YELLOW}You can access the services at:${NC}"
echo -e "  Prometheus: http://localhost:9090"
echo -e "  Grafana: http://localhost:3000 (admin/[generated password])"
echo -e "  Alertmanager: http://localhost:9093"

# Optionally create ingresses
read -p "Do you want to create ingresses for external access? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    read -p "Enter your domain (e.g., example.com): " DOMAIN
    
    cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: prometheus
  namespace: monitoring
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/auth-type: basic
    nginx.ingress.kubernetes.io/auth-secret: basic-auth
    nginx.ingress.kubernetes.io/auth-realm: 'Authentication Required'
spec:
  tls:
    - hosts:
        - prometheus.$DOMAIN
      secretName: prometheus-tls
  rules:
    - host: prometheus.$DOMAIN
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: prometheus
                port:
                  number: 9090
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: grafana
  namespace: monitoring
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
    - hosts:
        - grafana.$DOMAIN
      secretName: grafana-tls
  rules:
    - host: grafana.$DOMAIN
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: grafana
                port:
                  number: 3000
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: alertmanager
  namespace: monitoring
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/auth-type: basic
    nginx.ingress.kubernetes.io/auth-secret: basic-auth
    nginx.ingress.kubernetes.io/auth-realm: 'Authentication Required'
spec:
  tls:
    - hosts:
        - alertmanager.$DOMAIN
      secretName: alertmanager-tls
  rules:
    - host: alertmanager.$DOMAIN
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: alertmanager
                port:
                  number: 9093
EOF

    echo -e "\n${GREEN}Ingresses created:${NC}"
    echo -e "  Prometheus: https://prometheus.$DOMAIN"
    echo -e "  Grafana: https://grafana.$DOMAIN"
    echo -e "  Alertmanager: https://alertmanager.$DOMAIN"
fi

# Show monitoring status
echo -e "\n${GREEN}Monitoring stack status:${NC}"
kubectl get all -n monitoring

echo -e "\n${GREEN}Monitoring setup complete!${NC}"
echo -e "${YELLOW}Remember to:${NC}"
echo -e "  1. Configure Alertmanager with your Slack/PagerDuty credentials"
echo -e "  2. Update PCF-MCP deployment to include monitoring annotations"
echo -e "  3. Import additional Grafana dashboards as needed"
echo -e "  4. Set up backup for Prometheus data"