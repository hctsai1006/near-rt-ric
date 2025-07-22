# O-RAN Near-RT RIC Deployment Guide

## Overview

This guide provides step-by-step instructions for deploying the O-RAN Near-RT RIC in various environments. The system now includes genuine O-RAN interface implementations and production-ready security configurations.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    O-RAN Near-RT RIC Platform                   │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │      E2      │  │      A1      │  │      O1      │          │
│  │  Interface   │  │  Interface   │  │  Interface   │          │
│  │   :36421     │  │   :10020     │  │    :830      │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                 xApp Framework                            │   │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐           │   │
│  │  │    xApp    │ │    xApp    │ │    xApp    │    ...    │   │
│  │  │ Management │ │ Lifecycle  │ │  Conflict  │           │   │
│  │  └────────────┘ └────────────┘ └────────────┘           │   │
│  └──────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────┤
│                      Infrastructure                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ Kubernetes   │  │ PostgreSQL   │  │ Redis/Cache  │          │
│  │   Cluster    │  │   Database   │  │              │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

### System Requirements

#### Minimum Requirements (Development)
- **CPU**: 4 cores
- **Memory**: 8 GB RAM
- **Storage**: 50 GB SSD
- **Network**: 1 Gbps

#### Production Requirements
- **CPU**: 16 cores (64-bit x86)
- **Memory**: 32 GB RAM
- **Storage**: 500 GB NVMe SSD
- **Network**: 10 Gbps with SCTP support
- **OS**: Linux (Ubuntu 20.04+ or RHEL 8+)

#### For Kubernetes Deployment
- **Minimum Kubernetes Version**: 1.25+
- **Node Count**: 3+ nodes for HA
- **Pod Network**: Calico, Flannel, or Weave
- **Storage Class**: Dynamic provisioning support
- **LoadBalancer**: MetalLB, HAProxy, or cloud provider

### Software Dependencies

#### Required
- **Docker**: 20.10+
- **Kubernetes**: 1.25+
- **Helm**: 3.8+
- **Git**: Latest
- **Go**: 1.21+ (for development)
- **Node.js**: 18+ (for frontend development)

#### Optional but Recommended
- **Istio**: 1.17+ (service mesh)
- **Prometheus**: Monitoring
- **Grafana**: Visualization
- **Jaeger**: Distributed tracing
- **cert-manager**: TLS certificate management

## Deployment Options

### Option 1: Docker Compose (Development)

Quick setup for development and testing:

```bash
# Clone the repository
git clone https://github.com/hctsai1006/near-rt-ric.git
cd near-rt-ric

# Configure environment
cp .env.example .env

# Edit .env file with your secure passwords
nano .env

# Required environment variables:
# POSTGRES_PASSWORD=SecureOran2024!
# REDIS_PASSWORD=SecureOran2024!
# GRAFANA_ADMIN_PASSWORD=SecureOran2024!
# NETCONF_USER=oranadmin
# NETCONF_PASS=SecureOran2024!

# Start the platform
docker-compose up -d

# Verify deployment
docker-compose ps
docker-compose logs -f ric-main
```

**Services Started:**
- **Main RIC**: http://localhost:8080 (E2 Interface: :36421)
- **A1 Interface**: http://localhost:10020
- **xApp Dashboard**: http://localhost:4200
- **Grafana**: http://localhost:3000
- **Prometheus**: http://localhost:9090

### Option 2: Kubernetes with Helm (Production)

#### Step 1: Prepare Kubernetes Cluster

```bash
# For cloud providers
# AWS EKS
eksctl create cluster --name oran-ric --region us-west-2 --nodes 3

# Azure AKS
az aks create --resource-group oran-ric --name oran-ric --node-count 3

# Google GKE
gcloud container clusters create oran-ric --num-nodes=3

# For on-premises, use kubeadm, Rancher, or OpenShift
```

#### Step 2: Install Prerequisites

