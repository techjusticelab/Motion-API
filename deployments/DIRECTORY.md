# Deployments Directory (`/deployments`)

This directory contains deployment configurations, scripts, and platform-specific deployment files for the Motion-Index Fiber application across different environments and platforms.

## Structure

```
deployments/
├── digitalocean/          # DigitalOcean App Platform deployment
├── docker/                # Docker containerization files
└── k8s/                   # Kubernetes deployment manifests
```

## Platform-Specific Deployments

### `/digitalocean` - DigitalOcean App Platform
**Purpose**: Native DigitalOcean App Platform deployment configuration
**Contains**:
- App Platform specification files (YAML)
- DigitalOcean-specific environment configurations
- CDN and Spaces integration settings
- Auto-scaling and resource allocation configs

**Usage**: Primary production deployment platform with integrated DigitalOcean services including Spaces storage and Managed OpenSearch.

### `/docker` - Docker Containerization
**Purpose**: Docker-based deployment configurations
**Contains**:
- `Dockerfile` - Production container definition
- `Dockerfile.dev` - Development container definition
- `docker-compose.yml` - Multi-service orchestration
- Container optimization and security configurations

**Usage**: 
- Local development environment
- Alternative cloud platform deployment
- CI/CD pipeline integration

### `/k8s` - Kubernetes Deployment
**Purpose**: Kubernetes cluster deployment manifests
**Contains**:
- Deployment manifests
- Service definitions
- ConfigMap and Secret templates
- Ingress configurations
- Resource quotas and limits

**Usage**: 
- Enterprise Kubernetes environments
- Multi-cloud deployment scenarios
- Advanced orchestration requirements

## Deployment Strategies

### Production (DigitalOcean)
1. **Platform**: DigitalOcean App Platform
2. **Storage**: DigitalOcean Spaces with CDN
3. **Search**: DigitalOcean Managed OpenSearch
4. **Scaling**: Auto-scaling based on CPU/memory
5. **Monitoring**: Built-in platform monitoring

### Development
1. **Platform**: Local Docker or direct Go execution
2. **Storage**: Local filesystem or development Spaces bucket
3. **Search**: Shared development OpenSearch cluster
4. **Configuration**: Environment-based with `.env` files

### Enterprise/Kubernetes
1. **Platform**: Kubernetes cluster (any provider)
2. **Storage**: S3-compatible storage
3. **Search**: Elasticsearch/OpenSearch cluster
4. **Configuration**: ConfigMaps and Secrets
5. **Monitoring**: Prometheus/Grafana stack

## Configuration Management

### Environment Variables
Each deployment method uses environment variables for configuration:
- **DigitalOcean**: Set via App Platform dashboard
- **Docker**: Passed via environment files or runtime
- **Kubernetes**: Managed through ConfigMaps and Secrets

### Secrets Management
- **DigitalOcean**: Native secrets management
- **Docker**: External secret management or environment files
- **Kubernetes**: Kubernetes Secrets with optional external secret operators

## Deployment Workflow

### Continuous Deployment (DigitalOcean)
```yaml
# GitHub Actions integration
on:
  push:
    branches: [main]
  
# Auto-deploy to DigitalOcean App Platform
```

### Manual Deployment
```bash
# DigitalOcean
doctl apps create --spec deployments/digitalocean/app.yaml

# Docker
docker build -f deployments/docker/Dockerfile .
docker run --env-file .env motion-index:latest

# Kubernetes
kubectl apply -f deployments/k8s/
```

## Best Practices

1. **Infrastructure as Code**: All deployment configurations are version controlled
2. **Environment Parity**: Development and production environments mirror each other
3. **Security**: Secrets are never committed to version control
4. **Monitoring**: Health checks and observability in all environments
5. **Rollback**: Support for quick rollback in production deployments