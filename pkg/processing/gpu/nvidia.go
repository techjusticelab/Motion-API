package gpu

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type nvidiaAccelerator struct {
	config      *GPUConfig
	info        *GPUInfo
	utilization *GPUUtilization
	metrics     *PerformanceMetrics
	mutex       sync.RWMutex
	initialized bool
	available   bool

	// Performance tracking
	operationCount int64
	totalTime      time.Duration
	startTime      time.Time
}

func NewNVIDIAAccelerator(config *GPUConfig) (GPUAccelerator, error) {
	accelerator := &nvidiaAccelerator{
		config:    config,
		startTime: time.Now(),
		metrics: &PerformanceMetrics{
			StartTime: time.Now(),
		},
	}

	// Check if NVIDIA GPU is available
	if err := accelerator.checkAvailability(); err != nil {
		if config.FallbackToCPU {
			accelerator.available = false
			return accelerator, nil // Return with CPU fallback
		}
		return nil, fmt.Errorf("NVIDIA GPU not available: %w", err)
	}

	accelerator.available = true
	return accelerator, nil
}

func (n *nvidiaAccelerator) checkAvailability() error {
	// Check nvidia-smi availability
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,driver_version", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("nvidia-smi not available: %w", err)
	}

	if len(strings.TrimSpace(string(output))) == 0 {
		return fmt.Errorf("no NVIDIA GPUs detected")
	}

	return nil
}

func (n *nvidiaAccelerator) IsAvailable() bool {
	return n.available
}

func (n *nvidiaAccelerator) Initialize() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.initialized {
		return nil
	}

	if !n.available {
		return fmt.Errorf("NVIDIA GPU not available")
	}

	// Get GPU information
	info, err := n.getGPUInfo()
	if err != nil {
		return fmt.Errorf("failed to get GPU info: %w", err)
	}
	n.info = info

	// Initialize CUDA context (in a real implementation)
	// This would involve calling CUDA driver API or using a library like gorgonia/cu

	// For now, we'll simulate initialization
	if n.config.WarmupIterations > 0 {
		if err := n.performWarmup(); err != nil {
			return fmt.Errorf("GPU warmup failed: %w", err)
		}
	}

	n.initialized = true
	return nil
}

func (n *nvidiaAccelerator) getGPUInfo() (*GPUInfo, error) {
	// Query GPU properties using nvidia-smi
	queries := []string{
		"name",
		"driver_version",
		"memory.total",
		"memory.free",
		"compute_cap",
		"clocks.current.graphics",
		"clocks.current.memory",
		"temperature.gpu",
	}

	queryStr := strings.Join(queries, ",")
	cmd := exec.Command("nvidia-smi",
		fmt.Sprintf("--query-gpu=%s", queryStr),
		"--format=csv,noheader,nounits")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query GPU info: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("no GPU information returned")
	}

	// Parse first GPU (device 0)
	fields := strings.Split(lines[0], ", ")
	if len(fields) < len(queries) {
		return nil, fmt.Errorf("incomplete GPU information")
	}

	info := &GPUInfo{
		Name:          strings.TrimSpace(fields[0]),
		DriverVersion: strings.TrimSpace(fields[1]),
		Available:     true,
		LastUpdate:    time.Now(),
	}

	// Parse memory info
	if memTotal, err := strconv.Atoi(strings.TrimSpace(fields[2])); err == nil {
		info.MemoryTotalMB = memTotal
	}
	if memFree, err := strconv.Atoi(strings.TrimSpace(fields[3])); err == nil {
		info.MemoryFreeMB = memFree
	}

	// Parse compute capability
	info.ComputeCapability = strings.TrimSpace(fields[4])

	// Parse clock rates
	if clockRate, err := strconv.Atoi(strings.TrimSpace(fields[5])); err == nil {
		info.ClockRateMHz = clockRate
	}
	if memClock, err := strconv.Atoi(strings.TrimSpace(fields[6])); err == nil {
		info.MemoryClockMHz = memClock
	}

	// For detailed compute capability info, we'd need additional queries
	// This is simplified for the example

	return info, nil
}

func (n *nvidiaAccelerator) performWarmup() error {
	// In a real implementation, this would:
	// 1. Allocate GPU memory
	// 2. Perform simple operations to warm up the GPU
	// 3. Measure performance baselines

	// Simulate warmup delay
	time.Sleep(time.Duration(n.config.WarmupIterations) * 100 * time.Millisecond)

	return nil
}

func (n *nvidiaAccelerator) Shutdown() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if !n.initialized {
		return nil
	}

	// In a real implementation, this would:
	// 1. Free GPU memory
	// 2. Destroy CUDA contexts
	// 3. Clean up resources

	n.initialized = false
	return nil
}

func (n *nvidiaAccelerator) GetInfo() *GPUInfo {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	if n.info == nil {
		return &GPUInfo{Available: false}
	}

	return n.info
}

