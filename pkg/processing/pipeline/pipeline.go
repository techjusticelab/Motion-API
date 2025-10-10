package pipeline

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"motion-index-fiber/pkg/processing/classifier"
	"motion-index-fiber/pkg/processing/extractor"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/storage"
)

// pipeline implements the Pipeline interface
type pipeline struct {
	extractorService  extractor.Service
	classifierService classifier.Service
	searchService     search.Service
	storageService    storage.Service
	workerPool        *WorkerPool
	processors        map[ProcessorType]Processor

	// Statistics
	completedJobs int64
	failedJobs    int64
	running       bool
	mu            sync.RWMutex

	config *Config
}

// Config holds pipeline configuration
type Config struct {
	MaxWorkers     int           `json:"max_workers"`
	QueueSize      int           `json:"queue_size"`
	ProcessTimeout time.Duration `json:"process_timeout"`
	RetryAttempts  int           `json:"retry_attempts"`
	RetryDelay     time.Duration `json:"retry_delay"`
	EnableMetrics  bool          `json:"enable_metrics"`
	MaxPDFPages    int           `json:"max_pdf_pages"`
}

// NewPipeline creates a new document processing pipeline
func NewPipeline(
	extractorSvc extractor.Service,
	classifierSvc classifier.Service,
	searchSvc search.Service,
	storageSvc storage.Service,
	config *Config,
) (Pipeline, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create worker pool
	workerPool := NewWorkerPool(config.MaxWorkers, config.QueueSize)

	// Create processors
	processors := make(map[ProcessorType]Processor)
	processors[ProcessorTypeExtraction] = NewExtractionProcessor(extractorSvc, config)
	processors[ProcessorTypeClassification] = NewClassificationProcessor(classifierSvc)
	processors[ProcessorTypeIndexing] = NewIndexingProcessor(searchSvc)
	processors[ProcessorTypeStorage] = NewStorageProcessor(storageSvc)

	return &pipeline{
		extractorService:  extractorSvc,
		classifierService: classifierSvc,
		searchService:     searchSvc,
		storageService:    storageSvc,
		workerPool:        workerPool,
		processors:        processors,
		config:            config,
		running:           false,
	}, nil
}

