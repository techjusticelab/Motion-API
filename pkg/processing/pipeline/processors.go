package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"motion-index-fiber/pkg/processing/classifier"
	"motion-index-fiber/pkg/processing/extractor"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/storage"
)

// extractionProcessor handles text extraction
type extractionProcessor struct {
	service extractor.Service
	config  *Config
}

// NewExtractionProcessor creates a new extraction processor
func NewExtractionProcessor(service extractor.Service, config *Config) Processor {
	return &extractionProcessor{
		service: service,
		config:  config,
	}
}

// Process executes text extraction
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

// GetType returns the processor type
func (p *extractionProcessor) GetType() ProcessorType {
	return ProcessorTypeExtraction
}

// IsHealthy returns true if the processor is healthy
func (p *extractionProcessor) IsHealthy() bool {
	return p.service != nil
}

// classificationProcessor handles document classification
type classificationProcessor struct {
	service classifier.Service
}

// NewClassificationProcessor creates a new classification processor
func NewClassificationProcessor(service classifier.Service) Processor {
	return &classificationProcessor{
		service: service,
	}
}

// Process executes document classification
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

// GetType returns the processor type
func (p *classificationProcessor) GetType() ProcessorType {
	return ProcessorTypeClassification
}

// IsHealthy returns true if the processor is healthy
func (p *classificationProcessor) IsHealthy() bool {
	return p.service != nil && p.service.IsHealthy()
}

// indexingProcessor handles document indexing
type indexingProcessor struct {
	service search.Service
}

// NewIndexingProcessor creates a new indexing processor
func NewIndexingProcessor(service search.Service) Processor {
	return &indexingProcessor{
		service: service,
	}
}

// Process executes document indexing
// This function receives a ProcessRequest with accumulated results from previous pipeline steps
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

