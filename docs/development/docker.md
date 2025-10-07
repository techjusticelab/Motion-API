# Docker Development Environment

## Overview

This guide covers setting up a local development environment using Docker for Motion-Index Fiber. The development environment includes:

- Hot reloading with Air
- Debugging support with Delve
- Local service orchestration
- DigitalOcean service integration

## Quick Start

### 1. Prerequisites
- Docker and Docker Compose installed
- Git repository cloned
- Environment variables configured

### 2. Environment Setup
```bash
# Copy environment template
cp .env.example .env

# Edit environment variables
# Add your DigitalOcean credentials and service endpoints
vim .env
```

### 3. Start Development Environment
```bash
# Start with hot reloading
docker-compose -f deployments/docker/docker-compose.yml up

# Start with additional services
docker-compose -f deployments/docker/docker-compose.yml --profile with-redis --profile with-opensearch up
```

### 4. Access Services
- **API**: http://localhost:8003
- **Health Check**: http://localhost:8003/health
- **OpenAPI Docs**: http://localhost:6000/docs (if enabled)
- **Redis** (optional): localhost:6379
- **OpenSearch** (optional): http://localhost:9200

## Development Features

### Hot Reloading
The development container uses [Air](https://github.com/cosmtrek/air) for automatic reloading:

- Watches for changes in `.go` files
- Automatically rebuilds and restarts the application
- Excludes test files and vendor directories
- Build errors displayed in container logs

### Debugging Support
Debug with Delve debugger:

```bash
# Start container in debug mode
docker-compose -f deployments/docker/docker-compose.yml exec motion-index-api dlv debug ./cmd/server/main.go --headless --listen=:2345 --api-version=2

# Connect from your IDE
# Debug port: localhost:2345
```

### Code Synchronization
- Source code is mounted as volume
- Changes are immediately reflected in container
- Go modules cache is preserved between runs

## Environment Configuration

### Required Environment Variables
```bash
# DigitalOcean Spaces
DO_SPACES_KEY=your_spaces_access_key
DO_SPACES_SECRET=your_spaces_secret_key
DO_SPACES_BUCKET=motion-index-docs
DO_SPACES_REGION=nyc3

# OpenSearch/Elasticsearch
ES_HOST=your-opensearch-host.k.db.ondigitalocean.com
ES_PORT=25060
ES_USERNAME=doadmin
ES_PASSWORD=your_opensearch_password
ES_USE_SSL=true
ES_INDEX=documents

# Supabase Authentication
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your_supabase_anon_key
SUPABASE_SERVICE_KEY=your_supabase_service_key
JWT_SECRET=dev-jwt-secret-change-in-production

# OpenAI (optional)
OPENAI_API_KEY=your_openai_api_key
OPENAI_MODEL=gpt-4
```

### Development-Specific Settings
```bash
# Server Configuration
PORT=8003
ENVIRONMENT=local
PRODUCTION=false
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173,http://localhost:5174

# Logging (verbose for development)
LOG_LEVEL=debug
LOG_FORMAT=text
ENABLE_REQUEST_LOGGING=true
ENABLE_ERROR_DETAILS=true
ENABLE_STACK_TRACE=true

# Processing (reduced for development)
MAX_FILE_SIZE=52428800  # 50MB
MAX_WORKERS=5
BATCH_SIZE=25
PROCESS_TIMEOUT=2m
```

## Docker Services

### Main API Service
- **Image**: Built from `Dockerfile.dev`
- **Ports**: 6000 (API), 2345 (debugger)
- **Volumes**: Source code, excluding `tmp/` directory
- **Features**: Hot reloading, debugging, development dependencies

### Optional Services

#### Redis (with-redis profile)
```bash
# Start with Redis
docker-compose -f deployments/docker/docker-compose.yml --profile with-redis up
```
- **Use case**: Caching, session storage
- **Port**: 6379
- **Image**: redis:7-alpine

#### Local OpenSearch (with-opensearch profile)
```bash
# Start with local OpenSearch
docker-compose -f deployments/docker/docker-compose.yml --profile with-opensearch up
```
- **Use case**: Local development without DigitalOcean dependency
- **Ports**: 9200 (API), 9600 (performance)
- **Image**: opensearchproject/opensearch:2.11.0
- **Config**: Single node, security disabled

## Common Development Tasks

### Running Tests
```bash
# Run tests in container
docker-compose -f deployments/docker/docker-compose.yml exec motion-index-api go test ./... -v

# Run tests with coverage
docker-compose -f deployments/docker/docker-compose.yml exec motion-index-api go test ./... -v -coverprofile=coverage.out

# Run short tests only
docker-compose -f deployments/docker/docker-compose.yml exec motion-index-api go test ./... -v -short
```

### Building Production Image
```bash
# Build production image locally
docker build -t motion-index-fiber:latest .

# Test production build
docker run -p 8003:8003 --env-file .env.production motion-index-fiber:latest
```

### Database Operations
```bash
# Access container shell
docker-compose -f deployments/docker/docker-compose.yml exec motion-index-api sh

# Run Go commands
docker-compose -f deployments/docker/docker-compose.yml exec motion-index-api go mod tidy
docker-compose -f deployments/docker/docker-compose.yml exec motion-index-api go fmt ./...
```

### Log Monitoring
```bash
# Follow all logs
docker-compose -f deployments/docker/docker-compose.yml logs -f

# Follow specific service
docker-compose -f deployments/docker/docker-compose.yml logs -f motion-index-api

# View recent logs
docker-compose -f deployments/docker/docker-compose.yml logs --tail=100 motion-index-api
```

## IDE Integration

### VS Code
Add to `.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Connect to Docker",
      "type": "go",
      "request": "attach",
      "mode": "remote",
      "remotePath": "/app",
      "port": 2345,
      "host": "127.0.0.1"
    }
  ]
}
```

### GoLand/IntelliJ
1. Go to Run → Edit Configurations
2. Add new "Go Remote" configuration
3. Set Host: `localhost`, Port: `2345`
4. Set Path mappings: Local: `./` → Remote: `/app`

## Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```bash
   # Check what's using the port
   lsof -i :6000
   
   # Kill process or change port in docker-compose.yml
   ```

2. **Permission Issues**
   ```bash
   # Fix file permissions
   sudo chown -R $USER:$USER .
   
   # On Linux, you may need to adjust user ID in Dockerfile.dev
   ```

3. **Module Download Issues**
   ```bash
   # Clear go module cache
   docker-compose -f deployments/docker/docker-compose.yml exec motion-index-api go clean -modcache
   
   # Rebuild container
   docker-compose -f deployments/docker/docker-compose.yml build --no-cache
   ```

4. **Air Not Working**
   ```bash
   # Check Air configuration
   docker-compose -f deployments/docker/docker-compose.yml exec motion-index-api cat .air.toml
   
   # Manually run Air
   docker-compose -f deployments/docker/docker-compose.yml exec motion-index-api air -c .air.toml
   ```

### Debug Commands
```bash
# Container health
docker-compose -f deployments/docker/docker-compose.yml ps

# Container logs
docker-compose -f deployments/docker/docker-compose.yml logs motion-index-api

# Execute commands in container
docker-compose -f deployments/docker/docker-compose.yml exec motion-index-api /bin/sh

# Inspect container
docker inspect $(docker-compose -f deployments/docker/docker-compose.yml ps -q motion-index-api)
```

## Performance Optimization

### Development Performance
- Use go module proxy: `GOPROXY=https://proxy.golang.org,direct`
- Enable build cache mounting
- Use multi-stage builds for faster rebuilds

### Resource Limits
```yaml
# Add to docker-compose.yml if needed
deploy:
  resources:
    limits:
      cpus: '2.0'
      memory: 4G
    reservations:
      cpus: '1.0'
      memory: 2G
```

## Testing Workflows

### Unit Testing
```bash
# Run all tests
make test

# Run specific package tests
docker-compose -f deployments/docker/docker-compose.yml exec motion-index-api go test ./internal/handlers/... -v

# Run tests with race detection
docker-compose -f deployments/docker/docker-compose.yml exec motion-index-api go test ./... -race
```

### Integration Testing
```bash
# Set up test environment
export RUN_INTEGRATION_TESTS=true

# Run integration tests
docker-compose -f deployments/docker/docker-compose.yml exec motion-index-api go test ./... -tags=integration -v
```

### API Testing
```bash
# Test health endpoint
curl http://localhost:6000/health

# Test search endpoint
curl -X POST http://localhost:6000/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "test", "limit": 5}'

# Test document upload
curl -X POST http://localhost:6000/api/v1/categorise \
  -F "file=@test.pdf" \
  -F "case_id=test-case"
```

## Cleanup

### Stop Services
```bash
# Stop all services
docker-compose -f deployments/docker/docker-compose.yml down

# Stop and remove volumes
docker-compose -f deployments/docker/docker-compose.yml down -v

# Stop and remove images
docker-compose -f deployments/docker/docker-compose.yml down --rmi all
```

### Clean Docker Environment
```bash
# Remove unused containers, networks, images
docker system prune

# Remove unused volumes
docker volume prune

# Remove development images
docker rmi motion-index-fiber:dev
```