```bash
# Install Helm
curl https://get.helm.sh/helm-v3.12.0-linux-amd64.tar.gz | tar xz
sudo mv linux-amd64/helm /usr/local/bin/

# Add Helm repositories
helm repo add stable https://charts.helm.sh/stable
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

# Install cert-manager (for TLS)
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.12.0/cert-manager.yaml

# Install Prometheus and Grafana (optional)
helm install prometheus prometheus-community/kube-prometheus-stack \\
  --namespace monitoring --create-namespace

# Install Istio (optional but recommended)
curl -L https://istio.io/downloadIstio | sh -
istioctl install --set values.defaultRevision=default
```

#### Step 3: Create Secrets

```bash
# Create namespace
kubectl create namespace oran-nearrt-ric

# Create secrets for passwords
kubectl create secret generic oran-secrets \\
  --from-literal=postgres-password=SecureOran2024! \\
  --from-literal=redis-password=SecureOran2024! \\
  --from-literal=grafana-admin-password=SecureOran2024! \\
  --namespace=oran-nearrt-ric

# Create TLS certificates (if not using cert-manager)
kubectl create secret tls oran-tls-secret \\
  --cert=path/to/tls.crt \\
  --key=path/to/tls.key \\
  --namespace=oran-nearrt-ric
```

#### Step 4: Configure Values

Create `values-production.yaml`:

```yaml
# Production values for O-RAN Near-RT RIC
replicaCount: 3

image:
  repository: ghcr.io/near-rt-ric/ric
  tag: "v1.0.0"
  pullPolicy: IfNotPresent

# Security Configuration
security:
  tls:
    enabled: true
    secretName: "oran-tls-secret"
  authentication:
    enabled: true
    type: "jwt"
  networkPolicy:
    enabled: true
  podSecurityPolicy:
    enabled: true

# High Availability
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

# Resource Configuration
resources:
  limits:
    cpu: 2000m
    memory: 4Gi
  requests:
    cpu: 500m
    memory: 1Gi

# Storage
persistence:
  enabled: true
  storageClass: "fast-ssd"
  accessMode: ReadWriteOnce
  size: 100Gi

# Database Configuration
postgresql:
  enabled: true
  auth:
    existingSecret: "oran-secrets"
    secretKeys:
      adminPasswordKey: "postgres-password"
  primary:
    persistence:
      enabled: true
      size: 100Gi
      storageClass: "fast-ssd"
  tls:
    enabled: true

# Redis Configuration  
redis:
  enabled: true
  auth:
    enabled: true
    existingSecret: "oran-secrets"
    existingSecretPasswordKey: "redis-password"
  master:
    persistence:
      enabled: true
      size: 20Gi
      storageClass: "fast-ssd"
  tls:
    enabled: true

# O-RAN Interface Configuration
oran:
  interfaces:
    e2:
      enabled: true
      port: 36421
      tls: true
      maxNodes: 1000
    a1:
      enabled: true
      port: 10020
      tls: true
      auth: true
    o1:
      enabled: true
      port: 830
      tls: true

# Monitoring
monitoring:
  prometheus:
    enabled: true
  grafana:
    enabled: true
  jaeger:
    enabled: true

# Service Mesh
serviceMesh:
  enabled: true
  mtls: "STRICT"

# Ingress
ingress:
  enabled: true
  className: "nginx"
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
  hosts:
    - host: ric.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: ric-tls
      hosts:
        - ric.example.com
```

#### Step 5: Deploy RIC

```bash
# Deploy the O-RAN Near-RT RIC
helm install oran-nearrt-ric ./helm/oran-nearrt-ric \\
  --namespace oran-nearrt-ric \\
  --values values-production.yaml \\
  --wait

# Verify deployment
kubectl get pods -n oran-nearrt-ric
kubectl get services -n oran-nearrt-ric
kubectl get ingress -n oran-nearrt-ric

# Check logs
kubectl logs -f deployment/oran-nearrt-ric-main -n oran-nearrt-ric
```

### Option 3: OpenShift Deployment

```bash
# Login to OpenShift
oc login --server=https://api.openshift.example.com

# Create project
oc new-project oran-nearrt-ric

# Create security context constraints
oc create -f openshift/scc.yaml

# Deploy using Helm
helm install oran-nearrt-ric ./helm/oran-nearrt-ric \\
  --namespace oran-nearrt-ric \\
  --set openshift.enabled=true \\
  --values values-openshift.yaml
```

## Configuration

### Environment Variables

