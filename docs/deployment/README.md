# Deployment Guide

Comprehensive deployment guide for Motion-Index Fiber across different environments and platforms.

## Overview

Motion-Index Fiber is designed for cloud-native deployment with DigitalOcean as the primary platform. The application supports multiple deployment patterns:

- **DigitalOcean App Platform** (Recommended): Managed PaaS with auto-scaling
- **Docker Containers**: Containerized deployment on any platform
- **Kubernetes**: Container orchestration for complex deployments
- **Direct Binary**: Traditional server deployment

## Prerequisites

### Required Services
- **DigitalOcean Spaces**: Document storage with CDN
- **DigitalOcean Managed OpenSearch**: Search and indexing
- **Supabase**: Authentication and user management
- **OpenAI**: Document classification (optional)

### Required Tools
- **doctl**: DigitalOcean CLI tool
- **Docker**: For containerized deployments
- **kubectl**: For Kubernetes deployments
- **Go 1.21+**: For building from source

## DigitalOcean App Platform (Recommended)

### 1. Initial Setup

#### Create Required Infrastructure
```bash
# Install doctl CLI
curl -sL https://github.com/digitalocean/doctl/releases/download/v1.98.1/doctl-1.98.1-linux-amd64.tar.gz | tar -xzv
sudo mv doctl /usr/local/bin

# Authenticate
doctl auth init

# Create Spaces bucket
doctl spaces buckets create motion-index-docs --region nyc3

# Create OpenSearch cluster
doctl databases create motion-search \
  --engine opensearch \
  --version 2 \
  --size db-s-2vcpu-4gb \
  --num-nodes 3 \
  --region nyc3

# Get OpenSearch connection details
doctl databases connection motion-search --format Host,Port,Username,Password

# Create CDN for Spaces (automatic via MCP integration)
```

#### Configure Environment Variables
Create environment variables in App Platform dashboard or via API:

```bash
# Core Application
ENVIRONMENT=production
PRODUCTION=true
PORT=8080
JWT_SECRET=your-production-jwt-secret

# DigitalOcean Services
DO_SPACES_KEY=your-spaces-access-key
DO_SPACES_SECRET=your-spaces-secret-key
DO_SPACES_BUCKET=motion-index-docs
DO_SPACES_REGION=nyc3

# OpenSearch (from cluster connection details)
OPENSEARCH_HOST=your-cluster.k.db.ondigitalocean.com
OPENSEARCH_PORT=25060
OPENSEARCH_USERNAME=doadmin
OPENSEARCH_PASSWORD=your-opensearch-password
OPENSEARCH_USE_SSL=true
OPENSEARCH_INDEX=documents

# Supabase
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-anon-key
SUPABASE_SERVICE_KEY=your-service-key

# OpenAI (optional)
OPENAI_API_KEY=your-openai-api-key
OPENAI_MODEL=gpt-4

# Performance Settings
MAX_FILE_SIZE=104857600
MAX_WORKERS=10
BATCH_SIZE=50
PROCESS_TIMEOUT=5m
```

### 2. Deployment

#### Option A: Deploy via GitHub (Recommended)
```bash
# 1. Push code to GitHub
git add .
git commit -m "Production deployment"
git push origin main

# 2. Create app spec
cat > app.yaml << EOF
name: motion-index-fiber
services:
- name: api
  source_dir: /
  github:
    repo: your-username/motion-index-fiber
    branch: main
    deploy_on_push: true
  build_command: go build -ldflags="-s -w" -o bin/server cmd/server/main.go
  run_command: ./bin/server
  environment_slug: go
  instance_count: 3
  instance_size_slug: basic-xxs
  http_port: 8080
  health_check:
    http_path: /health
  envs:
  - key: ENVIRONMENT
    value: production
  - key: PRODUCTION
    value: "true"
  # Add all other environment variables here
EOF

# 3. Deploy to App Platform
doctl apps create --spec app.yaml

# 4. Get app info
doctl apps list
doctl apps get <app-id>
```

#### Option B: Deploy via doctl CLI
```bash
# Build and deploy directly
doctl apps create --spec app.yaml

# Update existing deployment
doctl apps update <app-id> --spec app.yaml

# Monitor deployment
doctl apps logs <app-id> --type build
doctl apps logs <app-id> --type deploy
doctl apps logs <app-id> --type run
```

### 3. Production Configuration