// ProcessDocument processes a single document through the complete pipeline
func (p *pipeline) ProcessDocument(ctx context.Context, req *ProcessRequest) (*ProcessResult, error) {
	startTime := time.Now()

	result := &ProcessResult{
		ID:        req.ID,
		StartTime: startTime,
		Steps:     []*ProcessStep{},
	}

	// Set default options if not provided
	if req.Options == nil {
		req.Options = DefaultProcessOptions()
	}

	// Create processing context with timeout
	processCtx := ctx
	if req.Options.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		processCtx, cancel = context.WithTimeout(ctx, time.Duration(req.Options.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	// Execute processing steps
	if err := p.executeProcessingSteps(processCtx, req, result); err != nil {
		result.Success = false
		result.Error = err.Error()
		result.EndTime = time.Now()
		result.ProcessingTime = time.Since(startTime).Milliseconds()
		atomic.AddInt64(&p.failedJobs, 1)
		return result, err
	}

	result.Success = true
	result.EndTime = time.Now()
	result.ProcessingTime = time.Since(startTime).Milliseconds()
	atomic.AddInt64(&p.completedJobs, 1)

	return result, nil
}

// ProcessBatch processes multiple documents concurrently
func (p *pipeline) ProcessBatch(ctx context.Context, requests []*ProcessRequest) (*BatchResult, error) {
	startTime := time.Now()

	batchResult := &BatchResult{
		TotalCount: len(requests),
		Results:    make([]*ProcessResult, len(requests)),
		StartTime:  startTime,
	}

	// Process documents concurrently
	var wg sync.WaitGroup
	resultsChan := make(chan struct {
		index  int
		result *ProcessResult
		err    error
	}, len(requests))

	for i, req := range requests {
		wg.Add(1)
		go func(index int, request *ProcessRequest) {
			defer wg.Done()
			result, err := p.ProcessDocument(ctx, request)
			resultsChan <- struct {
				index  int
				result *ProcessResult
				err    error
			}{index, result, err}
		}(i, req)
	}

	// Wait for all processing to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for item := range resultsChan {
		batchResult.Results[item.index] = item.result
		if item.result.Success {
			batchResult.SuccessCount++
		} else {
			batchResult.FailureCount++
		}
	}

	batchResult.EndTime = time.Now()
	batchResult.ProcessingTime = time.Since(startTime).Milliseconds()

	return batchResult, nil
}

// executeProcessingSteps executes the complete processing pipeline
func (p *pipeline) executeProcessingSteps(ctx context.Context, req *ProcessRequest, result *ProcessResult) error {
	// Initialize request metadata if not present
	if req.Metadata == nil {
		req.Metadata = make(map[string]string)
	}

	// Step 1: Text Extraction
	if req.Options.ExtractText {
		if err := p.executeStep(ctx, ProcessorTypeExtraction, req, result); err != nil {
			return NewPipelineError("extraction_failed", "text extraction failed", ProcessorTypeExtraction, err)
		}

		// Pass extraction results to subsequent steps
		if result.ExtractionResult != nil {
			req.Metadata["extracted_text"] = result.ExtractionResult.Text
			req.Metadata["word_count"] = fmt.Sprintf("%d", result.ExtractionResult.WordCount)
			req.Metadata["page_count"] = fmt.Sprintf("%d", result.ExtractionResult.PageCount)
			req.Metadata["char_count"] = fmt.Sprintf("%d", result.ExtractionResult.CharCount)
			if result.ExtractionResult.Language != "" {
				req.Metadata["language"] = result.ExtractionResult.Language
			}
		}
	}

	// Step 2: Document Classification (requires extracted text)
	if req.Options.ClassifyDoc {
		if err := p.executeStep(ctx, ProcessorTypeClassification, req, result); err != nil {
			return NewPipelineError("classification_failed", "document classification failed", ProcessorTypeClassification, err)
		}

		// Pass classification results to subsequent steps
		if result.ClassificationResult != nil {
			req.Metadata["document_type"] = result.ClassificationResult.DocumentType
			req.Metadata["legal_category"] = result.ClassificationResult.LegalCategory
			req.Metadata["confidence"] = fmt.Sprintf("%.2f", result.ClassificationResult.Confidence)
			if result.ClassificationResult.SubCategory != "" {
				req.Metadata["sub_category"] = result.ClassificationResult.SubCategory
			}
			if result.ClassificationResult.Summary != "" {
				req.Metadata["summary"] = result.ClassificationResult.Summary
			}
			if result.ClassificationResult.Subject != "" {
				req.Metadata["subject"] = result.ClassificationResult.Subject
			}
			if result.ClassificationResult.Status != "" {
				req.Metadata["status"] = result.ClassificationResult.Status
			}
			
			// Transfer all date fields to metadata (CRITICAL FIX)
			if result.ClassificationResult.FilingDate != nil {
				req.Metadata["filing_date"] = *result.ClassificationResult.FilingDate
			}
			if result.ClassificationResult.EventDate != nil {
				req.Metadata["event_date"] = *result.ClassificationResult.EventDate
			}
			if result.ClassificationResult.HearingDate != nil {
				req.Metadata["hearing_date"] = *result.ClassificationResult.HearingDate
			}
			if result.ClassificationResult.DecisionDate != nil {
				req.Metadata["decision_date"] = *result.ClassificationResult.DecisionDate
			}
			if result.ClassificationResult.ServedDate != nil {
				req.Metadata["served_date"] = *result.ClassificationResult.ServedDate
			}
		}
	}

	// Step 3: Document Storage
	if req.Options.StoreDocument {
		if err := p.executeStep(ctx, ProcessorTypeStorage, req, result); err != nil {
			return NewPipelineError("storage_failed", "document storage failed", ProcessorTypeStorage, err)
		}

		// Pass storage results to subsequent steps
		if result.StorageResult != nil {
			req.Metadata["storage_path"] = result.StorageResult.StoragePath
			if result.StorageResult.URL != "" {
				req.Metadata["storage_url"] = result.StorageResult.URL
			}
		}
	}

	// Step 4: Document Indexing (uses all previous results)
	if req.Options.IndexDocument {
		// Special handling for indexing processor to pass full ClassificationResult
		if err := p.executeIndexingStep(ctx, req, result); err != nil {
			return NewPipelineError("indexing_failed", "document indexing failed", ProcessorTypeIndexing, err)
		}
	}

	return nil
}

// executeIndexingStep executes the indexing step with access to full ProcessResult
func (p *pipeline) executeIndexingStep(ctx context.Context, req *ProcessRequest, result *ProcessResult) error {
	stepStart := time.Now()

	processor, exists := p.processors[ProcessorTypeIndexing]
	if !exists {
		return fmt.Errorf("indexing processor not found")
	}

	// Check if processor is healthy
	if !processor.IsHealthy() {
		return fmt.Errorf("indexing processor is not healthy")
	}

	// Create a specialized indexing processor that can handle full ProcessResult
	if indexingProc, ok := processor.(*indexingProcessor); ok {
		// Call specialized indexing method with full result
		stepResult, err := indexingProc.ProcessWithFullResult(ctx, req, result)
		if err != nil {
			// Record failed step
			step := &ProcessStep{
				Type:      ProcessorTypeIndexing,
				Success:   false,
				Error:     err.Error(),
				Duration:  time.Since(stepStart).Milliseconds(),
				Timestamp: time.Now(),
			}
			result.Steps = append(result.Steps, step)
			return err
		}

		// Record successful step
		step := &ProcessStep{
			Type:      ProcessorTypeIndexing,
			Success:   true,
			Duration:  time.Since(stepStart).Milliseconds(),
			Timestamp: time.Now(),
		}
		result.Steps = append(result.Steps, step)

		// Merge step results into main result
		p.mergeStepResult(ProcessorTypeIndexing, stepResult, result)
		return nil
	}

	// Fallback to regular processing if not the specialized indexing processor
	return p.executeStep(ctx, ProcessorTypeIndexing, req, result)
}

// executeStep executes a single processing step
func (p *pipeline) executeStep(ctx context.Context, stepType ProcessorType, req *ProcessRequest, result *ProcessResult) error {
	stepStart := time.Now()

	processor, exists := p.processors[stepType]
	if !exists {
		return fmt.Errorf("processor not found for type: %s", stepType)
	}

	// Check if processor is healthy
	if !processor.IsHealthy() {
		return fmt.Errorf("processor %s is not healthy", stepType)
	}

	// Execute the processor
	stepResult, err := processor.Process(ctx, req)
	if err != nil {
		// Record failed step
		step := &ProcessStep{
			Type:      stepType,
			Success:   false,
			Error:     err.Error(),
			Duration:  time.Since(stepStart).Milliseconds(),
			Timestamp: time.Now(),
		}
		result.Steps = append(result.Steps, step)
		return err
	}

	// Record successful step
	step := &ProcessStep{
		Type:      stepType,
		Success:   true,
		Duration:  time.Since(stepStart).Milliseconds(),
		Timestamp: time.Now(),
	}
	result.Steps = append(result.Steps, step)

	// Merge step results into main result
	p.mergeStepResult(stepType, stepResult, result)

	return nil
}

// mergeStepResult merges step results into the main result
func (p *pipeline) mergeStepResult(stepType ProcessorType, stepResult *ProcessResult, mainResult *ProcessResult) {
	switch stepType {
	case ProcessorTypeExtraction:
		mainResult.ExtractionResult = stepResult.ExtractionResult
	case ProcessorTypeClassification:
		mainResult.ClassificationResult = stepResult.ClassificationResult
	case ProcessorTypeIndexing:
		mainResult.IndexResult = stepResult.IndexResult
	case ProcessorTypeStorage:
		mainResult.StorageResult = stepResult.StorageResult
	}

	if stepResult.Document != nil {
		mainResult.Document = stepResult.Document
	}
}

// GetStatus returns the current status of the pipeline
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

// Stop gracefully stops the pipeline
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

// IsHealthy returns true if the pipeline is healthy
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

// Start starts the pipeline
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

// DefaultConfig returns default pipeline configuration
func DefaultConfig() *Config {
	return &Config{
		MaxWorkers:     10,
		QueueSize:      100,
		ProcessTimeout: 5 * time.Minute,
		RetryAttempts:  3,
		RetryDelay:     time.Second,
		EnableMetrics:  true,
		MaxPDFPages:    25, // Limit PDF processing to 25 pages by default
	}
}
