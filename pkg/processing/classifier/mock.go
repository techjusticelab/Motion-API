package classifier

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// mockClassifier provides a mock implementation for testing
type mockClassifier struct {
	configured bool
}

// NewMockClassifier creates a new mock classifier
func NewMockClassifier() Classifier {
	return &mockClassifier{
		configured: true,
	}
}

// Classify provides mock classification results for testing
func (m *mockClassifier) Classify(ctx context.Context, text string, metadata *DocumentMetadata) (*ClassificationResult, error) {
	// Simulate processing time based on document size
	processingTime := m.calculateProcessingTime(metadata)
	time.Sleep(time.Duration(processingTime) * time.Millisecond)

	// Determine document type based on text content and metadata
	documentType := m.determineDocumentType(text, metadata)
	legalCategory := m.determineLegalCategory(text)

	// Extract mock entities with metadata context
	entities := m.extractMockEntities(text, metadata)

	// Generate mock keywords
	keywords := m.extractKeywords(text)

	// Generate mock legal tags
	legalTags := m.generateLegalTags(documentType, legalCategory)

	// Generate enhanced summary using metadata
	summary := m.generateEnhancedSummary(text, documentType, metadata)
	subject := m.generateSubject(documentType, metadata)

	// Generate mock case info based on document characteristics
	caseInfo := m.generateMockCaseInfo(text, metadata)
	courtInfo := m.generateMockCourtInfo(text, metadata)
	parties := m.generateMockParties(text, metadata)
	attorneys := m.generateMockAttorneys(text, metadata)
	judge := m.generateMockJudge(text, metadata)

	return &ClassificationResult{
		DocumentType:  documentType,
		LegalCategory: legalCategory,
		SubCategory:   m.getSubCategory(legalCategory),
		Subject:       subject,
		Summary:       summary,
		Confidence:    m.calculateConfidence(text, metadata),
		Keywords:      keywords,
		Entities:      entities,
		LegalTags:     legalTags,
		CaseInfo:      caseInfo,
		CourtInfo:     courtInfo,
		Parties:       parties,
		Attorneys:     attorneys,
		Judge:         judge,
		FilingDate:    stringPtr("2024-01-15"),
		Status:        "filed",
		Metadata: map[string]interface{}{
			"classifier":    "mock",
			"version":       "2.0",
			"model":         "enhanced-mock-classifier",
			"word_count":    getMetadataInt(metadata, "word_count"),
			"page_count":    getMetadataInt(metadata, "page_count"),
			"processing_ms": processingTime,
		},
	}, nil
}

// GetSupportedCategories returns the categories this classifier can identify
func (m *mockClassifier) GetSupportedCategories() []string {
	return GetDefaultCategories()
}

// IsConfigured returns true if the classifier is properly configured
func (m *mockClassifier) IsConfigured() bool {
	return m.configured
}

// determineDocumentType analyzes text and metadata to determine document type
func (m *mockClassifier) determineDocumentType(text string, metadata *DocumentMetadata) string {
	text = strings.ToLower(text)

	switch {
	case strings.Contains(text, "motion") && strings.Contains(text, "suppress"):
		return DocumentTypeMotionToSuppress
	case strings.Contains(text, "motion") && strings.Contains(text, "dismiss"):
		return DocumentTypeMotionToDismiss
	case strings.Contains(text, "motion") && strings.Contains(text, "compel"):
		return DocumentTypeMotionToCompel
	case strings.Contains(text, "motion") && strings.Contains(text, "limine"):
		return DocumentTypeMotionInLimine
	case strings.Contains(text, "motion") && strings.Contains(text, "summary"):
		return DocumentTypeMotionForSummaryJudgment
	case strings.Contains(text, "motion"):
		return DocumentTypeMotionToSuppress // Default motion type
	case strings.Contains(text, "order") && strings.Contains(text, "court"):
		return DocumentTypeOrder
	case strings.Contains(text, "ruling"):
		return DocumentTypeRuling
	case strings.Contains(text, "judgment"):
		return DocumentTypeJudgment
	case strings.Contains(text, "brief") || strings.Contains(text, "memorandum"):
		return DocumentTypeBrief
	case strings.Contains(text, "complaint") || strings.Contains(text, "plaintiff"):
		return DocumentTypeComplaint
	case strings.Contains(text, "answer") && strings.Contains(text, "defendant"):
		return DocumentTypeAnswer
	case strings.Contains(text, "plea"):
		return DocumentTypePlea
	case strings.Contains(text, "notice"):
		return DocumentTypeNotice
	case strings.Contains(text, "letter") || strings.Contains(text, "correspondence"):
		return DocumentTypeCorrespondence
	default:
		return m.determineTypeByMetadata(metadata)
	}
}

