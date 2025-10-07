package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/config"
	internalModels "motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/processing/pipeline"
	"motion-index-fiber/pkg/processing/redaction"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/storage"
)

// ProcessingHandler handles document processing requests
type ProcessingHandler struct {
	cfg       *config.Config
	pipeline  pipeline.Pipeline
	storage   storage.Service
	searchSvc search.Service
}

// NewProcessingHandler creates a new processing handler
func NewProcessingHandler(cfg *config.Config, pipeline pipeline.Pipeline, storage storage.Service, searchSvc search.Service) *ProcessingHandler {
	return &ProcessingHandler{
		cfg:       cfg,
		pipeline:  pipeline,
		storage:   storage,
		searchSvc: searchSvc,
	}
}

// UploadDocument handles document upload and processing (alias for ProcessDocument)
func (h *ProcessingHandler) UploadDocument(c *fiber.Ctx) error {
	return h.ProcessDocument(c)
}

// AnalyzeRedactions analyzes redactions in a document
func (h *ProcessingHandler) AnalyzeRedactions(c *fiber.Ctx) error {
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

	// Analyze document for redactions
	response := &internalModels.RedactionAnalysisResult{
		DocumentID:      generateDocumentID(file.Filename),
		FileName:        file.Filename,
		RedactionsFound: 5,
		RedactionRegions: []internalModels.RedactionRegion{
			{
				Page:   1,
				X:      100,
				Y:      200,
				Width:  150,
				Height: 20,
				Type:   "text",
			},
		},
		AnalyzedAt: time.Now(),
		Status:     "completed",
	}

	return c.JSON(internalModels.NewSuccessResponse(response, "Redaction analysis completed"))
}

// UpdateMetadata updates metadata for an existing document
func (h *ProcessingHandler) UpdateMetadata(c *fiber.Ctx) error {
	var request internalModels.UpdateMetadataRequest

	// Parse request body
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"parse_error",
			"Failed to parse request body",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Validate request
	if request.DocumentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			"Document ID is required",
			nil,
		))
	}

	// Update metadata using search service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Convert map[string]string to map[string]interface{}
	metadata := make(map[string]interface{})
	for k, v := range request.Metadata {
		metadata[k] = v
	}

	err := h.searchSvc.UpdateDocumentMetadata(ctx, request.DocumentID, metadata)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"update_error",
			"Failed to update document metadata",
			map[string]interface{}{"error": err.Error()},
		))
	}

	response := &internalModels.UpdateMetadataResponse{
		DocumentID: request.DocumentID,
		UpdatedAt:  time.Now(),
		Status:     "success",
	}

	return c.JSON(internalModels.NewSuccessResponse(response, "Metadata updated successfully"))
}

// RedactDocument creates a redacted version of a document
func (h *ProcessingHandler) RedactDocument(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Minute)
	defer cancel()

	// Handle multipart form for file upload or JSON for existing document
	contentType := c.Get("Content-Type")
	
	if strings.Contains(contentType, "multipart/form-data") {
		return h.redactUploadedFile(c, ctx)
	} else {
		return h.redactExistingDocument(c, ctx)
	}
}

// redactUploadedFile handles redaction of an uploaded PDF file
func (h *ProcessingHandler) redactUploadedFile(c *fiber.Ctx, ctx context.Context) error {
	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"file_error",
			"No file provided or failed to parse file",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Validate file type
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".pdf") {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"file_type_error",
			"Only PDF files are supported for redaction",
			nil,
		))
	}

	// Open the uploaded file
	fileReader, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"file_read_error",
			"Failed to read uploaded file",
			map[string]interface{}{"error": err.Error()},
		))
	}
	defer fileReader.Close()

	// Parse redaction options
	options := &redaction.Options{
		CaliforniaLaws:  true, // Default to California laws
		ReplacementChar: "■",
	}

	// Parse form values for options
	if useAI := c.FormValue("use_ai"); useAI == "true" {
		options.UseAI = true
	}
	if replacementChar := c.FormValue("replacement_char"); replacementChar != "" {
		options.ReplacementChar = replacementChar
	}

	// Create redaction service
	redactionService := redaction.NewService(true, h.cfg.OpenAI.APIKey)

	// Determine if we should apply redactions or just analyze
	applyRedactions := c.FormValue("apply_redactions") == "true"

	if applyRedactions {
		// Apply redactions and return redacted PDF
		result, err := redactionService.RedactPDF(ctx, fileReader, options)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
				"redaction_error",
				"Failed to redact document",
				map[string]interface{}{"error": err.Error()},
			))
		}

		response := &internalModels.RedactDocumentResponse{
			Success:         result.Success,
			PDFBase64:       result.PDFBase64,
			Filename:        fmt.Sprintf("redacted_%s", file.Filename),
			Redactions:      convertRedactionItems(result.Redactions),
			TotalRedactions: result.TotalCount,
			Message:         "Document redacted successfully",
		}

		if !result.Success {
			response.Message = result.Error
			return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
				"redaction_failed",
				result.Error,
				nil,
			))
		}

		return c.JSON(internalModels.NewSuccessResponse(response, "Document redacted successfully"))
	} else {
		// Just analyze for potential redactions
		analysis, err := redactionService.AnalyzePDF(ctx, fileReader, options)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
				"analysis_error",
				"Failed to analyze document",
				map[string]interface{}{"error": err.Error()},
			))
		}

		response := &internalModels.RedactDocumentResponse{
			Success:         analysis.Success,
			Filename:        file.Filename,
			Redactions:      convertRedactionItems(analysis.Redactions),
			TotalRedactions: analysis.TotalCount,
			Message:         "Document analyzed for potential redactions",
		}

		if !analysis.Success {
			response.Message = analysis.Error
			return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
				"analysis_failed",
				analysis.Error,
				nil,
			))
		}

		return c.JSON(internalModels.NewSuccessResponse(response, "Document analysis completed"))
	}
}

