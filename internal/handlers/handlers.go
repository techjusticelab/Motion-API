package handlers

import (
	"context"
	"fmt"
	"time"

	"motion-index-fiber/internal/config"
	"motion-index-fiber/pkg/cloud/digitalocean"
	"motion-index-fiber/pkg/processing"
	"motion-index-fiber/pkg/processing/classifier"
	"motion-index-fiber/pkg/processing/extractor"
	"motion-index-fiber/pkg/processing/pipeline"
	"motion-index-fiber/pkg/processing/queue"
)

type Handlers struct {
	Health       *HealthHandler
	Processing   *ProcessingHandler
	Search       *SearchHandler
	Storage      *StorageHandler
	Batch        *BatchHandler
	Indexing     *IndexingHandler
	queueManager queue.QueueManager
}

func New(cfg *config.Config) (*Handlers, error) {
	// Initialize services using DigitalOcean service factory
	if cfg.DigitalOcean == nil {
		return nil, fmt.Errorf("DigitalOcean configuration is required")
	}

	// Create DigitalOcean service factory
	doFactory := digitalocean.NewServiceFactory(cfg.DigitalOcean)

	// Create storage service through factory
	storageService, err := doFactory.CreateStorageService()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage service: %w", err)
	}

	// Create search service through factory
	searchService, err := doFactory.CreateSearchService()
	if err != nil {
		return nil, fmt.Errorf("failed to create search service: %w", err)
	}

	// Initialize text extraction service
	extractorService := extractor.NewService()

	// Initialize classification service with fallback support
	classifierService, err := createClassificationService(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create classification service: %w", err)
	}

	// Initialize processing pipeline
	pipelineConfig := &pipeline.Config{
		MaxWorkers:     cfg.Processing.MaxWorkers,
		QueueSize:      cfg.Processing.BatchSize,
		ProcessTimeout: cfg.Processing.ProcessTimeout,
		RetryAttempts:  3,
		RetryDelay:     1 * time.Second,
		EnableMetrics:  true,
	}

	processingPipeline, err := pipeline.NewPipeline(
		extractorService,
		classifierService,
		searchService,
		storageService,
		pipelineConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create processing pipeline: %w", err)
	}

	// Initialize queue manager for batch processing
	queueManager := queue.NewQueueManager()

	// Create indexing queue with real processor
	indexingConfig := &queue.QueueConfig{
		Name:            "indexing",
		Type:            queue.QueueTypeIndexing,
		MaxSize:         300,
		WorkerCount:     6,
		ProcessTimeout:  30 * time.Second,
		RetryAttempts:   3,
		RetryDelay:      5 * time.Second,
		EnableMetrics:   true,
	}

	// Create indexing processor
	indexingProcessorConfig := &processing.IndexingProcessorConfig{
		APIBaseURL:     "http://localhost:6000", // TODO: Make this configurable
		RequestTimeout: 30 * time.Second,
		MaxRetries:     3,
		RetryDelay:     5 * time.Second,
	}
	indexingProcessor := processing.NewIndexingProcessor(indexingProcessorConfig)

	// Create the indexing queue
	_, err = queueManager.CreateQueue(indexingConfig, indexingProcessor.ProcessIndexingItem)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexing queue: %w", err)
	}

	return &Handlers{
		Health:       NewHealthHandler(storageService, searchService),
		Processing:   NewProcessingHandler(cfg, processingPipeline, storageService, searchService),
		Search:       NewSearchHandler(searchService),
		Storage:      NewStorageHandler(cfg, storageService),
		Batch:        NewBatchHandler(queueManager, storageService, searchService, classifierService, extractorService),
		Indexing:     NewIndexingHandler(searchService),
		queueManager: queueManager,
	}, nil
}

// StartQueueProcessing starts the queue manager and all worker pools
func (h *Handlers) StartQueueProcessing(ctx context.Context) error {
	if h.queueManager == nil {
		return fmt.Errorf("queue manager not initialized")
	}
	return h.queueManager.Start(ctx)
}

// StopQueueProcessing stops the queue manager gracefully
func (h *Handlers) StopQueueProcessing(ctx context.Context) error {
	if h.queueManager == nil {
		return nil
	}
	return h.queueManager.Stop(ctx)
}

// GetQueueStats returns statistics for all queues
func (h *Handlers) GetQueueStats() map[string]*queue.QueueStats {
	if h.queueManager == nil {
		return nil
	}
	return h.queueManager.GetAllStats()
}