// ProcessWithFullResult executes document indexing with access to full ProcessResult
// This allows the indexing processor to access the complete ClassificationResult
func (p *indexingProcessor) ProcessWithFullResult(ctx context.Context, req *ProcessRequest, fullResult *ProcessResult) (*ProcessResult, error) {
	if p.service == nil {
		return nil, fmt.Errorf("search service not available")
	}

	// Extract data from previous processing steps
	extractedText := req.Metadata["extracted_text"]
	if extractedText == "" {
		extractedText = "No text extracted"
	}

	// Create document for indexing with all collected data
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

	// Use full ClassificationResult if available (THIS IS THE KEY FIX)
	if fullResult != nil && fullResult.ClassificationResult != nil {
		classResult := fullResult.ClassificationResult
		
		// Map core classification fields
		doc.DocType = classResult.DocumentType
		doc.Category = classResult.LegalCategory
		
		// Map DocumentType to metadata as well
		if classResult.DocumentType != "" {
			doc.Metadata.DocumentType = models.ParseDocumentType(classResult.DocumentType)
		}
		
		// Properly map both Subject and Summary fields
		if classResult.Subject != "" {
			doc.Metadata.Subject = classResult.Subject
		}
		if classResult.Summary != "" {
			doc.Metadata.Summary = classResult.Summary
			// If no explicit subject, use summary as fallback for subject
			if doc.Metadata.Subject == "" {
				doc.Metadata.Subject = classResult.Summary
			}
		}
		if classResult.Status != "" {
			doc.Metadata.Status = classResult.Status
		}
		doc.Metadata.Confidence = classResult.Confidence
		doc.Metadata.AIClassified = classResult.Confidence > 0.5

		// Map all date fields from ClassificationResult
		if classResult.FilingDate != nil {
			if parsedDate, err := time.Parse("2006-01-02", *classResult.FilingDate); err == nil {
				doc.Metadata.FilingDate = &parsedDate
			}
		}
		if classResult.EventDate != nil {
			if parsedDate, err := time.Parse("2006-01-02", *classResult.EventDate); err == nil {
				doc.Metadata.EventDate = &parsedDate
			}
		}
		if classResult.HearingDate != nil {
			if parsedDate, err := time.Parse("2006-01-02", *classResult.HearingDate); err == nil {
				doc.Metadata.HearingDate = &parsedDate
			}
		}
		if classResult.DecisionDate != nil {
			if parsedDate, err := time.Parse("2006-01-02", *classResult.DecisionDate); err == nil {
				doc.Metadata.DecisionDate = &parsedDate
			}
		}
		if classResult.ServedDate != nil {
			if parsedDate, err := time.Parse("2006-01-02", *classResult.ServedDate); err == nil {
				doc.Metadata.ServedDate = &parsedDate
			}
		}

		// Map complex legal entities (THIS FIXES THE MISSING FIELDS ISSUE)
		// Note: We need to convert between classifier types and models types
		if classResult.CaseInfo != nil {
			doc.Metadata.Case = convertCaseInfo(classResult.CaseInfo)
		}
		if classResult.CourtInfo != nil {
			doc.Metadata.Court = convertCourtInfo(classResult.CourtInfo)
		}
		
		// Always initialize arrays even if empty to ensure consistent structure
		doc.Metadata.Parties = convertParties(classResult.Parties)
		doc.Metadata.Attorneys = convertAttorneys(classResult.Attorneys)
		doc.Metadata.Charges = convertCharges(classResult.Charges)
		doc.Metadata.Authorities = convertAuthorities(classResult.Authorities)
		
		// Map LegalTags array (copy directly as it's already []string)
		if classResult.LegalTags != nil {
			doc.Metadata.LegalTags = classResult.LegalTags
		} else {
			doc.Metadata.LegalTags = []string{} // Initialize empty slice
		}
		
		if classResult.Judge != nil {
			doc.Metadata.Judge = convertJudge(classResult.Judge)
		}
	} else {
		// Fallback to string metadata parsing (for backwards compatibility)
		if documentType, exists := req.Metadata["document_type"]; exists {
			doc.DocType = documentType
		} else {
			doc.DocType = "Other"
		}
		if legalCategory, exists := req.Metadata["legal_category"]; exists {
			doc.Category = legalCategory
		} else {
			doc.Category = "Civil"
		}
		if subCategory, exists := req.Metadata["sub_category"]; exists {
			doc.Metadata.Subject = subCategory
		}
		if summary, exists := req.Metadata["summary"]; exists && doc.Metadata.Subject == "" {
			doc.Metadata.Subject = summary
		}
		
		// Parse date fields from string metadata
		if filingDateStr, exists := req.Metadata["filing_date"]; exists {
			if parsedDate, err := time.Parse("2006-01-02", filingDateStr); err == nil {
				doc.Metadata.FilingDate = &parsedDate
			}
		}
		if eventDateStr, exists := req.Metadata["event_date"]; exists {
			if parsedDate, err := time.Parse("2006-01-02", eventDateStr); err == nil {
				doc.Metadata.EventDate = &parsedDate
			}
		}
		if hearingDateStr, exists := req.Metadata["hearing_date"]; exists {
			if parsedDate, err := time.Parse("2006-01-02", hearingDateStr); err == nil {
				doc.Metadata.HearingDate = &parsedDate
			}
		}
		if decisionDateStr, exists := req.Metadata["decision_date"]; exists {
			if parsedDate, err := time.Parse("2006-01-02", decisionDateStr); err == nil {
				doc.Metadata.DecisionDate = &parsedDate
			}
		}
		if servedDateStr, exists := req.Metadata["served_date"]; exists {
			if parsedDate, err := time.Parse("2006-01-02", servedDateStr); err == nil {
				doc.Metadata.ServedDate = &parsedDate
			}
		}
		
		if status, exists := req.Metadata["status"]; exists {
			doc.Metadata.Status = status
		}
		if subject, exists := req.Metadata["subject"]; exists {
			doc.Metadata.Subject = subject
		}
		if confidence, exists := req.Metadata["confidence"]; exists {
			if conf, err := fmt.Sscanf(confidence, "%f", &doc.Metadata.Confidence); err == nil && conf == 1 {
				doc.Metadata.AIClassified = true
			}
		}
	}

	// Add storage metadata
	if storagePath, exists := req.Metadata["storage_path"]; exists {
		doc.FilePath = storagePath
	}
	if storageURL, exists := req.Metadata["storage_url"]; exists {
		doc.FileURL = storageURL
	}

	// Set processing timestamp (remove redundant timestamp field)
	now := time.Now()
	doc.Metadata.ProcessedAt = now
	
	// Populate legacy fields for backward compatibility
	doc.Metadata.SetLegacyFields()

	// Index document
	docID, err := p.service.IndexDocument(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("document indexing failed: %w", err)
	}

	return &ProcessResult{
		ID: req.ID,
		IndexResult: &IndexResult{
			DocumentID: docID,
			Success:    true,
		},
		Document: doc,
	}, nil
}