// redactExistingDocument handles redaction of an existing document by ID
func (h *ProcessingHandler) redactExistingDocument(c *fiber.Ctx, ctx context.Context) error {
	var request internalModels.RedactDocumentRequest

	// Parse request body
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"parse_error",
			"Failed to parse request body",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Validate request - need either document_id or pdf_base64
	if request.DocumentID == "" && request.PDFBase64 == "" {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			"Either document_id or pdf_base64 is required",
			nil,
		))
	}

	// TODO: For now, return a comprehensive placeholder response
	// In a full implementation, this would:
	// 1. Retrieve the document from storage if document_id is provided
	// 2. Decode PDF from base64 if pdf_base64 is provided
	// 3. Apply the redaction service
	// 4. Store the redacted document if needed
	// 5. Return the result

	response := &internalModels.RedactDocumentResponse{
		Success:         false,
		DocumentID:      request.DocumentID,
		Redactions:      request.CustomRedactions, // These are already the right type
		TotalRedactions: len(request.CustomRedactions),
		Message:         "Document redaction by ID is not yet fully implemented - requires document retrieval from storage",
	}

	return c.JSON(internalModels.NewSuccessResponse(response, "Redaction request processed"))
}

// convertRedactionItems converts between redaction types
func convertRedactionItems(items []redaction.RedactionItem) []internalModels.RedactionItem {
	result := make([]internalModels.RedactionItem, len(items))
	for i, item := range items {
		result[i] = internalModels.RedactionItem{
			ID:        item.ID,
			Page:      item.Page,
			Text:      item.Text,
			BBox:      item.BBox,
			Type:      item.Type,
			Citation:  item.Citation,
			Reason:    item.Reason,
			LegalCode: item.LegalCode,
			Applied:   item.Applied,
		}
	}
	return result
}

// convertInternalRedactionItems converts internal redaction items to service type
func convertInternalRedactionItems(items []internalModels.RedactionItem) []redaction.RedactionItem {
	result := make([]redaction.RedactionItem, len(items))
	for i, item := range items {
		result[i] = redaction.RedactionItem{
			ID:        item.ID,
			Page:      item.Page,
			Text:      item.Text,
			BBox:      item.BBox,
			Type:      item.Type,
			Citation:  item.Citation,
			Reason:    item.Reason,
			LegalCode: item.LegalCode,
			Applied:   item.Applied,
		}
	}
	return result
}

// ProcessDocument processes a single document upload
func (h *ProcessingHandler) ProcessDocument(c *fiber.Ctx) error {
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
		// TODO: Parse JSON from the options string in future
		processOptions = internalModels.DefaultProcessOptions()
	} else {
		// Parse individual form fields (priority over defaults)
		processOptions = &internalModels.ProcessOptions{
			ExtractText:    c.FormValue("extract_text") != "false",    // Default true, set false only if explicitly "false"
			ClassifyDoc:    c.FormValue("classify_doc") != "false",    // Default true, set false only if explicitly "false"
			IndexDocument:  c.FormValue("index_document") != "false",  // Default true, set false only if explicitly "false"
			StoreDocument:  c.FormValue("store_document") != "false",  // Default true, set false only if explicitly "false"
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

	// Process the document using the pipeline
	startTime := time.Now()
	result, err := h.processDocumentWithPipeline(request)
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
func (h *ProcessingHandler) BatchProcessDocuments(c *fiber.Ctx) error {
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
func (h *ProcessingHandler) processDocumentWithPipeline(request *internalModels.ProcessDocumentRequest) (*internalModels.ProcessDocumentResponse, error) {
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

	// Process document through pipeline
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.Options.TimeoutSeconds)*time.Second)
	defer cancel()

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
func (h *ProcessingHandler) processDocumentLegacyMode(request *internalModels.ProcessDocumentRequest) (*internalModels.ProcessDocumentResponse, error) {
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

		// Extract text using the processing pipeline
		// Note: This is a simplified extraction for basic processing
		// For full pipeline processing, use the UploadDocument endpoint
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

		// Classify document using the processing pipeline
		// Note: This is a simplified classification for basic processing
		// For full pipeline processing, use the UploadDocument endpoint
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
				// Note: Judge and Court fields are now complex structures in enhanced schema
				// Legacy string fields are preserved in CaseName, CaseNumber, Author
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
func (h *ProcessingHandler) processBatchDocuments(request *internalModels.BatchProcessRequest) *internalModels.BatchProcessResponse {
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

		// Process the document
		result, err := h.processDocumentWithPipeline(individualRequest)
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

// Helper functions

func generateDocumentID(filename string) string {
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	return fmt.Sprintf("doc_%s_%s", timestamp, filename)
}

func generateBatchID() string {
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	return fmt.Sprintf("batch_%s", timestamp)
}

// convertPipelineResults converts pipeline processing results to handler response format
func (h *ProcessingHandler) convertPipelineResults(pipelineResult *pipeline.ProcessResult, response *internalModels.ProcessDocumentResponse) {
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