#### Auto-Scaling Configuration
```yaml
# app.yaml
services:
- name: api
  instance_count: 3
  instance_size_slug: basic-xxs
  autoscaling:
    min_instance_count: 2
    max_instance_count: 10
    metrics:
      cpu:
        percent: 70
```

#### Health Checks
```yaml
services:
- name: api
  health_check:
    http_path: /health/ready
    initial_delay_seconds: 60
    period_seconds: 10
    timeout_seconds: 5
    failure_threshold: 3
    success_threshold: 2
```

#### Custom Domains
```bash
# Add custom domain
doctl apps update <app-id> --spec app.yaml

# app.yaml with custom domain
domains:
- name: api.motionindex.com
  type: PRIMARY
  zone: motionindex.com
```

## Docker Deployment

### 1. Production Dockerfile

Create `Dockerfile.prod`:
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION:-dev}" \
    -o bin/server cmd/server/main.go

# Production stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S app && \
    adduser -u 1001 -S app -G app

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/server ./server

# Change ownership
RUN chown app:app ./server

# Switch to non-root user
USER app

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ./server --health-check || exit 1

# Run application
CMD ["./server"]
```

### 2. Docker Compose for Development

```yaml
# docker-compose.yml
version: '3.8'

services:
  app:
    build:
      context: .
      dockerfile: Dockerfile.prod
    ports:
      - "8080:8080"
    environment:
      - ENVIRONMENT=production
      - PORT=8080
      # Add other environment variables
    volumes:
      - ./logs:/app/logs
    depends_on:
      - opensearch
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "./server", "--health-check"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  opensearch:
    image: opensearchproject/opensearch:2.11.0
    environment:
      - discovery.type=single-node
      - bootstrap.memory_lock=true
      - "OPENSEARCH_JAVA_OPTS=-Xms512m -Xmx512m"
      - "DISABLE_INSTALL_DEMO_CONFIG=true"
      - "DISABLE_SECURITY_PLUGIN=true"
    ulimits:
      memlock:
        soft: -1
        hard: -1
    ports:
      - "9200:9200"
    volumes:
      - opensearch_data:/usr/share/opensearch/data

  opensearch-dashboards:
    image: opensearchproject/opensearch-dashboards:2.11.0
    ports:
      - "5601:5601"
    environment:
      - 'OPENSEARCH_HOSTS=["http://opensearch:9200"]'
      - "DISABLE_SECURITY_DASHBOARDS_PLUGIN=true"
    depends_on:
      - opensearch

volumes:
  opensearch_data:
```

### 3. Docker Deployment Commands

```bash
# Build production image
docker build -f Dockerfile.prod -t motion-index:latest .

# Run container
docker run -d \
  --name motion-index \
  -p 8080:8080 \
  --env-file .env.production \
  --restart unless-stopped \
  motion-index:latest

# Using Docker Compose
docker-compose -f docker-compose.prod.yml up -d

# Monitor logs
docker logs -f motion-index

# Check health
docker exec motion-index ./server --health-check

# Update deployment
docker pull motion-index:latest
docker stop motion-index
docker rm motion-index
docker run -d \
  --name motion-index \
  -p 8080:8080 \
  --env-file .env.production \
  --restart unless-stopped \
  motion-index:latest
```

## Kubernetes Deployment

### 1. Kubernetes Manifests

#### Namespace
```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: motion-index
  labels:
    name: motion-index
```

#### ConfigMap
```yaml
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: motion-index-config
  namespace: motion-index
data:
  ENVIRONMENT: "production"
  PORT: "8080"
  PRODUCTION: "true"
  DO_SPACES_BUCKET: "motion-index-docs"
  DO_SPACES_REGION: "nyc3"
  OPENSEARCH_PORT: "25060"
  OPENSEARCH_USE_SSL: "true"
  OPENSEARCH_INDEX: "documents"
  MAX_WORKERS: "10"
  BATCH_SIZE: "50"
  PROCESS_TIMEOUT: "5m"
```

#### Secret
```yaml
# k8s/secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: motion-index-secrets
  namespace: motion-index
type: Opaque
stringData:
  JWT_SECRET: "your-production-jwt-secret"
  DO_SPACES_KEY: "your-spaces-access-key"
  DO_SPACES_SECRET: "your-spaces-secret-key"
  OPENSEARCH_HOST: "your-cluster.k.db.ondigitalocean.com"
  OPENSEARCH_USERNAME: "doadmin"
  OPENSEARCH_PASSWORD: "your-opensearch-password"
  SUPABASE_URL: "https://your-project.supabase.co"
  SUPABASE_ANON_KEY: "your-anon-key"
  SUPABASE_SERVICE_KEY: "your-service-key"
  OPENAI_API_KEY: "your-openai-api-key"
