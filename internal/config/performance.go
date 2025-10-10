package config

import (
	"fmt"
	"runtime"
	"time"
)

// PerformanceConfig holds hardware-optimized performance settings
type PerformanceConfig struct {
	// Hardware detection
	CPUCores         int `json:"cpu_cores"`
	MemoryGB         int `json:"memory_gb"`
	AvailableMemoryGB int `json:"available_memory_gb"`
	
	// Worker pool configuration
	ExtractionWorkers    int `json:"extraction_workers"`
	ClassificationWorkers int `json:"classification_workers"`
	IndexingWorkers      int `json:"indexing_workers"`
	DownloadWorkers      int `json:"download_workers"`
	
	// Queue configurations
	ExtractionQueueSize    int `json:"extraction_queue_size"`
	ClassificationQueueSize int `json:"classification_queue_size"`
	IndexingQueueSize      int `json:"indexing_queue_size"`
	
	// Batch sizing
	DownloadBatchSize     int `json:"download_batch_size"`
	ProcessingBatchSize   int `json:"processing_batch_size"`
	IndexingBatchSize     int `json:"indexing_batch_size"`
	
	// Rate limiting
	GPTRequestsPerMinute  int           `json:"gpt_requests_per_minute"`
	GPTBurstSize         int           `json:"gpt_burst_size"`
	GPTRetryDelay        time.Duration `json:"gpt_retry_delay"`
	
	// Memory management
	MaxMemoryUsagePercent int   `json:"max_memory_usage_percent"`
	DocumentMemoryLimitMB int   `json:"document_memory_limit_mb"`
	GCInterval           time.Duration `json:"gc_interval"`
	
	// Timeouts
	DownloadTimeout      time.Duration `json:"download_timeout"`
	ExtractionTimeout    time.Duration `json:"extraction_timeout"`
	ClassificationTimeout time.Duration `json:"classification_timeout"`
	IndexingTimeout      time.Duration `json:"indexing_timeout"`
	
	// GPU settings
	EnableGPU           bool `json:"enable_gpu"`
	GPUMemoryLimitMB    int  `json:"gpu_memory_limit_mb"`
	GPUBatchSize        int  `json:"gpu_batch_size"`
	
	// PDF processing limits
	MaxPDFPages         int  `json:"max_pdf_pages"`
}

