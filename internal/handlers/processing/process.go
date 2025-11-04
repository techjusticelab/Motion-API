package processing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	internalModels "motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/processing/pipeline"
	"motion-index-fiber/pkg/storage"
)

// UploadDocument handles document upload and processing (alias for ProcessDocument)
func (h *Handler) UploadDocument(c *fiber.Ctx) error {
	return h.ProcessDocument(c)
}

// ProcessDocument processes a single document upload with timeout and error recovery
func (h *Handler) ProcessDocument(c *fiber.Ctx) error {
	// Create a timeout context for the entire processing operation
	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second) // 60 second timeout
	defer cancel()

	// Add panic recovery for this handler
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PROCESSING-HANDLER] Panic recovered in ProcessDocument: %v", r)
			// Return error response for panic
			_ = c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
				"processing_panic",
				"Processing failed due to internal error",
				map[string]interface{}{"panic": fmt.Sprintf("%v", r)},
			))
		}
	}()

	// Parse the multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"multipart_error",
			"Failed to parse multipart form",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Get the uploaded file
	files := form.File["file"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"missing_file",
			"No file provided",
			nil,
		))
	}

	file := files[0]

	// Parse processing options from individual form fields or JSON string
	var processOptions *internalModels.ProcessOptions
	if optionsStr := c.FormValue("options"); optionsStr != "" {
		parsedOptions, err := parseProcessOptionsJSON(optionsStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
				"options_parse_error",
				"Failed to parse processing options",
				map[string]interface{}{"error": err.Error()},
			))
		}
		processOptions = parsedOptions
	} else {
		// Parse individual form fields (priority over defaults)
		processOptions = &internalModels.ProcessOptions{
			ExtractText:    c.FormValue("extract_text") != "false",   // Default true, set false only if explicitly "false"
			ClassifyDoc:    c.FormValue("classify_doc") != "false",   // Default true, set false only if explicitly "false"
			IndexDocument:  c.FormValue("index_document") != "false", // Default true, set false only if explicitly "false"
			StoreDocument:  c.FormValue("store_document") != "false", // Default true, set false only if explicitly "false"
			TimeoutSeconds: 120,
			RetryCount:     1,
		}

		// Override defaults if explicit values provided
		if c.FormValue("extract_text") == "false" {
			processOptions.ExtractText = false
		}
		if c.FormValue("classify_doc") == "false" {
			processOptions.ClassifyDoc = false
		}
		if c.FormValue("index_document") == "false" {
			processOptions.IndexDocument = false
		}
		if c.FormValue("store_document") == "false" {
			processOptions.StoreDocument = false
		}
	}

	// Validate and apply defaults
	if err := processOptions.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			err.Error(),
			nil,
		))
	}
	processOptions.ApplyDefaults()

	// Build processing request
	request := &internalModels.ProcessDocumentRequest{
		File:        file,
		Category:    c.FormValue("category"),
		Description: c.FormValue("description"),
		CaseName:    c.FormValue("case_name"),
		CaseNumber:  c.FormValue("case_number"),
		Author:      c.FormValue("author"),
		Judge:       c.FormValue("judge"),
		Court:       c.FormValue("court"),
		Options:     processOptions,
	}

	// Validate the request
	if err := internalModels.ValidateStruct(request); err != nil {
		validationErrors := internalModels.FormatValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(&models.APIResponse{
			Success:   false,
			Timestamp: time.Now(),
			Error: &models.APIError{
				Code:    "validation_error",
				Message: "Request validation failed",
				Details: map[string]interface{}{
					"validation_errors": validationErrors,
				},
			},
		})
	}

	// Process the document using the pipeline with context
	startTime := time.Now()
	result, err := h.processDocumentWithPipeline(ctx, request)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"processing_error",
			err.Error(),
			nil,
		))
	}

	result.ProcessingTime = time.Since(startTime).Milliseconds()
	result.CreatedAt = time.Now()

	// Return successful response
	return c.JSON(internalModels.NewSuccessResponse(result, "Document processed successfully"))
}