```

#### Deployment
```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: motion-index
  namespace: motion-index
  labels:
    app: motion-index
spec:
  replicas: 3
  selector:
    matchLabels:
      app: motion-index
  template:
    metadata:
      labels:
        app: motion-index
    spec:
      containers:
      - name: motion-index
        image: motion-index:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: motion-index-config
        - secretRef:
            name: motion-index-secrets
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 30
          timeoutSeconds: 5
          failureThreshold: 5
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
          timeoutSeconds: 3
          failureThreshold: 3
        lifecycle:
          preStop:
            exec:
              command: ["/bin/sh", "-c", "sleep 15"]
```

#### Service
```yaml
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: motion-index-service
  namespace: motion-index
  labels:
    app: motion-index
spec:
  selector:
    app: motion-index
  ports:
  - name: http
    port: 80
    targetPort: 8080
    protocol: TCP
  type: ClusterIP
```

#### Ingress
```yaml
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: motion-index-ingress
  namespace: motion-index
  annotations:
    kubernetes.io/ingress.class: "nginx"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "300"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "300"
spec:
  tls:
  - hosts:
    - api.motionindex.com
    secretName: motion-index-tls
  rules:
  - host: api.motionindex.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: motion-index-service
            port:
              number: 80
```

#### HPA (Horizontal Pod Autoscaler)
```yaml
# k8s/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: motion-index-hpa
  namespace: motion-index
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: motion-index
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

### 2. Kubernetes Deployment Commands

```bash
# Create namespace
kubectl apply -f k8s/namespace.yaml

# Create secrets (edit with real values first)
kubectl apply -f k8s/secret.yaml

# Deploy configuration
kubectl apply -f k8s/configmap.yaml

# Deploy application
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml
kubectl apply -f k8s/hpa.yaml

# Monitor deployment
kubectl get pods -n motion-index -w
kubectl logs -f deployment/motion-index -n motion-index

# Check service status
kubectl get svc -n motion-index
kubectl get ingress -n motion-index

# Scale deployment
kubectl scale deployment motion-index --replicas=5 -n motion-index

# Rolling update
kubectl set image deployment/motion-index motion-index=motion-index:v1.1.0 -n motion-index
kubectl rollout status deployment/motion-index -n motion-index

# Rollback if needed
kubectl rollout undo deployment/motion-index -n motion-index
```

## Environment-Specific Configurations

### Development Environment
```bash
# .env.development
ENVIRONMENT=local
PRODUCTION=false
PORT=6000
USE_MOCK_SERVICES=true
LOG_LEVEL=debug

# Use local OpenSearch
OPENSEARCH_HOST=localhost
OPENSEARCH_PORT=9200
OPENSEARCH_USE_SSL=false
```

### Staging Environment
```bash
# .env.staging
ENVIRONMENT=staging
PRODUCTION=false
PORT=8080

# Use staging DigitalOcean services
DO_SPACES_BUCKET=motion-index-staging
OPENSEARCH_INDEX=documents-staging

# Reduced resource limits for cost optimization
MAX_WORKERS=5
BATCH_SIZE=25
```

### Production Environment
```bash
# .env.production
ENVIRONMENT=production
PRODUCTION=true
PORT=8080

# Full production services
DO_SPACES_BUCKET=motion-index-docs
OPENSEARCH_INDEX=documents

# Optimized for performance
MAX_WORKERS=20
BATCH_SIZE=100
PROCESS_TIMEOUT=10m
```

## Monitoring & Observability

### 1. Health Checks
Configure health check monitoring:

```bash
# Basic health check
curl https://api.motionindex.com/health

# Detailed status
curl https://api.motionindex.com/health/detailed

# Readiness check (for load balancers)
curl https://api.motionindex.com/health/ready

# Metrics endpoint
curl https://api.motionindex.com/metrics
```

### 2. Logging

#### Application Logs
```bash
# DigitalOcean App Platform
doctl apps logs <app-id> --type run --follow

# Docker
docker logs -f motion-index

# Kubernetes
kubectl logs -f deployment/motion-index -n motion-index
```

