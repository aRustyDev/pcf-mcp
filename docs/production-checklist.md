# PCF-MCP Production Checklist

This checklist ensures PCF-MCP is properly configured and secured for production deployment.

## Pre-Deployment

### Security

- [ ] **API Keys**: Ensure all API keys are stored in secret management system
- [ ] **TLS/SSL**: Configure HTTPS with valid certificates
- [ ] **Authentication**: Enable and configure authentication for HTTP transport
- [ ] **Network Policies**: Apply Kubernetes network policies
- [ ] **Security Scanning**: Run security scans and address findings
  ```bash
  make security
  ```
- [ ] **Dependency Audit**: Check for vulnerable dependencies
  ```bash
  go list -json -m all | nancy sleuth
  ```
- [ ] **Container Scanning**: Scan Docker image for vulnerabilities
  ```bash
  trivy image pcf-mcp:latest
  ```

### Configuration

- [ ] **Production Config**: Use production configuration values
- [ ] **Resource Limits**: Set appropriate CPU/memory limits
- [ ] **Timeouts**: Configure appropriate timeout values
- [ ] **Rate Limiting**: Enable and configure rate limiting
- [ ] **Log Level**: Set to "info" or "warn" (not "debug")
- [ ] **Metrics**: Enable metrics collection
- [ ] **Tracing**: Configure distributed tracing with appropriate sampling

### Infrastructure

- [ ] **High Availability**: Deploy multiple replicas (minimum 3)
- [ ] **Pod Disruption Budget**: Configure PDB for availability
- [ ] **Anti-Affinity**: Ensure pods are distributed across nodes
- [ ] **Health Checks**: Configure liveness and readiness probes
- [ ] **Autoscaling**: Configure HPA for automatic scaling
- [ ] **Backup**: Set up configuration and secret backups

## Deployment

### Initial Deployment

- [ ] **Namespace**: Create dedicated namespace with proper labels
- [ ] **RBAC**: Apply least-privilege RBAC policies
- [ ] **Secrets**: Deploy secrets using external secret manager
- [ ] **ConfigMaps**: Deploy configuration with production values
- [ ] **Network Policies**: Apply network segmentation rules
- [ ] **Service Accounts**: Use dedicated service accounts

### Validation

- [ ] **Health Check**: Verify all pods are healthy
  ```bash
  kubectl get pods -n pcf-mcp
  kubectl describe pods -n pcf-mcp
  ```
- [ ] **Endpoints**: Test all service endpoints
  ```bash
  curl https://pcf-mcp.example.com/health
  ```
- [ ] **Metrics**: Verify metrics are being collected
  ```bash
  curl https://pcf-mcp.example.com/metrics
  ```
- [ ] **Logs**: Check logs for errors
  ```bash
  kubectl logs -f deployment/pcf-mcp -n pcf-mcp
  ```

## Monitoring Setup

### Metrics

- [ ] **Prometheus**: Configure Prometheus scraping
- [ ] **Grafana Dashboard**: Import PCF-MCP dashboard
- [ ] **Alerts**: Configure alerting rules
  - High error rate (> 1%)
  - High latency (P99 > 5s)
  - Pod restarts
  - Certificate expiration

### Logging

- [ ] **Log Aggregation**: Configure log forwarding (Fluentd/Fluentbit)
- [ ] **Log Retention**: Set appropriate retention policies
- [ ] **Log Alerts**: Configure alerts for ERROR logs
- [ ] **Audit Logs**: Enable and monitor audit logs

### Tracing

- [ ] **Trace Collection**: Configure trace collector (Jaeger/Zipkin)
- [ ] **Sampling Rate**: Set production sampling rate (0.1-1%)
- [ ] **Trace Retention**: Configure retention period
- [ ] **Performance Analysis**: Set up trace analysis dashboards

## Performance

### Load Testing

- [ ] **Baseline**: Establish performance baseline
- [ ] **Load Test**: Run load tests to verify capacity
  ```bash
  k6 run scripts/load-test.js
  ```
