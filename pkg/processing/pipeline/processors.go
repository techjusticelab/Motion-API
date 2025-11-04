package pipeline

import (
	"context"
	"fmt"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/processing/classifier"
	"motion-index-fiber/pkg/processing/extractor"
	"motion-index-fiber/pkg/search"
	"time"
)

type extractionProcessor struct {
	service extractor.Service
	config  *Config
}

func NewExtractionProcessor(service extractor.Service, config *Config) Processor {
	return &extractionProcessor{
		service: service,
		config:  config,
	}
}

func (p *extractionProcessor) Process(ctx context.Context, req *ProcessRequest) (*ProcessResult, error) {
	if p.service == nil {
		return nil, fmt.Errorf("extraction service not available")
	}

	// Create extraction metadata
	metadata := &extractor.DocumentMetadata{
		FileName:   req.FileName,
		MimeType:   req.ContentType,
		Size:       req.Size,
		Properties: make(map[string]string),
	}

	// Add PDF page limit configuration
	if p.config != nil && p.config.MaxPDFPages > 0 {
		metadata.Properties["max_pdf_pages"] = fmt.Sprintf("%d", p.config.MaxPDFPages)
	}

	// Extract text
	result, err := p.service.ExtractText(ctx, req.Content, metadata)
	if err != nil {
		return nil, fmt.Errorf("text extraction failed: %w", err)
	}

	return &ProcessResult{
		ID:               req.ID,
		ExtractionResult: result,
	}, nil
}

func (p *extractionProcessor) GetType() ProcessorType {
	return ProcessorTypeExtraction
}

func (p *extractionProcessor) IsHealthy() bool {
	return p.service != nil
}

type classificationProcessor struct {
	service classifier.Service
}

func NewClassificationProcessor(service classifier.Service) Processor {
	return &classificationProcessor{
		service: service,
	}
}

func (p *classificationProcessor) Process(ctx context.Context, req *ProcessRequest) (*ProcessResult, error) {
	if p.service == nil {
		return nil, fmt.Errorf("classification service not available")
	}

	// Get extracted text from request context or extract it ourselves
	var text string
	var wordCount, pageCount int

	// Check if we have extracted text in the metadata
	if extractedText, exists := req.Metadata["extracted_text"]; exists {
		text = extractedText
		if wc, exists := req.Metadata["word_count"]; exists {
			fmt.Sscanf(wc, "%d", &wordCount)
		}
		if pc, exists := req.Metadata["page_count"]; exists {
			fmt.Sscanf(pc, "%d", &pageCount)
		}
	} else {
		// Extract text first using basic extractor
		extractorSvc := extractor.NewService()
		extractorMetadata := &extractor.DocumentMetadata{
			FileName:   req.FileName,
			MimeType:   req.ContentType,
			Size:       req.Size,
			Properties: make(map[string]string),
		}

		// Apply default PDF page limit
		extractorMetadata.Properties["max_pdf_pages"] = "25"

		extractionResult, err := extractorSvc.ExtractText(ctx, req.Content, extractorMetadata)
		if err != nil {
			return nil, fmt.Errorf("failed to extract text for classification: %w", err)
		}

		text = extractionResult.Text
		wordCount = extractionResult.WordCount
		pageCount = extractionResult.PageCount
	}

	// Create classification metadata
	metadata := &classifier.DocumentMetadata{
		FileName:  req.FileName,
		FileType:  req.ContentType,
		Size:      req.Size,
		WordCount: wordCount,
		PageCount: pageCount,
	}

	// Classify document
	result, err := p.service.ClassifyDocument(ctx, text, metadata)
	if err != nil {
		return nil, fmt.Errorf("document classification failed: %w", err)
	}

	return &ProcessResult{
		ID:                   req.ID,
		ClassificationResult: result,
	}, nil
}

func (p *classificationProcessor) GetType() ProcessorType {
	return ProcessorTypeClassification
}

func (p *classificationProcessor) IsHealthy() bool {
	return p.service != nil && p.service.IsHealthy()
}

type indexingProcessor struct {
	service search.Service
}

func NewIndexingProcessor(service search.Service) Processor {
	return &indexingProcessor{
		service: service,
	}
}

func (p *indexingProcessor) Process(ctx context.Context, req *ProcessRequest) (*ProcessResult, error) {
	if p.service == nil {
		return nil, fmt.Errorf("search service not available")
	}

	// NOTE: This processor is called as part of the pipeline and needs to reconstruct
	// the full ProcessResult from the request metadata. However, in the pipeline context,
	// we don't have direct access to the ClassificationResult here.
	// The actual fix needs to be in the pipeline execution where this processor is called.

	// Extract data from previous processing steps
	extractedText := req.Metadata["extracted_text"]
	if extractedText == "" {
		extractedText = "No text extracted"
	}

	// Create document for indexing with all collected data
	// This will be populated with full classification results by the pipeline
	doc := &models.Document{
		ID:          req.ID,
		FileName:    req.FileName,
		FilePath:    req.ID, // Use ID as file path since documents are already stored
		ContentType: req.ContentType,
		Size:        req.Size,
		Text:        extractedText,
		Hash:        fmt.Sprintf("hash_%s", req.ID), // Generate a basic hash
		Metadata:    &models.DocumentMetadata{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Populate metadata from processing results
	doc.Metadata.DocumentName = req.FileName

	// DEPRECATED: This method now delegates to ProcessWithFullResult for better metadata handling
	// For backwards compatibility, we'll call ProcessWithFullResult with a nil fullResult
	return p.ProcessWithFullResult(ctx, req, nil)
}
