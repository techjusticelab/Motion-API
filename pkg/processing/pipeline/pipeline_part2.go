package pipeline

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

func (p *pipeline) GetStatus() *PipelineStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Get processor statuses
	processorStatuses := make([]*ProcessorStatus, 0, len(p.processors))
	for procType, processor := range p.processors {
		status := &ProcessorStatus{
			Type:    procType,
			Healthy: processor.IsHealthy(),
		}
		if !status.Healthy {
			status.Error = "processor unhealthy"
		}
		processorStatuses = append(processorStatuses, status)
	}

	// Get worker pool stats
	var poolStats *PoolStats
	if p.workerPool != nil {
		poolStats = p.workerPool.GetStats()
	}

	return &PipelineStatus{
		Running:         p.running,
		ActiveJobs:      0, // Would be tracked by worker pool
		QueuedJobs:      0, // Would be tracked by worker pool
		CompletedJobs:   atomic.LoadInt64(&p.completedJobs),
		FailedJobs:      atomic.LoadInt64(&p.failedJobs),
		ProcessorStatus: processorStatuses,
		WorkerPoolStats: poolStats,
		LastUpdate:      time.Now(),
	}
}

func (p *pipeline) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	// Stop worker pool
	if p.workerPool != nil {
		if err := p.workerPool.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop worker pool: %w", err)
		}
	}

	p.running = false
	return nil
}

func (p *pipeline) IsHealthy() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check if all processors are healthy
	for _, processor := range p.processors {
		if !processor.IsHealthy() {
			return false
		}
	}

	return true
}

func (p *pipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}

	// Start worker pool
	if p.workerPool != nil {
		if err := p.workerPool.Start(ctx); err != nil {
			return fmt.Errorf("failed to start worker pool: %w", err)
		}
	}

	p.running = true
	return nil
}

func DefaultConfig() *Config {
	return &Config{
		MaxWorkers:     10,
		QueueSize:      100,
		ProcessTimeout: 5 * time.Minute,
		RetryAttempts:  3,
		RetryDelay:     time.Second,
		EnableMetrics:  true,
		MaxPDFPages:    20, // Reduced from 25 to 20 for memory efficiency
	}
}
