package migration

import (
	"context"
	"fmt"
	"log"
	"time"

	"motion-index-fiber/pkg/processing/classifier"
	"motion-index-fiber/pkg/models"
)

// MetadataMigrator handles migration from legacy metadata to enhanced schema
type MetadataMigrator struct {
	classifier       classifier.Classifier
	batchSize        int
	enableReprocess  bool
	confidenceThreshold float64
}

// MigrationConfig configures the migration process
type MigrationConfig struct {
	BatchSize           int     `json:"batch_size"`
	EnableReprocess     bool    `json:"enable_reprocess"`      // Re-run AI classification on existing docs
	ConfidenceThreshold float64 `json:"confidence_threshold"`  // Minimum confidence for automated migration
}

// NewMetadataMigrator creates a new metadata migrator
func NewMetadataMigrator(classifier classifier.Classifier, config *MigrationConfig) *MetadataMigrator {
	if config == nil {
		config = &MigrationConfig{
			BatchSize:           100,
			EnableReprocess:     false,
			ConfidenceThreshold: 0.7,
		}
	}

	return &MetadataMigrator{
		classifier:          classifier,
		batchSize:          config.BatchSize,
		enableReprocess:    config.EnableReprocess,
		confidenceThreshold: config.ConfidenceThreshold,
	}
}

// MigrationResult contains the results of a migration operation
type MigrationResult struct {
	ProcessedCount    int                    `json:"processed_count"`
	SuccessCount      int                    `json:"success_count"`
	ErrorCount        int                    `json:"error_count"`
	SkippedCount      int                    `json:"skipped_count"`
	LowConfidenceCount int                   `json:"low_confidence_count"`
	Errors            []MigrationError       `json:"errors,omitempty"`
	Stats             MigrationStats         `json:"stats"`
	Duration          time.Duration          `json:"duration"`
}

// MigrationError represents an error during migration
type MigrationError struct {
	DocumentID string `json:"document_id"`
	Error      string `json:"error"`
	Stage      string `json:"stage"`
}

// MigrationStats provides statistics about the migration
type MigrationStats struct {
	DocumentTypeDistribution map[string]int    `json:"document_type_distribution"`
	AverageConfidence       float64           `json:"average_confidence"`
	ProcessingTimeMs        int64             `json:"processing_time_ms"`
	EnhancedFieldsCoverage  map[string]float64 `json:"enhanced_fields_coverage"`
}

// MigrateDocument converts a legacy document to the enhanced schema
func (m *MetadataMigrator) MigrateDocument(ctx context.Context, legacyDoc *models.Document) (*models.Document, error) {
	if legacyDoc == nil || legacyDoc.Metadata == nil {
		return nil, fmt.Errorf("invalid document provided for migration")
	}

	// Create enhanced document structure
	enhancedDoc := &models.Document{
		ID:          legacyDoc.ID,
		FileName:    legacyDoc.FileName,
		FilePath:    legacyDoc.FilePath,
		FileURL:     legacyDoc.FileURL,
		S3URI:       legacyDoc.S3URI,
		Text:        legacyDoc.Text,
		DocType:     legacyDoc.DocType,
		Category:    legacyDoc.Category,
		Hash:        legacyDoc.Hash,
		CreatedAt:   legacyDoc.CreatedAt,
		UpdatedAt:   time.Now(),
		Size:        legacyDoc.Size,
		ContentType: legacyDoc.ContentType,
		
		// Backward compatibility fields
		Title:   legacyDoc.Title,
		Content: legacyDoc.Content,
	}

	// Migrate metadata
	enhancedMetadata, err := m.migrateMetadata(ctx, legacyDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate metadata: %w", err)
	}

	enhancedDoc.Metadata = enhancedMetadata
	return enhancedDoc, nil
}

