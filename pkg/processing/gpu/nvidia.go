package gpu

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// nvidiaAccelerator implements GPU acceleration using NVIDIA GPUs
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

// NewNVIDIAAccelerator creates a new NVIDIA GPU accelerator
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

// checkAvailability checks if NVIDIA GPU is available
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

// IsAvailable returns true if GPU acceleration is available
func (n *nvidiaAccelerator) IsAvailable() bool {
	return n.available
}

// Initialize initializes the GPU accelerator
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

// getGPUInfo retrieves detailed GPU information
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

// performWarmup performs GPU warmup operations
func (n *nvidiaAccelerator) performWarmup() error {
	// In a real implementation, this would:
	// 1. Allocate GPU memory
	// 2. Perform simple operations to warm up the GPU
	// 3. Measure performance baselines
	
	// Simulate warmup delay
	time.Sleep(time.Duration(n.config.WarmupIterations) * 100 * time.Millisecond)
	
	return nil
}

// Shutdown gracefully shuts down the GPU accelerator
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

// GetInfo returns GPU information
func (n *nvidiaAccelerator) GetInfo() *GPUInfo {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	
	if n.info == nil {
		return &GPUInfo{Available: false}
	}
	
	return n.info
}

// GetUtilization returns current GPU utilization
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

// IsHealthy returns true if the GPU is healthy
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

// GetCapabilities returns GPU capabilities
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
		UnifiedMemory:      false,
		SupportedPrecisions: []string{"fp32", "fp16"},
		MaxBatchSize:       n.config.BatchSize,
		MaxTextLength:      1000000, // 1M characters
	}
}

// EstimatePerformance estimates performance for given workload
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
		estimatedTimeMS = dataSize / 1000 // 1ms per KB
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
		Confidence:       0.7, // Medium confidence for estimates
	}
}

// PerformanceEstimate contains performance estimation
type PerformanceEstimate struct {
	UseCPU           bool    `json:"use_cpu"`
	UseGPU           bool    `json:"use_gpu"`
	EstimatedTimeMS  int     `json:"estimated_time_ms"`
	RecommendedBatch int     `json:"recommended_batch"`
	MemoryRequiredMB int     `json:"memory_required_mb"`
	Confidence       float64 `json:"confidence"`
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetPerformanceMetrics returns current performance metrics
func (n *nvidiaAccelerator) GetPerformanceMetrics() *PerformanceMetrics {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	
	return n.metrics
}

// UpdatePerformanceMetrics updates performance metrics
func (n *nvidiaAccelerator) UpdatePerformanceMetrics(duration time.Duration, success bool) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	
	n.metrics.TotalOperations++
	if success {
		n.metrics.SuccessfulOps++
	} else {
		n.metrics.FailedOps++
	}
	
	n.metrics.TotalGPUTime += duration
	n.metrics.LastOperation = time.Now()
	
	// Update average latency
	if n.metrics.TotalOperations > 0 {
		n.metrics.AverageLatency = n.metrics.TotalGPUTime / time.Duration(n.metrics.TotalOperations)
		
		// Calculate throughput
		totalTime := time.Since(n.metrics.StartTime)
		if totalTime > 0 {
			n.metrics.ThroughputPerSec = float64(n.metrics.TotalOperations) / totalTime.Seconds()
		}
	}
}

// Benchmark runs a performance benchmark
func (n *nvidiaAccelerator) Benchmark(ctx context.Context, workloadType string, iterations int) (*BenchmarkResult, error) {
	if !n.available {
		return nil, fmt.Errorf("GPU not available for benchmarking")
	}
	
	result := &BenchmarkResult{
		WorkloadType: workloadType,
		Iterations:   iterations,
		StartTime:    time.Now(),
	}
	
	var totalTime time.Duration
	successCount := 0
	
	for i := 0; i < iterations; i++ {
		start := time.Now()
		
		// Simulate workload (in real implementation, this would run actual GPU operations)
		success := n.simulateWorkload(ctx, workloadType)
		
		duration := time.Since(start)
		totalTime += duration
		
		if success {
			successCount++
		}
		
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
	
	result.EndTime = time.Now()
	result.TotalTime = totalTime
	result.AverageTime = totalTime / time.Duration(iterations)
	result.SuccessRate = float64(successCount) / float64(iterations)
	result.ThroughputPerSec = float64(iterations) / result.EndTime.Sub(result.StartTime).Seconds()
	
	return result, nil
}

// simulateWorkload simulates a GPU workload for benchmarking
func (n *nvidiaAccelerator) simulateWorkload(ctx context.Context, workloadType string) bool {
	// Simulate different types of GPU work
	switch workloadType {
	case "text_processing":
		time.Sleep(time.Millisecond * 10) // Simulate 10ms text processing
	case "image_processing":
		time.Sleep(time.Millisecond * 5) // Simulate 5ms image processing
	case "memory_bandwidth":
		time.Sleep(time.Millisecond * 2) // Simulate 2ms memory operations
	default:
		time.Sleep(time.Millisecond * 8) // Default simulation
	}
	
	// Simulate 95% success rate
	return time.Now().UnixNano()%100 < 95
}

// BenchmarkResult contains benchmark results
type BenchmarkResult struct {
	WorkloadType     string        `json:"workload_type"`
	Iterations       int           `json:"iterations"`
	StartTime        time.Time     `json:"start_time"`
	EndTime          time.Time     `json:"end_time"`
	TotalTime        time.Duration `json:"total_time"`
	AverageTime      time.Duration `json:"average_time"`
	SuccessRate      float64       `json:"success_rate"`
	ThroughputPerSec float64       `json:"throughput_per_sec"`
}

// Register NVIDIA accelerator
func init() {
	RegisterAccelerator("nvidia", func(config *GPUConfig) (GPUAccelerator, error) {
		return NewNVIDIAAccelerator(config)
	})
}

// GetRecommendedGPUConfig returns recommended GPU configuration for the current system
func GetRecommendedGPUConfig() *GPUConfig {
	config := &GPUConfig{
		Enabled:          false, // Default to disabled for safety
		DeviceID:         0,     // Use first GPU
		MemoryLimitMB:    4096,  // Conservative 4GB limit
		BatchSize:        32,    // Reasonable batch size
		StreamCount:      2,     // Multiple streams for concurrency
		EnableProfiling:  false, // Disable profiling by default
		LogLevel:         "INFO",
		FallbackToCPU:    true,  // Always allow CPU fallback
		WarmupIterations: 5,     // Quick warmup
		BenchmarkMode:    false,
		OptimizeFor:      "speed", // Optimize for speed by default
	}
	
	// Try to detect GPU and adjust config
	if accelerator, err := NewNVIDIAAccelerator(config); err == nil {
		if accelerator.IsAvailable() {
			config.Enabled = true
			
			if err := accelerator.Initialize(); err == nil {
				info := accelerator.GetInfo()
				if info != nil && info.MemoryTotalMB > 0 {
					// Use up to 75% of GPU memory
					config.MemoryLimitMB = int(float64(info.MemoryTotalMB) * 0.75)
					
					// Adjust batch size based on memory
					if info.MemoryTotalMB >= 16384 { // 16GB+
						config.BatchSize = 128
					} else if info.MemoryTotalMB >= 8192 { // 8GB+
						config.BatchSize = 64
					} else if info.MemoryTotalMB >= 4096 { // 4GB+
						config.BatchSize = 32
					} else {
						config.BatchSize = 16 // Conservative for <4GB
					}
				}
				accelerator.Shutdown()
			}
		}
	}
	
	return config
}