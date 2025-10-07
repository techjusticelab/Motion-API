package handlers

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	internalModels "motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/processing"
	"motion-index-fiber/pkg/processing/classifier"
	"motion-index-fiber/pkg/processing/extractor"
	"motion-index-fiber/pkg/processing/queue"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/storage"
)

// PendingDocument represents a document ready for batch indexing
type PendingDocument struct {
	Document         *BatchDocumentInput              `json:"document"`
	Text             string                          `json:"text"`
	Classification   *classifier.ClassificationResult `json:"classification"`
}

// BatchHandler handles async batch processing operations
type BatchHandler struct {
	queueManager     queue.QueueManager
	storage          storage.Service
	search           search.Service
	classifier       classifier.Service
	extractor        extractor.Service
	jobs             map[string]*BatchJob
	jobsMutex        sync.RWMutex
	pendingDocs      map[string][]*PendingDocument // jobID -> documents for batch indexing
	pendingDocsMutex sync.RWMutex
}

// BatchJob represents an async batch processing job
type BatchJob struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Progress    BatchProgress          `json:"progress"`
	Results     []BatchResult          `json:"results,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Options     map[string]interface{} `json:"options"`
}

// BatchProgress tracks the progress of a batch job
type BatchProgress struct {
	TotalDocuments    int     `json:"total_documents"`
	ProcessedCount    int     `json:"processed_count"`
	SuccessCount      int     `json:"success_count"`
	ErrorCount        int     `json:"error_count"`
	SkippedCount      int     `json:"skipped_count"`
	IndexedCount      int     `json:"indexed_count"`
	IndexErrorCount   int     `json:"index_error_count"`
	PercentComplete   float64 `json:"percent_complete"`
	EstimatedDuration string  `json:"estimated_duration,omitempty"`
}

// BatchResult represents the result of processing a single document in a batch
type BatchResult struct {
	DocumentID           string                           `json:"document_id"`
	DocumentPath         string                           `json:"document_path"`
	Status               string                           `json:"status"`
	ClassificationResult *classifier.ClassificationResult `json:"classification_result,omitempty"`
	Error                string                           `json:"error,omitempty"`
	Indexed              bool                             `json:"indexed"`
	IndexError           string                           `json:"index_error,omitempty"`
	IndexID              string                           `json:"index_id,omitempty"`
	ProcessedAt          time.Time                        `json:"processed_at"`
}

// BatchClassifyRequest represents a request to classify multiple documents
type BatchClassifyRequest struct {
	Documents []BatchDocumentInput   `json:"documents"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// BatchDocumentInput represents a document to be processed
type BatchDocumentInput struct {
	DocumentID   string `json:"document_id"`
	DocumentPath string `json:"document_path,omitempty"`
	Text         string `json:"text,omitempty"`
}

// NewBatchHandler creates a new batch handler
func NewBatchHandler(queueManager queue.QueueManager, storage storage.Service, search search.Service, classifier classifier.Service, extractor extractor.Service) *BatchHandler {
	return &BatchHandler{
		queueManager: queueManager,
		storage:      storage,
		search:       search,
		classifier:   classifier,
		extractor:    extractor,
		jobs:         make(map[string]*BatchJob),
		pendingDocs:  make(map[string][]*PendingDocument),
	}
}

// StartBatchClassification handles POST /api/batch/classify - Start async classification job
func (h *BatchHandler) StartBatchClassification(c *fiber.Ctx) error {
	var request BatchClassifyRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"parse_error",
			"Failed to parse request body",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Validate request
	if len(request.Documents) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			"No documents provided for classification",
			nil,
		))
	}

	if len(request.Documents) > 1000 {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			"Maximum 1000 documents per batch",
			nil,
		))
	}

	// Create batch job
	jobID := uuid.New().String()
	job := &BatchJob{
		ID:     jobID,
		Type:   "classification",
		Status: "queued",
		Progress: BatchProgress{
			TotalDocuments: len(request.Documents),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Options:   request.Options,
	}

	// Store job
	h.jobsMutex.Lock()
	h.jobs[jobID] = job
	h.jobsMutex.Unlock()

	// Start async processing
	go h.processBatchClassification(jobID, request.Documents)

	response := map[string]interface{}{
		"job_id":          jobID,
		"status":          job.Status,
		"total_documents": len(request.Documents),
		"created_at":      job.CreatedAt,
	}

	return c.Status(fiber.StatusAccepted).JSON(internalModels.NewSuccessResponse(response, "Batch classification job started"))
}

