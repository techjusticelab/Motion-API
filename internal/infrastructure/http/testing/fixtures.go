package testing

import (
	"time"

	"motion-index-fiber/internal/application/dto"
)

// DocumentFixtures provides test fixtures for document-related tests
type DocumentFixtures struct{}

// NewDocumentFixtures creates a new document fixtures instance
func NewDocumentFixtures() *DocumentFixtures {
	return &DocumentFixtures{}
}

// ValidProcessRequest returns a valid ProcessDocumentRequest for testing
func (f *DocumentFixtures) ValidProcessRequest() *dto.ProcessDocumentRequest {
	return &dto.ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "test.pdf",
		ContentSize:   1024,
		ContentType:   "application/pdf",
		StoragePath:   "documents/test.pdf",
		HashValue:     "abc123",
		HashAlgorithm: "SHA256",
		StoreBinary:   true,
		Classify:      true,
		Index:         true,
		Metadata: map[string]string{
			"case_name": "Test Case",
		},
	}
}

// ValidProcessResponse returns a valid ProcessDocumentResponse for testing
func (f *DocumentFixtures) ValidProcessResponse() *dto.ProcessDocumentResponse {
	now := time.Now()
	return &dto.ProcessDocumentResponse{
		DocumentID:  "doc_123",
		Stored:      true,
		StorageURL:  "https://storage.example.com/doc_123",
		Classified:  true,
		Indexed:     true,
		CreatedAt:   now,
		ProcessedAt: &now,
	}
}

// ValidBatchRequest returns a valid BatchProcessDocumentRequest for testing
func (f *DocumentFixtures) ValidBatchRequest() *dto.BatchProcessDocumentRequest {
	return &dto.BatchProcessDocumentRequest{
		Documents: []*dto.ProcessDocumentRequest{
			f.ValidProcessRequest(),
		},
	}
}

// ValidIndexRequest returns a valid IndexDocumentRequest for testing
func (f *DocumentFixtures) ValidIndexRequest() *dto.IndexDocumentRequest {
	return &dto.IndexDocumentRequest{
		DocumentID: "doc_123",
		Force:      false,
	}
}

// ValidIndexResponse returns a valid IndexDocumentResponse for testing
func (f *DocumentFixtures) ValidIndexResponse() *dto.IndexDocumentResponse {
	return &dto.IndexDocumentResponse{
		DocumentID: "doc_123",
		Indexed:    true,
	}
}

// ValidUpdateMetadataRequest returns a valid UpdateMetadataRequest for testing
func (f *DocumentFixtures) ValidUpdateMetadataRequest() *dto.UpdateMetadataRequest {
	return &dto.UpdateMetadataRequest{
		DocumentID: "doc_123",
		Language:   "en",
		LegalTags:  []string{"motion", "evidence"},
	}
}

// ClassificationFixtures provides test fixtures for classification tests
type ClassificationFixtures struct{}

// NewClassificationFixtures creates a new classification fixtures instance
func NewClassificationFixtures() *ClassificationFixtures {
	return &ClassificationFixtures{}
}

// ValidClassifyRequest returns a valid ClassifyDocumentRequest for testing
func (f *ClassificationFixtures) ValidClassifyRequest() *dto.ClassifyDocumentRequest {
	return &dto.ClassifyDocumentRequest{
		DocumentID: "doc_123",
		Force:      false,
	}
}

// ValidClassificationResponse returns a valid ClassificationResponse for testing
func (f *ClassificationFixtures) ValidClassificationResponse() *dto.ClassificationResponse {
	return &dto.ClassificationResponse{
		DocumentID:   "doc_123",
		DocumentType: "motion",
		Category:     "criminal",
		Confidence:   0.92,
		LegalTags:    []string{"motion to suppress", "evidence"},
		ClassifiedBy: "openai",
		ClassifiedAt: time.Now(),
	}
}

// RedactionFixtures provides test fixtures for redaction tests
type RedactionFixtures struct{}

// NewRedactionFixtures creates a new redaction fixtures instance
func NewRedactionFixtures() *RedactionFixtures {
	return &RedactionFixtures{}
}

// ValidAnalyzeRequest returns a valid AnalyzeRedactionsRequest for testing
func (f *RedactionFixtures) ValidAnalyzeRequest() *dto.AnalyzeRedactionsRequest {
	return &dto.AnalyzeRedactionsRequest{
		DocumentID: "doc_123",
		Options: &dto.RedactionOptions{
			UseAI:           true,
			CaliforniaLaws:  true,
			ReplacementChar: "■",
		},
	}
}

// ValidAnalyzeResponse returns a valid AnalyzeRedactionsResponse for testing
func (f *RedactionFixtures) ValidAnalyzeResponse() *dto.AnalyzeRedactionsResponse {
	return &dto.AnalyzeRedactionsResponse{
		DocumentID: "doc_123",
		FileName:   "test.pdf",
		Redactions: []dto.RedactionItem{
			{
				ID:        "red_1",
				Page:      1,
				Text:      "SSN: 123-45-6789",
				Type:      "ssn",
				Citation:  "CA Penal Code 293",
				Reason:    "Personal Information",
				LegalCode: "PC293",
				Applied:   false,
			},
		},
		TotalCount: 1,
	}
}

