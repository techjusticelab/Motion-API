package gpu

import (
	"context"
	"fmt"
	"time"
)

// GPUAccelerator interface defines GPU acceleration capabilities
type GPUAccelerator interface {
	// IsAvailable returns true if GPU acceleration is available
	IsAvailable() bool
	
	// Initialize initializes the GPU accelerator
	Initialize() error
	
	// Shutdown gracefully shuts down the GPU accelerator
	Shutdown() error
	
	// GetInfo returns GPU information
	GetInfo() *GPUInfo
	
	// GetUtilization returns current GPU utilization
	GetUtilization() *GPUUtilization
	
	// IsHealthy returns true if the GPU is healthy
	IsHealthy() bool
}

// TextProcessor interface defines GPU-accelerated text processing
type TextProcessor interface {
	GPUAccelerator
	
	// ProcessTextBatch processes multiple texts in parallel on GPU
	ProcessTextBatch(ctx context.Context, texts []string, options *ProcessingOptions) (*BatchResult, error)
	
	// ExtractFeatures extracts text features using GPU
	ExtractFeatures(ctx context.Context, text string) (*TextFeatures, error)
	
	// GenerateEmbeddings generates text embeddings using GPU
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
	
	// GetSupportedOperations returns list of supported GPU operations
	GetSupportedOperations() []string
}

// DocumentProcessor interface defines GPU-accelerated document processing
type DocumentProcessor interface {
	GPUAccelerator
	
	// ProcessPDFBatch processes multiple PDFs in parallel
	ProcessPDFBatch(ctx context.Context, documents []*DocumentData, options *ProcessingOptions) (*BatchResult, error)
	
	// ExtractTextGPU extracts text from documents using GPU acceleration
	ExtractTextGPU(ctx context.Context, documentData []byte, docType string) (*ExtractionResult, error)
	
	// PreprocessImage preprocesses document images for better OCR
	PreprocessImage(ctx context.Context, imageData []byte) ([]byte, error)
	
	// BatchOCR performs OCR on multiple images using GPU
	BatchOCR(ctx context.Context, images [][]byte) ([]*OCRResult, error)
}

// GPUInfo contains information about the GPU
type GPUInfo struct {
	Name              string    `json:"name"`
	DriverVersion     string    `json:"driver_version"`
	CUDAVersion       string    `json:"cuda_version"`
	MemoryTotalMB     int       `json:"memory_total_mb"`
	MemoryFreeMB      int       `json:"memory_free_mb"`
	ComputeCapability string    `json:"compute_capability"`
	MultiProcessors   int       `json:"multi_processors"`
	CoresPerMP        int       `json:"cores_per_mp"`
	TotalCores        int       `json:"total_cores"`
	MaxThreadsPerBlock int      `json:"max_threads_per_block"`
	MaxBlockDimensions [3]int   `json:"max_block_dimensions"`
	MaxGridDimensions  [3]int   `json:"max_grid_dimensions"`
	ClockRateMHz      int       `json:"clock_rate_mhz"`
	MemoryClockMHz    int       `json:"memory_clock_mhz"`
	MemoryBusWidth    int       `json:"memory_bus_width"`
	L2CacheSizeKB     int       `json:"l2_cache_size_kb"`
	TextureAlignment  int       `json:"texture_alignment"`
	Available         bool      `json:"available"`
	LastUpdate        time.Time `json:"last_update"`
}

// GPUUtilization contains GPU utilization metrics
type GPUUtilization struct {
	GPUUsagePercent     float64   `json:"gpu_usage_percent"`
	MemoryUsagePercent  float64   `json:"memory_usage_percent"`
	MemoryUsedMB        int       `json:"memory_used_mb"`
	TemperatureCelsius  float64   `json:"temperature_celsius"`
	PowerDrawWatts      float64   `json:"power_draw_watts"`
	FanSpeedPercent     float64   `json:"fan_speed_percent"`
	ProcessCount        int       `json:"process_count"`
	Timestamp          time.Time `json:"timestamp"`
}