// GetBatchJobStatus handles GET /api/batch/{job_id}/status - Get job progress
func (h *BatchHandler) GetBatchJobStatus(c *fiber.Ctx) error {
	jobID := c.Params("job_id")
	if jobID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			"Job ID is required",
			nil,
		))
	}

	h.jobsMutex.RLock()
	job, exists := h.jobs[jobID]
	h.jobsMutex.RUnlock()

	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(internalModels.NewErrorResponse(
			"job_not_found",
			"Batch job not found",
			nil,
		))
	}

	return c.JSON(internalModels.NewSuccessResponse(job, "Job status retrieved successfully"))
}

// GetBatchJobResults handles GET /api/batch/{job_id}/results - Get completed results
func (h *BatchHandler) GetBatchJobResults(c *fiber.Ctx) error {
	jobID := c.Params("job_id")
	if jobID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			"Job ID is required",
			nil,
		))
	}

	h.jobsMutex.RLock()
	job, exists := h.jobs[jobID]
	h.jobsMutex.RUnlock()

	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(internalModels.NewErrorResponse(
			"job_not_found",
			"Batch job not found",
			nil,
		))
	}

	if job.Status != "completed" && job.Status != "failed" {
		return c.Status(fiber.StatusConflict).JSON(internalModels.NewErrorResponse(
			"job_not_ready",
			"Job is not yet completed",
			map[string]interface{}{"current_status": job.Status},
		))
	}

	response := map[string]interface{}{
		"job_id":       jobID,
		"status":       job.Status,
		"progress":     job.Progress,
		"results":      job.Results,
		"completed_at": job.CompletedAt,
	}

	return c.JSON(internalModels.NewSuccessResponse(response, "Job results retrieved successfully"))
}

// CancelBatchJob handles DELETE /api/batch/{job_id} - Cancel running job
func (h *BatchHandler) CancelBatchJob(c *fiber.Ctx) error {
	jobID := c.Params("job_id")
	if jobID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			"Job ID is required",
			nil,
		))
	}

	h.jobsMutex.Lock()
	job, exists := h.jobs[jobID]
	if exists && (job.Status == "queued" || job.Status == "running") {
		job.Status = "cancelled"
		job.UpdatedAt = time.Now()
		now := time.Now()
		job.CompletedAt = &now
	}
	h.jobsMutex.Unlock()

	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(internalModels.NewErrorResponse(
			"job_not_found",
			"Batch job not found",
			nil,
		))
	}

	return c.JSON(internalModels.NewSuccessResponse(map[string]interface{}{
		"job_id": jobID,
		"status": job.Status,
	}, "Job cancelled successfully"))
}

// processBatchClassification processes a batch of documents for classification
func (h *BatchHandler) processBatchClassification(jobID string, documents []BatchDocumentInput) {
	ctx := context.Background()

	// Update job status to running
	h.updateJobStatus(jobID, "running", "")

	var results []BatchResult
	var successCount, errorCount, skippedCount int

	for i, doc := range documents {
		// Check if job was cancelled
		h.jobsMutex.RLock()
		job := h.jobs[jobID]
		cancelled := job.Status == "cancelled"
		h.jobsMutex.RUnlock()

		if cancelled {
			break
		}

		result := h.processDocument(ctx, jobID, doc, job.Options)
		results = append(results, result)

		// Track all metrics
		var indexedCount, indexErrorCount int
		for _, r := range results {
			if r.Indexed {
				indexedCount++
			}
			if r.IndexError != "" {
				indexErrorCount++
			}
		}

		switch result.Status {
		case "success":
			successCount++
		case "error":
			errorCount++
		case "skipped":
			skippedCount++
		}

		// Update progress with indexing metrics
		h.updateJobProgress(jobID, i+1, successCount, errorCount, skippedCount, indexedCount, indexErrorCount, results)

		// Log detailed progress every 10 documents
		if (i+1)%10 == 0 || i+1 == len(documents) {
			percentComplete := float64(i+1) / float64(len(documents)) * 100
			log.Printf("[BATCH-PROGRESS] 📊 Job %s: %.1f%% complete (%d/%d) | ✅ %d classified | 🚫 %d errors | 📦 %d queued | ❌ %d queue errors",
				jobID, percentComplete, i+1, len(documents), successCount, errorCount, indexedCount, indexErrorCount)
		}
	}

	// Mark job as completed
	h.finalizeJob(jobID, results, successCount, errorCount, skippedCount)
}