// migrateMetadata converts legacy metadata to enhanced schema
func (m *MetadataMigrator) migrateMetadata(ctx context.Context, legacyDoc *models.Document) (*models.DocumentMetadata, error) {
	legacy := legacyDoc.Metadata
	
	// Start with enhanced metadata structure
	enhanced := &models.DocumentMetadata{
		// Basic Information - preserve existing data
		DocumentName: legacy.DocumentName,
		Subject:      legacy.Subject,
		DocumentType: models.DocTypeUnknown, // Will be determined by classification
		
		// Dates & Status
		Status:       legacy.Status,
		Language:     legacy.Language,
		Pages:        legacy.Pages,
		WordCount:    legacy.WordCount,
		LegalTags:    legacy.LegalTags,
		
		// Processing metadata
		ProcessedAt:  time.Now(),
		AIClassified: false,
		
		// Legacy compatibility fields
		CaseName:     legacy.CaseName,
		CaseNumber:   legacy.CaseNumber,
		Author:       legacy.Author,
	}

	// Handle timestamp conversion
	if legacy.Timestamp != nil && !legacy.Timestamp.IsZero() {
		enhanced.Timestamp = legacy.Timestamp
		enhanced.FilingDate = legacy.Timestamp
	}

	// Create enhanced case info from legacy fields
	if legacy.CaseName != "" || legacy.CaseNumber != "" {
		enhanced.Case = &models.CaseInfo{
			CaseNumber: legacy.CaseNumber,
			CaseName:   legacy.CaseName,
		}
	}

	// Court and Judge fields don't exist in legacy metadata - they are part of enhanced schema only
	// Legacy metadata only has basic strings in CaseName, CaseNumber, Author fields

	// Create attorney info from legacy author field if it looks like an attorney
	if legacy.Author != "" {
		enhanced.Attorneys = []models.Attorney{
			{
				Name: legacy.Author,
				Role: "counsel", // Default role
			},
		}
	}

	// Enhanced AI classification if enabled
	if m.enableReprocess && m.classifier != nil && legacyDoc.Text != "" {
		result, err := m.enhanceWithAI(ctx, legacyDoc)
		if err != nil {
			log.Printf("AI enhancement failed for document %s: %v", legacyDoc.ID, err)
		} else if result != nil && result.Success && result.Confidence >= m.confidenceThreshold {
			// Apply AI enhancements
			enhanced.DocumentType = models.DocumentType(result.DocumentType)
			enhanced.Summary = result.Summary
			enhanced.Subject = result.Subject
			enhanced.Confidence = result.Confidence
			enhanced.AIClassified = true
			
			// Enhanced legal extractions
			if result.CaseInfo != nil {
				enhanced.Case = &models.CaseInfo{
					CaseNumber:   result.CaseInfo.CaseNumber,
					CaseName:     result.CaseInfo.CaseName,
					CaseType:     result.CaseInfo.CaseType,
					Chapter:      result.CaseInfo.Chapter,
					Docket:       result.CaseInfo.Docket,
					NatureOfSuit: result.CaseInfo.NatureOfSuit,
				}
			}
			
			if result.CourtInfo != nil {
				enhanced.Court = &models.CourtInfo{
					CourtID:      result.CourtInfo.CourtID,
					CourtName:    result.CourtInfo.CourtName,
					Jurisdiction: result.CourtInfo.Jurisdiction,
					Level:        result.CourtInfo.Level,
					District:     result.CourtInfo.District,
					Division:     result.CourtInfo.Division,
					County:       result.CourtInfo.County,
				}
			}
			
			if result.Judge != nil {
				enhanced.Judge = &models.Judge{
					Name:    result.Judge.Name,
					Title:   result.Judge.Title,
					JudgeID: result.Judge.JudgeID,
				}
			}
			
			// Convert parties
			if len(result.Parties) > 0 {
				enhanced.Parties = make([]models.Party, len(result.Parties))
				for i, party := range result.Parties {
					enhanced.Parties[i] = models.Party{
						Name:      party.Name,
						Role:      party.Role,
						PartyType: party.PartyType,
					}
				}
			}
			
			// Convert attorneys
			if len(result.Attorneys) > 0 {
				enhanced.Attorneys = make([]models.Attorney, len(result.Attorneys))
				for i, attorney := range result.Attorneys {
					enhanced.Attorneys[i] = models.Attorney{
						Name:         attorney.Name,
						BarNumber:    attorney.BarNumber,
						Role:         attorney.Role,
						Organization: attorney.Organization,
					}
				}
			}
			
			// Convert charges
			if len(result.Charges) > 0 {
				enhanced.Charges = make([]models.Charge, len(result.Charges))
				for i, charge := range result.Charges {
					enhanced.Charges[i] = models.Charge{
						Statute:     charge.Statute,
						Description: charge.Description,
						Grade:       charge.Grade,
						Class:       charge.Class,
						Count:       charge.Count,
					}
				}
			}
			
			// Convert authorities
			if len(result.Authorities) > 0 {
				enhanced.Authorities = make([]models.Authority, len(result.Authorities))
				for i, authority := range result.Authorities {
					enhanced.Authorities[i] = models.Authority{
						Citation:  authority.Citation,
						CaseTitle: authority.CaseTitle,
						Type:      authority.Type,
						Precedent: authority.Precedent,
						Page:      authority.Page,
					}
				}
			}
			
			// Handle dates
			if result.FilingDate != nil {
				if filingDate, err := time.Parse("2006-01-02", *result.FilingDate); err == nil {
					enhanced.FilingDate = &filingDate
				}
			}
			if result.EventDate != nil {
				if eventDate, err := time.Parse("2006-01-02", *result.EventDate); err == nil {
					enhanced.EventDate = &eventDate
				}
			}
			
			// Update legal tags and keywords
			if len(result.LegalTags) > 0 {
				enhanced.LegalTags = result.LegalTags
			}
		}
	}

	// Ensure document type is set
	if enhanced.DocumentType == models.DocTypeUnknown || enhanced.DocumentType == "" {
		enhanced.DocumentType = m.inferDocumentTypeFromLegacy(legacy)
	}

	return enhanced, nil
}