#### Core Configuration
```bash
# RIC Configuration
RIC_LISTEN_ADDRESS=0.0.0.0
RIC_LISTEN_PORT=8080
E2_LISTEN_PORT=36421
A1_LISTEN_PORT=10020
O1_LISTEN_PORT=830

# Security
TLS_ENABLED=true
TLS_CERT_PATH=/certs/tls.crt
TLS_KEY_PATH=/certs/tls.key
JWT_SIGNING_KEY=base64-encoded-key
ENABLE_AUTHENTICATION=true
ENABLE_AUTHORIZATION=true

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=oran_nearrt_ric
POSTGRES_USER=oran
POSTGRES_PASSWORD=SecureOran2024!

# Redis
REDIS_HOST=localhost  
REDIS_PORT=6379
REDIS_PASSWORD=SecureOran2024!

# Monitoring
PROMETHEUS_ENABLED=true
GRAFANA_ENABLED=true
JAEGER_ENABLED=true
LOG_LEVEL=info
```

#### xApp Framework Configuration
```bash
# xApp Framework
XAPP_REGISTRY_URL=https://registry.example.com
XAPP_NAMESPACE=oran-xapps
XAPP_IMAGE_PULL_SECRETS=regcred
XAPP_HEALTH_CHECK_INTERVAL=30s
XAPP_METRICS_INTERVAL=60s
XAPP_CONFLICT_DETECTION=true
XAPP_AUTO_SCALE=true
```

### NETCONF/YANG Configuration (O1 Interface)

Create `o1-config.xml`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <oran-ric xmlns="urn:o-ran:ric:1.0">
    <interfaces>
      <e2-interface>
        <enabled>true</enabled>
        <listen-port>36421</listen-port>
        <max-connections>1000</max-connections>
        <heartbeat-interval>30</heartbeat-interval>
      </e2-interface>
      <a1-interface>
        <enabled>true</enabled>
        <listen-port>10020</listen-port>
        <authentication-enabled>true</authentication-enabled>
        <rate-limiting-enabled>true</rate-limiting-enabled>
      </a1-interface>
    </interfaces>
    <xapp-framework>
      <enabled>true</enabled>
      <max-xapps>100</max-xapps>
      <conflict-detection>true</conflict-detection>
      <auto-scaling>true</auto-scaling>
    </xapp-framework>
  </oran-ric>
</config>
```

## Verification and Testing

### Health Checks

```bash
# Check main RIC health
curl -k https://ric.example.com/health

# Check E2 interface
curl -k https://ric.example.com:36421/health

# Check A1 interface  
curl -k https://ric.example.com:10020/a1-p/healthcheck

# Check all services in Kubernetes
kubectl get pods -n oran-nearrt-ric
kubectl top pods -n oran-nearrt-ric
```

### E2 Interface Testing

```bash
# Deploy E2 simulator
kubectl apply -f k8s/oran/e2-simulator.yaml

# Check E2 connections
kubectl logs -f deployment/e2-simulator -n oran-nearrt-ric

# Verify E2 metrics
curl -k https://ric.example.com:9090/metrics | grep e2_
```

### A1 Interface Testing

```bash
# Test policy type creation
curl -X PUT https://ric.example.com:10020/a1-p/policytypes/qos-policy-v1 \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer $JWT_TOKEN" \\
  -d '{
    "type": "object",
    "properties": {
      "qci": {"type": "integer", "minimum": 1, "maximum": 9},
      "priority_level": {"type": "integer", "minimum": 1, "maximum": 15}
    },
    "required": ["qci", "priority_level"]
  }'

# Test policy creation
curl -X PUT https://ric.example.com:10020/a1-p/policytypes/qos-policy-v1/policies/policy-1 \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer $JWT_TOKEN" \\
  -d '{
    "qci": 5,
    "priority_level": 3
  }'
```

### xApp Framework Testing

```bash
# Deploy sample xApp
kubectl apply -f k8s/oran/sample-xapps/resource-allocation-xapp.yaml

# Check xApp status
curl -k https://ric.example.com/xapps