// processDocument processes a single document for classification
func (h *BatchHandler) processDocument(ctx context.Context, jobID string, doc BatchDocumentInput, jobOptions map[string]interface{}) BatchResult {
	log.Printf("[BATCH-DOC] 🔄 Starting processing for document: %s", doc.DocumentID)

	result := BatchResult{
		DocumentID:   doc.DocumentID,
		DocumentPath: doc.DocumentPath,
		ProcessedAt:  time.Now(),
	}

	// Get text content
	text := doc.Text
	if text == "" && doc.DocumentPath != "" {
		// Download and extract text from document
		log.Printf("[BATCH-EXTRACT] 📥 Downloading document: %s", doc.DocumentID)
		reader, err := h.storage.Download(ctx, doc.DocumentPath)
		if err != nil {
			log.Printf("[BATCH-EXTRACT] ❌ Download failed for document %s: %v", doc.DocumentID, err)
			result.Status = "error"
			result.Error = fmt.Sprintf("Failed to download document: %v", err)
			return result
		}
		defer reader.Close()

		// Extract text using the extractor service
		log.Printf("[BATCH-EXTRACT] 📄 Extracting text from document: %s", doc.DocumentID)
		metadata := &extractor.DocumentMetadata{
			FileName: doc.DocumentID,
			Format:   getFileFormat(doc.DocumentPath),
		}

		extractionResult, err := h.extractor.ExtractText(ctx, reader, metadata)
		if err != nil {
			log.Printf("[BATCH-EXTRACT] ❌ Text extraction failed for document %s: %v", doc.DocumentID, err)
			result.Status = "error"
			result.Error = fmt.Sprintf("Failed to extract text: %v", err)
			return result
		}

		if !extractionResult.Success {
			log.Printf("[BATCH-EXTRACT] ❌ Text extraction unsuccessful for document %s: %s", doc.DocumentID, extractionResult.Error)
			result.Status = "error"
			result.Error = fmt.Sprintf("Text extraction failed: %s", extractionResult.Error)
			return result
		}

		log.Printf("[BATCH-EXTRACT] ✅ Text extraction successful for document %s (%d chars)", doc.DocumentID, len(extractionResult.Text))
		text = extractionResult.Text
	}

	// Handle text processing and validation
	var originalText string
	var classificationText string
	var isActualContent bool

	if text == "" {
		// Try to salvage by using filename and basic metadata
		fallbackText := h.generateFallbackText(doc.DocumentPath, doc.DocumentID)
		if fallbackText != "" {
			originalText = fallbackText
			classificationText = fallbackText
			isActualContent = false
			log.Printf("[BATCH] Using fallback text for document: %s", doc.DocumentID)
		} else {
			result.Status = "skipped"
			result.Error = "No text content provided and no fallback available"
			return result
		}
	} else {
		// We have actual extracted text - apply user's text selection requirements
		originalText = text
		isActualContent = true

		// Skip first 500 characters, then take 1000 characters for AI classification
		if len(text) > 500 {
			startPos := 500
			endPos := startPos + 1000
			if endPos > len(text) {
				endPos = len(text)
			}
			classificationText = text[startPos:endPos]
			log.Printf("[BATCH-EXTRACT] 📝 Using text substring for AI classification on document %s: chars %d-%d (from %d total chars)",
				doc.DocumentID, startPos, endPos-1, len(originalText))
		} else {
			// Document is too short, use all available text
			classificationText = text
			log.Printf("[BATCH-EXTRACT] ⚠️  Document %s has only %d chars, using all available text for classification",
				doc.DocumentID, len(text))
		}
	}

	// Check if AI classification should be skipped
	var classificationResult *classifier.ClassificationResult
	if skipAI, ok := jobOptions["skip_ai"].(bool); ok && skipAI {
		log.Printf("[BATCH] Skipping AI classification for document: %s (skip_ai option)", doc.DocumentID)
		// Create a default classification result for indexing
		classificationResult = &classifier.ClassificationResult{
			DocumentType:  "other",
			LegalCategory: "unknown",
			Summary:       "Document processed without AI classification",
			Confidence:    0.0,
			Success:       true,
		}
	} else {
		// Classify document using the processed text (actual content from chars 500-1500, or fallback metadata)
		metadata := &classifier.DocumentMetadata{
			FileName:     doc.DocumentID,
			FileType:     "unknown",
			SourceSystem: "batch-processor",
		}

		log.Printf("[BATCH-CLASSIFY] Starting classification for document: %s (using %d chars)", doc.DocumentID, len(classificationText))
		var err error
		classificationResult, err = h.classifier.ClassifyDocument(ctx, classificationText, metadata)
		if err != nil {
			// Enhanced error logging for OpenAI API issues
			errorType := h.categorizeClassificationError(err)
			log.Printf("[BATCH-CLASSIFY] ❌ Classification failed for document %s: %s - %v", doc.DocumentID, errorType, err)

			result.Status = "error"
			result.Error = fmt.Sprintf("Classification failed (%s): %v", errorType, err)
			return result
		}

		log.Printf("[BATCH-CLASSIFY] ✅ Classification successful for document %s (confidence: %.2f, type: %s)",
			doc.DocumentID, classificationResult.Confidence, classificationResult.DocumentType)
	}

	result.Status = "success"
	result.ClassificationResult = classificationResult

	// Store document for batch indexing instead of immediate indexing
	if shouldIndex := h.shouldIndexDocument(jobOptions); shouldIndex {
		log.Printf("[BATCH-DEFER] 💾 Storing document for batch indexing: %s", doc.DocumentID)
		// Use original text for indexing, not the processed text used for classification
		indexingText := originalText
		if !isActualContent {
			// For fallback text, we still want to index it for searchability
			indexingText = originalText // Use originalText consistently
		}
		log.Printf("[BATCH-DEFER] 📝 Stored %d chars for batch indexing (isActualContent: %t)", len(indexingText), isActualContent)
		
		// Store document for batch indexing after classification phase completes
		h.storePendingDocument(jobID, &doc, indexingText, classificationResult)
		
		result.Indexed = false // Will be indexed in batch after classification completes
		result.IndexID = ""    // Will be set when batch indexed
		log.Printf("[BATCH-DEFER] ✅ Document stored for batch indexing: %s", doc.DocumentID)
	} else {
		log.Printf("[BATCH-DEFER] ⏭️  Indexing not requested for document: %s", doc.DocumentID)
		result.Indexed = false
	}

	return result
}