// BatchProcessDocuments processes multiple documents
func (h *Handler) BatchProcessDocuments(c *fiber.Ctx) error {
	// Parse the multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"multipart_error",
			"Failed to parse multipart form",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Get the uploaded files
	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"missing_files",
			"No files provided",
			nil,
		))
	}

	// Parse processing options
	processOptions := internalModels.DefaultProcessOptions()
	if err := processOptions.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			err.Error(),
			nil,
		))
	}
	processOptions.ApplyDefaults()

	// Build batch processing request
	request := &internalModels.BatchProcessRequest{
		Files:       files,
		Category:    c.FormValue("category"),
		Description: c.FormValue("description"),
		CaseName:    c.FormValue("case_name"),
		CaseNumber:  c.FormValue("case_number"),
		Options:     processOptions,
	}

	// Validate the request
	if err := internalModels.ValidateStruct(request); err != nil {
		validationErrors := internalModels.FormatValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(&models.APIResponse{
			Success:   false,
			Timestamp: time.Now(),
			Error: &models.APIError{
				Code:    "validation_error",
				Message: "Batch request validation failed",
				Details: map[string]interface{}{
					"validation_errors": validationErrors,
				},
			},
		})
	}

	// Process documents in batch
	startTime := time.Now()
	batchResult := h.processBatchDocuments(request)
	batchResult.ProcessingTime = time.Since(startTime).Milliseconds()
	batchResult.CompletedAt = time.Now()

	// Return batch response
	return c.JSON(internalModels.NewSuccessResponse(batchResult, "Batch processing completed"))
}

// processDocumentWithPipeline processes a single document through the pipeline
func (h *Handler) processDocumentWithPipeline(ctx context.Context, request *internalModels.ProcessDocumentRequest) (*internalModels.ProcessDocumentResponse, error) {
	file := request.File

	// Generate document ID
	documentID := generateDocumentID(file.Filename)

	response := &internalModels.ProcessDocumentResponse{
		DocumentID: documentID,
		FileName:   file.Filename,
		Status:     "processing",
		Steps:      []*internalModels.ProcessingStep{},
		Metadata: &models.DocumentMetadata{
			DocumentName: file.Filename,
			CaseName:     request.CaseName,
			CaseNumber:   request.CaseNumber,
			Author:       request.Author,
			ProcessedAt:  time.Now(),
		},
	}

	// Add Judge if provided
	if request.Judge != "" {
		response.Metadata.Judge = &models.Judge{
			Name: request.Judge,
		}
	}

	// Add Court if provided
	if request.Court != "" {
		response.Metadata.Court = &models.CourtInfo{
			CourtName: request.Court,
		}
	}

	// Check if pipeline is available
	if h.pipeline == nil {
		return h.processDocumentLegacyMode(request)
	}

	// Read file content
	fileReader, err := file.Open()
	if err != nil {
		response.Status = "failed"
		return response, fmt.Errorf("failed to open file: %w", err)
	}
	defer fileReader.Close()

	// Read content into memory
	content, err := io.ReadAll(fileReader)
	if err != nil {
		response.Status = "failed"
		return response, fmt.Errorf("failed to read file content: %w", err)
	}

	// Create pipeline processing request
	pipelineRequest := &pipeline.ProcessRequest{
		ID:          documentID,
		FileName:    file.Filename,
		ContentType: file.Header.Get("Content-Type"),
		Size:        file.Size,
		Content:     bytes.NewReader(content),
		Options: &pipeline.ProcessOptions{
			ExtractText:    request.Options.ExtractText,
			ClassifyDoc:    request.Options.ClassifyDoc,
			StoreDocument:  request.Options.StoreDocument,
			IndexDocument:  request.Options.IndexDocument,
			TimeoutSeconds: int(request.Options.TimeoutSeconds),
		},
		Metadata: map[string]string{
			"case_name":   request.CaseName,
			"case_number": request.CaseNumber,
			"author":      request.Author,
			"judge":       request.Judge,
			"court":       request.Court,
			"category":    request.Category,
		},
	}

	// Process document through pipeline (using the context passed from handler)
	pipelineResult, err := h.pipeline.ProcessDocument(ctx, pipelineRequest)
	if err != nil {
		response.Status = "failed"
		return response, fmt.Errorf("pipeline processing failed: %w", err)
	}

	// Convert pipeline results to handler response format
	h.convertPipelineResults(pipelineResult, response)

	return response, nil
}

