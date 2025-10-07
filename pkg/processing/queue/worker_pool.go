package queue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// workerPool implements the WorkerPool interface
type workerPool struct {
	name           string
	workerCount    int
	workers        []*worker
	jobQueue       chan *QueueItem
	resultQueue    chan *ProcessingResult
	processor      ProcessorFunc
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	running        bool
	mutex          sync.RWMutex
	
	// Statistics
	processedJobs  int64
	failedJobs     int64
	totalJobTime   int64
	lastJobTime    time.Time
	
	// Configuration
	config *WorkerPoolConfig
}

// worker represents a single worker in the pool
type worker struct {
	id         int
	workerPool *workerPool
	active     bool
	mutex      sync.RWMutex
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(config *WorkerPoolConfig, processor ProcessorFunc) (WorkerPool, error) {
	if config.WorkerCount <= 0 {
		return nil, fmt.Errorf("worker count must be positive")
	}
	
	if processor == nil {
		return nil, fmt.Errorf("processor function is required")
	}
	
	queueSize := config.QueueSize
	if queueSize <= 0 {
		queueSize = config.WorkerCount * 2 // Default queue size
	}
	
	wp := &workerPool{
		name:        config.Name,
		workerCount: config.WorkerCount,
		workers:     make([]*worker, config.WorkerCount),
		jobQueue:    make(chan *QueueItem, queueSize),
		resultQueue: make(chan *ProcessingResult, queueSize),
		processor:   processor,
		config:      config,
	}
	
	// Create workers
	for i := 0; i < config.WorkerCount; i++ {
		wp.workers[i] = &worker{
			id:         i,
			workerPool: wp,
		}
	}
	
	return wp, nil
}

// Start starts the worker pool
func (wp *workerPool) Start(ctx context.Context) error {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	
	if wp.running {
		return fmt.Errorf("worker pool %s is already running", wp.name)
	}
	
	wp.ctx, wp.cancel = context.WithCancel(ctx)
	
	// Start workers
	for _, worker := range wp.workers {
		wp.wg.Add(1)
		go worker.start()
	}
	
	// Start result processor
	wp.wg.Add(1)
	go wp.processResults()
	
	wp.running = true
	return nil
}

// Stop stops the worker pool gracefully
func (wp *workerPool) Stop(ctx context.Context) error {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	
	if !wp.running {
		return nil
	}
	
	// Cancel context to signal workers to stop
	wp.cancel()
	
	// Close job queue to prevent new submissions
	close(wp.jobQueue)
	
	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// All workers finished gracefully
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for workers to stop")
	}
	
	wp.running = false
	return nil
}

// Submit submits work to the pool
func (wp *workerPool) Submit(ctx context.Context, item *QueueItem) error {
	wp.mutex.RLock()
	defer wp.mutex.RUnlock()
	
	if !wp.running {
		return fmt.Errorf("worker pool %s is not running", wp.name)
	}
	
	select {
	case wp.jobQueue <- item:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout submitting job to worker pool %s", wp.name)
	}
}

// GetStats returns worker pool statistics
func (wp *workerPool) GetStats() *WorkerPoolStats {
	wp.mutex.RLock()
	defer wp.mutex.RUnlock()
	
	activeWorkers := 0
	for _, worker := range wp.workers {
		if worker.isActive() {
			activeWorkers++
		}
	}
	
	processedJobs := atomic.LoadInt64(&wp.processedJobs)
	failedJobs := atomic.LoadInt64(&wp.failedJobs)
	totalJobTime := atomic.LoadInt64(&wp.totalJobTime)
	
	var avgJobTime time.Duration
	if processedJobs > 0 {
		avgJobTime = time.Duration(totalJobTime / processedJobs)
	}
	
	return &WorkerPoolStats{
		WorkerCount:    wp.workerCount,
		ActiveWorkers:  activeWorkers,
		IdleWorkers:    wp.workerCount - activeWorkers,
		QueueSize:      len(wp.jobQueue),
		ProcessedJobs:  processedJobs,
		FailedJobs:     failedJobs,
		AverageJobTime: avgJobTime,
		LastJobTime:    wp.lastJobTime,
	}
}

// IsHealthy returns true if the worker pool is healthy
func (wp *workerPool) IsHealthy() bool {
	wp.mutex.RLock()
	defer wp.mutex.RUnlock()
	
	if !wp.running {
		return false
	}
	
	// Check if at least half the workers are available
	availableWorkers := 0
	for _, worker := range wp.workers {
		if !worker.isActive() {
			availableWorkers++
		}
	}
	
	return availableWorkers >= wp.workerCount/2
}