// updateJobStatus updates the status of a batch job
func (h *BatchHandler) updateJobStatus(jobID, status, errorMsg string) {
	h.jobsMutex.Lock()
	defer h.jobsMutex.Unlock()

	if job, exists := h.jobs[jobID]; exists {
		job.Status = status
		job.UpdatedAt = time.Now()
		if errorMsg != "" {
			job.Error = errorMsg
		}
		if status == "completed" || status == "failed" || status == "cancelled" {
			now := time.Now()
			job.CompletedAt = &now
		}
	}
}

// updateJobProgress updates the progress of a batch job
func (h *BatchHandler) updateJobProgress(jobID string, processed, success, errors, skipped, indexed, indexErrors int, results []BatchResult) {
	h.jobsMutex.Lock()
	defer h.jobsMutex.Unlock()

	if job, exists := h.jobs[jobID]; exists {
		job.Progress.ProcessedCount = processed
		job.Progress.SuccessCount = success
		job.Progress.ErrorCount = errors
		job.Progress.SkippedCount = skipped
		job.Progress.IndexedCount = indexed
		job.Progress.IndexErrorCount = indexErrors
		job.Progress.PercentComplete = float64(processed) / float64(job.Progress.TotalDocuments) * 100
		job.Results = results
		job.UpdatedAt = time.Now()
	}
}

