package processing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	internalModels "motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/processing/pipeline"
	"time"
)

func (h *Handler) UploadDocument(c *fiber.Ctx) error {
	return h.ProcessDocument(c)
}

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
