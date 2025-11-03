package handlers

import (
	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/application/usecase/system"
	"motion-index-fiber/internal/infrastructure/http/presenter"
)

// HealthHandlerRefactored handles health check and status endpoints using DDD principles.
type HealthHandlerRefactored struct {
	healthService *system.HealthService
}

// NewHealthHandlerRefactored creates a new refactored health handler.
func NewHealthHandlerRefactored(healthService *system.HealthService) *HealthHandlerRefactored {
	return &HealthHandlerRefactored{
		healthService: healthService,
	}
}

// Root returns basic service information for the root endpoint.
func (h *HealthHandlerRefactored) Root(c *fiber.Ctx) error {
	response := h.healthService.GetBasicHealth()
	return presenter.Success(c, response)
}

// Health returns basic health status.
func (h *HealthHandlerRefactored) Health(c *fiber.Ctx) error {
	response := h.healthService.GetBasicHealth()
	return presenter.Success(c, response)
}

// HealthCheck returns basic health status.
func (h *HealthHandlerRefactored) HealthCheck(c *fiber.Ctx) error {
	response := h.healthService.GetBasicHealth()
	return presenter.Success(c, response)
}

// DetailedStatus returns comprehensive system status.
func (h *HealthHandlerRefactored) DetailedStatus(c *fiber.Ctx) error {
	status := h.healthService.GetDetailedStatus()

	if status.Status == "degraded" {
		return presenter.SuccessWithStatus(c, fiber.StatusServiceUnavailable, status)
	}

	return presenter.Success(c, status)
}

// ReadinessCheck returns readiness status for orchestration systems.
func (h *HealthHandlerRefactored) ReadinessCheck(c *fiber.Ctx) error {
	response := h.healthService.CheckReadiness()

	if !response.Ready {
		return presenter.SuccessWithStatus(c, fiber.StatusServiceUnavailable, response)
	}

	return presenter.Success(c, response)
}

// LivenessCheck returns liveness status for orchestration systems.
func (h *HealthHandlerRefactored) LivenessCheck(c *fiber.Ctx) error {
	response := h.healthService.CheckLiveness()
	return presenter.Success(c, response)
}

// Metrics returns basic application metrics.
func (h *HealthHandlerRefactored) Metrics(c *fiber.Ctx) error {
	metrics := h.healthService.CollectMetrics()
	return presenter.Success(c, metrics)
}