# View xApp logs
kubectl logs -f deployment/resource-allocation-xapp -n oran-xapps
```

## Performance Tuning

### Kubernetes Resource Optimization

```yaml
# Resource requests and limits
resources:
  requests:
    cpu: 100m
    memory: 256Mi
  limits:
    cpu: 2000m
    memory: 4Gi

# JVM tuning for Java components
env:
  - name: JAVA_OPTS
    value: "-Xms512m -Xmx2g -XX:+UseG1GC"

# Node affinity for performance
nodeAffinity:
  preferredDuringSchedulingIgnoredDuringExecution:
  - weight: 100
    preference:
      matchExpressions:
      - key: node-type
        operator: In
        values: ["performance"]
```

### Database Tuning

```postgresql
-- PostgreSQL performance tuning
ALTER SYSTEM SET shared_buffers = '1GB';
ALTER SYSTEM SET effective_cache_size = '3GB';  
ALTER SYSTEM SET maintenance_work_mem = '256MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
SELECT pg_reload_conf();
```

### Network Optimization

```bash
# SCTP kernel parameters
echo 'net.sctp.max_autoclose = 300' >> /etc/sysctl.conf
echo 'net.sctp.sctp_rmem = 4096 87380 16777216' >> /etc/sysctl.conf
echo 'net.sctp.sctp_wmem = 4096 65536 16777216' >> /etc/sysctl.conf
sysctl -p

# Network interface tuning
ethtool -G eth0 rx 4096 tx 4096
ethtool -C eth0 rx-usecs 100 tx-usecs 100
```

## Monitoring and Alerting

### Prometheus Metrics

Key metrics to monitor:

```promql
# E2 Interface metrics
e2_connections_total
e2_messages_per_second
e2_message_processing_duration_seconds
e2_setup_requests_total
e2_subscription_requests_total

# A1 Interface metrics  
a1_policy_types_total
a1_policies_total
a1_policy_violations_total
a1_api_request_duration_seconds

# xApp Framework metrics
xapp_instances_total
xapp_deployments_total
xapp_conflicts_detected_total
xapp_health_check_failures_total

