package pipeline

import (
	"context"
	"io"
	"time"

	"motion-index-fiber/pkg/processing/classifier"
	"motion-index-fiber/pkg/processing/extractor"
	"motion-index-fiber/pkg/models"
)

// Pipeline defines the interface for document processing pipeline
type Pipeline interface {
	// ProcessDocument processes a single document through the complete pipeline
	ProcessDocument(ctx context.Context, req *ProcessRequest) (*ProcessResult, error)

	// ProcessBatch processes multiple documents concurrently
	ProcessBatch(ctx context.Context, requests []*ProcessRequest) (*BatchResult, error)

	// GetStatus returns the current status of the pipeline
	GetStatus() *PipelineStatus

	// Stop gracefully stops the pipeline
	Stop(ctx context.Context) error

	// IsHealthy returns true if the pipeline is healthy
	IsHealthy() bool
}

// Processor handles individual document processing steps
type Processor interface {
	// Process executes a processing step
	Process(ctx context.Context, req *ProcessRequest) (*ProcessResult, error)

	// GetType returns the processor type
	GetType() ProcessorType

	// IsHealthy returns true if the processor is healthy
	IsHealthy() bool
}

// WorkerPoolInterface manages a pool of workers for concurrent processing
type WorkerPoolInterface interface {
	// Submit submits a job to the worker pool
	Submit(job Job) error

	// Start starts the worker pool
	Start(ctx context.Context) error

	// Stop stops the worker pool
	Stop(ctx context.Context) error

	// GetStats returns worker pool statistics
	GetStats() *PoolStats

	// IsRunning returns true if the worker pool is running
	IsRunning() bool
}

// ProcessRequest contains the document and metadata for processing
type ProcessRequest struct {
	ID          string            `json:"id"`
	FileName    string            `json:"file_name"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	Content     io.Reader         `json:"-"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Options     *ProcessOptions   `json:"options,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// ProcessOptions contains processing configuration
type ProcessOptions struct {
	ExtractText    bool `json:"extract_text"`
	ClassifyDoc    bool `json:"classify_doc"`
	IndexDocument  bool `json:"index_document"`
	StoreDocument  bool `json:"store_document"`
	SkipIfExists   bool `json:"skip_if_exists"`
	Priority       int  `json:"priority"`
	TimeoutSeconds int  `json:"timeout_seconds"`
	RetryCount     int  `json:"retry_count"`
}

// ProcessResult contains the result of document processing
type ProcessResult struct {
	ID                   string                           `json:"id"`
	Success              bool                             `json:"success"`
	Error                string                           `json:"error,omitempty"`
	Steps                []*ProcessStep                   `json:"steps"`
	ExtractionResult     *extractor.ExtractionResult      `json:"extraction_result,omitempty"`
	ClassificationResult *classifier.ClassificationResult `json:"classification_result,omitempty"`
	IndexResult          *IndexResult                     `json:"index_result,omitempty"`
	StorageResult        *StorageResult                   `json:"storage_result,omitempty"`
	Document             *models.Document                 `json:"document,omitempty"`
	ProcessingTime       int64                            `json:"processing_time_ms"`
	StartTime            time.Time                        `json:"start_time"`
	EndTime              time.Time                        `json:"end_time"`
}

// ProcessStep represents a single processing step
type ProcessStep struct {
	Type      ProcessorType `json:"type"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
	Duration  int64         `json:"duration_ms"`
	Timestamp time.Time     `json:"timestamp"`
}

// BatchResult contains results from batch processing
type BatchResult struct {
	TotalCount     int              `json:"total_count"`
	SuccessCount   int              `json:"success_count"`
	FailureCount   int              `json:"failure_count"`
	Results        []*ProcessResult `json:"results"`
	ProcessingTime int64            `json:"processing_time_ms"`
	StartTime      time.Time        `json:"start_time"`
	EndTime        time.Time        `json:"end_time"`
}

// IndexResult contains the result of document indexing
type IndexResult struct {
	DocumentID string `json:"document_id"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// StorageResult contains the result of document storage
type StorageResult struct {
	StoragePath string `json:"storage_path"`
	URL         string `json:"url,omitempty"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

// PipelineStatus contains the current status of the pipeline
type PipelineStatus struct {
	Running         bool               `json:"running"`
	ActiveJobs      int                `json:"active_jobs"`
	QueuedJobs      int                `json:"queued_jobs"`
	CompletedJobs   int64              `json:"completed_jobs"`
	FailedJobs      int64              `json:"failed_jobs"`
	ProcessorStatus []*ProcessorStatus `json:"processor_status"`
	WorkerPoolStats *PoolStats         `json:"worker_pool_stats,omitempty"`
	LastUpdate      time.Time          `json:"last_update"`
}

// ProcessorStatus contains the status of a processor
type ProcessorStatus struct {
	Type    ProcessorType `json:"type"`
	Healthy bool          `json:"healthy"`
	Error   string        `json:"error,omitempty"`
}

// PoolStats contains worker pool statistics
type PoolStats struct {
	WorkerCount    int   `json:"worker_count"`
	ActiveWorkers  int   `json:"active_workers"`
	QueueSize      int   `json:"queue_size"`
	ProcessedJobs  int64 `json:"processed_jobs"`
	FailedJobs     int64 `json:"failed_jobs"`
	AverageLatency int64 `json:"average_latency_ms"`
}

// Job represents a processing job
type Job interface {
	// Execute executes the job
	Execute(ctx context.Context) error

	// GetID returns the job ID
	GetID() string

	// GetPriority returns the job priority
	GetPriority() int

	// GetTimeout returns the job timeout
	GetTimeout() time.Duration
}

// ProcessorType represents the type of processor
type ProcessorType string

const (
	ProcessorTypeExtraction     ProcessorType = "extraction"
	ProcessorTypeClassification ProcessorType = "classification"
	ProcessorTypeIndexing       ProcessorType = "indexing"
	ProcessorTypeStorage        ProcessorType = "storage"
	ProcessorTypeValidation     ProcessorType = "validation"
)

// Default processing options
func DefaultProcessOptions() *ProcessOptions {
	return &ProcessOptions{
		ExtractText:    true,
		ClassifyDoc:    true,
		IndexDocument:  true,
		StoreDocument:  true,
		SkipIfExists:   false,
		Priority:       5,   // Medium priority
		TimeoutSeconds: 300, // 5 minutes
		RetryCount:     3,
	}
}

// PipelineError represents errors that occur during pipeline processing
type PipelineError struct {
	Type    string
	Message string
	Cause   error
	Step    ProcessorType
}

func (e *PipelineError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *PipelineError) Unwrap() error {
	return e.Cause
}

// NewPipelineError creates a new pipeline error
func NewPipelineError(errorType, message string, step ProcessorType, cause error) *PipelineError {
	return &PipelineError{
		Type:    errorType,
		Message: message,
		Step:    step,
		Cause:   cause,
	}
}