// processDocumentLegacyMode processes document using the legacy implementation (fallback)
func (h *Handler) processDocumentLegacyMode(request *internalModels.ProcessDocumentRequest) (*internalModels.ProcessDocumentResponse, error) {
	file := request.File

	// Generate document ID
	documentID := generateDocumentID(file.Filename)

	response := &internalModels.ProcessDocumentResponse{
		DocumentID: documentID,
		FileName:   file.Filename,
		Status:     "processing",
		Steps:      []*internalModels.ProcessingStep{},
		Metadata: &models.DocumentMetadata{
			DocumentName: file.Filename,
			CaseName:     request.CaseName,
			CaseNumber:   request.CaseNumber,
			Author:       request.Author,
			ProcessedAt:  time.Now(),
		},
	}

	// Add Judge if provided
	if request.Judge != "" {
		response.Metadata.Judge = &models.Judge{
			Name: request.Judge,
		}
	}

	// Add Court if provided
	if request.Court != "" {
		response.Metadata.Court = &models.CourtInfo{
			CourtName: request.Court,
		}
	}

	// Step 1: Text Extraction (if enabled)
	if request.Options.ExtractText {
		step := &internalModels.ProcessingStep{
			Name:      "text_extraction",
			Status:    "running",
			StartTime: time.Now(),
		}
		response.Steps = append(response.Steps, step)

		// Open file for reading
		fileReader, err := file.Open()
		if err != nil {
			step.Status = "failed"
			step.Error = err.Error()
			step.EndTime = time.Now()
			step.Duration = step.EndTime.Sub(step.StartTime).Milliseconds()
			response.Status = "failed"
			return response, fmt.Errorf("failed to open file: %w", err)
		}
		defer fileReader.Close()

		// Extract text using the processing pipeline (placeholder)
		extractionResult := &internalModels.ExtractionResult{
			Text:      "Extracted text content will be processed by the pipeline",
			PageCount: 1,
			Language:  "en",
		}

		step.Status = "completed"
		step.EndTime = time.Now()
		step.Duration = step.EndTime.Sub(step.StartTime).Milliseconds()
		response.ExtractionResult = extractionResult

		// Update metadata with extraction results
		response.Metadata.WordCount = len(extractionResult.Text) / 5
		response.Metadata.Pages = extractionResult.PageCount
		response.Metadata.Language = extractionResult.Language
	}

	// Step 2: Document Classification (if enabled)
	if request.Options.ClassifyDoc && response.ExtractionResult != nil {
		step := &internalModels.ProcessingStep{
			Name:      "document_classification",
			Status:    "running",
			StartTime: time.Now(),
		}
		response.Steps = append(response.Steps, step)

		classificationResult := &internalModels.ClassificationResult{
			Category:   "document",
			Confidence: 0.75,
			Tags:       []string{"legal", "processed"},
		}

		step.Status = "completed"
		step.EndTime = time.Now()
		step.Duration = step.EndTime.Sub(step.StartTime).Milliseconds()
		response.ClassificationResult = classificationResult

		// Update metadata with classification results
		response.Metadata.LegalTags = classificationResult.Tags
	}

	// Step 3: Document Storage (if enabled)
	if request.Options.StoreDocument {
		step := &internalModels.ProcessingStep{
			Name:      "document_storage",
			Status:    "running",
			StartTime: time.Now(),
		}
		response.Steps = append(response.Steps, step)

		// Store document using the storage service
		fileReader, err := file.Open()
		if err != nil {
			step.Status = "failed"
			step.Error = err.Error()
			step.EndTime = time.Now()
			step.Duration = step.EndTime.Sub(step.StartTime).Milliseconds()
			response.Status = "failed"
			return response, fmt.Errorf("failed to open file for storage: %w", err)
		}
		defer fileReader.Close()

		// Create storage path
		storagePath := fmt.Sprintf("documents/%s/%s", documentID, file.Filename)

		// Create upload metadata
		uploadMetadata := &storage.UploadMetadata{
			ContentType: file.Header.Get("Content-Type"),
			Size:        file.Size,
			FileName:    file.Filename,
			Tags: map[string]string{
				"document_id": documentID,
				"category":    request.Category,
				"case_name":   request.CaseName,
			},
		}

		// Upload with context
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		uploadResult, err := h.storage.Upload(ctx, storagePath, fileReader, uploadMetadata)
		if err != nil {
			step.Status = "failed"
			step.Error = err.Error()
			step.EndTime = time.Now()
			step.Duration = step.EndTime.Sub(step.StartTime).Milliseconds()
			response.Status = "failed"
			return response, fmt.Errorf("document storage failed: %w", err)
		}

		step.Status = "completed"
		step.EndTime = time.Now()
		step.Duration = step.EndTime.Sub(step.StartTime).Milliseconds()
		response.StorageResult = uploadResult
		response.URL = uploadResult.URL
		response.CDN_URL = uploadResult.URL // CDN URL same as URL for now
	}

	// Step 4: Document Indexing (if enabled)
	if request.Options.IndexDocument && response.ExtractionResult != nil {
		step := &internalModels.ProcessingStep{
			Name:      "document_indexing",
			Status:    "running",
			StartTime: time.Now(),
		}
		response.Steps = append(response.Steps, step)

		// Create index document
		indexDoc := &models.Document{
			ID:        documentID,
			FileName:  file.Filename,
			Text:      response.ExtractionResult.Text,
			Category:  request.Category,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata: &models.DocumentMetadata{
				DocumentName: file.Filename,
				CaseName:     request.CaseName,
				CaseNumber:   request.CaseNumber,
				Author:       request.Author,
			},
		}

		// Map legacy Judge and Court strings to enhanced structures
		if request.Judge != "" {
			indexDoc.Metadata.Judge = &models.Judge{
				Name: request.Judge,
			}
		}

		if request.Court != "" {
			indexDoc.Metadata.Court = &models.CourtInfo{
				CourtName: request.Court,
			}
		}

		// Add classification results if available
		if response.ClassificationResult != nil {
			indexDoc.Category = response.ClassificationResult.Category
			if indexDoc.Metadata == nil {
				indexDoc.Metadata = &models.DocumentMetadata{}
			}
			indexDoc.Metadata.LegalTags = response.ClassificationResult.Tags
		}

		// Index the document
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := h.searchSvc.IndexDocument(ctx, indexDoc)
		if err != nil {
			step.Status = "failed"
			step.Error = err.Error()
			step.EndTime = time.Now()
			step.Duration = step.EndTime.Sub(step.StartTime).Milliseconds()
			// Don't fail the entire process for indexing errors
		} else {
			step.Status = "completed"
			step.EndTime = time.Now()
			step.Duration = step.EndTime.Sub(step.StartTime).Milliseconds()
			response.IndexResult = &internalModels.IndexResult{
				DocumentID: documentID,
				IndexName:  "documents", // Default index name
				Success:    true,
			}
		}
	}

	response.Status = "completed"
	return response, nil
}

