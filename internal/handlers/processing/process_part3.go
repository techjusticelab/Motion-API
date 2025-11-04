package processing

import (
	internalModels "motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/processing/pipeline"
	"motion-index-fiber/pkg/storage"
	"time"
)

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
