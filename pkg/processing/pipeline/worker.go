package pipeline

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool implements the WorkerPool interface
type WorkerPool struct {
	maxWorkers  int
	queueSize   int
	jobs        chan Job
	workers     []*worker
	stats       *poolStats
	running     bool
	mu          sync.RWMutex
	wg          sync.WaitGroup
}

// Alias for backward compatibility
type workerPool = WorkerPool

// poolStats tracks worker pool statistics
type poolStats struct {
	processedJobs int64
	failedJobs    int64
	totalLatency  int64
	activeWorkers int32
}

// worker represents a single worker
type worker struct {
	id       int
	jobQueue chan Job
	quit     chan bool
	stats    *poolStats
}

// processingJob implements the Job interface
type processingJob struct {
	id          string
	priority    int
	timeout     time.Duration
	executeFunc func(ctx context.Context) error
}

// WorkerJob represents a job for worker processing (for tests compatibility)
type WorkerJob struct {
	ID         string          `json:"id"`
	Request    *ProcessRequest `json:"request,omitempty"`
	Result     *ProcessResult  `json:"result,omitempty"`
	Error      error           `json:"error,omitempty"`
	Priority   int             `json:"priority"`
	Timeout    time.Duration   `json:"timeout"`
	executeFunc func(ctx context.Context) error
}

// Execute implements the Job interface
func (w *WorkerJob) Execute(ctx context.Context) error {
	if w.executeFunc != nil {
		return w.executeFunc(ctx)
	}
	// Default implementation - just return nil
	return nil
}

// GetID implements the Job interface
func (w *WorkerJob) GetID() string {
	return w.ID
}

// GetPriority implements the Job interface
func (w *WorkerJob) GetPriority() int {
	return w.Priority
}

// GetTimeout implements the Job interface
func (w *WorkerJob) GetTimeout() time.Duration {
	return w.Timeout
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workerCount, queueSize int) *WorkerPool {
	return &WorkerPool{
		maxWorkers: workerCount,
		queueSize:  queueSize,
		jobs:       make(chan Job, queueSize),
		workers:    make([]*worker, workerCount),
		stats: &poolStats{
			processedJobs: 0,
			failedJobs:    0,
			totalLatency:  0,
			activeWorkers: 0,
		},
		running: false,
	}
}

// Submit submits a job to the worker pool
func (p *WorkerPool) Submit(job Job) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.running {
		return NewPipelineError("pool_not_running", "worker pool is not running", "", nil)
	}

	select {
	case p.jobs <- job:
		return nil
	default:
		return NewPipelineError("queue_full", "job queue is full", "", nil)
	}
}

// Start starts the worker pool
func (p *WorkerPool) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}

	// Start workers
	for i := 0; i < p.maxWorkers; i++ {
		worker := &worker{
			id:       i,
			jobQueue: p.jobs,
			quit:     make(chan bool),
			stats:    p.stats,
		}

		p.workers[i] = worker
		p.wg.Add(1)
		go worker.start(ctx, &p.wg)
	}

	p.running = true
	return nil
}

// Stop stops the worker pool
func (p *WorkerPool) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	// Signal all workers to quit
	for _, worker := range p.workers {
		close(worker.quit)
	}

	// Wait for all workers to finish
	done := make(chan bool)
	go func() {
		p.wg.Wait()
		done <- true
	}()

	// Wait for shutdown or timeout
	select {
	case <-done:
		// All workers stopped successfully
	case <-ctx.Done():
		// Timeout reached
		return ctx.Err()
	}

	// Close job queue
	close(p.jobs)

	p.running = false
	return nil
}

// GetStats returns worker pool statistics
func (p *WorkerPool) GetStats() *PoolStats {
	processedJobs := atomic.LoadInt64(&p.stats.processedJobs)
	failedJobs := atomic.LoadInt64(&p.stats.failedJobs)
	totalLatency := atomic.LoadInt64(&p.stats.totalLatency)
	activeWorkers := atomic.LoadInt32(&p.stats.activeWorkers)

	var averageLatency int64
	if processedJobs > 0 {
		averageLatency = totalLatency / processedJobs
	}

	return &PoolStats{
		WorkerCount:    p.maxWorkers,
		ActiveWorkers:  int(activeWorkers),
		QueueSize:      len(p.jobs),
		ProcessedJobs:  processedJobs,
		FailedJobs:     failedJobs,
		AverageLatency: averageLatency,
	}
}

// IsRunning returns true if the worker pool is running
func (p *WorkerPool) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// start starts the worker
func (w *worker) start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case job := <-w.jobQueue:
			if job != nil {
				w.processJob(ctx, job)
			}
		case <-w.quit:
			return
		case <-ctx.Done():
			return
		}
	}
}

// processJob processes a single job
func (w *worker) processJob(ctx context.Context, job Job) {
	startTime := time.Now()

	// Mark worker as active
	atomic.AddInt32(&w.stats.activeWorkers, 1)
	defer atomic.AddInt32(&w.stats.activeWorkers, -1)

	// Create job context with timeout
	jobCtx := ctx
	if timeout := job.GetTimeout(); timeout > 0 {
		var cancel context.CancelFunc
		jobCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Execute job
	err := job.Execute(jobCtx)

	// Update statistics
	latency := time.Since(startTime).Milliseconds()
	atomic.AddInt64(&w.stats.totalLatency, latency)

	if err != nil {
		atomic.AddInt64(&w.stats.failedJobs, 1)
	} else {
		atomic.AddInt64(&w.stats.processedJobs, 1)
	}
}

// NewProcessingJob creates a new processing job
func NewProcessingJob(id string, priority int, timeout time.Duration, executeFunc func(ctx context.Context) error) Job {
	return &processingJob{
		id:          id,
		priority:    priority,
		timeout:     timeout,
		executeFunc: executeFunc,
	}
}

// Execute executes the job
func (j *processingJob) Execute(ctx context.Context) error {
	if j.executeFunc == nil {
		return NewPipelineError("no_execute_func", "job has no execute function", "", nil)
	}
	return j.executeFunc(ctx)
}

// GetID returns the job ID
func (j *processingJob) GetID() string {
	return j.id
}

// GetPriority returns the job priority
func (j *processingJob) GetPriority() int {
	return j.priority
}

// GetTimeout returns the job timeout
func (j *processingJob) GetTimeout() time.Duration {
	return j.timeout
}