// GetType returns the processor type
func (p *indexingProcessor) GetType() ProcessorType {
	return ProcessorTypeIndexing
}

// IsHealthy returns true if the processor is healthy
func (p *indexingProcessor) IsHealthy() bool {
	return p.service != nil && p.service.IsHealthy()
}

// storageProcessor handles document storage
type storageProcessor struct {
	service storage.Service
}

// NewStorageProcessor creates a new storage processor
func NewStorageProcessor(service storage.Service) Processor {
	return &storageProcessor{
		service: service,
	}
}

// Process executes document storage
func (p *storageProcessor) Process(ctx context.Context, req *ProcessRequest) (*ProcessResult, error) {
	if p.service == nil {
		return nil, fmt.Errorf("storage service not available")
	}

	// Generate storage path
	storagePath := p.generateStoragePath(req.FileName, req.ID)

	// Store document (this is a placeholder implementation)
	// In a real implementation, you would upload the document to storage
	url := fmt.Sprintf("https://storage.example.com/%s", storagePath)

	return &ProcessResult{
		ID: req.ID,
		StorageResult: &StorageResult{
			StoragePath: storagePath,
			URL:         url,
			Success:     true,
		},
	}, nil
}

// generateStoragePath generates a storage path for the document
func (p *storageProcessor) generateStoragePath(fileName, docID string) string {
	// Create a path based on document ID and filename
	// Format: documents/{year}/{month}/{docID}/{filename}
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")

	// Clean filename
	cleanName := strings.ReplaceAll(fileName, " ", "_")
	cleanName = strings.ReplaceAll(cleanName, "/", "_")

	return fmt.Sprintf("documents/%s/%s/%s/%s", year, month, docID, cleanName)
}

// GetType returns the processor type
func (p *storageProcessor) GetType() ProcessorType {
	return ProcessorTypeStorage
}

// IsHealthy returns true if the processor is healthy
func (p *storageProcessor) IsHealthy() bool {
	return p.service != nil
}

// validationProcessor handles document validation (optional)
type validationProcessor struct{}

// NewValidationProcessor creates a new validation processor
func NewValidationProcessor() Processor {
	return &validationProcessor{}
}

// Process executes document validation
func (p *validationProcessor) Process(ctx context.Context, req *ProcessRequest) (*ProcessResult, error) {
	// Validate file size
	if req.Size > 100*1024*1024 { // 100MB limit
		return nil, fmt.Errorf("file size too large: %d bytes", req.Size)
	}

	// Validate file type
	allowedTypes := map[string]bool{
		"application/pdf":  true,
		"application/docx": true,
		"text/plain":       true,
	}

	if !allowedTypes[req.ContentType] {
		return nil, fmt.Errorf("unsupported file type: %s", req.ContentType)
	}

	// Validate filename
	if req.FileName == "" {
		return nil, fmt.Errorf("filename is required")
	}

	return &ProcessResult{
		ID: req.ID,
	}, nil
}

// GetType returns the processor type
func (p *validationProcessor) GetType() ProcessorType {
	return ProcessorTypeValidation
}

