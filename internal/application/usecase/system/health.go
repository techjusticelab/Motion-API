package system

import (
	"context"
	"os"
	"runtime"
	"time"

	"motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/storage"
)

// HealthService provides health check operations.
type HealthService struct {
	storage   storage.Service
	searchSvc search.Service
	startTime time.Time
}

// NewHealthService creates a new health service.
func NewHealthService(storage storage.Service, searchSvc search.Service) *HealthService {
	return &HealthService{
		storage:   storage,
		searchSvc: searchSvc,
		startTime: time.Now(),
	}
}

// GetBasicHealth returns basic health status.
func (s *HealthService) GetBasicHealth() *models.HealthResponse {
	return &models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Service:   "motion-index-fiber",
	}
}

// GetDetailedStatus returns comprehensive system status.
func (s *HealthService) GetDetailedStatus() *models.SystemStatus {
	storageStatus := s.checkComponentHealth("storage", s.storage)
	searchStatus := s.checkComponentHealth("search", s.searchSvc)

	status := &models.SystemStatus{
		Service:   "motion-index-fiber",
		Version:   "1.0.0",
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    time.Since(s.startTime),
		System:    buildSystemInfo(),
		Storage:   storageStatus,
		Indexer:   searchStatus,
	}

	// Determine overall health
	if storageStatus.Status != "healthy" || searchStatus.Status != "healthy" {
		status.Status = "degraded"
	}

	return status
}

// CheckReadiness checks if the system is ready to accept traffic.
func (s *HealthService) CheckReadiness() *models.ReadinessResponse {
	storageHealthy := s.isServiceHealthy(s.storage)
	searchHealthy := s.isServiceHealthy(s.searchSvc)

	return &models.ReadinessResponse{
		Ready:     storageHealthy && searchHealthy,
		Timestamp: time.Now(),
		Checks: map[string]bool{
			"storage": storageHealthy,
			"search":  searchHealthy,
		},
	}
}

// CheckLiveness checks if the application is alive.
func (s *HealthService) CheckLiveness() *models.LivenessResponse {
	return &models.LivenessResponse{
		Alive:     true,
		Timestamp: time.Now(),
		PID:       os.Getpid(),
	}
}

// CollectMetrics gathers application metrics.
func (s *HealthService) CollectMetrics() *models.MetricsResponse {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &models.MetricsResponse{
		Timestamp:  time.Now(),
		Memory:     buildMemoryInfo(&m),
		Goroutines: runtime.NumGoroutine(),
		GC:         buildGCStats(&m),
		Storage:    s.getStorageMetrics(),
		Indexer:    s.getSearchMetrics(),
	}
}

// Helper methods

func (s *HealthService) checkComponentHealth(name string, svc interface{ IsHealthy() bool }) *models.ComponentStatus {
	status := &models.ComponentStatus{
		Name:      name,
		Status:    "healthy",
		Timestamp: time.Now(),
	}

	if svc == nil {
		status.Status = "unhealthy"
		status.Error = name + " not initialized"
		status.LastError = time.Now()
	} else if !svc.IsHealthy() {
		status.Status = "unhealthy"
		status.Error = name + " service is not healthy"
		status.LastError = time.Now()
	}

	return status
}

func (s *HealthService) isServiceHealthy(svc interface{ IsHealthy() bool }) bool {
	return svc != nil && svc.IsHealthy()
}

func (s *HealthService) getStorageMetrics() map[string]interface{} {
	if s.storage == nil {
		return make(map[string]interface{})
	}
	if metrics := s.storage.GetMetrics(); metrics != nil {
		return metrics
	}
	return make(map[string]interface{})
}

func (s *HealthService) getSearchMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})
	if s.searchSvc == nil {
		return metrics
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if healthStatus, err := s.searchSvc.Health(ctx); err == nil {
		metrics["cluster_name"] = healthStatus.ClusterName
		metrics["number_of_nodes"] = healthStatus.NumberOfNodes
		metrics["active_shards"] = healthStatus.ActiveShards
		metrics["index_exists"] = healthStatus.IndexExists
		metrics["index_health"] = healthStatus.IndexHealth
	}

	return metrics
}

// Utility functions

func buildSystemInfo() *models.SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &models.SystemInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		GoVersion:    runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		Goroutines:   runtime.NumGoroutine(),
		Memory:       buildMemoryInfo(&m),
	}
}

func buildMemoryInfo(m *runtime.MemStats) *models.MemoryInfo {
	return &models.MemoryInfo{
		Alloc:      m.Alloc,
		TotalAlloc: m.TotalAlloc,
		Sys:        m.Sys,
		NumGC:      m.NumGC,
	}
}

func buildGCStats(m *runtime.MemStats) *models.GCStats {
	return &models.GCStats{
		NumGC:      m.NumGC,
		PauseTotal: time.Duration(m.PauseTotalNs),
		LastGC:     time.Unix(0, int64(m.LastGC)),
		NextGC:     m.NextGC,
	}
}