func (n *nvidiaAccelerator) GetUtilization() *GPUUtilization {
	if !n.available {
		return &GPUUtilization{Timestamp: time.Now()}
	}

	cmd := exec.Command("nvidia-smi",
		"--query-gpu=utilization.gpu,utilization.memory,memory.used,temperature.gpu,power.draw,fan.speed",
		"--format=csv,noheader,nounits")

	output, err := cmd.Output()
	if err != nil {
		return &GPUUtilization{Timestamp: time.Now()}
	}

	fields := strings.Split(strings.TrimSpace(string(output)), ", ")
	if len(fields) < 6 {
		return &GPUUtilization{Timestamp: time.Now()}
	}

	util := &GPUUtilization{Timestamp: time.Now()}

	if gpuUsage, err := strconv.ParseFloat(strings.TrimSpace(fields[0]), 64); err == nil {
		util.GPUUsagePercent = gpuUsage
	}
	if memUsage, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64); err == nil {
		util.MemoryUsagePercent = memUsage
	}
	if memUsed, err := strconv.Atoi(strings.TrimSpace(fields[2])); err == nil {
		util.MemoryUsedMB = memUsed
	}
	if temp, err := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64); err == nil {
		util.TemperatureCelsius = temp
	}
	if power, err := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64); err == nil {
		util.PowerDrawWatts = power
	}
	if fan, err := strconv.ParseFloat(strings.TrimSpace(fields[5]), 64); err == nil {
		util.FanSpeedPercent = fan
	}

	n.mutex.Lock()
	n.utilization = util
	n.mutex.Unlock()

	return util
}

func (n *nvidiaAccelerator) IsHealthy() bool {
	if !n.available {
		return n.config.FallbackToCPU // Healthy if we can fallback to CPU
	}

	util := n.GetUtilization()

	// Check temperature (assume unhealthy if > 85°C)
	if util.TemperatureCelsius > 85.0 {
		return false
	}

	// Check if GPU is responsive
	return util.Timestamp.After(time.Now().Add(-30 * time.Second))
}

func (n *nvidiaAccelerator) GetCapabilities() *GPUCapabilities {
	if !n.available {
		return &GPUCapabilities{}
	}

	// This would be determined based on actual GPU compute capability
	// For now, return conservative capabilities
	return &GPUCapabilities{
		TextProcessing:      true,
		ImageProcessing:     true,
		OCR:                 false, // Requires additional libraries
		Embeddings:          true,
		TensorOperations:    true,
		ConcurrentStreams:   true,
		UnifiedMemory:       false,
		SupportedPrecisions: []string{"fp32", "fp16"},
		MaxBatchSize:        n.config.BatchSize,
		MaxTextLength:       1000000, // 1M characters
	}
}

func (n *nvidiaAccelerator) EstimatePerformance(workloadType string, dataSize int) *PerformanceEstimate {
	if !n.available {
		return &PerformanceEstimate{
			UseCPU:           true,
			EstimatedTimeMS:  dataSize * 10, // CPU fallback estimate
			RecommendedBatch: 1,
		}
	}

	// Simple performance model based on GPU memory and compute
	memoryMB := n.info.MemoryFreeMB

	var estimatedTimeMS int
	var recommendedBatch int

	switch workloadType {
	case "text_extraction":
		// Text extraction is typically CPU-bound, GPU helps with parallel processing
		estimatedTimeMS = dataSize / 1000                   // 1ms per KB
		recommendedBatch = min(dataSize/1024, memoryMB/100) // Conservative memory usage

	case "text_embedding":
		// Embeddings benefit significantly from GPU
		estimatedTimeMS = dataSize / 10000 // Much faster on GPU
		recommendedBatch = min(dataSize/100, memoryMB/50)

	case "image_processing":
		// Image processing is highly GPU-optimized
		estimatedTimeMS = dataSize / 50000 // Very fast on GPU
		recommendedBatch = min(dataSize/1024, memoryMB/200)

	default:
		estimatedTimeMS = dataSize / 1000
		recommendedBatch = min(dataSize/1024, 32)
	}

	if recommendedBatch < 1 {
		recommendedBatch = 1
	}

	return &PerformanceEstimate{
		UseCPU:           false,
		UseGPU:           true,
		EstimatedTimeMS:  estimatedTimeMS,
		RecommendedBatch: recommendedBatch,
		MemoryRequiredMB: recommendedBatch * 50, // Estimate 50MB per batch item
		Confidence:       0.7,                   // Medium confidence for estimates
	}
}

type PerformanceEstimate struct {
	UseCPU           bool    `json:"use_cpu"`
	UseGPU           bool    `json:"use_gpu"`
	EstimatedTimeMS  int     `json:"estimated_time_ms"`
	RecommendedBatch int     `json:"recommended_batch"`
	MemoryRequiredMB int     `json:"memory_required_mb"`
	Confidence       float64 `json:"confidence"`
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (n *nvidiaAccelerator) GetPerformanceMetrics() *PerformanceMetrics {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	return n.metrics
}
