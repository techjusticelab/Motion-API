package models

import (
	"time"

	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/storage"
)

// ExtractionResult represents the result of text extraction
type ExtractionResult struct {
	Text      string `json:"text"`
	PageCount int    `json:"page_count"`
	Language  string `json:"language"`
}

// ClassificationResult represents the result of document classification
type ClassificationResult struct {
	Category   string   `json:"category"`
	Confidence float64  `json:"confidence"`
	Tags       []string `json:"tags"`
}

// Re-export types from pkg/models for internal use
type APIResponse = models.APIResponse
type APIError = models.APIError

// ProcessDocumentResponse represents the response from document processing
type ProcessDocumentResponse struct {
	DocumentID           string                `json:"document_id"`
	FileName             string                `json:"file_name"`
	Status               string                `json:"status"`
	ProcessingTime       int64                 `json:"processing_time_ms"`
	ExtractionResult     *ExtractionResult     `json:"extraction_result,omitempty"`
	ClassificationResult *ClassificationResult `json:"classification_result,omitempty"`
	IndexResult          *IndexResult          `json:"index_result,omitempty"`
	StorageResult        *storage.UploadResult `json:"storage_result,omitempty"`
	URL                  string                `json:"url,omitempty"`
	CDN_URL              string                `json:"cdn_url,omitempty"`
	Steps                []*ProcessingStep     `json:"steps,omitempty"`
	Metadata             *DocumentMetadata     `json:"metadata,omitempty"`
	CreatedAt            time.Time             `json:"created_at"`
}

// BatchProcessResponse represents the response from batch processing
type BatchProcessResponse struct {
	BatchID        string                     `json:"batch_id"`
	TotalCount     int                        `json:"total_count"`
	SuccessCount   int                        `json:"success_count"`
	FailureCount   int                        `json:"failure_count"`
	Results        []*ProcessDocumentResponse `json:"results"`
	Errors         []*BatchProcessError       `json:"errors,omitempty"`
	ProcessingTime int64                      `json:"processing_time_ms"`
	Status         string                     `json:"status"`
	CompletedAt    time.Time                  `json:"completed_at"`
}

// BatchProcessError represents an error in batch processing
type BatchProcessError struct {
	FileName string `json:"file_name"`
	Error    string `json:"error"`
	Code     string `json:"code"`
}

