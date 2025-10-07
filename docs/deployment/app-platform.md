# DigitalOcean App Platform Deployment Guide

## Overview

This guide covers deploying Motion-Index Fiber to DigitalOcean App Platform using Docker containers. The deployment includes:

- Production-optimized Docker container
- Automated CI/CD pipeline
- Integration with DigitalOcean managed services
- Comprehensive monitoring and logging

## Prerequisites

### 1. DigitalOcean Account Setup
- DigitalOcean account with billing enabled
- DigitalOcean API token with read/write permissions
- Access to DigitalOcean Spaces and Managed OpenSearch

### 2. Required Services
- **DigitalOcean Spaces**: Document storage with CDN
- **DigitalOcean Managed OpenSearch**: Search and indexing
- **Supabase**: User authentication and management
- **OpenAI**: AI-powered document classification (optional)

### 3. GitHub Repository Setup
- Repository with GitHub Actions enabled
- Required secrets configured (see below)

## Step 1: Configure GitHub Secrets

Add these secrets to your GitHub repository (`Settings > Secrets and variables > Actions`):

```bash
# DigitalOcean
DIGITALOCEAN_ACCESS_TOKEN=your_do_api_token

# DigitalOcean Spaces
DO_SPACES_KEY=your_spaces_access_key
DO_SPACES_SECRET=your_spaces_secret_key

# OpenSearch
ES_HOST=your-opensearch-host.k.db.ondigitalocean.com
ES_PASSWORD=your_opensearch_password

# Supabase
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your_supabase_anon_key
SUPABASE_SERVICE_KEY=your_supabase_service_key
JWT_SECRET=your_jwt_secret

# OpenAI (optional)
OPENAI_API_KEY=your_openai_api_key
```

## Step 2: Update App Specification

Edit `deployments/digitalocean/app.yaml` and update:

1. **GitHub repository**: Change `repo` to your repository
2. **Environment variables**: Update placeholder values
3. **Domains**: Configure your custom domains if needed
4. **Instance sizing**: Adjust based on your needs

```yaml
# Example updates
services:
  - name: api
    github:
      repo: your-username/your-repo  # Update this
      branch: main
    envs:
      - key: ALLOWED_ORIGINS
        value: "https://your-domain.com"  # Update this
```

## Step 3: Deploy Using CI/CD Pipeline

### Automatic Deployment
1. Push to `main` branch → deploys to production
2. Push to `staging` branch → deploys to staging environment
3. Pull requests → runs tests only

### Manual Deployment
```bash
# Install doctl
curl -sL https://github.com/digitalocean/doctl/releases/download/v1.100.0/doctl-1.100.0-linux-amd64.tar.gz | tar -xzv
sudo mv doctl /usr/local/bin

# Authenticate
doctl auth init --access-token your_do_api_token

# Deploy new app
doctl apps create --spec deployments/digitalocean/app.yaml --wait

# Update existing app
APP_ID=$(doctl apps list --format ID,Name --no-header | grep motion-index-fiber | awk '{print $1}')
doctl apps update $APP_ID --spec deployments/digitalocean/app.yaml --wait
```

## Step 4: Configure Environment Variables

After deployment, configure secrets in the App Platform dashboard:

1. Go to DigitalOcean Console → Apps → Your App
2. Click "Settings" → "App-Level Environment Variables"
3. Add/update environment variables marked as `SECRET` in the app spec

### Critical Environment Variables
```bash
# Storage
DO_SPACES_KEY=dop_v1_xxx
DO_SPACES_SECRET=xxx

# Database
ES_PASSWORD=xxx
SUPABASE_SERVICE_KEY=xxx

# Security
JWT_SECRET=xxx  # 256-bit minimum
OPENAI_API_KEY=xxx
```

## Step 5: Verify Deployment

### Health Check
```bash
# Get app URL
APP_ID=$(doctl apps list --format ID,Name --no-header | grep motion-index-fiber | awk '{print $1}')
APP_URL=$(doctl apps get $APP_ID --format LiveURL --no-header)

# Test health endpoint
curl "$APP_URL/health"
```

Expected response:
```json
{
  "status": "success",
  "data": {
    "service": "motion-index-fiber",
    "status": "healthy",
    "timestamp": "2024-01-01T00:00:00Z",
    "storage": {"status": "healthy"},
    "search": {"status": "healthy"}
  }
}
```