// determineTypeByMetadata uses metadata characteristics to help determine document type
func (m *mockClassifier) determineTypeByMetadata(metadata *DocumentMetadata) string {
	if metadata == nil {
		return DocumentTypeOther
	}
	
	wordCount := metadata.WordCount
	pageCount := metadata.PageCount
	
	// Use document characteristics to make educated guesses
	switch {
	case wordCount < 200: // Very short documents
		return DocumentTypeNotice
	case wordCount < 500: // Short documents
		return DocumentTypeCorrespondence
	case wordCount > 5000: // Long documents
		return DocumentTypeBrief
	case pageCount == 1 && wordCount < 1000: // Single page, moderate length
		return DocumentTypeOrder
	case pageCount > 10: // Multi-page documents
		return DocumentTypeBrief
	default:
		return DocumentTypeOther
	}
}

// determineLegalCategory analyzes text to determine legal category
func (m *mockClassifier) determineLegalCategory(text string) string {
	text = strings.ToLower(text)

	switch {
	case strings.Contains(text, "criminal") || strings.Contains(text, "prosecution"):
		return LegalCategoryCriminal
	case strings.Contains(text, "contract") || strings.Contains(text, "breach"):
		return LegalCategoryContract
	case strings.Contains(text, "family") || strings.Contains(text, "divorce") || strings.Contains(text, "custody"):
		return LegalCategoryFamily
	case strings.Contains(text, "property") || strings.Contains(text, "real estate"):
		return LegalCategoryProperty
	case strings.Contains(text, "employment") || strings.Contains(text, "workplace"):
		return LegalCategoryEmployment
	case strings.Contains(text, "intellectual property") || strings.Contains(text, "patent") || strings.Contains(text, "trademark"):
		return LegalCategoryIntellectual
	case strings.Contains(text, "tax") || strings.Contains(text, "irs"):
		return LegalCategoryTax
	case strings.Contains(text, "bankruptcy") || strings.Contains(text, "chapter 7") || strings.Contains(text, "chapter 11"):
		return LegalCategoryBankruptcy
	case strings.Contains(text, "personal injury") || strings.Contains(text, "negligence"):
		return LegalCategoryPersonalInjury
	default:
		return LegalCategoryCivil
	}
}

// getSubCategory returns a subcategory based on the main category
func (m *mockClassifier) getSubCategory(category string) string {
	switch category {
	case LegalCategoryCriminal:
		return "Felony"
	case LegalCategoryContract:
		return "Commercial"
	case LegalCategoryFamily:
		return "Divorce"
	case LegalCategoryProperty:
		return "Commercial Real Estate"
	case LegalCategoryEmployment:
		return "Wrongful Termination"
	default:
		return ""
	}
}

// calculateConfidence returns a mock confidence score based on text and metadata
func (m *mockClassifier) calculateConfidence(text string, metadata *DocumentMetadata) float64 {
	// Base confidence on text length and content
	baseConfidence := 0.7

	if len(text) > 1000 {
		baseConfidence += 0.1
	}
	if len(text) > 5000 {
		baseConfidence += 0.1
	}

	// Look for specific legal terms
	legalTerms := []string{"court", "plaintiff", "defendant", "motion", "order", "contract"}
	foundTerms := 0
	text = strings.ToLower(text)

	for _, term := range legalTerms {
		if strings.Contains(text, term) {
			foundTerms++
		}
	}

	termBonus := float64(foundTerms) * 0.05
	confidence := baseConfidence + termBonus

	if confidence > 0.95 {
		confidence = 0.95
	}

	// Adjust confidence based on metadata quality
	if metadata != nil {
		// Higher confidence for documents with good metadata
		if metadata.WordCount > 0 && metadata.PageCount > 0 {
			confidence += 0.05
		}
		
		// Lower confidence for very short or very long documents
		if metadata.WordCount < 100 || metadata.WordCount > 20000 {
			confidence -= 0.1
		}
	}
	
	if confidence > 0.95 {
		confidence = 0.95
	}
	if confidence < 0.3 {
		confidence = 0.3
	}
	
	return confidence
}

