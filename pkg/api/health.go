package api

import (
	"context"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type HealthService struct{}

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

func NewHealthService() *HealthService {
	return &HealthService{}
}

func (h *HealthService) GetHealth(ctx context.Context, includeDetails bool) (*HealthResponse, error) {
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

		// TODO: Add actual service checks
		services["opensearch"] = ServiceInfo{
			Status:  "ok",
			Message: "Connected to OpenSearch cluster",
		}

		services["storage"] = ServiceInfo{
			Status:  "ok",
			Message: "DigitalOcean Spaces accessible",
		}

		response.Services = services
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
