package handlers

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/storage"
)

// HealthHandler handles health check and status endpoints
type HealthHandler struct {
	storage   storage.Service
	searchSvc search.Service
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(storage storage.Service, searchSvc search.Service) *HealthHandler {
	return &HealthHandler{
		storage:   storage,
		searchSvc: searchSvc,
	}
}

// Root returns basic service information for the root endpoint
func (h *HealthHandler) Root(c *fiber.Ctx) error {
	response := &models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Service:   "motion-index-fiber",
	}

	return c.JSON(models.NewSuccessResponse(response, "Motion Index API is running"))
}

// Health returns basic health status
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	response := &models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Service:   "motion-index-fiber",
	}

	return c.JSON(models.NewSuccessResponse(response, "Service is healthy"))
}

// HealthCheck returns basic health status
func (h *HealthHandler) HealthCheck(c *fiber.Ctx) error {
	response := &models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Service:   "motion-index-fiber",
	}

	return c.JSON(models.NewSuccessResponse(response, "Service is healthy"))
}

// DetailedStatus returns comprehensive system status
func (h *HealthHandler) DetailedStatus(c *fiber.Ctx) error {
	status := &models.SystemStatus{
		Service:   "motion-index-fiber",
		Version:   "1.0.0",
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    getUptime(),
		System:    getSystemInfo(),
		Storage:   h.getStorageStatus(),
		Indexer:   h.getSearchStatus(),
	}

	// Overall health determination
	if status.Storage.Status != "healthy" || status.Indexer.Status != "healthy" {
		status.Status = "degraded"
	}

	httpStatus := fiber.StatusOK
	if status.Status == "degraded" {
		httpStatus = fiber.StatusServiceUnavailable
	}

	return c.Status(httpStatus).JSON(models.NewSuccessResponse(status, "System status"))
}

// ReadinessCheck returns readiness status for orchestration systems
func (h *HealthHandler) ReadinessCheck(c *fiber.Ctx) error {
	// Check if all dependencies are ready
	storageStatus := h.getStorageStatus()
	searchStatus := h.getSearchStatus()

	ready := storageStatus.Status == "healthy" && searchStatus.Status == "healthy"

	response := &models.ReadinessResponse{
		Ready:     ready,
		Timestamp: time.Now(),
		Checks: map[string]bool{
			"storage": storageStatus.Status == "healthy",
			"search":  searchStatus.Status == "healthy",
		},
	}

	httpStatus := fiber.StatusOK
	if !ready {
		httpStatus = fiber.StatusServiceUnavailable
	}

	return c.Status(httpStatus).JSON(models.NewSuccessResponse(response, "Readiness status"))
}

// LivenessCheck returns liveness status for orchestration systems
func (h *HealthHandler) LivenessCheck(c *fiber.Ctx) error {
	// Simple liveness check - if we can respond, we're alive
	response := &models.LivenessResponse{
		Alive:     true,
		Timestamp: time.Now(),
		PID:       os.Getpid(),
	}

	return c.JSON(models.NewSuccessResponse(response, "Service is alive"))
}

// Metrics returns basic application metrics
func (h *HealthHandler) Metrics(c *fiber.Ctx) error {
	metrics := &models.MetricsResponse{
		Timestamp:  time.Now(),
		Memory:     getMemoryStats(),
		Goroutines: runtime.NumGoroutine(),
		GC:         getGCStats(),
		Storage:    h.getStorageMetrics(),
		Indexer:    h.getSearchMetrics(),
	}

	return c.JSON(models.NewSuccessResponse(metrics, "Application metrics"))
}

// getUptime calculates service uptime
var startTime = time.Now()

func getUptime() time.Duration {
	return time.Since(startTime)
}

// getSystemInfo returns basic system information
func getSystemInfo() *models.SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &models.SystemInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		GoVersion:    runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		Goroutines:   runtime.NumGoroutine(),
		Memory: &models.MemoryInfo{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			NumGC:      m.NumGC,
		},
	}
}

// getStorageStatus checks storage health
func (h *HealthHandler) getStorageStatus() *models.ComponentStatus {
	status := &models.ComponentStatus{
		Name:      "storage",
		Status:    "healthy",
		Timestamp: time.Now(),
	}

	// Test storage connectivity
	if h.storage != nil {
		// Try a simple health check operation
		if !h.storage.IsHealthy() {
			status.Status = "unhealthy"
			status.Error = "storage service is not healthy"
			status.LastError = time.Now()
		}
	} else {
		status.Status = "unhealthy"
		status.Error = "storage not initialized"
		status.LastError = time.Now()
	}

	return status
}

// getSearchStatus checks search service health
func (h *HealthHandler) getSearchStatus() *models.ComponentStatus {
	status := &models.ComponentStatus{
		Name:      "search",
		Status:    "healthy",
		Timestamp: time.Now(),
	}

	// Test search service connectivity
	if h.searchSvc != nil {
		// Try a simple health check operation
		if !h.searchSvc.IsHealthy() {
			status.Status = "unhealthy"
			status.Error = "search service is not healthy"
			status.LastError = time.Now()
		}
	} else {
		status.Status = "unhealthy"
		status.Error = "search service not initialized"
		status.LastError = time.Now()
	}

	return status
}

// getMemoryStats returns current memory statistics
func getMemoryStats() *models.MemoryInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &models.MemoryInfo{
		Alloc:      m.Alloc,
		TotalAlloc: m.TotalAlloc,
		Sys:        m.Sys,
		NumGC:      m.NumGC,
	}
}

// getGCStats returns garbage collection statistics
func getGCStats() *models.GCStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &models.GCStats{
		NumGC:      m.NumGC,
		PauseTotal: time.Duration(m.PauseTotalNs),
		LastGC:     time.Unix(0, int64(m.LastGC)),
		NextGC:     m.NextGC,
	}
}

// getStorageMetrics returns storage-specific metrics
func (h *HealthHandler) getStorageMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	if h.storage != nil {
		// Get storage metrics if available
		if storageMetrics := h.storage.GetMetrics(); storageMetrics != nil {
			metrics = storageMetrics
		}
	}

	return metrics
}

// getSearchMetrics returns search service-specific metrics
func (h *HealthHandler) getSearchMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	if h.searchSvc != nil {
		// Get health status with detailed metrics
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if healthStatus, err := h.searchSvc.Health(ctx); err == nil {
			metrics["cluster_name"] = healthStatus.ClusterName
			metrics["number_of_nodes"] = healthStatus.NumberOfNodes
			metrics["active_shards"] = healthStatus.ActiveShards
			metrics["index_exists"] = healthStatus.IndexExists
			metrics["index_health"] = healthStatus.IndexHealth
		}
	}

	return metrics
}