// extractKeywords finds key legal terms in the text
func (m *mockClassifier) extractKeywords(text string) []string {
	keywords := []string{}
	text = strings.ToLower(text)

	commonLegalTerms := []string{
		"plaintiff", "defendant", "court", "judge", "jury", "motion", "order",
		"contract", "agreement", "breach", "damages", "liability", "negligence",
		"evidence", "discovery", "deposition", "trial", "settlement", "appeal",
	}

	for _, term := range commonLegalTerms {
		if strings.Contains(text, term) {
			keywords = append(keywords, term)
		}
	}

	// Limit to 10 keywords
	if len(keywords) > 10 {
		keywords = keywords[:10]
	}

	return keywords
}

// extractMockEntities finds mock entities in the text using metadata context
func (m *mockClassifier) extractMockEntities(text string, metadata *DocumentMetadata) []*Entity {
	entities := []*Entity{}

	// Look for common patterns
	if strings.Contains(strings.ToLower(text), "john") {
		entities = append(entities, &Entity{
			Text:       "John Smith",
			Type:       EntityTypePerson,
			Confidence: 0.9,
		})
	}

	if strings.Contains(strings.ToLower(text), "court") {
		entities = append(entities, &Entity{
			Text:       "Superior Court",
			Type:       EntityTypeOrganization,
			Confidence: 0.8,
		})
	}

	if strings.Contains(text, "$") {
		entities = append(entities, &Entity{
			Text:       "$10,000",
			Type:       EntityTypeMoney,
			Confidence: 0.7,
		})
	}

	// Add metadata-based entities
	if metadata != nil {
		if metadata.FileName != "" {
			entities = append(entities, &Entity{
				Text:       metadata.FileName,
				Type:       "DOCUMENT",
				Confidence: 1.0,
			})
		}
		
		if metadata.WordCount > 0 {
			entities = append(entities, &Entity{
				Text:       fmt.Sprintf("%d words", metadata.WordCount),
				Type:       "DOCUMENT_METRIC",
				Confidence: 1.0,
			})
		}
	}
	
	return entities
}

// generateLegalTags creates legal tags based on document type and category
func (m *mockClassifier) generateLegalTags(documentType, legalCategory string) []string {
	tags := []string{}

	// Add tags based on document type
	switch documentType {
	case DocumentTypeMotionToSuppress, DocumentTypeMotionToDismiss, DocumentTypeMotionToCompel, DocumentTypeMotionInLimine:
		tags = append(tags, "Pre-Trial", "Motion Practice")
	case DocumentTypeOrder:
		tags = append(tags, "Court Order", "Judicial Decision")
	case DocumentTypeBrief:
		tags = append(tags, "Legal Argument", "Case Law")
	case DocumentTypeComplaint, DocumentTypeAnswer:
		tags = append(tags, "Pleadings", "Case Initiation")
	}

	// Add tags based on legal category
	switch legalCategory {
	case LegalCategoryCriminal:
		tags = append(tags, "Criminal Procedure", "Constitutional Law")
	case LegalCategoryContract:
		tags = append(tags, "Commercial Law", "Business Disputes")
	case LegalCategoryFamily:
		tags = append(tags, "Domestic Relations", "Child Custody")
	}

	return tags
}

// generateEnhancedSummary creates a comprehensive summary using metadata
func (m *mockClassifier) generateEnhancedSummary(text, documentType string, metadata *DocumentMetadata) string {
	baseSummary := m.generateSummary(text, documentType)
	
	if metadata == nil {
		return baseSummary
	}
	
	// Enhance summary with metadata context
	wordCount := metadata.WordCount
	pageCount := metadata.PageCount
	
	enhancement := ""
	switch {
	case wordCount < 300:
		enhancement = " This brief document contains essential case information."
	case wordCount > 3000:
		enhancement = " This comprehensive document provides detailed legal analysis and extensive case documentation."
	case pageCount > 5:
		enhancement = fmt.Sprintf(" This multi-page document (%d pages) contains substantial legal content and procedural details.", pageCount)
	default:
		enhancement = fmt.Sprintf(" Document contains %d words across %d pages with relevant case details.", wordCount, pageCount)
	}
	
	return baseSummary + enhancement
}

