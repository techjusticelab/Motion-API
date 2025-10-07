package gpu

import (
	"runtime"
	"time"
)

// cpuFallbackAccelerator provides CPU-based processing when GPU is not available
type cpuFallbackAccelerator struct {
	config      *GPUConfig
	metrics     *PerformanceMetrics
	initialized bool
}

// NewCPUFallbackAccelerator creates a CPU fallback accelerator
func NewCPUFallbackAccelerator(config *GPUConfig) GPUAccelerator {
	return &cpuFallbackAccelerator{
		config: config,
		metrics: &PerformanceMetrics{
			StartTime: time.Now(),
		},
	}
}

// IsAvailable always returns true for CPU fallback
func (c *cpuFallbackAccelerator) IsAvailable() bool {
	return true
}

// Initialize initializes the CPU fallback (no-op)
func (c *cpuFallbackAccelerator) Initialize() error {
	c.initialized = true
	return nil
}

// Shutdown shuts down the CPU fallback (no-op)
func (c *cpuFallbackAccelerator) Shutdown() error {
	c.initialized = false
	return nil
}

// GetInfo returns CPU information as GPU info
func (c *cpuFallbackAccelerator) GetInfo() *GPUInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return &GPUInfo{
		Name:              "CPU Fallback",
		DriverVersion:     "N/A",
		CUDAVersion:       "N/A",
		MemoryTotalMB:     int(m.Sys / 1024 / 1024),
		MemoryFreeMB:      int((m.Sys - m.Alloc) / 1024 / 1024),
		ComputeCapability: "N/A",
		MultiProcessors:   runtime.NumCPU(),
		TotalCores:        runtime.NumCPU(),
		Available:         true,
		LastUpdate:        time.Now(),
	}
}

// GetUtilization returns simulated utilization for CPU
func (c *cpuFallbackAccelerator) GetUtilization() *GPUUtilization {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	memUsagePercent := float64(m.Alloc) / float64(m.Sys) * 100
	
	return &GPUUtilization{
		GPUUsagePercent:    50.0, // Simulated CPU usage
		MemoryUsagePercent: memUsagePercent,
		MemoryUsedMB:       int(m.Alloc / 1024 / 1024),
		TemperatureCelsius: 45.0, // Simulated temperature
		PowerDrawWatts:     100.0, // Simulated power
		FanSpeedPercent:    40.0,  // Simulated fan speed
		ProcessCount:       1,
		Timestamp:          time.Now(),
	}
}

// IsHealthy always returns true for CPU fallback
func (c *cpuFallbackAccelerator) IsHealthy() bool {
	return c.initialized
}

// Register CPU fallback accelerator
func init() {
	RegisterAccelerator("cpu_fallback", func(config *GPUConfig) (GPUAccelerator, error) {
		return NewCPUFallbackAccelerator(config), nil
	})
}

// CreateBestAvailableAccelerator creates the best available accelerator
func CreateBestAvailableAccelerator(config *GPUConfig) GPUAccelerator {
	// Try NVIDIA first
	if nvidia, err := CreateAccelerator("nvidia", config); err == nil {
		if nvidia.IsAvailable() {
			return nvidia
		}
	}
	
	// Fallback to CPU
	return NewCPUFallbackAccelerator(config)
}