- [ ] **Stress Test**: Verify behavior under stress
- [ ] **Soak Test**: Run extended duration tests

### Optimization

- [ ] **Resource Tuning**: Optimize resource requests/limits
- [ ] **Connection Pooling**: Verify connection pool settings
- [ ] **Caching**: Enable appropriate caching
- [ ] **Compression**: Enable response compression

## Security Hardening

### Runtime Security

- [ ] **Security Context**: Enforce security contexts
  - Non-root user
  - Read-only root filesystem
  - No privilege escalation
  - Drop all capabilities
- [ ] **Pod Security Policy**: Apply PSP or Pod Security Standards
- [ ] **Admission Controllers**: Enable security admission controllers
- [ ] **Runtime Protection**: Consider runtime security tools (Falco)

### Network Security

- [ ] **Ingress Rules**: Restrict ingress to required sources
- [ ] **Egress Rules**: Limit egress to required destinations
- [ ] **Service Mesh**: Consider service mesh for zero-trust
- [ ] **WAF**: Deploy Web Application Firewall

## Operational Readiness

### Documentation

- [ ] **Runbook**: Create operational runbook
- [ ] **Architecture Diagram**: Document deployed architecture
- [ ] **Contact List**: Maintain on-call contact list
- [ ] **SLOs**: Define Service Level Objectives

### Disaster Recovery

- [ ] **Backup Strategy**: Document backup procedures
- [ ] **Recovery Plan**: Test disaster recovery plan
- [ ] **RTO/RPO**: Define and test recovery objectives
- [ ] **Failover**: Test failover procedures

### Maintenance

- [ ] **Update Process**: Document update procedures
- [ ] **Rollback Plan**: Test rollback procedures
- [ ] **Maintenance Windows**: Define maintenance windows
- [ ] **Change Management**: Establish change process

## Post-Deployment

### Validation

- [ ] **Smoke Tests**: Run production smoke tests
- [ ] **Integration Tests**: Verify PCF connectivity
- [ ] **Performance**: Verify meets SLOs
- [ ] **Security Scan**: Run post-deployment security scan

### Monitoring

- [ ] **Dashboard Review**: Verify all metrics visible
- [ ] **Alert Testing**: Test critical alerts
- [ ] **Log Review**: Check for any errors/warnings
- [ ] **Trace Sampling**: Verify traces being collected

### Sign-off

- [ ] **Security Review**: Security team approval
- [ ] **Operations Review**: Operations team approval
- [ ] **Performance Review**: Performance meets requirements
- [ ] **Documentation**: All documentation complete

## Ongoing Maintenance

### Daily
- [ ] Review error logs
- [ ] Check metrics dashboards
- [ ] Verify backup completion

### Weekly
- [ ] Review performance trends
- [ ] Check for security updates
- [ ] Review capacity metrics

### Monthly
- [ ] Security vulnerability scan
- [ ] Disaster recovery test
- [ ] Performance baseline review
- [ ] Certificate expiration check

### Quarterly
- [ ] Full security audit
- [ ] Load testing
- [ ] Documentation review
- [ ] Dependency updates

## Emergency Procedures

### Incident Response

1. **Identify**: Detect and classify incident
2. **Contain**: Isolate affected components
3. **Investigate**: Analyze logs and traces
4. **Remediate**: Apply fixes
5. **Document**: Create incident report

### Rollback Procedure

```bash
# Helm rollback
helm rollback pcf-mcp -n pcf-mcp

# Kubectl rollback
kubectl rollout undo deployment/pcf-mcp -n pcf-mcp
```

### Emergency Contacts

- On-call Engineer: [PHONE]
- Security Team: [EMAIL]
- Platform Team: [SLACK]
- Escalation: [MANAGER]

## Compliance

- [ ] **Data Residency**: Verify data location compliance
- [ ] **Audit Logging**: Enable audit trail
- [ ] **Access Control**: Implement least privilege
- [ ] **Encryption**: Verify encryption at rest and in transit
- [ ] **Compliance Scan**: Run compliance verification