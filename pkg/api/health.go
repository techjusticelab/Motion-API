package api

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/storage"
)

type HealthService struct {
	searchSvc  search.Service
	storageSvc storage.Service
}

type HealthResponse struct {
	Status      string                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version"`
	Uptime      string                 `json:"uptime"`
	Environment string                 `json:"environment"`
	Services    map[string]ServiceInfo `json:"services,omitempty"`
	System      *SystemInfo            `json:"system,omitempty"`
}

type ServiceInfo struct {
	Status       string        `json:"status"`
	ResponseTime time.Duration `json:"response_time,omitempty"`
	Message      string        `json:"message,omitempty"`
}

type SystemInfo struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	MemoryTotal uint64  `json:"memory_total"`
	MemoryUsed  uint64  `json:"memory_used"`
	Goroutines  int     `json:"goroutines"`
	GoVersion   string  `json:"go_version"`
}

var startTime = time.Now()

func NewHealthService(storageSvc storage.Service, searchSvc search.Service) *HealthService {
	return &HealthService{
		searchSvc:  searchSvc,
		storageSvc: storageSvc,
	}
}

func (h *HealthService) GetHealth(ctx context.Context, includeDetails bool) (*HealthResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	response := &HealthResponse{
		Status:      "ok",
		Timestamp:   time.Now(),
		Version:     "1.0.0",
		Uptime:      time.Since(startTime).String(),
		Environment: "development",
	}

	if includeDetails {
		// Get system information
		systemInfo, err := h.getSystemInfo(ctx)
		if err == nil {
			response.System = systemInfo
		}

		// Check service dependencies
		services := make(map[string]ServiceInfo)

		services["opensearch"] = h.checkSearchService(ctx)
		services["storage"] = h.checkStorageService()

		response.Services = services

		if isDegraded(services) {
			response.Status = "degraded"
		}
	}

	return response, nil
}

func (h *HealthService) getSystemInfo(ctx context.Context) (*SystemInfo, error) {
	// Get CPU usage
	cpuPercent, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return nil, err
	}

	// Get memory usage
	memInfo, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, err
	}

	return &SystemInfo{
		CPUUsage:    cpuPercent[0],
		MemoryUsage: memInfo.UsedPercent,
		MemoryTotal: memInfo.Total,
		MemoryUsed:  memInfo.Used,
		Goroutines:  runtime.NumGoroutine(),
		GoVersion:   runtime.Version(),
	}, nil
}

func (h *HealthService) checkSearchService(ctx context.Context) ServiceInfo {
	if h.searchSvc == nil {
		return ServiceInfo{
			Status:  "unavailable",
			Message: "Search service not configured",
		}
	}

	healthCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	start := time.Now()
	status, err := h.searchSvc.Health(healthCtx)
	responseTime := time.Since(start)

	if err != nil {
		return ServiceInfo{
			Status:       "error",
			ResponseTime: responseTime,
			Message:      fmt.Sprintf("OpenSearch health check failed: %v", err),
		}
	}

	if status == nil {
		return ServiceInfo{
			Status:       "error",
			ResponseTime: responseTime,
			Message:      "OpenSearch health check returned no status",
		}
	}

	messageBuilder := strings.Builder{}
	messageBuilder.WriteString("Cluster healthy")
	if status.ClusterName != "" {
		messageBuilder.WriteString(fmt.Sprintf(" (%s)", status.ClusterName))
	}

	serviceInfo := ServiceInfo{
		Status:       "ok",
		ResponseTime: responseTime,
		Message:      messageBuilder.String(),
	}

	if !status.IndexExists {
		serviceInfo.Status = "degraded"
		serviceInfo.Message = "OpenSearch index missing"
		return serviceInfo
	}

	if status.IndexHealth != "" {
		indexHealth := strings.ToLower(status.IndexHealth)
		switch indexHealth {
		case "green":
			serviceInfo.Message = fmt.Sprintf("Index health: %s", status.IndexHealth)
		case "yellow":
			serviceInfo.Status = "degraded"
			serviceInfo.Message = "Index health: yellow"
		default:
			serviceInfo.Status = "error"
			serviceInfo.Message = fmt.Sprintf("Index health: %s", status.IndexHealth)
		}
	}

	return serviceInfo
}

func (h *HealthService) checkStorageService() ServiceInfo {
	if h.storageSvc == nil {
		return ServiceInfo{
			Status:  "unavailable",
			Message: "Storage service not configured",
		}
	}

	start := time.Now()
	healthy := h.storageSvc.IsHealthy()
	responseTime := time.Since(start)

	if healthy {
		return ServiceInfo{
			Status:       "ok",
			ResponseTime: responseTime,
			Message:      "Storage service healthy",
		}
	}

	return ServiceInfo{
		Status:       "error",
		ResponseTime: responseTime,
		Message:      "Storage service reported unhealthy",
	}
}

func isDegraded(services map[string]ServiceInfo) bool {
	for _, service := range services {
		if service.Status != "ok" {
			return true
		}
	}
	return false
}
