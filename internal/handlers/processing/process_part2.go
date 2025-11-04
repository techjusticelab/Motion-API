package processing

import (
	"context"
	"fmt"
	internalModels "motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/storage"
	"time"
)

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