// storePendingDocument stores a document for batch indexing after classification completes
func (h *BatchHandler) storePendingDocument(jobID string, doc *BatchDocumentInput, text string, classification *classifier.ClassificationResult) {
	h.pendingDocsMutex.Lock()
	defer h.pendingDocsMutex.Unlock()
	
	pendingDoc := &PendingDocument{
		Document:       doc,
		Text:          text,
		Classification: classification,
	}
	
	h.pendingDocs[jobID] = append(h.pendingDocs[jobID], pendingDoc)
	log.Printf("[BATCH-DEFER] Added document %s to pending batch for job %s (total pending: %d)", 
		doc.DocumentID, jobID, len(h.pendingDocs[jobID]))
}

// finalizeJob marks a batch job as completed and triggers batch indexing
func (h *BatchHandler) finalizeJob(jobID string, results []BatchResult, success, errors, skipped int) {
	// Trigger batch indexing before marking job as complete
	indexedCount, indexErrorCount := h.performBatchIndexing(jobID, results)

	status := "completed"
	if errors > 0 && success == 0 {
		status = "failed"
	}

	h.jobsMutex.Lock()
	defer h.jobsMutex.Unlock()

	if job, exists := h.jobs[jobID]; exists {
		job.Status = status
		job.Results = results
		job.Progress.SuccessCount = success
		job.Progress.ErrorCount = errors
		job.Progress.SkippedCount = skipped
		job.Progress.IndexedCount = indexedCount
		job.Progress.IndexErrorCount = indexErrorCount
		job.Progress.PercentComplete = 100.0
		job.UpdatedAt = time.Now()
		now := time.Now()
		job.CompletedAt = &now

		// Log final statistics
		log.Printf("[BATCH] Job %s completed: %d processed, %d classified, %d indexed, %d index errors",
			jobID, len(results), success, indexedCount, indexErrorCount)
	}
}

// performBatchIndexing performs bulk indexing for all pending documents in a job
func (h *BatchHandler) performBatchIndexing(jobID string, results []BatchResult) (indexedCount, indexErrorCount int) {
	h.pendingDocsMutex.Lock()
	pendingDocs, exists := h.pendingDocs[jobID]
	if exists {
		delete(h.pendingDocs, jobID) // Clean up pending docs
	}
	h.pendingDocsMutex.Unlock()
	
	if !exists || len(pendingDocs) == 0 {
		log.Printf("[BATCH-INDEX] No pending documents to index for job %s", jobID)
		return 0, 0
	}
	
	log.Printf("[BATCH-INDEX] 🚀 Starting batch indexing for job %s (%d documents)", jobID, len(pendingDocs))
	
	// Convert pending documents to search documents
	var searchDocs []*models.Document
	docMap := make(map[string]*PendingDocument) // Track pending docs by ID
	
	for _, pendingDoc := range pendingDocs {
		// Create search document from classification result
		searchDoc := &models.Document{
			ID:            pendingDoc.Document.DocumentID,
			FileName:      pendingDoc.Document.DocumentID, // Use ID as filename if no filename
			FilePath:      pendingDoc.Document.DocumentPath,
			Text:          pendingDoc.Text,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		
		// Add classification metadata
		if pendingDoc.Classification != nil {
			// Create metadata if not exists
			if searchDoc.Metadata == nil {
				searchDoc.Metadata = &models.DocumentMetadata{}
			}
			searchDoc.DocType = pendingDoc.Classification.DocumentType
			searchDoc.Metadata.DocumentType = models.DocumentType(pendingDoc.Classification.DocumentType)
			searchDoc.Metadata.Subject = pendingDoc.Classification.Subject
			searchDoc.Metadata.Summary = pendingDoc.Classification.Summary
			searchDoc.Metadata.Confidence = pendingDoc.Classification.Confidence
			searchDoc.Metadata.AIClassified = true
			searchDoc.Metadata.ProcessedAt = time.Now()
		}
		
		searchDocs = append(searchDocs, searchDoc)
		docMap[searchDoc.ID] = pendingDoc
	}
	
	// Perform bulk indexing
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // Longer timeout for bulk operations
	defer cancel()
	
	bulkResult, err := h.search.BulkIndexDocuments(ctx, searchDocs)
	if err != nil {
		log.Printf("[BATCH-INDEX] ❌ Bulk indexing failed for job %s: %v", jobID, err)
		indexErrorCount = len(pendingDocs)
		
		// Update results with indexing errors
		for i := range results {
			if docMap[results[i].DocumentID] != nil {
				results[i].IndexError = err.Error()
				results[i].Indexed = false
			}
		}
		return 0, indexErrorCount
	}
	
	// Process bulk indexing results
	log.Printf("[BATCH-INDEX] ✅ Bulk indexing completed for job %s: %d indexed, %d failed", 
		jobID, bulkResult.Indexed, bulkResult.Failed)
	
	// Create a map of failed documents for quick lookup
	failedDocs := make(map[string]string)
	if bulkResult.FailedDocs != nil {
		for _, failedDoc := range bulkResult.FailedDocs {
			failedDocs[failedDoc.ID] = failedDoc.Error
		}
	}
	
	// Update results with indexing status
	for i := range results {
		if docMap[results[i].DocumentID] != nil {
			if errorMsg, failed := failedDocs[results[i].DocumentID]; failed {
				results[i].IndexError = errorMsg
				results[i].Indexed = false
				indexErrorCount++
			} else {
				results[i].Indexed = true
				results[i].IndexID = results[i].DocumentID // Use document ID as index ID
				indexedCount++
			}
		}
	}
	
	return indexedCount, indexErrorCount
}

// getFileFormat extracts the file format from a file path
func getFileFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "unknown"
	}
	return ext[1:] // Remove the leading dot
}

