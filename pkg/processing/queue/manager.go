package queue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// queueManager implements the QueueManager interface
type queueManager struct {
	queues      map[string]Queue
	processors  map[string]ProcessorFunc
	workerPools map[string]WorkerPool
	mutex       sync.RWMutex
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewQueueManager creates a new queue manager
func NewQueueManager() QueueManager {
	return &queueManager{
		queues:      make(map[string]Queue),
		processors:  make(map[string]ProcessorFunc),
		workerPools: make(map[string]WorkerPool),
	}
}

// CreateQueue creates a new queue with the given configuration
func (qm *queueManager) CreateQueue(config *QueueConfig, processor ProcessorFunc) (Queue, error) {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()
	
	if _, exists := qm.queues[config.Name]; exists {
		return nil, fmt.Errorf("queue %s already exists", config.Name)
	}
	
	// Create the priority queue
	queue := NewPriorityQueue(config)
	
	// Create worker pool for this queue
	var workerPool WorkerPool
	var err error
	
	if config.WorkerCount > 0 {
		workerPoolConfig := &WorkerPoolConfig{
			Name:           config.Name + "_workers",
			WorkerCount:    config.WorkerCount,
			QueueSize:      config.MaxSize,
			ProcessTimeout: config.ProcessTimeout,
			EnableMetrics:  config.EnableMetrics,
		}
		
		workerPool, err = NewWorkerPool(workerPoolConfig, processor)
		if err != nil {
			return nil, fmt.Errorf("failed to create worker pool for queue %s: %w", config.Name, err)
		}
	}
	
	// Store references
	qm.queues[config.Name] = queue
	qm.processors[config.Name] = processor
	if workerPool != nil {
		qm.workerPools[config.Name] = workerPool
	}
	
	return queue, nil
}

// GetQueue returns a queue by name
func (qm *queueManager) GetQueue(name string) (Queue, error) {
	qm.mutex.RLock()
	defer qm.mutex.RUnlock()
	
	queue, exists := qm.queues[name]
	if !exists {
		return nil, fmt.Errorf("queue %s not found", name)
	}
	
	return queue, nil
}

// ListQueues returns all queue names
func (qm *queueManager) ListQueues() []string {
	qm.mutex.RLock()
	defer qm.mutex.RUnlock()
	
	names := make([]string, 0, len(qm.queues))
	for name := range qm.queues {
		names = append(names, name)
	}
	
	return names
}

// GetAllStats returns statistics for all queues
func (qm *queueManager) GetAllStats() map[string]*QueueStats {
	qm.mutex.RLock()
	defer qm.mutex.RUnlock()
	
	stats := make(map[string]*QueueStats)
	for name, queue := range qm.queues {
		stats[name] = queue.GetStats()
	}
	
	return stats
}

// Start starts all queues and their worker pools
func (qm *queueManager) Start(ctx context.Context) error {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()
	
	if qm.running {
		return fmt.Errorf("queue manager is already running")
	}
	
	qm.ctx, qm.cancel = context.WithCancel(ctx)
	
	// Start all worker pools
	for name, workerPool := range qm.workerPools {
		if err := workerPool.Start(qm.ctx); err != nil {
			// Stop already started pools
			for stopName, stopPool := range qm.workerPools {
				if stopName == name {
					break
				}
				stopPool.Stop(context.Background())
			}
			return fmt.Errorf("failed to start worker pool %s: %w", name, err)
		}
	}
	
	// Start queue processing loops
	for name, queue := range qm.queues {
		qm.wg.Add(1)
		go qm.processQueue(name, queue)
	}
	
	qm.running = true
	return nil
}

// Stop stops all queues gracefully
func (qm *queueManager) Stop(ctx context.Context) error {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()
	
	if !qm.running {
		return nil
	}
	
	// Cancel context to stop processing
	qm.cancel()
	
	// Wait for processing goroutines to finish
	done := make(chan struct{})
	go func() {
		qm.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// All goroutines finished
	case <-ctx.Done():
		return ctx.Err()
	}
	
	// Stop worker pools
	for _, workerPool := range qm.workerPools {
		if err := workerPool.Stop(ctx); err != nil {
			// Log error but continue stopping others
			fmt.Printf("Error stopping worker pool: %v\n", err)
		}
	}
	
	// Close queues
	for _, queue := range qm.queues {
		if err := queue.Close(); err != nil {
			// Log error but continue closing others
			fmt.Printf("Error closing queue: %v\n", err)
		}
	}
	
	qm.running = false
	return nil
}

// IsHealthy returns true if all queues are healthy
func (qm *queueManager) IsHealthy() bool {
	qm.mutex.RLock()
	defer qm.mutex.RUnlock()
	
	if !qm.running {
		return false
	}
	
	// Check worker pools
	for _, workerPool := range qm.workerPools {
		if !workerPool.IsHealthy() {
			return false
		}
	}
	
	return true
}

// processQueue processes items from a queue using its worker pool
func (qm *queueManager) processQueue(name string, queue Queue) {
	defer qm.wg.Done()
	
	workerPool, hasWorkerPool := qm.workerPools[name]
	processor, hasProcessor := qm.processors[name]
	
	for {
		select {
		case <-qm.ctx.Done():
			return
		default:
			// Try to get an item from the queue
			item, err := queue.Dequeue(qm.ctx)
			if err != nil {
				if qm.ctx.Err() != nil {
					return // Context cancelled
				}
				// Log error and continue
				time.Sleep(100 * time.Millisecond)
				continue
			}
			
			// Process the item
			if hasWorkerPool {
				// Submit to worker pool
				if err := workerPool.Submit(qm.ctx, item); err != nil {
					// Re-queue for retry or mark as failed
					qm.handleProcessingError(queue, item, err)
				}
			} else if hasProcessor {
				// Process directly
				go qm.processItemDirectly(queue, item, processor)
			} else {
				// No processor configured, mark as failed
				fmt.Printf("No processor configured for queue %s\n", name)
			}
		}
	}
}

// processItemDirectly processes an item directly without worker pool
func (qm *queueManager) processItemDirectly(queue Queue, item *QueueItem, processor ProcessorFunc) {
	startTime := time.Now()
	
	// Process the item
	result := processor(qm.ctx, item)
	
	if result.Success {
		if pq, ok := queue.(*priorityQueue); ok {
			pq.MarkCompleted(item)
		}
	} else {
		if result.ShouldRetry {
			if pq, ok := queue.(*priorityQueue); ok {
				if err := pq.RequeueForRetry(item); err != nil {
					// Failed to retry, mark as failed
					pq.MarkFailed(item)
				}
			}
		} else {
			if pq, ok := queue.(*priorityQueue); ok {
				pq.MarkFailed(item)
			}
		}
	}
	
	// Log processing time
	duration := time.Since(startTime)
	if duration > 10*time.Second {
		fmt.Printf("Slow processing detected: item %s took %v\n", item.ID, duration)
	}
}

// handleProcessingError handles errors during item processing
func (qm *queueManager) handleProcessingError(queue Queue, item *QueueItem, err error) {
	fmt.Printf("Processing error for item %s: %v\n", item.ID, err)
	
	if pq, ok := queue.(*priorityQueue); ok {
		if err := pq.RequeueForRetry(item); err != nil {
			// Failed to retry, mark as failed
			pq.MarkFailed(item)
		}
	}
}

// GetQueueMetrics returns detailed metrics for all queues
func (qm *queueManager) GetQueueMetrics() map[string]interface{} {
	qm.mutex.RLock()
	defer qm.mutex.RUnlock()
	
	metrics := make(map[string]interface{})
	
	// Queue statistics
	queueStats := make(map[string]*QueueStats)
	for name, queue := range qm.queues {
		queueStats[name] = queue.GetStats()
	}
	metrics["queues"] = queueStats
	
	// Worker pool statistics
	workerStats := make(map[string]*WorkerPoolStats)
	for name, workerPool := range qm.workerPools {
		workerStats[name] = workerPool.GetStats()
	}
	metrics["worker_pools"] = workerStats
	
	// Overall metrics
	totalQueueSize := 0
	totalCompleted := int64(0)
	totalFailed := int64(0)
	
	for _, stats := range queueStats {
		totalQueueSize += stats.Size
		totalCompleted += stats.CompletedItems
		totalFailed += stats.FailedItems
	}
	
	metrics["totals"] = map[string]interface{}{
		"queue_size":      totalQueueSize,
		"completed_items": totalCompleted,
		"failed_items":    totalFailed,
		"running":         qm.running,
		"queue_count":     len(qm.queues),
	}
	
	return metrics
}

// WorkerPoolConfig holds configuration for a worker pool
type WorkerPoolConfig struct {
	Name           string
	WorkerCount    int
	QueueSize      int
	ProcessTimeout time.Duration
	EnableMetrics  bool
}

// CreateStandardQueues creates a standard set of queues for document processing
func (qm *queueManager) CreateStandardQueues(performanceConfig interface{}) error {
	// This would create standard queues based on performance configuration
	// Implementation would depend on specific performance config structure
	
	// Example queue configurations
	configs := []*QueueConfig{
		{
			Name:            "download",
			Type:            QueueTypeDownload,
			MaxSize:         1000,
			WorkerCount:     10,
			ProcessTimeout:  60 * time.Second,
			RetryAttempts:   3,
			RetryDelay:      5 * time.Second,
			EnableMetrics:   true,
		},
		{
			Name:            "extraction", 
			Type:            QueueTypeExtraction,
			MaxSize:         500,
			WorkerCount:     8,
			ProcessTimeout:  120 * time.Second,
			RetryAttempts:   3,
			RetryDelay:      10 * time.Second,
			EnableMetrics:   true,
		},
		{
			Name:            "classification",
			Type:            QueueTypeClassification,
			MaxSize:         200,
			WorkerCount:     4,
			ProcessTimeout:  300 * time.Second,
			RetryAttempts:   5,
			RetryDelay:      30 * time.Second,
			EnableRateLimit: true,
			RateLimit:       50, // 50 requests per minute
			BurstSize:       10,
			EnableMetrics:   true,
		},
		{
			Name:            "indexing",
			Type:            QueueTypeIndexing,
			MaxSize:         300,
			WorkerCount:     6,
			ProcessTimeout:  30 * time.Second,
			RetryAttempts:   3,
			RetryDelay:      5 * time.Second,
			EnableMetrics:   true,
		},
	}
	
	// Create mock processors for now (would be replaced with real implementations)
	mockProcessor := func(ctx context.Context, item *QueueItem) *ProcessingResult {
		return &ProcessingResult{
			Success:  true,
			Duration: time.Millisecond * 100,
		}
	}
	
	for _, config := range configs {
		if _, err := qm.CreateQueue(config, mockProcessor); err != nil {
			return fmt.Errorf("failed to create queue %s: %w", config.Name, err)
		}
	}
	
	return nil
}