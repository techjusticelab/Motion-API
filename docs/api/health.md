# Health & Status Endpoints

Monitor system health, performance metrics, and service status.

## Endpoints

### `GET /`
Root status endpoint providing basic service information.

**Response:**
```json
{
  "success": true,
  "message": "Motion Index API is running",
  "data": {
    "status": "healthy",
    "timestamp": "2024-01-01T12:00:00Z",
    "version": "1.0.0",
    "service": "motion-index-fiber"
  }
}
```

### `GET /health`
Basic health check for load balancers and monitoring systems.

**Response:**
```json
{
  "success": true,
  "message": "Service is healthy",
  "data": {
    "status": "healthy",
    "timestamp": "2024-01-01T12:00:00Z",
    "version": "1.0.0",
    "service": "motion-index-fiber"
  }
}
```

### `GET /health/detailed`
Comprehensive system status with dependency health checks.

**Response (Healthy):**
```json
{
  "success": true,
  "message": "System status",
  "data": {
    "service": "motion-index-fiber",
    "version": "1.0.0",
    "status": "healthy",
    "timestamp": "2024-01-01T12:00:00Z",
    "uptime": "72h30m15s",
    "system": {
      "os": "linux",
      "architecture": "amd64",
      "go_version": "go1.21.0",
      "num_cpu": 4,
      "goroutines": 25,
      "memory": {
        "alloc": 15728640,
        "total_alloc": 157286400,
        "sys": 71303168,
        "num_gc": 142
      }
    },
    "storage": {
      "name": "storage",
      "status": "healthy",
      "timestamp": "2024-01-01T12:00:00Z",
      "last_error": null
    },
    "indexer": {
      "name": "search",
      "status": "healthy", 
      "timestamp": "2024-01-01T12:00:00Z",
      "last_error": null
    }
  }
}
```

**Response (Degraded) - HTTP 503:**
```json
{
  "success": true,
  "message": "System status",
  "data": {
    "service": "motion-index-fiber",
    "status": "degraded",
    "timestamp": "2024-01-01T12:00:00Z",
    "storage": {
      "name": "storage",
      "status": "unhealthy",
      "timestamp": "2024-01-01T12:00:00Z",
      "error": "connection timeout",
      "last_error": "2024-01-01T11:58:30Z"
    },
    "indexer": {
      "name": "search",
      "status": "healthy",
      "timestamp": "2024-01-01T12:00:00Z"
    }
  }
}
```

### `GET /health/ready`
Readiness check for Kubernetes and orchestration systems.

**Purpose:** Indicates if the service is ready to accept traffic.

**Response (Ready):**
```json
{
  "success": true,
  "message": "Readiness status",
  "data": {
    "ready": true,
    "timestamp": "2024-01-01T12:00:00Z",
    "checks": {
      "storage": true,
      "search": true
    }
  }
}
```

**Response (Not Ready) - HTTP 503:**
```json
{
  "success": true,
  "message": "Readiness status", 
  "data": {
    "ready": false,
    "timestamp": "2024-01-01T12:00:00Z",
    "checks": {
      "storage": false,
      "search": true
    }
  }
}
```

### `GET /health/live`
Liveness check for Kubernetes and orchestration systems.

**Purpose:** Indicates if the service is alive and should not be restarted.

**Response:**
```json
{
  "success": true,
  "message": "Service is alive",
  "data": {
    "alive": true,
    "timestamp": "2024-01-01T12:00:00Z",
    "pid": 12345
  }
}
```

### `GET /metrics`
Application metrics and performance data.

**Response:**
```json
{
  "success": true,
  "message": "Application metrics",
  "data": {
    "timestamp": "2024-01-01T12:00:00Z",
    "memory": {
      "alloc": 15728640,
      "total_alloc": 157286400,
      "sys": 71303168,
      "num_gc": 142
    },
    "goroutines": 25,
    "gc": {
      "num_gc": 142,
      "pause_total": "5.2ms",
      "last_gc": "2024-01-01T11:59:45Z",
      "next_gc": 33554432
    },
    "storage": {
      "total_requests": 15420,
      "successful_uploads": 1205,
      "failed_uploads": 12,
      "avg_upload_time": "1.2s",
      "cache_hit_rate": 0.85
    },
    "indexer": {
      "cluster_name": "motion-index-cluster",
      "number_of_nodes": 3,
      "active_shards": 15,
      "index_exists": true,
      "index_health": "green",
      "total_docs": 23547,
      "search_requests": 8420,
      "avg_search_time": "85ms"
    }
  }
}
```

## Status Codes

| Status | Meaning | When Used |
|--------|---------|-----------|
| `healthy` | All systems operational | All dependencies are working |
| `degraded` | Some issues but service operational | Non-critical dependencies down |
| `unhealthy` | Service not functioning properly | Critical dependencies down |

## Monitoring Integration

### Prometheus Metrics
The `/metrics` endpoint can be scraped by Prometheus for monitoring:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'motion-index'
    static_configs:
      - targets: ['motion-index:6000']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

### Health Check Intervals
- **Load Balancer**: Check `/health` every 10 seconds
- **Kubernetes Liveness**: Check `/health/live` every 30 seconds  
- **Kubernetes Readiness**: Check `/health/ready` every 10 seconds
- **Monitoring Systems**: Check `/health/detailed` every 60 seconds

### Alerting Rules

#### Critical Alerts
- Service returns 503 for more than 2 minutes
- Memory usage > 1.5GB for more than 5 minutes
- Error rate > 5% for more than 2 minutes

#### Warning Alerts  
- Storage dependency unhealthy for more than 1 minute
- Search dependency unhealthy for more than 1 minute
- GC pause time > 100ms average over 5 minutes
- Goroutine count > 1000

## Example Usage

### Basic Health Check
```bash
curl -s http://localhost:6000/health | jq '.data.status'
# Output: "healthy"
```

### Monitor Storage Health
```bash
curl -s http://localhost:6000/health/detailed | jq '.data.storage.status'
# Output: "healthy" or "unhealthy"
```

### Get System Metrics
```bash
curl -s http://localhost:6000/metrics | jq '.data.memory.alloc'
# Output: 15728640
```

### Kubernetes Readiness Probe
```yaml
readinessProbe:
  httpGet:
    path: /health/ready
    port: 6000
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 3
  failureThreshold: 3
```

### Kubernetes Liveness Probe
```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 6000
  initialDelaySeconds: 30
  periodSeconds: 30
  timeoutSeconds: 5
  failureThreshold: 5
```