// ValidApplyRequest returns a valid ApplyRedactionsRequest for testing
func (f *RedactionFixtures) ValidApplyRequest() *dto.ApplyRedactionsRequest {
	return &dto.ApplyRedactionsRequest{
		DocumentID: "doc_123",
		Options: &dto.RedactionOptions{
			UseAI:           true,
			CaliforniaLaws:  true,
			ReplacementChar: "■",
		},
		CustomRedactions: []dto.RedactionItem{
			{
				ID:   "red_1",
				Page: 1,
				Type: "ssn",
			},
		},
		ReturnBase64: true,
	}
}

// ValidApplyResponse returns a valid ApplyRedactionsResponse for testing
func (f *RedactionFixtures) ValidApplyResponse() *dto.ApplyRedactionsResponse {
	return &dto.ApplyRedactionsResponse{
		DocumentID: "doc_123",
		FileName:   "redacted_test.pdf",
		Redactions: []dto.RedactionItem{
			{
				ID:      "red_1",
				Page:    1,
				Type:    "ssn",
				Applied: true,
			},
		},
		TotalRedactions: 1,
		PDFBase64:       "base64encodedpdf",
	}
}

// TestFileContent provides common test file content
func TestFileContent() []byte {
	return []byte(`
		MOTION TO SUPPRESS EVIDENCE

		Case No: 2024-CR-12345
		Court: Superior Court of California

		Defendant: John Doe
		Attorney: Jane Smith, Esq.

		The defendant hereby moves to suppress evidence obtained during
		an unlawful search and seizure on January 15, 2024.

		SSN: 123-45-6789
		Phone: (555) 123-4567
		Address: 123 Main St, Anytown, CA 90210
	`)
}

// TestPDFContent provides test PDF file content (minimal valid PDF)
func TestPDFContent() []byte {
	return []byte("%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj 2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj 3 0 obj<</Type/Page/MediaBox[0 0 612 792]/Parent 2 0 R/Resources<<>>>>endobj\nxref\n0 4\n0000000000 65535 f\n0000000009 00000 n\n0000000052 00000 n\n0000000101 00000 n\ntrailer<</Size 4/Root 1 0 R>>\nstartxref\n178\n%%EOF")
}

// TestDocxContent provides test DOCX file signature
func TestDocxContent() []byte {
	// DOCX files are ZIP archives, this is a minimal ZIP signature
	return []byte("PK\x03\x04")
}

// MockUseCaseBuilder provides a fluent API for building mock use cases
type MockUseCaseBuilder struct {
	processResponse        *dto.ProcessDocumentResponse
	processError           error
	classifyResponse       *dto.ClassificationResponse
	classifyError          error
	analyzeResponse        *dto.AnalyzeRedactionsResponse
	analyzeError           error
	applyResponse          *dto.ApplyRedactionsResponse
	applyError             error
	indexResponse          *dto.IndexDocumentResponse
	indexError             error
	updateMetadataResponse *dto.UpdateMetadataResponse
	updateMetadataError    error
}

// NewMockUseCaseBuilder creates a new mock use case builder
func NewMockUseCaseBuilder() *MockUseCaseBuilder {
	return &MockUseCaseBuilder{}
}

// WithProcessSuccess sets up successful process response
func (b *MockUseCaseBuilder) WithProcessSuccess() *MockUseCaseBuilder {
	b.processResponse = NewDocumentFixtures().ValidProcessResponse()
	b.processError = nil
	return b
}

// WithProcessError sets up process error
func (b *MockUseCaseBuilder) WithProcessError(err error) *MockUseCaseBuilder {
	b.processResponse = nil
	b.processError = err
	return b
}

// WithClassifySuccess sets up successful classify response
func (b *MockUseCaseBuilder) WithClassifySuccess() *MockUseCaseBuilder {
	b.classifyResponse = NewClassificationFixtures().ValidClassificationResponse()
	b.classifyError = nil
	return b
}

// WithClassifyError sets up classify error
func (b *MockUseCaseBuilder) WithClassifyError(err error) *MockUseCaseBuilder {
	b.classifyResponse = nil
	b.classifyError = err
	return b
}

// WithAnalyzeSuccess sets up successful analyze response
func (b *MockUseCaseBuilder) WithAnalyzeSuccess() *MockUseCaseBuilder {
	b.analyzeResponse = NewRedactionFixtures().ValidAnalyzeResponse()
	b.analyzeError = nil
	return b
}

// WithAnalyzeError sets up analyze error
func (b *MockUseCaseBuilder) WithAnalyzeError(err error) *MockUseCaseBuilder {
	b.analyzeResponse = nil
	b.analyzeError = err
	return b
}

// ProcessResponse returns the configured process response
func (b *MockUseCaseBuilder) ProcessResponse() (*dto.ProcessDocumentResponse, error) {
	return b.processResponse, b.processError
}

// ClassifyResponse returns the configured classify response
func (b *MockUseCaseBuilder) ClassifyResponse() (*dto.ClassificationResponse, error) {
	return b.classifyResponse, b.classifyError
}

// AnalyzeResponse returns the configured analyze response
func (b *MockUseCaseBuilder) AnalyzeResponse() (*dto.AnalyzeRedactionsResponse, error) {
	return b.analyzeResponse, b.analyzeError
}