// HardwareInfo contains detected hardware information
type HardwareInfo struct {
	CPUCores          int     `json:"cpu_cores"`
	MemoryTotalGB     float64 `json:"memory_total_gb"`
	MemoryAvailableGB float64 `json:"memory_available_gb"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	HasNVIDIAGPU      bool    `json:"has_nvidia_gpu"`
	GPUMemoryMB       int     `json:"gpu_memory_mb"`
}

// DetectHardware automatically detects system hardware capabilities
func DetectHardware() (*HardwareInfo, error) {
	info := &HardwareInfo{
		CPUCores: runtime.NumCPU(),
	}
	
	// Detect memory (this is a simplified version - in production you'd use more sophisticated methods)
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// Get system memory info from /proc/meminfo if available
	totalMemory, availableMemory, err := getSystemMemoryInfo()
	if err != nil {
		// Fallback to runtime memory stats
		info.MemoryTotalGB = float64(memStats.Sys) / (1024 * 1024 * 1024)
		info.MemoryAvailableGB = float64(memStats.Sys-memStats.Alloc) / (1024 * 1024 * 1024)
	} else {
		info.MemoryTotalGB = totalMemory
		info.MemoryAvailableGB = availableMemory
	}
	
	if info.MemoryTotalGB > 0 {
		info.MemoryUsagePercent = ((info.MemoryTotalGB - info.MemoryAvailableGB) / info.MemoryTotalGB) * 100
	}
	
	// Check for NVIDIA GPU (placeholder - would need CUDA/nvidia-ml-go library in production)
	info.HasNVIDIAGPU, info.GPUMemoryMB = detectNVIDIAGPU()
	
	return info, nil
}

// OptimizeForHardware creates performance configuration optimized for detected hardware
func OptimizeForHardware(hardware *HardwareInfo) *PerformanceConfig {
	config := &PerformanceConfig{
		CPUCores:             hardware.CPUCores,
		MemoryGB:             int(hardware.MemoryTotalGB),
		AvailableMemoryGB:    int(hardware.MemoryAvailableGB),
		
		// Rate limiting for OpenAI GPT-4 (conservative estimates)
		GPTRequestsPerMinute: 50,  // GPT-4 tier 1 rate limit
		GPTBurstSize:        10,   // Allow burst of requests
		GPTRetryDelay:       2 * time.Second,
		
		// Memory management
		MaxMemoryUsagePercent: 70, // Use max 70% of available memory
		DocumentMemoryLimitMB: 100, // Max 100MB per document in memory
		GCInterval:           30 * time.Second,
		
		// Timeouts
		DownloadTimeout:       60 * time.Second,
		ExtractionTimeout:     120 * time.Second,
		ClassificationTimeout: 300 * time.Second,  // GPT can be slow
		IndexingTimeout:       30 * time.Second,
		
		// GPU settings
		EnableGPU:        hardware.HasNVIDIAGPU,
		GPUMemoryLimitMB: hardware.GPUMemoryMB / 2, // Use half of GPU memory
		GPUBatchSize:     32,
		
		// PDF processing limits
		MaxPDFPages:      25, // Limit to 25 pages to prevent memory exhaustion
	}
	
	// Calculate optimal worker counts based on hardware
	config.calculateOptimalWorkers(hardware)
	
	// Calculate optimal queue sizes
	config.calculateOptimalQueues(hardware)
	
	// Calculate optimal batch sizes
	config.calculateOptimalBatches(hardware)
	
	return config
}

// calculateOptimalWorkers determines optimal worker counts for each stage
func (c *PerformanceConfig) calculateOptimalWorkers(hardware *HardwareInfo) {
	cores := hardware.CPUCores
	
	// Download workers: I/O bound, can use many workers
	c.DownloadWorkers = cores * 2  // Up to 2x CPU cores for I/O
	if c.DownloadWorkers > 40 {
		c.DownloadWorkers = 40     // Cap at reasonable limit
	}
	
	// Extraction workers: CPU intensive but can be parallel
	c.ExtractionWorkers = cores - 2  // Leave 2 cores for other operations
	if c.ExtractionWorkers < 4 {
		c.ExtractionWorkers = 4
	}
	
	// Classification workers: Rate limited by OpenAI API
	// Conservative: 3-5 concurrent requests to avoid rate limits
	c.ClassificationWorkers = 4
	if cores >= 16 {
		c.ClassificationWorkers = 5
	}
	
	// Indexing workers: Network and CPU bound
	c.IndexingWorkers = cores / 3  // Use ~1/3 of cores for indexing
	if c.IndexingWorkers < 4 {
		c.IndexingWorkers = 4
	}
	if c.IndexingWorkers > 12 {
		c.IndexingWorkers = 12
	}
}

// calculateOptimalQueues determines optimal queue sizes
func (c *PerformanceConfig) calculateOptimalQueues(hardware *HardwareInfo) {
	memGB := int(hardware.MemoryAvailableGB)
	
	// Base queue size on available memory
	// Assume average document ~2MB in memory
	baseQueueSize := (memGB * 1024) / 10  // Conservative memory usage
	
	c.ExtractionQueueSize = baseQueueSize
	if c.ExtractionQueueSize > 1000 {
		c.ExtractionQueueSize = 1000
	}
	
	c.ClassificationQueueSize = baseQueueSize / 2  // Classification queue smaller
	if c.ClassificationQueueSize > 500 {
		c.ClassificationQueueSize = 500
	}
	
	c.IndexingQueueSize = baseQueueSize / 4
	if c.IndexingQueueSize < 100 {
		c.IndexingQueueSize = 100
	}
}

// calculateOptimalBatches determines optimal batch sizes for different operations
func (c *PerformanceConfig) calculateOptimalBatches(hardware *HardwareInfo) {
	memGB := int(hardware.MemoryAvailableGB)
	
	// Download batch: Large batches for efficient S3 operations
	c.DownloadBatchSize = 50
	if memGB >= 32 {
		c.DownloadBatchSize = 100
	}
	if memGB >= 64 {
		c.DownloadBatchSize = 200
	}
	
	// Processing batch: Balance memory usage and throughput
	c.ProcessingBatchSize = 20
	if memGB >= 32 {
		c.ProcessingBatchSize = 50
	}
	if memGB >= 64 {
		c.ProcessingBatchSize = 100
	}
	
	// Indexing batch: Efficient OpenSearch bulk operations
	c.IndexingBatchSize = 25
	if memGB >= 32 {
		c.IndexingBatchSize = 50
	}
}

// GetSummary returns a human-readable summary of the performance configuration
func (c *PerformanceConfig) GetSummary() string {
	return fmt.Sprintf(`Performance Configuration Summary:
Hardware: %d CPU cores, %d GB memory (%d GB available)
Worker Pools:
  - Download: %d workers
  - Extraction: %d workers  
  - Classification: %d workers (GPT rate-limited)
  - Indexing: %d workers
Queue Sizes:
  - Extraction: %d
  - Classification: %d
  - Indexing: %d
Batch Sizes:
  - Download: %d documents
  - Processing: %d documents
  - Indexing: %d documents
Rate Limiting:
  - GPT requests: %d/minute (burst: %d)
GPU: %v`,
		c.CPUCores, c.MemoryGB, c.AvailableMemoryGB,
		c.DownloadWorkers, c.ExtractionWorkers, c.ClassificationWorkers, c.IndexingWorkers,
		c.ExtractionQueueSize, c.ClassificationQueueSize, c.IndexingQueueSize,
		c.DownloadBatchSize, c.ProcessingBatchSize, c.IndexingBatchSize,
		c.GPTRequestsPerMinute, c.GPTBurstSize,
		c.EnableGPU)
}

// ValidateConfiguration ensures the configuration is reasonable
func (c *PerformanceConfig) ValidateConfiguration() error {
	if c.CPUCores <= 0 {
		return fmt.Errorf("invalid CPU cores count: %d", c.CPUCores)
	}
	
	if c.AvailableMemoryGB <= 0 {
		return fmt.Errorf("invalid available memory: %d GB", c.AvailableMemoryGB)
	}
	
	// Ensure worker counts are reasonable
	totalWorkers := c.DownloadWorkers + c.ExtractionWorkers + c.ClassificationWorkers + c.IndexingWorkers
	if totalWorkers > c.CPUCores*3 {
		return fmt.Errorf("total workers (%d) exceeds reasonable limit for %d cores", totalWorkers, c.CPUCores)
	}
	
	if c.GPTRequestsPerMinute <= 0 || c.GPTRequestsPerMinute > 1000 {
		return fmt.Errorf("invalid GPT requests per minute: %d", c.GPTRequestsPerMinute)
	}
	
	return nil
}