// shouldIndexDocument checks if indexing is requested in the job options
func (h *BatchHandler) shouldIndexDocument(options map[string]interface{}) bool {
	if options == nil {
		return false
	}

	// Check for "update_index" option (used by api-batch-classifier)
	if updateIndex, exists := options["update_index"]; exists {
		if val, ok := updateIndex.(bool); ok && val {
			return true
		}
	}

	// Check for "index_document" option (standard option)
	if indexDoc, exists := options["index_document"]; exists {
		if val, ok := indexDoc.(bool); ok && val {
			return true
		}
	}

	return false
}

// generateFallbackText creates basic text content from filename when extraction fails
func (h *BatchHandler) generateFallbackText(documentPath, documentID string) string {
	if documentPath == "" && documentID == "" {
		return ""
	}

	// Use path if available, otherwise use ID
	source := documentPath
	if source == "" {
		source = documentID
	}

	// Extract filename
	fileName := filepath.Base(source)

	// Clean up filename to make it readable
	cleanName := strings.ReplaceAll(fileName, "_", " ")
	cleanName = strings.ReplaceAll(cleanName, "-", " ")

	// Remove file extension
	if ext := filepath.Ext(cleanName); ext != "" {
		cleanName = strings.TrimSuffix(cleanName, ext)
	}

	// Generate basic text content for classification
	fallbackText := fmt.Sprintf("Document: %s\nFile Path: %s\nFilename: %s",
		cleanName, source, fileName)

	// Try to infer document type from filename
	lowerName := strings.ToLower(cleanName)
	switch {
	case strings.Contains(lowerName, "motion"):
		fallbackText += "\nDocument Type: Legal Motion"
	case strings.Contains(lowerName, "complaint"):
		fallbackText += "\nDocument Type: Legal Complaint"
	case strings.Contains(lowerName, "order"):
		fallbackText += "\nDocument Type: Court Order"
	case strings.Contains(lowerName, "brief"):
		fallbackText += "\nDocument Type: Legal Brief"
	case strings.Contains(lowerName, "petition"):
		fallbackText += "\nDocument Type: Legal Petition"
	case strings.Contains(lowerName, "notice"):
		fallbackText += "\nDocument Type: Legal Notice"
	case strings.Contains(lowerName, "filing"):
		fallbackText += "\nDocument Type: Court Filing"
	default:
		fallbackText += "\nDocument Type: Legal Document"
	}

	return fallbackText
}

