package handlers

import (
	"os"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/infrastructure/http/presenter"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/storage"
)

// HealthHandler handles health check and status endpoints
type HealthHandler struct {
	storage   storage.Service
	searchSvc search.Service
	startTime time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(storage storage.Service, searchSvc search.Service) *HealthHandler {
	return &HealthHandler{
		storage:   storage,
		searchSvc: searchSvc,
		startTime: time.Now(),
	}
}

// Health returns basic health status
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return presenter.Success(c, map[string]interface{}{
		"status":  "healthy",
		"service": "motion-index-fiber",
		"version": "1.0.0",
	})
}

// DetailedStatus returns comprehensive system status
func (h *HealthHandler) DetailedStatus(c *fiber.Ctx) error {
	storageHealthy := h.storage != nil && h.storage.IsHealthy()
	searchHealthy := h.searchSvc != nil && h.searchSvc.IsHealthy()
	overallHealthy := storageHealthy && searchHealthy

	status := map[string]interface{}{
		"service": "motion-index-fiber",
		"version": "1.0.0",
		"status":  h.statusString(overallHealthy),
		"uptime":  time.Since(h.startTime).String(),
		"storage": h.componentStatus("storage", storageHealthy),
		"search":  h.componentStatus("search", searchHealthy),
		"system":  h.systemInfo(),
	}

	if !overallHealthy {
		return presenter.SuccessWithStatus(c, fiber.StatusServiceUnavailable, status)
	}

	return presenter.Success(c, status)
}

// ReadinessCheck returns readiness status for orchestration systems
func (h *HealthHandler) ReadinessCheck(c *fiber.Ctx) error {
	storageReady := h.storage != nil && h.storage.IsHealthy()
	searchReady := h.searchSvc != nil && h.searchSvc.IsHealthy()
	ready := storageReady && searchReady

	response := map[string]interface{}{
		"ready": ready,
		"checks": map[string]bool{
			"storage": storageReady,
			"search":  searchReady,
		},
	}

	if !ready {
		return presenter.SuccessWithStatus(c, fiber.StatusServiceUnavailable, response)
	}

	return presenter.Success(c, response)
}

// LivenessCheck returns liveness status for orchestration systems
func (h *HealthHandler) LivenessCheck(c *fiber.Ctx) error {
	return presenter.Success(c, map[string]interface{}{
		"alive": true,
		"pid":   os.Getpid(),
	})
}

// statusString returns "healthy" or "degraded" based on health status
func (h *HealthHandler) statusString(healthy bool) string {
	if healthy {
		return "healthy"
	}
	return "degraded"
}

// componentStatus returns status information for a component
func (h *HealthHandler) componentStatus(name string, healthy bool) map[string]interface{} {
	status := map[string]interface{}{
		"name":   name,
		"status": h.statusString(healthy),
	}

	if !healthy {
		status["error"] = name + " service is not healthy"
	}

	return status
}

// systemInfo returns basic system information
func (h *HealthHandler) systemInfo() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"os":           runtime.GOOS,
		"architecture": runtime.GOARCH,
		"go_version":   runtime.Version(),
		"num_cpu":      runtime.NumCPU(),
		"goroutines":   runtime.NumGoroutine(),
		"memory": map[string]interface{}{
			"alloc":       m.Alloc,
			"total_alloc": m.TotalAlloc,
			"sys":         m.Sys,
			"num_gc":      m.NumGC,
		},
	}
}