// IsQueueHealthy returns true if all queues are healthy
func (h *Handlers) IsQueueHealthy() bool {
	if h.queueManager == nil {
		return false
	}
	return h.queueManager.IsHealthy()
}

// createClassificationService creates a classification service with fallback support
func createClassificationService(cfg *config.Config) (classifier.Service, error) {
	// Check if fallback is enabled and we have multiple providers configured
	if cfg.AI.EnableFallback && (cfg.AI.Claude.APIKey != "" || cfg.AI.Ollama.BaseURL != "") {
		// Create fallback classifier
		fallbackConfig := &classifier.FallbackConfig{
			EnableFallback: cfg.AI.EnableFallback,
			RetryAttempts:  cfg.AI.RetryAttempts,
			RetryDelay:     cfg.AI.RetryDelay,
		}

		// Configure Ollama (primary - cost-effective local model)
		if cfg.AI.Ollama.BaseURL != "" {
			fallbackConfig.Ollama = &classifier.OllamaConfig{
				BaseURL: cfg.AI.Ollama.BaseURL,
				Model:   cfg.AI.Ollama.Model,
				Timeout: cfg.AI.Ollama.Timeout,
			}
		}
		
		// Configure OpenAI (fallback only)
		if cfg.AI.OpenAI.APIKey != "" {
			fallbackConfig.OpenAI = &classifier.Config{
				Provider:   "openai",
				APIKey:     cfg.AI.OpenAI.APIKey,
				Model:      cfg.AI.OpenAI.Model,
				MaxRetries: 3,
				Timeout:    30 * time.Second,
			}
		} else if cfg.OpenAI.APIKey != "" {
			// Backward compatibility
			fallbackConfig.OpenAI = &classifier.Config{
				Provider:   "openai",
				APIKey:     cfg.OpenAI.APIKey,
				Model:      cfg.OpenAI.Model,
				MaxRetries: 3,
				Timeout:    30 * time.Second,
			}
		}

		// Configure Claude (first fallback)
		if cfg.AI.Claude.APIKey != "" {
			fallbackConfig.Claude = &classifier.ClaudeConfig{
				APIKey:  cfg.AI.Claude.APIKey,
				Model:   cfg.AI.Claude.Model,
				BaseURL: cfg.AI.Claude.BaseURL,
			}
		}

		// Configure Ollama (local fallback)
		if cfg.AI.Ollama.BaseURL != "" {
			fallbackConfig.Ollama = &classifier.OllamaConfig{
				BaseURL: cfg.AI.Ollama.BaseURL,
				Model:   cfg.AI.Ollama.Model,
				Timeout: cfg.AI.Ollama.Timeout,
			}
		}

		// Create fallback classifier directly
		fallbackClassifier, err := classifier.NewFallbackClassifier(fallbackConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create fallback classifier: %w", err)
		}

		// Wrap in a service
		return &classifier.ServiceWrapper{Classifier: fallbackClassifier}, nil
	}

	// Fall back to single provider - prioritize Ollama for cost savings
	if cfg.AI.Ollama.BaseURL != "" {
		// Use Ollama as single provider (cost-effective local model)
		ollamaConfig := &classifier.OllamaConfig{
			BaseURL: cfg.AI.Ollama.BaseURL,
			Model:   cfg.AI.Ollama.Model,
			Timeout: cfg.AI.Ollama.Timeout,
		}
		
		ollamaClassifier, err := classifier.NewOllamaClassifier(ollamaConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Ollama classifier: %w", err)
		}
		
		return &classifier.ServiceWrapper{Classifier: ollamaClassifier}, nil
	}
	
	// Fallback to OpenAI if Ollama not configured
	var primaryAPIKey, primaryModel string
	if cfg.AI.OpenAI.APIKey != "" {
		primaryAPIKey = cfg.AI.OpenAI.APIKey
		primaryModel = cfg.AI.OpenAI.Model
	} else if cfg.OpenAI.APIKey != "" {
		primaryAPIKey = cfg.OpenAI.APIKey
		primaryModel = cfg.OpenAI.Model
	} else {
		return nil, fmt.Errorf("Ollama is not configured and no AI provider API key is available")
	}

	classifierConfig := &classifier.Config{
		Provider:   "openai",
		APIKey:     primaryAPIKey,
		Model:      primaryModel,
		MaxRetries: 3,
		Timeout:    30 * time.Second,
	}

	return classifier.NewService(classifierConfig)
}