// ProcessingOptions defines options for GPU processing
type ProcessingOptions struct {
	BatchSize         int           `json:"batch_size"`
	Timeout          time.Duration `json:"timeout"`
	UseGPUMemory     bool          `json:"use_gpu_memory"`
	MaxGPUMemoryMB   int           `json:"max_gpu_memory_mb"`
	EnableProfiling  bool          `json:"enable_profiling"`
	OptimizeFor      string        `json:"optimize_for"` // "speed", "memory", "quality"
	RetryOnError     bool          `json:"retry_on_error"`
	FallbackToCPU    bool          `json:"fallback_to_cpu"`
	ParallelStreams  int           `json:"parallel_streams"`
}

// BatchResult contains results from batch processing
type BatchResult struct {
	Results           []*ProcessingResult `json:"results"`
	TotalProcessed    int                 `json:"total_processed"`
	SuccessCount      int                 `json:"success_count"`
	ErrorCount        int                 `json:"error_count"`
	TotalTime         time.Duration       `json:"total_time"`
	AverageTime       time.Duration       `json:"average_time"`
	GPUMemoryUsedMB   int                 `json:"gpu_memory_used_mb"`
	ThroughputPerSec  float64             `json:"throughput_per_sec"`
	Errors            []error             `json:"errors"`
}

