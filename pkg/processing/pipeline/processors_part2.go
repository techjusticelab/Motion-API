package pipeline

import (
	"context"
	"fmt"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/processing/classifier"
	"motion-index-fiber/pkg/storage"
	"strings"
	"time"
)

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
		// Generate S3 URI in the format: s3://bucket-name/path
		// Note: In a production system, we'd get bucket name from configuration
		// For now, we'll extract it from the storage URL if available
		if storageURL, urlExists := req.Metadata["storage_url"]; urlExists {
			// Try to extract bucket name from URL like: https://bucket.region.digitaloceanspaces.com/path
			if idx := strings.Index(storageURL, ".digitaloceanspaces.com/"); idx > 0 {
				// Extract bucket name from URL
				bucketPart := storageURL[8:idx] // Skip "https://"
				if dotIdx := strings.Index(bucketPart, "."); dotIdx > 0 {
					bucketName := bucketPart[:dotIdx]
					doc.S3URI = fmt.Sprintf("s3://%s/%s", bucketName, storagePath)
				}
			}
		}
		// Fallback: if we couldn't extract bucket name, use path only
		if doc.S3URI == "" {
			doc.S3URI = fmt.Sprintf("s3://unknown-bucket/%s", storagePath)
		}
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

func (p *indexingProcessor) GetType() ProcessorType {
	return ProcessorTypeIndexing
}

func (p *indexingProcessor) IsHealthy() bool {
	return p.service != nil && p.service.IsHealthy()
}

type storageProcessor struct {
	service storage.Service
}

func NewStorageProcessor(service storage.Service) Processor {
	return &storageProcessor{
		service: service,
	}
}

func (p *storageProcessor) Process(ctx context.Context, req *ProcessRequest) (*ProcessResult, error) {
	if p.service == nil {
		return nil, fmt.Errorf("storage service not available")
	}

	// Generate storage path where the document will be/is stored
	storagePath := p.generateStoragePath(req.FileName, req.ID)

	// Generate the public URL (this matches what SpacesService.GetURL() returns)
	// Note: In production, bucket and region should come from configuration
	// For now, we'll use a placeholder that will be replaced by the actual storage URL
	url := p.service.GetURL(storagePath)

	return &ProcessResult{
		ID: req.ID,
		StorageResult: &StorageResult{
			StoragePath: storagePath,
			URL:         url,
			Success:     true,
		},
	}, nil
}

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

func (p *storageProcessor) GetType() ProcessorType {
	return ProcessorTypeStorage
}

func (p *storageProcessor) IsHealthy() bool {
	return p.service != nil
}

type validationProcessor struct{}

func NewValidationProcessor() Processor {
	return &validationProcessor{}
}

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

func (p *validationProcessor) GetType() ProcessorType {
	return ProcessorTypeValidation
}

func (p *validationProcessor) IsHealthy() bool {
	return true
}

func convertCaseInfo(classifierCase *classifier.CaseInfo) *models.CaseInfo {
	if classifierCase == nil {
		return nil
	}
	return &models.CaseInfo{
		CaseNumber:   classifierCase.CaseNumber,
		CaseName:     classifierCase.CaseName,
		CaseType:     classifierCase.CaseType,
		Chapter:      classifierCase.Chapter,
		Docket:       classifierCase.Docket,
		NatureOfSuit: classifierCase.NatureOfSuit,
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