#### Structured Logging Configuration
```go
// Configure structured logging for production
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
    AddSource: true,
}))

slog.SetDefault(logger)
```

### 3. Metrics Collection

#### Prometheus Configuration
```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'motion-index'
    static_configs:
      - targets: ['motion-index:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

#### Grafana Dashboard
Key metrics to monitor:
- Request rate and latency
- Error rate
- Document processing throughput
- Storage and search service health
- Memory and CPU usage
- Goroutine count and GC metrics

## Security Configuration

### 1. Network Security
```bash
# DigitalOcean Firewall Rules
doctl compute firewall create motion-index-fw \
  --inbound-rules "protocol:tcp,ports:443,source_addresses:0.0.0.0/0" \
  --inbound-rules "protocol:tcp,ports:80,source_addresses:0.0.0.0/0" \
  --outbound-rules "protocol:tcp,ports:443,destination_addresses:0.0.0.0/0" \
  --outbound-rules "protocol:tcp,ports:25060,destination_addresses:your-opensearch-ip"
```

### 2. SSL/TLS Configuration
```yaml
# Kubernetes ingress with TLS
spec:
  tls:
  - hosts:
    - api.motionindex.com
    secretName: motion-index-tls
```

### 3. Secret Management
```bash
# Kubernetes secrets
kubectl create secret generic motion-index-secrets \
  --from-env-file=.env.production \
  --namespace=motion-index

# Rotate secrets
kubectl delete secret motion-index-secrets -n motion-index
kubectl create secret generic motion-index-secrets \
  --from-env-file=.env.production.new \
  --namespace=motion-index
kubectl rollout restart deployment/motion-index -n motion-index
```

## Troubleshooting

### Common Issues

#### 1. Service Connection Issues
```bash
# Check DigitalOcean service status
doctl databases get motion-search
doctl spaces ls

# Test OpenSearch connectivity
curl -u admin:password https://your-cluster.k.db.ondigitalocean.com:25060

# Check application logs
kubectl logs deployment/motion-index -n motion-index --previous
```

#### 2. Performance Issues
```bash
# Check resource usage
kubectl top pods -n motion-index
kubectl describe pod <pod-name> -n motion-index

# Monitor metrics
curl https://api.motionindex.com/metrics | grep memory
```

#### 3. Deployment Issues
```bash
# Check deployment status
kubectl get events -n motion-index --sort-by='.lastTimestamp'
kubectl describe deployment motion-index -n motion-index

# Rollback problematic deployment
kubectl rollout undo deployment/motion-index -n motion-index
```

### Recovery Procedures

#### 1. Service Recovery
```bash
# Restart services
kubectl rollout restart deployment/motion-index -n motion-index

# Scale down and up
kubectl scale deployment motion-index --replicas=0 -n motion-index
kubectl scale deployment motion-index --replicas=3 -n motion-index
```

#### 2. Data Recovery
```bash
# Backup OpenSearch indices
curl -X PUT "your-cluster.k.db.ondigitalocean.com:25060/_snapshot/backup_repo" \
  -H 'Content-Type: application/json' \
  -d '{"type": "s3", "settings": {"bucket": "backup-bucket"}}'

# Restore from backup
curl -X POST "your-cluster.k.db.ondigitalocean.com:25060/_snapshot/backup_repo/snapshot_1/_restore"
```

## Maintenance

### 1. Regular Updates
```bash
# Update dependencies
go mod tidy
go mod verify

# Security updates
go list -u -m all
go get -u all

# Rebuild and deploy
docker build -f Dockerfile.prod -t motion-index:v1.1.0 .
kubectl set image deployment/motion-index motion-index=motion-index:v1.1.0 -n motion-index
```

### 2. Database Maintenance
```bash
# OpenSearch cluster maintenance
doctl databases maintenance motion-search

# Index optimization
curl -X POST "your-cluster.k.db.ondigitalocean.com:25060/documents/_forcemerge?max_num_segments=1"
```

### 3. Monitoring Health
```bash
# Automated health checks
#!/bin/bash
HEALTH_URL="https://api.motionindex.com/health"
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" $HEALTH_URL)

if [ $RESPONSE -ne 200 ]; then
    echo "Health check failed: $RESPONSE"
    # Send alert
    exit 1
fi

echo "Health check passed"
```

This deployment guide provides comprehensive coverage for deploying Motion-Index Fiber in production environments with proper monitoring, security, and maintenance procedures.