// processBatchDocuments processes multiple documents
func (h *Handler) processBatchDocuments(request *internalModels.BatchProcessRequest) *internalModels.BatchProcessResponse {
	batchID := generateBatchID()

	response := &internalModels.BatchProcessResponse{
		BatchID:      batchID,
		TotalCount:   len(request.Files),
		SuccessCount: 0,
		FailureCount: 0,
		Results:      make([]*internalModels.ProcessDocumentResponse, 0, len(request.Files)),
		Errors:       make([]*internalModels.BatchProcessError, 0),
		Status:       "processing",
	}

	// Process each file
	for _, file := range request.Files {
		// Create individual processing request
		individualRequest := &internalModels.ProcessDocumentRequest{
			File:        file,
			Category:    request.Category,
			Description: request.Description,
			CaseName:    request.CaseName,
			CaseNumber:  request.CaseNumber,
			Options:     request.Options,
		}

		// Process the document with context
		ctx := context.Background() // Create a new context for each document in batch
		result, err := h.processDocumentWithPipeline(ctx, individualRequest)
		if err != nil {
			response.FailureCount++
			response.Errors = append(response.Errors, &internalModels.BatchProcessError{
				FileName: file.Filename,
				Error:    err.Error(),
				Code:     "processing_error",
			})
		} else {
			response.SuccessCount++
			response.Results = append(response.Results, result)
		}
	}

	if response.FailureCount > 0 && response.SuccessCount == 0 {
		response.Status = "failed"
	} else if response.FailureCount > 0 {
		response.Status = "partial_success"
	} else {
		response.Status = "completed"
	}

	return response
}