// IndexResult represents the result of document indexing
type IndexResult struct {
	DocumentID string `json:"document_id"`
	IndexName  string `json:"index_name"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// ProcessingStep represents a single step in the processing pipeline
type ProcessingStep struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Duration    int64     `json:"duration_ms"`
	Error       string    `json:"error,omitempty"`
	ProcessedBy string    `json:"processed_by,omitempty"`
}

// Re-export DocumentMetadata from pkg/models
type DocumentMetadata = models.DocumentMetadata

// SearchDocumentsResponse represents the response from document search
type SearchDocumentsResponse struct {
	Query        string                   `json:"query"`
	TotalHits    int64                    `json:"total_hits"`
	MaxScore     float64                  `json:"max_score,omitempty"`
	SearchTime   int64                    `json:"search_time_ms"`
	Page         int                      `json:"page"`
	Size         int                      `json:"size"`
	Documents    []*models.Document `json:"documents"`
	Aggregations map[string]interface{}   `json:"aggregations,omitempty"`
	Suggestions  []string                 `json:"suggestions,omitempty"`
}

// DocumentStatsResponse represents document statistics
type DocumentStatsResponse struct {
	TotalDocuments   int64            `json:"total_documents"`
	TotalSize        int64            `json:"total_size_bytes"`
	AverageSize      int64            `json:"average_size_bytes"`
	DocumentTypes    []*TypeCount     `json:"document_types"`
	Categories       []*CategoryCount `json:"categories"`
	LegalTags        []*TagCount      `json:"legal_tags"`
	Courts           []*CourtCount    `json:"courts"`
	Authors          []*AuthorCount   `json:"authors"`
	Judges           []*JudgeCount    `json:"judges"`
	DateDistribution []*DateCount     `json:"date_distribution,omitempty"`
	GeneratedAt      time.Time        `json:"generated_at"`
}

// Re-export types from pkg/models that exist there
type TypeCount = models.TypeCount
type TagCount = models.TagCount

// CategoryCount represents category statistics (internal only)
type CategoryCount struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

// CourtCount represents court statistics
type CourtCount struct {
	Court string `json:"court"`
	Count int64  `json:"count"`
}

// AuthorCount represents author statistics
type AuthorCount struct {
	Author string `json:"author"`
	Count  int64  `json:"count"`
}

// JudgeCount represents judge statistics
type JudgeCount struct {
	Judge string `json:"judge"`
	Count int64  `json:"count"`
}

// DateCount represents date-based statistics
type DateCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// HealthCheckResponse represents a health check response
type HealthCheckResponse struct {
	Status     string                      `json:"status"`
	Version    string                      `json:"version,omitempty"`
	Uptime     int64                       `json:"uptime_seconds"`
	Components map[string]*ComponentHealth `json:"components"`
	SystemInfo *SystemInfo                 `json:"system_info,omitempty"`
	CheckedAt  time.Time                   `json:"checked_at"`
}

// ComponentHealth represents the health of a system component
type ComponentHealth struct {
	Status       string            `json:"status"`
	Message      string            `json:"message,omitempty"`
	ResponseTime int64             `json:"response_time_ms,omitempty"`
	Details      map[string]string `json:"details,omitempty"`
	LastChecked  time.Time         `json:"last_checked"`
}

// SystemInfo represents system information
type SystemInfo struct {
	OS           string      `json:"os"`
	Architecture string      `json:"architecture"`
	GoVersion    string      `json:"go_version"`
	NumCPU       int         `json:"num_cpu"`
	Goroutines   int         `json:"goroutines"`
	Memory       *MemoryInfo `json:"memory"`
}

// MemoryUsage represents memory usage statistics
type MemoryUsage struct {
	Allocated   uint64 `json:"allocated_bytes"`
	TotalAlloc  uint64 `json:"total_alloc_bytes"`
	SystemMem   uint64 `json:"system_bytes"`
	GCCycles    uint32 `json:"gc_cycles"`
	HeapObjects uint64 `json:"heap_objects"`
}

// UploadResponse represents file upload response
type UploadResponse struct {
	FileName    string    `json:"file_name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	URL         string    `json:"url"`
	CDN_URL     string    `json:"cdn_url,omitempty"`
	StoragePath string    `json:"storage_path"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// AnalyzeRedactionsResponse represents redaction analysis response
type AnalyzeRedactionsResponse struct {
	FileName        string           `json:"file_name"`
	TotalPages      int              `json:"total_pages"`
	RedactionsFound int              `json:"redactions_found"`
	Redactions      []*RedactionInfo `json:"redactions"`
	Confidence      float64          `json:"confidence"`
	Recommendations []string         `json:"recommendations,omitempty"`
	AnalyzedAt      time.Time        `json:"analyzed_at"`
}

// RedactionInfo represents information about a redaction
type RedactionInfo struct {
	Page       int     `json:"page"`
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
	Area       *Area   `json:"area,omitempty"`
	Text       string  `json:"text,omitempty"`
}

// Area represents a rectangular area on a page
type Area struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// PipelineStatusResponse represents pipeline status
type PipelineStatusResponse struct {
	Status          string                      `json:"status"`
	ActiveJobs      int64                       `json:"active_jobs"`
	QueuedJobs      int64                       `json:"queued_jobs"`
	CompletedJobs   int64                       `json:"completed_jobs"`
	FailedJobs      int64                       `json:"failed_jobs"`
	WorkerCount     int                         `json:"worker_count"`
	ProcessorStatus map[string]*ProcessorStatus `json:"processor_status"`
	Uptime          int64                       `json:"uptime_seconds"`
	LastUpdate      time.Time                   `json:"last_update"`
}

// ProcessorStatus represents the status of a processor
type ProcessorStatus struct {
	Type              string    `json:"type"`
	Status            string    `json:"status"`
	ProcessedCount    int64     `json:"processed_count"`
	ErrorCount        int64     `json:"error_count"`
	AvgProcessingTime int64     `json:"avg_processing_time_ms"`
	LastProcessed     time.Time `json:"last_processed,omitempty"`
}


// HealthResponse represents a basic health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Service   string    `json:"service"`
}

// SystemStatus represents comprehensive system status
type SystemStatus struct {
	Service   string           `json:"service"`
	Version   string           `json:"version"`
	Status    string           `json:"status"`
	Timestamp time.Time        `json:"timestamp"`
	Uptime    time.Duration    `json:"uptime"`
	System    *SystemInfo      `json:"system"`
	Storage   *ComponentStatus `json:"storage"`
	Indexer   *ComponentStatus `json:"indexer"`
}

// ComponentStatus represents the status of a system component
type ComponentStatus struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
	LastError time.Time `json:"last_error,omitempty"`
}

// ReadinessResponse represents readiness check response
type ReadinessResponse struct {
	Ready     bool            `json:"ready"`
	Timestamp time.Time       `json:"timestamp"`
	Checks    map[string]bool `json:"checks"`
}

// LivenessResponse represents liveness check response
type LivenessResponse struct {
	Alive     bool      `json:"alive"`
	Timestamp time.Time `json:"timestamp"`
	PID       int       `json:"pid"`
}

// MetricsResponse represents application metrics
type MetricsResponse struct {
	Timestamp  time.Time              `json:"timestamp"`
	Memory     *MemoryInfo            `json:"memory"`
	Goroutines int                    `json:"goroutines"`
	GC         *GCStats               `json:"gc"`
	Storage    map[string]interface{} `json:"storage"`
	Indexer    map[string]interface{} `json:"indexer"`
}

// MemoryInfo represents memory statistics
type MemoryInfo struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
}

// GCStats represents garbage collection statistics
type GCStats struct {
	NumGC      uint32        `json:"num_gc"`
	PauseTotal time.Duration `json:"pause_total"`
	LastGC     time.Time     `json:"last_gc"`
	NextGC     uint64        `json:"next_gc"`
}

// RedactionAnalysisResult represents the result of redaction analysis
type RedactionAnalysisResult struct {
	DocumentID       string            `json:"document_id"`
	FileName         string            `json:"file_name"`
	RedactionsFound  int               `json:"redactions_found"`
	RedactionRegions []RedactionRegion `json:"redaction_regions"`
	AnalyzedAt       time.Time         `json:"analyzed_at"`
	Status           string            `json:"status"`
}

// RedactionRegion represents a redacted region in a document
type RedactionRegion struct {
	Page   int     `json:"page"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Type   string  `json:"type"`
}

// UpdateMetadataResponse represents the response from updating metadata
type UpdateMetadataResponse struct {
	DocumentID string    `json:"document_id"`
	UpdatedAt  time.Time `json:"updated_at"`
	Status     string    `json:"status"`
}

// RedactDocumentResponse represents the response from creating a redacted document
type RedactDocumentResponse struct {
	Success          bool            `json:"success"`
	DocumentID       string          `json:"document_id,omitempty"`
	RedactedURL      *string         `json:"redacted_url,omitempty"`
	PDFBase64        string          `json:"pdf_base64,omitempty"`
	Filename         string          `json:"filename,omitempty"`
	Redactions       []RedactionItem `json:"redactions"`
	TotalRedactions  int             `json:"total_redactions"`
	Message          string          `json:"message"`
}

// Re-export helper functions from pkg/models for convenience
var (
	NewSuccessResponse          = models.NewSuccessResponse
	NewErrorResponse            = models.NewErrorResponse
	NewValidationErrorResponse  = models.NewValidationErrorResponse
)