// enqueueForIndexing enqueues a document for asynchronous indexing
func (h *BatchHandler) enqueueForIndexing(ctx context.Context, doc BatchDocumentInput, text string, classificationResult *classifier.ClassificationResult, jobOptions map[string]interface{}) error {
	// Prepare queue options
	queueOptions := map[string]interface{}{
		"file_name":    filepath.Base(doc.DocumentPath),
		"content_type": h.determineContentType(doc.DocumentPath),
		"size":         int64(len(text)),
		"file_url":     fmt.Sprintf("/api/documents/%s", doc.DocumentPath),
	}

	// Add source job ID if available
	if jobID, exists := jobOptions["job_id"]; exists {
		queueOptions["source_job_id"] = jobID
	}

	// Create indexing queue item
	queueItem := processing.CreateIndexingQueueItem(
		doc.DocumentID,
		doc.DocumentPath,
		text,
		classificationResult,
		queueOptions,
	)

	// Get or create the indexing queue
	indexingQueue, err := h.queueManager.GetQueue("indexing")
	if err != nil {
		// Queue doesn't exist yet, will be created when queue system is initialized
		return fmt.Errorf("indexing queue not available: %w", err)
	}

	// Enqueue the item
	if err := indexingQueue.Enqueue(ctx, queueItem); err != nil {
		return fmt.Errorf("failed to enqueue item: %w", err)
	}

	log.Printf("[BATCH] Document %s enqueued for indexing (queue item: %s)", doc.DocumentID, queueItem.ID)
	return nil
}

// determineContentType determines the content type from file path
func (h *BatchHandler) determineContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".doc":
		return "application/msword"
	case ".txt":
		return "text/plain"
	case ".rtf":
		return "application/rtf"
	default:
		return "application/octet-stream"
	}
}

// categorizeClassificationError categorizes OpenAI API and classification errors for better logging
func (h *BatchHandler) categorizeClassificationError(err error) string {
	if err == nil {
		return "unknown"
	}

	errStr := strings.ToLower(err.Error())

	// OpenAI quota and rate limit errors
	if strings.Contains(errStr, "quota") || strings.Contains(errStr, "billing") {
		return "QUOTA_EXCEEDED"
	}
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "status 429") {
		return "RATE_LIMIT"
	}
	if strings.Contains(errStr, "insufficient_quota") {
		return "INSUFFICIENT_QUOTA"
	}

	// OpenAI API errors
	if strings.Contains(errStr, "status 401") || strings.Contains(errStr, "unauthorized") {
		return "API_AUTH_ERROR"
	}
	if strings.Contains(errStr, "status 400") || strings.Contains(errStr, "bad request") {
		return "API_BAD_REQUEST"
	}
	if strings.Contains(errStr, "status 5") || strings.Contains(errStr, "server error") {
		return "API_SERVER_ERROR"
	}

	// Network and timeout errors
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "context deadline exceeded") {
		return "TIMEOUT"
	}
	if strings.Contains(errStr, "connection") || strings.Contains(errStr, "network") {
		return "NETWORK_ERROR"
	}

	// JSON parsing errors
	if strings.Contains(errStr, "json") || strings.Contains(errStr, "unmarshal") {
		return "RESPONSE_PARSE_ERROR"
	}

	// Classification validation errors
	if strings.Contains(errStr, "validation") {
		return "VALIDATION_ERROR"
	}

	return "UNKNOWN_ERROR"
}

// categorizeQueueError categorizes queue operation errors for better logging
func (h *BatchHandler) categorizeQueueError(err error) string {
	if err == nil {
		return "unknown"
	}

	errStr := strings.ToLower(err.Error())

	// Queue management errors
	if strings.Contains(errStr, "queue not available") || strings.Contains(errStr, "queue not found") {
		return "QUEUE_UNAVAILABLE"
	}
	if strings.Contains(errStr, "queue full") || strings.Contains(errStr, "max size") {
		return "QUEUE_FULL"
	}
	if strings.Contains(errStr, "enqueue") {
		return "ENQUEUE_FAILED"
	}

	// Context and timeout errors
	if strings.Contains(errStr, "context") || strings.Contains(errStr, "timeout") {
		return "CONTEXT_ERROR"
	}

	// Queue item creation errors
	if strings.Contains(errStr, "queue item") || strings.Contains(errStr, "item creation") {
		return "ITEM_CREATION_ERROR"
	}

	return "QUEUE_ERROR"
}