// start starts a worker
func (w *worker) start() {
	defer w.workerPool.wg.Done()
	
	for {
		select {
		case <-w.workerPool.ctx.Done():
			return
		case item, ok := <-w.workerPool.jobQueue:
			if !ok {
				return // Channel closed
			}
			
			w.processItem(item)
		}
	}
}

// processItem processes a single item
func (w *worker) processItem(item *QueueItem) {
	w.setActive(true)
	defer w.setActive(false)
	
	startTime := time.Now()
	
	// Create context with timeout
	ctx := w.workerPool.ctx
	if w.workerPool.config.ProcessTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, w.workerPool.config.ProcessTimeout)
		defer cancel()
	}
	
	// Process the item
	result := w.workerPool.processor(ctx, item)
	result.Duration = time.Since(startTime)
	
	// Update statistics
	atomic.AddInt64(&w.workerPool.totalJobTime, int64(result.Duration))
	w.workerPool.lastJobTime = time.Now()
	
	if result.Success {
		atomic.AddInt64(&w.workerPool.processedJobs, 1)
	} else {
		atomic.AddInt64(&w.workerPool.failedJobs, 1)
	}
	
	// Send result
	select {
	case w.workerPool.resultQueue <- result:
	default:
		// Result queue full, drop result
		fmt.Printf("Warning: result queue full for worker pool %s\n", w.workerPool.name)
	}
}

// setActive sets the worker's active status
func (w *worker) setActive(active bool) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.active = active
}

// isActive returns the worker's active status
func (w *worker) isActive() bool {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.active
}

// processResults processes results from workers
func (wp *workerPool) processResults() {
	defer wp.wg.Done()
	
	for {
		select {
		case <-wp.ctx.Done():
			return
		case result, ok := <-wp.resultQueue:
			if !ok {
				return // Channel closed
			}
			
			// Process result (log, metrics, etc.)
			wp.handleResult(result)
		}
	}
}

// handleResult handles a processing result
func (wp *workerPool) handleResult(result *ProcessingResult) {
	if wp.config.EnableMetrics {
		// Log slow jobs
		if result.Duration > 30*time.Second {
			fmt.Printf("Slow job detected in pool %s: %v\n", wp.name, result.Duration)
		}
		
		// Log errors
		if !result.Success && result.Error != nil {
			fmt.Printf("Job failed in pool %s: %v\n", wp.name, result.Error)
		}
	}
}

// GetWorkerStatus returns the status of individual workers
func (wp *workerPool) GetWorkerStatus() []map[string]interface{} {
	wp.mutex.RLock()
	defer wp.mutex.RUnlock()
	
	status := make([]map[string]interface{}, len(wp.workers))
	for i, worker := range wp.workers {
		status[i] = map[string]interface{}{
			"id":     worker.id,
			"active": worker.isActive(),
		}
	}
	
	return status
}

// Resize changes the number of workers in the pool
func (wp *workerPool) Resize(newWorkerCount int) error {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	
	if !wp.running {
		return fmt.Errorf("cannot resize stopped worker pool")
	}
	
	if newWorkerCount <= 0 {
		return fmt.Errorf("worker count must be positive")
	}
	
	currentCount := len(wp.workers)
	
	if newWorkerCount > currentCount {
		// Add workers
		for i := currentCount; i < newWorkerCount; i++ {
			worker := &worker{
				id:         i,
				workerPool: wp,
			}
			wp.workers = append(wp.workers, worker)
			wp.wg.Add(1)
			go worker.start()
		}
	} else if newWorkerCount < currentCount {
		// Remove workers (this is more complex in practice)
		// For simplicity, we'll just update the count
		// In a real implementation, you'd need to gracefully stop excess workers
		wp.workers = wp.workers[:newWorkerCount]
	}
	
	wp.workerCount = newWorkerCount
	return nil
}

// Pause pauses the worker pool (stops accepting new jobs)
func (wp *workerPool) Pause() error {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	
	if !wp.running {
		return fmt.Errorf("worker pool %s is not running", wp.name)
	}
	
	// Implementation would involve stopping job acceptance
	// while allowing current jobs to complete
	
	return nil
}

// Resume resumes the worker pool
func (wp *workerPool) Resume() error {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	
	// Implementation would involve resuming job acceptance
	
	return nil
}

// GetQueueCapacity returns the current queue capacity
func (wp *workerPool) GetQueueCapacity() (current, max int) {
	wp.mutex.RLock()
	defer wp.mutex.RUnlock()
	
	return len(wp.jobQueue), cap(wp.jobQueue)
}