### Test Key Endpoints
```bash
# Test document search
curl "$APP_URL/api/v1/search" \
  -H "Content-Type: application/json" \
  -d '{"query": "test", "limit": 5}'

# Test field options
curl "$APP_URL/api/v1/all-field-options"

# Test document serving
curl -I "$APP_URL/api/v1/documents/test/document.pdf"
```

## Step 6: Monitor Deployment

### App Platform Dashboard
- Monitor resource usage (CPU, memory, requests)
- View application logs
- Check deployment history
- Configure alerts

### Application Logs
```bash
# View recent logs
doctl apps logs $APP_ID --type=run --follow

# View build logs
doctl apps logs $APP_ID --type=build
```

### Key Metrics to Monitor
- Response time < 500ms for search queries
- Memory usage < 80% of allocated
- Error rate < 1%
- Storage connectivity
- OpenSearch response times

## Troubleshooting

### Common Issues

1. **Build Failures**
   ```bash
   # Check build logs
   doctl apps logs $APP_ID --type=build
   
   # Common causes:
   # - Missing dependencies in Dockerfile
   # - Go module issues
   # - Resource limits during build
   ```

2. **Health Check Failures**
   ```bash
   # Check app logs
   doctl apps logs $APP_ID --type=run
   
   # Common causes:
   # - Missing environment variables
   # - Service connectivity issues
   # - Port configuration problems
   ```

3. **Service Connectivity Issues**
   ```bash
   # Test individual services
   curl "$APP_URL/health"
   
   # Check environment variables
   doctl apps get $APP_ID
   ```

### Debug Commands
```bash
# Get app details
doctl apps get $APP_ID

# List all apps
doctl apps list

# Get app spec
doctl apps spec get $APP_ID

# Force rebuild
doctl apps create-deployment $APP_ID --force-rebuild
```

## Performance Optimization

### Scaling Configuration
```yaml
# In app.yaml
autoscaling:
  min_instance_count: 1
  max_instance_count: 5  # Increase for high traffic
  metrics:
    cpu:
      percent: 70  # Scale up at 70% CPU
```

### Resource Allocation
- **basic-xxs**: Development/testing
- **basic-xs**: Light production workloads  
- **basic-s**: Standard production
- **basic-m**: High-traffic production

### Caching Strategy
- Enable CDN for document serving
- Implement application-level caching for search results
- Use DigitalOcean Spaces CDN for static assets

## Security Considerations

### Secret Management
- Use App Platform environment variables for secrets
- Enable encryption for sensitive environment variables
- Regularly rotate API keys and secrets

### Network Security
- Configure proper CORS origins
- Enable rate limiting
- Use HTTPS for all external communications

### Access Control
- Implement proper JWT authentication
- Use Supabase row-level security
- Monitor access logs

## Maintenance

### Regular Tasks
1. **Monitor resource usage**: Weekly review of CPU/memory
2. **Update dependencies**: Monthly security updates
3. **Backup verification**: Ensure backups are working
4. **Log analysis**: Review error patterns and performance

### Update Process
1. Test changes in staging environment
2. Deploy to production during low-traffic periods
3. Monitor deployment for issues
4. Have rollback plan ready

### Rollback Procedure
```bash
# List deployments
doctl apps list-deployments $APP_ID

# Rollback to previous deployment
PREV_DEPLOYMENT=$(doctl apps list-deployments $APP_ID --format ID --no-header | sed -n '2p')
doctl apps create-deployment $APP_ID --deployment-id $PREV_DEPLOYMENT
```

## Cost Optimization

### Resource Management
- Use auto-scaling to handle traffic spikes
- Monitor and optimize instance sizes
- Implement efficient caching strategies

### Service Optimization
- Use DigitalOcean managed services for cost efficiency
- Monitor Spaces storage usage
- Optimize OpenSearch cluster sizing

## Support and Documentation

### Additional Resources
- [DigitalOcean App Platform Documentation](https://docs.digitalocean.com/products/app-platform/)
- [Motion-Index Fiber API Documentation](../api/README.md)
- [GitHub Actions Workflows](.github/workflows/)

### Getting Help
- Check application logs first
- Review DigitalOcean status page
- Contact support with specific error messages and timestamps