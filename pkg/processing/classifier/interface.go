package classifier

import (
	"context"
)

// Classifier defines the interface for document classification
type Classifier interface {
	// Classify analyzes document text and returns classification results
	Classify(ctx context.Context, text string, metadata *DocumentMetadata) (*ClassificationResult, error)

	// GetSupportedCategories returns the categories this classifier can identify
	GetSupportedCategories() []string

	// IsConfigured returns true if the classifier is properly configured
	IsConfigured() bool
}

// Service provides document classification functionality
type Service interface {
	// ClassifyDocument classifies a document and returns the results
	ClassifyDocument(ctx context.Context, text string, metadata *DocumentMetadata) (*ClassificationResult, error)

	// GetAvailableCategories returns all available classification categories
	GetAvailableCategories() []string

	// IsHealthy returns true if the classification service is healthy
	IsHealthy() bool

	// ValidateResult validates classification results (for testing)
	ValidateResult(result *ClassificationResult) error
}

// DocumentMetadata contains information about the document being classified
type DocumentMetadata struct {
	FileName     string            `json:"file_name"`
	FileType     string            `json:"file_type"`
	Size         int64             `json:"size"`
	WordCount    int               `json:"word_count"`
	PageCount    int               `json:"page_count"`
	Properties   map[string]string `json:"properties,omitempty"`
	SourceSystem string            `json:"source_system,omitempty"`
}

// ClassificationResult contains the result of document classification
type ClassificationResult struct {
	DocumentType  string    `json:"document_type"`
	LegalCategory string    `json:"legal_category"`
	SubCategory   string    `json:"sub_category,omitempty"`
	Subject       string    `json:"subject,omitempty"` // Brief subject line
	Summary       string    `json:"summary,omitempty"` // Enhanced legal summary
	Confidence    float64   `json:"confidence"`
	Keywords      []string  `json:"keywords,omitempty"`
	Entities      []*Entity `json:"entities,omitempty"`
	LegalTags     []string  `json:"legal_tags,omitempty"`

	// Enhanced Legal Extraction
	CaseInfo    *CaseInfo   `json:"case_info,omitempty"`
	CourtInfo   *CourtInfo  `json:"court_info,omitempty"`
	Parties     []Party     `json:"parties,omitempty"`
	Attorneys   []Attorney  `json:"attorneys,omitempty"`
	Judge       *Judge      `json:"judge,omitempty"`
	Charges     []Charge    `json:"charges,omitempty"`
	Authorities []Authority `json:"authorities,omitempty"`
	
	// Enhanced Date Fields - ISO date strings (YYYY-MM-DD)
	FilingDate   *string     `json:"filing_date,omitempty"`   // When document was filed with court
	EventDate    *string     `json:"event_date,omitempty"`    // Key event or action date
	HearingDate  *string     `json:"hearing_date,omitempty"`  // Scheduled court hearing date
	DecisionDate *string     `json:"decision_date,omitempty"` // When court decision was made
	ServedDate   *string     `json:"served_date,omitempty"`   // When documents were served
	DateRanges   []DateRange `json:"date_ranges,omitempty"`   // Multi-day events
	
	Status      string      `json:"status,omitempty"`

	// Processing metadata
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Success        bool                   `json:"success"`
	Error          string                 `json:"error,omitempty"`
	ProcessingTime int64                  `json:"processing_time_ms"`
}

// CaseInfo contains case-related information extracted from documents
type CaseInfo struct {
	CaseNumber   string `json:"case_number"`
	CaseName     string `json:"case_name"`
	CaseType     string `json:"case_type,omitempty"` // "criminal", "civil", "traffic"
	Chapter      string `json:"chapter,omitempty"`   // Bankruptcy chapter
	Docket       string `json:"docket,omitempty"`    // Full docket number
	NatureOfSuit string `json:"nature_of_suit,omitempty"`
}

// CourtInfo contains court information extracted from documents
type CourtInfo struct {
	CourtID      string `json:"court_id,omitempty"`
	CourtName    string `json:"court_name"`
	Jurisdiction string `json:"jurisdiction,omitempty"` // "federal", "state", "local"
	Level        string `json:"level,omitempty"`        // "trial", "appellate", "supreme"
	District     string `json:"district,omitempty"`
	Division     string `json:"division,omitempty"`
	County       string `json:"county,omitempty"`
}

// Party represents a party to the case
type Party struct {
	Name      string `json:"name"`
	Role      string `json:"role"`                 // "defendant", "plaintiff", "appellant"
	PartyType string `json:"party_type,omitempty"` // "individual", "corporation", "government"
}

// Attorney represents legal counsel
type Attorney struct {
	Name         string `json:"name"`
	BarNumber    string `json:"bar_number,omitempty"`
	Role         string `json:"role"` // "defense", "prosecution", "counsel"
	Organization string `json:"organization,omitempty"`
}

// Judge represents the presiding judge
type Judge struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	JudgeID string `json:"judge_id,omitempty"`
}

// Charge represents criminal charges
type Charge struct {
	Statute     string `json:"statute"`
	Description string `json:"description"`
	Grade       string `json:"grade,omitempty"` // "felony", "misdemeanor"
	Class       string `json:"class,omitempty"` // "A", "B", "C"
	Count       int    `json:"count,omitempty"` // Count number
}

