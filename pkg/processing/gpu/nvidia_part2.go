package gpu

import (
	"context"
	"fmt"
	"time"
)

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

func init() {
	RegisterAccelerator("nvidia", func(config *GPUConfig) (GPUAccelerator, error) {
		return NewNVIDIAAccelerator(config)
	})
}

func GetRecommendedGPUConfig() *GPUConfig {
	config := &GPUConfig{
		Enabled:          false, // Default to disabled for safety
		DeviceID:         0,     // Use first GPU
		MemoryLimitMB:    4096,  // Conservative 4GB limit
		BatchSize:        32,    // Reasonable batch size
		StreamCount:      2,     // Multiple streams for concurrency
		EnableProfiling:  false, // Disable profiling by default
		LogLevel:         "INFO",
		FallbackToCPU:    true, // Always allow CPU fallback
		WarmupIterations: 5,    // Quick warmup
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