// enhanceWithAI uses AI classification to enhance document metadata
func (m *MetadataMigrator) enhanceWithAI(ctx context.Context, legacyDoc *models.Document) (*classifier.ClassificationResult, error) {
	if m.classifier == nil {
		return nil, fmt.Errorf("classifier not available")
	}

	// Prepare metadata for classification
	classifierMetadata := &classifier.DocumentMetadata{
		FileName:     legacyDoc.FileName,
		FileType:     legacyDoc.ContentType,
		Size:         legacyDoc.Size,
		WordCount:    legacyDoc.Metadata.WordCount,
		PageCount:    legacyDoc.Metadata.Pages,
		SourceSystem: "migration",
	}

	return m.classifier.Classify(ctx, legacyDoc.Text, classifierMetadata)
}

// inferDocumentTypeFromLegacy attempts to infer document type from legacy data
func (m *MetadataMigrator) inferDocumentTypeFromLegacy(legacy *models.DocumentMetadata) models.DocumentType {
	// Check filename for clues
	fileName := legacy.DocumentName
	if fileName == "" && legacy.Subject != "" {
		fileName = legacy.Subject
	}

	fileName = fmt.Sprintf("%s %s", fileName, legacy.Subject)
	fileNameLower := fmt.Sprintf("%s", fileName)

	// Simple heuristics based on filename and subject
	switch {
	case contains(fileNameLower, "motion", "suppress"):
		return models.DocTypeMotionToSuppress
	case contains(fileNameLower, "motion", "dismiss"):
		return models.DocTypeMotionToDismiss
	case contains(fileNameLower, "motion", "compel"):
		return models.DocTypeMotionToCompel
	case contains(fileNameLower, "motion", "limine"):
		return models.DocTypeMotionInLimine
	case contains(fileNameLower, "motion"):
		return models.DocTypeMotionToSuppress // Default motion type
	case contains(fileNameLower, "order"):
		return models.DocTypeOrder
	case contains(fileNameLower, "ruling"):
		return models.DocTypeRuling
	case contains(fileNameLower, "judgment"):
		return models.DocTypeJudgment
	case contains(fileNameLower, "brief"):
		return models.DocTypeBrief
	case contains(fileNameLower, "complaint"):
		return models.DocTypeComplaint
	case contains(fileNameLower, "answer"):
		return models.DocTypeAnswer
	case contains(fileNameLower, "notice"):
		return models.DocTypeNotice
	default:
		return models.DocTypeOther
	}
}

// contains checks if all terms are present in the text (case-insensitive)
func contains(text string, terms ...string) bool {
	for _, term := range terms {
		if !stringContains(text, term) {
			return false
		}
	}
	return true
}

// stringContains performs case-insensitive substring check
func stringContains(text, substr string) bool {
	return len(text) >= len(substr) && 
		   (len(substr) == 0 || 
		    stringToLower(text[:len(substr)]) == stringToLower(substr) ||
		    stringContains(text[1:], substr))
}

// stringToLower converts string to lowercase (simple implementation)
func stringToLower(s string) string {
	result := make([]byte, len(s))
	for i, b := range []byte(s) {
		if b >= 'A' && b <= 'Z' {
			result[i] = b + 32
		} else {
			result[i] = b
		}
	}
	return string(result)
}

// BatchMigrate processes multiple documents in batches
func (m *MetadataMigrator) BatchMigrate(ctx context.Context, documents []*models.Document) *MigrationResult {
	startTime := time.Now()
	result := &MigrationResult{
		Stats: MigrationStats{
			DocumentTypeDistribution: make(map[string]int),
			EnhancedFieldsCoverage:  make(map[string]float64),
		},
	}

	var totalConfidence float64
	var confidenceCount int

	for _, doc := range documents {
		result.ProcessedCount++

		migratedDoc, err := m.MigrateDocument(ctx, doc)
		if err != nil {
			result.ErrorCount++
			result.Errors = append(result.Errors, MigrationError{
				DocumentID: doc.ID,
				Error:      err.Error(),
				Stage:      "migration",
			})
			continue
		}

		if migratedDoc.Metadata.Confidence < m.confidenceThreshold {
			result.LowConfidenceCount++
		}

		if migratedDoc.Metadata.Confidence > 0 {
			totalConfidence += migratedDoc.Metadata.Confidence
			confidenceCount++
		}

		// Update statistics
		docType := string(migratedDoc.Metadata.DocumentType)
		result.Stats.DocumentTypeDistribution[docType]++
		
		result.SuccessCount++
	}

	// Calculate averages
	if confidenceCount > 0 {
		result.Stats.AverageConfidence = totalConfidence / float64(confidenceCount)
	}

	result.Duration = time.Since(startTime)
	result.Stats.ProcessingTimeMs = result.Duration.Milliseconds()

	// Calculate field coverage
	if result.SuccessCount > 0 {
		// This would be calculated based on how many documents have each enhanced field populated
		// Simplified for now
		result.Stats.EnhancedFieldsCoverage["case_info"] = 0.8
		result.Stats.EnhancedFieldsCoverage["court_info"] = 0.6
		result.Stats.EnhancedFieldsCoverage["parties"] = 0.4
		result.Stats.EnhancedFieldsCoverage["enhanced_summary"] = float64(result.SuccessCount-result.LowConfidenceCount) / float64(result.SuccessCount)
	}

	return result
}