// Authority represents legal authorities cited
type Authority struct {
	Citation  string `json:"citation"`
	CaseTitle string `json:"case_title,omitempty"`
	Type      string `json:"type"` // "case_law", "statute", "regulation"
	Precedent bool   `json:"precedent"`
	Page      string `json:"page,omitempty"` // Page or section reference
}

// Entity represents a named entity found in the document
type Entity struct {
	Text       string  `json:"text"`
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
	StartPos   int     `json:"start_pos,omitempty"`
	EndPos     int     `json:"end_pos,omitempty"`
}


// Category represents a classification category with confidence
type Category struct {
	Name       string   `json:"name"`
	Confidence float64  `json:"confidence"`
	Tags       []string `json:"tags,omitempty"`
}

// ClassificationError represents errors that occur during classification
type ClassificationError struct {
	Type    string
	Message string
	Cause   error
}

func (e *ClassificationError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *ClassificationError) Unwrap() error {
	return e.Cause
}

// NewClassificationError creates a new classification error
func NewClassificationError(errorType, message string, cause error) *ClassificationError {
	return &ClassificationError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// Classification constants
const (
	// Enhanced Document types - aligned with models.DocumentType
	DocumentTypeMotionToSuppress         = "motion_to_suppress"
	DocumentTypeMotionToDismiss          = "motion_to_dismiss"
	DocumentTypeMotionToCompel           = "motion_to_compel"
	DocumentTypeMotionInLimine           = "motion_in_limine"
	DocumentTypeMotionForSummaryJudgment = "motion_summary_judgment"
	DocumentTypeMotionToStrike           = "motion_to_strike"
	DocumentTypeMotionForReconsideration = "motion_for_reconsideration"
	DocumentTypeMotionToAmend            = "motion_to_amend"
	DocumentTypeMotionForContinuance     = "motion_for_continuance"
	DocumentTypeOrder                    = "order"
	DocumentTypeRuling                   = "ruling"
	DocumentTypeJudgment                 = "judgment"
	DocumentTypeSentence                 = "sentence"
	DocumentTypeInjunction               = "injunction"
	DocumentTypeBrief                    = "brief"
	DocumentTypeComplaint                = "complaint"
	DocumentTypeAnswer                   = "answer"
	DocumentTypePlea                     = "plea"
	DocumentTypeReply                    = "reply"
	DocumentTypeDocketEntry              = "docket_entry"
	DocumentTypeNotice                   = "notice"
	DocumentTypeStipulation              = "stipulation"
	DocumentTypeCorrespondence           = "correspondence"
	DocumentTypeTranscript               = "transcript"
	DocumentTypeEvidence                 = "evidence"
	DocumentTypeOther                    = "other"

	// Legal categories
	LegalCategoryCriminal       = "Criminal Law"
	LegalCategoryCivil          = "Civil Law"
	LegalCategoryContract       = "Contract Law"
	LegalCategoryFamily         = "Family Law"
	LegalCategoryProperty       = "Property Law"
	LegalCategoryEmployment     = "Employment Law"
	LegalCategoryIntellectual   = "Intellectual Property"
	LegalCategoryTax            = "Tax Law"
	LegalCategoryBankruptcy     = "Bankruptcy"
	LegalCategoryPersonalInjury = "Personal Injury"

	// Entity types
	EntityTypePerson        = "PERSON"
	EntityTypeOrganization  = "ORGANIZATION"
	EntityTypeLocation      = "LOCATION"
	EntityTypeDate          = "DATE"
	EntityTypeMoney         = "MONEY"
	EntityTypeLegalCitation = "LEGAL_CITATION"
	EntityTypeCaseNumber    = "CASE_NUMBER"
	EntityTypeStatute       = "STATUTE"
)

// GetDefaultCategories returns the default legal categories
func GetDefaultCategories() []string {
	return []string{
		LegalCategoryCriminal,
		LegalCategoryCivil,
		LegalCategoryContract,
		LegalCategoryFamily,
		LegalCategoryProperty,
		LegalCategoryEmployment,
		LegalCategoryIntellectual,
		LegalCategoryTax,
		LegalCategoryBankruptcy,
		LegalCategoryPersonalInjury,
	}
}

// GetDefaultDocumentTypes returns the enhanced document types
func GetDefaultDocumentTypes() []string {
	return []string{
		// Motions
		DocumentTypeMotionToSuppress,
		DocumentTypeMotionToDismiss,
		DocumentTypeMotionToCompel,
		DocumentTypeMotionInLimine,
		DocumentTypeMotionForSummaryJudgment,
		DocumentTypeMotionToStrike,
		DocumentTypeMotionForReconsideration,
		DocumentTypeMotionToAmend,
		DocumentTypeMotionForContinuance,
		// Orders and Rulings
		DocumentTypeOrder,
		DocumentTypeRuling,
		DocumentTypeJudgment,
		DocumentTypeSentence,
		DocumentTypeInjunction,
		// Briefs and Pleadings
		DocumentTypeBrief,
		DocumentTypeComplaint,
		DocumentTypeAnswer,
		DocumentTypePlea,
		DocumentTypeReply,
		// Administrative
		DocumentTypeDocketEntry,
		DocumentTypeNotice,
		DocumentTypeStipulation,
		DocumentTypeCorrespondence,
		DocumentTypeTranscript,
		DocumentTypeEvidence,
		DocumentTypeOther,
	}
}