# System metrics
container_cpu_usage_seconds_total
container_memory_usage_bytes
container_network_receive_bytes_total
container_network_transmit_bytes_total
```

### Grafana Dashboards

Import the provided dashboards:

- **E2 Interface Dashboard**: `config/grafana/e2-interface-dashboard.json`
- **A1 Interface Dashboard**: `config/grafana/a1-o1-interface-dashboard.json`
- **xApp Framework Dashboard**: `config/grafana/dashboard-monitoring.json`
- **Federated Learning Dashboard**: `config/grafana/federated-learning-dashboard.json`

### Alerting Rules

```yaml
# Critical alerts
groups:
  - name: oran-ric-critical
    rules:
      - alert: E2InterfaceDown
        expr: up{job="e2-interface"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "E2 interface is down"
          
      - alert: HighE2MessageLatency
        expr: e2_message_processing_duration_seconds > 0.01
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "E2 message processing latency is high"
          
      - alert: A1PolicyViolations
        expr: increase(a1_policy_violations_total[5m]) > 10
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "High number of A1 policy violations detected"
```

## Troubleshooting

### Common Issues

#### 1. E2 Interface Connection Issues

```bash
# Check SCTP support
lsmod | grep sctp
modprobe sctp

# Check firewall
iptables -L | grep 36421
ufw allow 36421/sctp

# Check service
kubectl get svc e2-interface -n oran-nearrt-ric
kubectl describe pod -l app=e2-interface -n oran-nearrt-ric
```

#### 2. Certificate Issues

```bash
# Check certificate validity
openssl x509 -in /path/to/cert.pem -text -noout

# Renew certificates
kubectl delete secret oran-tls-secret -n oran-nearrt-ric
cert-manager will automatically renew

# Check cert-manager logs
kubectl logs -f deployment/cert-manager -n cert-manager
```

#### 3. Database Connection Issues

```bash
# Check PostgreSQL connectivity
kubectl exec -it postgresql-0 -n oran-nearrt-ric -- psql -U oran -d oran_nearrt_ric

# Check database logs
kubectl logs -f postgresql-0 -n oran-nearrt-ric

# Reset database password
kubectl delete secret oran-secrets -n oran-nearrt-ric
# Recreate with new password
```

#### 4. xApp Deployment Issues

```bash
# Check xApp manager logs
kubectl logs -f deployment/xapp-manager -n oran-nearrt-ric

# Check resource constraints
kubectl describe pod failing-xapp -n oran-xapps

# Check RBAC permissions
kubectl auth can-i create deployments --as=system:serviceaccount:oran-xapps:xapp-sa
```

### Log Analysis

```bash
# Centralized logging with ELK stack
kubectl apply -f monitoring/elk-stack.yaml

# View aggregated logs
kubectl port-forward svc/kibana 5601:5601 -n monitoring
# Access Kibana at http://localhost:5601

# Search for errors
kubectl logs -f -l app=oran-nearrt-ric -n oran-nearrt-ric | grep ERROR

# Export logs for analysis
kubectl logs deployment/oran-nearrt-ric-main -n oran-nearrt-ric --since=1h > ric-logs.txt
```

## Security Considerations

### Network Security

```yaml
# Network policies
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: oran-ric-network-policy
  namespace: oran-nearrt-ric
spec:
  podSelector:
    matchLabels:
      app: oran-nearrt-ric
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: oran-nearrt-ric
    ports:
    - protocol: TCP
      port: 8080
    - protocol: SCTP
      port: 36421
```

### Pod Security

```yaml
# Pod Security Policy
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: oran-ric-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'downwardAPI'
    - 'persistentVolumeClaim'
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
```

## Backup and Recovery

### Database Backup

```bash
# Create backup job
kubectl create job --from=cronjob/postgres-backup postgres-backup-manual -n oran-nearrt-ric

# Manual backup
kubectl exec -it postgresql-0 -n oran-nearrt-ric -- pg_dump -U oran oran_nearrt_ric > backup.sql

# Restore from backup
kubectl exec -i postgresql-0 -n oran-nearrt-ric -- psql -U oran oran_nearrt_ric < backup.sql
```

### Configuration Backup

```bash
# Backup all configurations
kubectl get all -n oran-nearrt-ric -o yaml > oran-ric-backup.yaml

# Backup secrets (encrypted)
kubectl get secrets -n oran-nearrt-ric -o yaml > oran-ric-secrets.yaml

# Backup persistent volumes
kubectl get pv -o yaml > oran-ric-pv-backup.yaml
```

## Scaling

### Horizontal Scaling

```bash
# Scale RIC components
kubectl scale deployment/oran-nearrt-ric-main --replicas=5 -n oran-nearrt-ric

# Enable autoscaling
kubectl autoscale deployment/oran-nearrt-ric-main --min=3 --max=10 --cpu-percent=70 -n oran-nearrt-ric

# Scale database
helm upgrade postgresql bitnami/postgresql \\
  --set primary.replicaCount=3 \\
  --namespace oran-nearrt-ric
```

### Vertical Scaling

```bash
# Update resource limits
kubectl patch deployment oran-nearrt-ric-main -n oran-nearrt-ric -p '{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "oran-nearrt-ric",
          "resources": {
            "limits": {"cpu": "4000m", "memory": "8Gi"},
            "requests": {"cpu": "1000m", "memory": "2Gi"}
          }
        }]
      }
    }
  }
}'
```

## Performance Benchmarks

### Expected Performance

| Component | Metric | Target | Production |
|-----------|---------|---------|------------|
| E2 Interface | Message Latency (P99) | <10ms | <5ms |
| E2 Interface | Concurrent Connections | 100+ | 1000+ |
| A1 Interface | API Response Time (P95) | <500ms | <200ms |
| A1 Interface | Policy Deployment Time | <1s | <500ms |
| xApp Framework | xApp Deployment Time | <30s | <15s |
| System | Availability | 99.9% | 99.99% |

### Load Testing

```bash
# E2 interface load test
./scripts/e2-load-test.sh --nodes 1000 --duration 300s

# A1 interface load test  
./scripts/a1-load-test.sh --policies 10000 --concurrent 100

# xApp framework load test
./scripts/xapp-load-test.sh --xapps 100 --deploy-rate 10/min
```

This deployment guide ensures you can successfully deploy and operate the O-RAN Near-RT RIC in production with all the modern security, scalability, and monitoring capabilities implemented.