// ProcessingResult contains result from single item processing
type ProcessingResult struct {
	ID            string                 `json:"id"`
	Success       bool                   `json:"success"`
	Error         error                  `json:"error,omitempty"`
	ProcessingTime time.Duration         `json:"processing_time"`
	Output        interface{}            `json:"output,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	GPUMemoryUsed int                    `json:"gpu_memory_used"`
	UsedGPU       bool                   `json:"used_gpu"`
}

// TextFeatures contains extracted text features
type TextFeatures struct {
	TokenCount       int                    `json:"token_count"`
	Vocabulary       []string               `json:"vocabulary"`
	FrequencyMap     map[string]int         `json:"frequency_map"`
	Embeddings       []float32              `json:"embeddings"`
	Sentiment        float32                `json:"sentiment"`
	Language         string                 `json:"language"`
	Complexity       float32                `json:"complexity"`
	Readability      float32                `json:"readability"`
	NamedEntities    []NamedEntity          `json:"named_entities"`
	Keywords         []string               `json:"keywords"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// NamedEntity represents a named entity in text
type NamedEntity struct {
	Text       string  `json:"text"`
	Label      string  `json:"label"`
	Confidence float32 `json:"confidence"`
	StartPos   int     `json:"start_pos"`
	EndPos     int     `json:"end_pos"`
}

// DocumentData represents document data for processing
type DocumentData struct {
	ID          string            `json:"id"`
	Data        []byte            `json:"data"`
	ContentType string            `json:"content_type"`
	Filename    string            `json:"filename"`
	Size        int64             `json:"size"`
	Metadata    map[string]string `json:"metadata"`
}

// ExtractionResult contains text extraction results
type ExtractionResult struct {
	Text           string                 `json:"text"`
	WordCount      int                    `json:"word_count"`
	CharCount      int                    `json:"char_count"`
	PageCount      int                    `json:"page_count"`
	Language       string                 `json:"language"`
	Confidence     float32                `json:"confidence"`
	ProcessingTime time.Duration          `json:"processing_time"`
	UsedGPU        bool                   `json:"used_gpu"`
	OCRResults     []*OCRResult           `json:"ocr_results,omitempty"`
	Metadata       map[string]interface{} `json:"metadata"`
	Error          error                  `json:"error,omitempty"`
}

// OCRResult contains OCR processing results
type OCRResult struct {
	PageNumber     int       `json:"page_number"`
	Text           string    `json:"text"`
	Confidence     float32   `json:"confidence"`
	BoundingBoxes  []BBox    `json:"bounding_boxes"`
	ProcessingTime time.Duration `json:"processing_time"`
	Error          error     `json:"error,omitempty"`
}

// BBox represents a bounding box for text detection
type BBox struct {
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Text   string  `json:"text"`
	Confidence float32 `json:"confidence"`
}

// GPUConfig contains GPU configuration options
type GPUConfig struct {
	Enabled           bool          `json:"enabled"`
	DeviceID          int           `json:"device_id"`
	MemoryLimitMB     int           `json:"memory_limit_mb"`
	BatchSize         int           `json:"batch_size"`
	StreamCount       int           `json:"stream_count"`
	EnableProfiling   bool          `json:"enable_profiling"`
	LogLevel          string        `json:"log_level"`
	FallbackToCPU     bool          `json:"fallback_to_cpu"`
	WarmupIterations  int           `json:"warmup_iterations"`
	BenchmarkMode     bool          `json:"benchmark_mode"`
	OptimizeFor       string        `json:"optimize_for"` // "speed", "memory", "quality"
}

// PerformanceMetrics contains GPU performance metrics
type PerformanceMetrics struct {
	TotalOperations     int64         `json:"total_operations"`
	SuccessfulOps       int64         `json:"successful_ops"`
	FailedOps           int64         `json:"failed_ops"`
	AverageLatency      time.Duration `json:"average_latency"`
	ThroughputPerSec    float64       `json:"throughput_per_sec"`
	GPUUtilizationAvg   float64       `json:"gpu_utilization_avg"`
	MemoryUtilizationAvg float64      `json:"memory_utilization_avg"`
	TotalGPUTime        time.Duration `json:"total_gpu_time"`
	TotalCPUTime        time.Duration `json:"total_cpu_time"`
	MemoryPeakUsageMB   int           `json:"memory_peak_usage_mb"`
	LastOperation       time.Time     `json:"last_operation"`
	StartTime           time.Time     `json:"start_time"`
}

// GPUError represents GPU-specific errors
type GPUError struct {
	Code        int    `json:"code"`
	Message     string `json:"message"`
	Operation   string `json:"operation"`
	DeviceID    int    `json:"device_id"`
	Recoverable bool   `json:"recoverable"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e *GPUError) Error() string {
	return fmt.Sprintf("GPU Error [%d]: %s (operation: %s, device: %d)", 
		e.Code, e.Message, e.Operation, e.DeviceID)
}

// GPUCapabilities represents what the GPU can do
type GPUCapabilities struct {
	TextProcessing     bool     `json:"text_processing"`
	ImageProcessing    bool     `json:"image_processing"`
	OCR                bool     `json:"ocr"`
	Embeddings         bool     `json:"embeddings"`
	TensorOperations   bool     `json:"tensor_operations"`
	ConcurrentStreams  bool     `json:"concurrent_streams"`
	UnifiedMemory      bool     `json:"unified_memory"`
	PeerToPeer         bool     `json:"peer_to_peer"`
	SupportedPrecisions []string `json:"supported_precisions"` // "fp16", "fp32", "int8"
	MaxBatchSize       int      `json:"max_batch_size"`
	MaxTextLength      int      `json:"max_text_length"`
}

// Factory function to create GPU accelerator
type GPUAcceleratorFactory func(config *GPUConfig) (GPUAccelerator, error)

// Registry for different GPU accelerator implementations
var acceleratorRegistry = make(map[string]GPUAcceleratorFactory)

// RegisterAccelerator registers a GPU accelerator implementation
func RegisterAccelerator(name string, factory GPUAcceleratorFactory) {
	acceleratorRegistry[name] = factory
}

// CreateAccelerator creates a GPU accelerator by name
func CreateAccelerator(name string, config *GPUConfig) (GPUAccelerator, error) {
	factory, exists := acceleratorRegistry[name]
	if !exists {
		return nil, fmt.Errorf("unknown GPU accelerator: %s", name)
	}
	
	return factory(config)
}

// GetAvailableAccelerators returns list of available accelerator implementations
func GetAvailableAccelerators() []string {
	names := make([]string, 0, len(acceleratorRegistry))
	for name := range acceleratorRegistry {
		names = append(names, name)
	}
	return names
}