// convertPipelineResults converts pipeline processing results to handler response format
func (h *Handler) convertPipelineResults(pipelineResult *pipeline.ProcessResult, response *internalModels.ProcessDocumentResponse) {
	// Set overall status
	if pipelineResult.Success {
		response.Status = "completed"
	} else {
		response.Status = "failed"
	}

	// Convert processing steps
	for _, step := range pipelineResult.Steps {
		handlerStep := &internalModels.ProcessingStep{
			Name:      string(step.Type),
			Status:    "completed",
			StartTime: step.Timestamp,
			EndTime:   step.Timestamp.Add(time.Duration(step.Duration) * time.Millisecond),
			Duration:  step.Duration,
		}

		if !step.Success {
			handlerStep.Status = "failed"
			handlerStep.Error = step.Error
		}

		response.Steps = append(response.Steps, handlerStep)
	}

	// Convert extraction results
	if pipelineResult.ExtractionResult != nil {
		response.ExtractionResult = &internalModels.ExtractionResult{
			Text:      pipelineResult.ExtractionResult.Text,
			PageCount: pipelineResult.ExtractionResult.PageCount,
			Language:  pipelineResult.ExtractionResult.Language,
		}

		// Update metadata with extraction results
		if response.Metadata != nil {
			response.Metadata.WordCount = pipelineResult.ExtractionResult.WordCount
			response.Metadata.Pages = pipelineResult.ExtractionResult.PageCount
			response.Metadata.Language = pipelineResult.ExtractionResult.Language
		}
	}

	// Convert classification results
	if pipelineResult.ClassificationResult != nil {
		response.ClassificationResult = &internalModels.ClassificationResult{
			Category:   pipelineResult.ClassificationResult.DocumentType,
			Confidence: pipelineResult.ClassificationResult.Confidence,
			Tags:       pipelineResult.ClassificationResult.Keywords,
		}

		// Update metadata with classification results
		if response.Metadata != nil {
			response.Metadata.LegalTags = pipelineResult.ClassificationResult.LegalTags
		}
	}

	// Convert storage results
	if pipelineResult.StorageResult != nil {
		response.StorageResult = &storage.UploadResult{
			URL:        pipelineResult.StorageResult.URL,
			Path:       pipelineResult.StorageResult.StoragePath,
			Size:       0, // Size not available in pipeline result
			Success:    pipelineResult.StorageResult.Success,
			UploadedAt: time.Now(),
		}
		response.URL = pipelineResult.StorageResult.URL
		response.CDN_URL = pipelineResult.StorageResult.URL
	}

	// Convert indexing results
	if pipelineResult.IndexResult != nil {
		response.IndexResult = &internalModels.IndexResult{
			DocumentID: pipelineResult.IndexResult.DocumentID,
			IndexName:  "documents", // Default index name
			Success:    pipelineResult.IndexResult.Success,
		}
	}

	// Set processing metadata
	response.ProcessingTime = pipelineResult.ProcessingTime
	response.CreatedAt = pipelineResult.StartTime
}
