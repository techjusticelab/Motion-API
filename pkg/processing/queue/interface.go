package queue

import (
	"context"
	"time"
)

// Priority levels for different queue operations
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// QueueType represents different types of processing queues
type QueueType string

const (
	QueueTypeDownload       QueueType = "download"
	QueueTypeExtraction     QueueType = "extraction"
	QueueTypeClassification QueueType = "classification"
	QueueTypeIndexing       QueueType = "indexing"
)

// QueueItem represents an item in a processing queue
type QueueItem struct {
	ID          string                 `json:"id"`
	Type        QueueType              `json:"type"`
	Priority    Priority               `json:"priority"`
	Data        interface{}            `json:"data"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
	NextRetry   *time.Time             `json:"next_retry,omitempty"`
}

// ProcessingResult represents the result of processing a queue item
type ProcessingResult struct {
	Success     bool                   `json:"success"`
	Error       error                  `json:"error,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Output      interface{}            `json:"output,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ShouldRetry bool                   `json:"should_retry"`
}

// QueueStats provides statistics about a queue
type QueueStats struct {
	Name            string    `json:"name"`
	Type            QueueType `json:"type"`
	Size            int       `json:"size"`
	PendingItems    int       `json:"pending_items"`
	ProcessingItems int       `json:"processing_items"`
	CompletedItems  int64     `json:"completed_items"`
	FailedItems     int64     `json:"failed_items"`
	RetryItems      int       `json:"retry_items"`
	AverageWaitTime time.Duration `json:"average_wait_time"`
	LastUpdate      time.Time `json:"last_update"`
}

// Queue interface defines the contract for a priority queue
type Queue interface {
	// Enqueue adds an item to the queue
	Enqueue(ctx context.Context, item *QueueItem) error
	
	// Dequeue removes and returns the highest priority item
	Dequeue(ctx context.Context) (*QueueItem, error)
	
	// Peek returns the highest priority item without removing it
	Peek(ctx context.Context) (*QueueItem, error)
	
	// Size returns the current number of items in the queue
	Size() int
	
	// IsEmpty returns true if the queue is empty
	IsEmpty() bool
	
	// Clear removes all items from the queue
	Clear() error
	
	// GetStats returns queue statistics
	GetStats() *QueueStats
	
	// Close closes the queue and releases resources
	Close() error
}

// WorkerPool interface defines the contract for a worker pool
type WorkerPool interface {
	// Start starts the worker pool
	Start(ctx context.Context) error
	
	// Stop stops the worker pool gracefully
	Stop(ctx context.Context) error
	
	// Submit submits work to the pool
	Submit(ctx context.Context, item *QueueItem) error
	
	// GetStats returns worker pool statistics
	GetStats() *WorkerPoolStats
	
	// IsHealthy returns true if the worker pool is healthy
	IsHealthy() bool
}

// WorkerPoolStats provides statistics about a worker pool
type WorkerPoolStats struct {
	WorkerCount     int           `json:"worker_count"`
	ActiveWorkers   int           `json:"active_workers"`
	IdleWorkers     int           `json:"idle_workers"`
	QueueSize       int           `json:"queue_size"`
	ProcessedJobs   int64         `json:"processed_jobs"`
	FailedJobs      int64         `json:"failed_jobs"`
	AverageJobTime  time.Duration `json:"average_job_time"`
	LastJobTime     time.Time     `json:"last_job_time"`
}

// RateLimiter interface defines rate limiting functionality
type RateLimiter interface {
	// Allow returns true if an operation is allowed
	Allow() bool
	
	// AllowN returns true if n operations are allowed
	AllowN(n int) bool
	
	// Reserve reserves permission for an operation
	Reserve() Reservation
	
	// ReserveN reserves permission for n operations
	ReserveN(n int) Reservation
	
	// Wait waits until permission is granted
	Wait(ctx context.Context) error
	
	// WaitN waits until permission for n operations is granted
	WaitN(ctx context.Context, n int) error
}

// Reservation represents a reserved permission to act
type Reservation interface {
	// OK returns true if the reservation is valid
	OK() bool
	
	// Delay returns the duration to wait before acting
	Delay() time.Duration
	
	// Cancel cancels the reservation
	Cancel()
}

// ProcessorFunc is a function that processes queue items
type ProcessorFunc func(ctx context.Context, item *QueueItem) *ProcessingResult

// QueueConfig holds configuration for a queue
type QueueConfig struct {
	Name              string        `json:"name"`
	Type              QueueType     `json:"type"`
	MaxSize           int           `json:"max_size"`
	WorkerCount       int           `json:"worker_count"`
	ProcessTimeout    time.Duration `json:"process_timeout"`
	RetryAttempts     int           `json:"retry_attempts"`
	RetryDelay        time.Duration `json:"retry_delay"`
	RetryBackoff      bool          `json:"retry_backoff"`
	EnableRateLimit   bool          `json:"enable_rate_limit"`
	RateLimit         int           `json:"rate_limit"`         // requests per minute
	BurstSize         int           `json:"burst_size"`
	MemoryLimitMB     int           `json:"memory_limit_mb"`
	EnableMetrics     bool          `json:"enable_metrics"`
	MetricsInterval   time.Duration `json:"metrics_interval"`
}

// QueueManager interface manages multiple queues
type QueueManager interface {
	// CreateQueue creates a new queue with the given configuration
	CreateQueue(config *QueueConfig, processor ProcessorFunc) (Queue, error)
	
	// GetQueue returns a queue by name
	GetQueue(name string) (Queue, error)
	
	// ListQueues returns all queue names
	ListQueues() []string
	
	// GetAllStats returns statistics for all queues
	GetAllStats() map[string]*QueueStats
	
	// Start starts all queues
	Start(ctx context.Context) error
	
	// Stop stops all queues gracefully
	Stop(ctx context.Context) error
	
	// IsHealthy returns true if all queues are healthy
	IsHealthy() bool
}

// BackpressureStrategy defines how to handle queue overflow
type BackpressureStrategy string

const (
	BackpressureBlock    BackpressureStrategy = "block"    // Block until space is available
	BackpressureDrop     BackpressureStrategy = "drop"     // Drop the item
	BackpressureReject   BackpressureStrategy = "reject"   // Reject with error
	BackpressureDegrade  BackpressureStrategy = "degrade"  // Lower priority items
)

// FlowControl manages backpressure and flow control
type FlowControl interface {
	// CheckCapacity returns true if there's capacity for more items
	CheckCapacity(queueType QueueType) bool
	
	// ApplyBackpressure applies backpressure strategy
	ApplyBackpressure(ctx context.Context, item *QueueItem, strategy BackpressureStrategy) error
	
	// GetSystemLoad returns current system load metrics
	GetSystemLoad() *SystemLoad
	
	// ShouldThrottle returns true if processing should be throttled
	ShouldThrottle() bool
}

// SystemLoad represents current system resource usage
type SystemLoad struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	QueueUtilization   float64 `json:"queue_utilization"`
	ActiveConnections  int     `json:"active_connections"`
	Timestamp          time.Time `json:"timestamp"`
}