// generateSummary creates a brief summary of the document
func (m *mockClassifier) generateSummary(text, documentType string) string {
	switch documentType {
	case DocumentTypeMotionToSuppress:
		return "Motion requesting exclusion of evidence from trial proceedings."
	case DocumentTypeMotionToDismiss:
		return "Motion seeking dismissal of charges or claims for legal insufficiency."
	case DocumentTypeMotionToCompel:
		return "Motion requesting court order to compel compliance with discovery requests."
	case DocumentTypeMotionInLimine:
		return "Pre-trial motion seeking to exclude prejudicial evidence or testimony."
	case DocumentTypeOrder:
		return "Court order directing parties to take or refrain from specific actions."
	case DocumentTypeRuling:
		return "Judicial decision on a matter before the court."
	case DocumentTypeBrief:
		return "Legal brief presenting arguments and case law analysis."
	case DocumentTypeComplaint:
		return "Initial pleading setting forth plaintiff's claims and requested relief."
	case DocumentTypeAnswer:
		return "Defendant's response to allegations in complaint or petition."
	default:
		return "Legal document containing case-related information and proceedings."
	}
}

// New helper functions for enhanced mock classification

// calculateProcessingTime simulates realistic processing time based on document size
func (m *mockClassifier) calculateProcessingTime(metadata *DocumentMetadata) int {
	baseTime := 10 // Base processing time in milliseconds
	
	if metadata == nil {
		return baseTime
	}
	
	// Simulate longer processing for larger documents
	wordCount := metadata.WordCount
	switch {
	case wordCount < 500:
		return baseTime
	case wordCount < 2000:
		return baseTime + 20
	case wordCount < 5000:
		return baseTime + 50
	default:
		return baseTime + 100
	}
}

// generateSubject creates a document subject line
func (m *mockClassifier) generateSubject(documentType string, metadata *DocumentMetadata) string {
	switch documentType {
	case DocumentTypeMotionToSuppress:
		return "Motion to Suppress Evidence - Criminal Case"
	case DocumentTypeMotionToDismiss:
		return "Motion to Dismiss Charges - Legal Insufficiency"
	case DocumentTypeOrder:
		return "Court Order - Judicial Ruling"
	case DocumentTypeBrief:
		return "Legal Brief - Case Analysis"
	default:
		return "Legal Document - Case Proceeding"
	}
}

// generateMockCaseInfo creates mock case information
func (m *mockClassifier) generateMockCaseInfo(text string, metadata *DocumentMetadata) *CaseInfo {
	return &CaseInfo{
		CaseNumber:   "CR-2024-001234",
		CaseName:     "People v. Smith",
		CaseType:     "criminal",
		Docket:       "Superior Court Case CR-2024-001234",
		NatureOfSuit: "Criminal prosecution",
	}
}

// generateMockCourtInfo creates mock court information
func (m *mockClassifier) generateMockCourtInfo(text string, metadata *DocumentMetadata) *CourtInfo {
	return &CourtInfo{
		CourtName:    "Superior Court of California",
		Jurisdiction: "state",
		Level:        "trial",
		County:       "Los Angeles",
	}
}

// generateMockParties creates mock party information
func (m *mockClassifier) generateMockParties(text string, metadata *DocumentMetadata) []Party {
	return []Party{
		{
			Name:      "John Smith",
			Role:      "defendant",
			PartyType: "individual",
		},
		{
			Name:      "People of the State of California",
			Role:      "plaintiff",
			PartyType: "government",
		},
	}
}

// generateMockAttorneys creates mock attorney information
func (m *mockClassifier) generateMockAttorneys(text string, metadata *DocumentMetadata) []Attorney {
	return []Attorney{
		{
			Name:         "Jane Doe",
			Role:         "defense",
			Organization: "Public Defender's Office",
		},
		{
			Name:         "Michael Johnson",
			Role:         "prosecution",
			Organization: "District Attorney's Office",
		},
	}
}

// generateMockJudge creates mock judge information
func (m *mockClassifier) generateMockJudge(text string, metadata *DocumentMetadata) *Judge {
	return &Judge{
		Name:  "Hon. Sarah Wilson",
		Title: "Superior Court Judge",
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func getMetadataInt(metadata *DocumentMetadata, field string) int {
	if metadata == nil {
		return 0
	}
	switch field {
	case "word_count":
		return metadata.WordCount
	case "page_count":
		return metadata.PageCount
	default:
		return 0
	}
}