// IsHealthy returns true if the processor is healthy
func (p *validationProcessor) IsHealthy() bool {
	return true
}

// Converter functions to transform classifier types to models types
// These functions handle the type conversion between classifier package and models package

func convertCaseInfo(classifierCase *classifier.CaseInfo) *models.CaseInfo {
	if classifierCase == nil {
		return nil
	}
	return &models.CaseInfo{
		CaseNumber:    classifierCase.CaseNumber,
		CaseName:      classifierCase.CaseName,
		CaseType:      classifierCase.CaseType,
		Chapter:       classifierCase.Chapter,
		Docket:        classifierCase.Docket,
		NatureOfSuit:  classifierCase.NatureOfSuit,
	}
}

func convertCourtInfo(classifierCourt *classifier.CourtInfo) *models.CourtInfo {
	if classifierCourt == nil {
		return nil
	}
	
	courtInfo := &models.CourtInfo{
		CourtName:    classifierCourt.CourtName,
		Jurisdiction: classifierCourt.Jurisdiction,
		Level:        classifierCourt.Level,
		District:     classifierCourt.District,
		Division:     classifierCourt.Division,
		County:       classifierCourt.County,
	}
	
	// Only set CourtID if it's not empty
	if classifierCourt.CourtID != "" {
		courtInfo.CourtID = classifierCourt.CourtID
	}
	
	return courtInfo
}

func convertParties(classifierParties []classifier.Party) []models.Party {
	if len(classifierParties) == 0 {
		return []models.Party{} // Return empty slice instead of nil
	}
	parties := make([]models.Party, len(classifierParties))
	for i, party := range classifierParties {
		parties[i] = models.Party{
			Name:      party.Name,
			Role:      party.Role,
			PartyType: party.PartyType,
			Date:      nil, // classifier.Party doesn't provide date info
		}
	}
	return parties
}

func convertAttorneys(classifierAttorneys []classifier.Attorney) []models.Attorney {
	if len(classifierAttorneys) == 0 {
		return []models.Attorney{} // Return empty slice instead of nil
	}
	attorneys := make([]models.Attorney, len(classifierAttorneys))
	for i, attorney := range classifierAttorneys {
		attorneys[i] = models.Attorney{
			Name:         attorney.Name,
			BarNumber:    attorney.BarNumber,
			Role:         attorney.Role,
			Organization: attorney.Organization,
			ContactInfo:  "", // classifier.Attorney doesn't provide contact info
		}
	}
	return attorneys
}

func convertJudge(classifierJudge *classifier.Judge) *models.Judge {
	if classifierJudge == nil || classifierJudge.Name == "" {
		return nil // Don't create Judge object if name is empty
	}
	return &models.Judge{
		Name:    classifierJudge.Name,
		Title:   classifierJudge.Title,
		JudgeID: classifierJudge.JudgeID,
	}
}

func convertCharges(classifierCharges []classifier.Charge) []models.Charge {
	if len(classifierCharges) == 0 {
		return []models.Charge{} // Return empty slice instead of nil
	}
	charges := make([]models.Charge, len(classifierCharges))
	for i, charge := range classifierCharges {
		charges[i] = models.Charge{
			Statute:     charge.Statute,
			Description: charge.Description,
			Grade:       charge.Grade,
			Class:       charge.Class,
			Count:       charge.Count,
		}
	}
	return charges
}

func convertAuthorities(classifierAuthorities []classifier.Authority) []models.Authority {
	if len(classifierAuthorities) == 0 {
		return []models.Authority{} // Return empty slice instead of nil
	}
	authorities := make([]models.Authority, len(classifierAuthorities))
	for i, authority := range classifierAuthorities {
		authorities[i] = models.Authority{
			Citation:  authority.Citation,
			CaseTitle: authority.CaseTitle,
			Type:      authority.Type,
			Precedent: authority.Precedent,
			Page:      authority.Page,
		}
	}
	return authorities
}
