package ai

import (
	"context"
	"errors"
	"testing"

	"motion-index-fiber/internal/domain/document"
	"motion-index-fiber/pkg/processing/classifier"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockClassifier is a mock implementation of classifier.Classifier
type mockClassifier struct {
	mock.Mock
}

func (m *mockClassifier) Classify(ctx context.Context, text string, metadata *classifier.DocumentMetadata) (*classifier.ClassificationResult, error) {
	args := m.Called(ctx, text, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*classifier.ClassificationResult), args.Error(1)
}

func (m *mockClassifier) GetSupportedCategories() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *mockClassifier) IsConfigured() bool {
	args := m.Called()
	return args.Bool(0)
}

func TestNewClassificationAdapter(t *testing.T) {
	mockClassifier := new(mockClassifier)
	adapter := NewClassificationAdapter(mockClassifier, "openai")

	assert.NotNil(t, adapter)
	assert.Equal(t, mockClassifier, adapter.classifier)
	assert.Equal(t, "openai", adapter.provider)
}

func TestClassificationAdapter_Classify_Success(t *testing.T) {
	mockClassifier := new(mockClassifier)
	adapter := NewClassificationAdapter(mockClassifier, "openai")

	ctx := context.Background()

	// Create domain document with text
	doc, err := document.NewDocument(
		document.WithID("doc_123"),
		document.WithFileName("motion.pdf"),
		document.WithStoragePath("documents/motion.pdf"),
		document.WithContentType("application/pdf"),
		document.WithHash("a1b2c3d4e5f60123456789abcdef0123456789abcdef0123456789abcdef0123", "SHA256"),
		document.WithFileSize(2048),
		document.WithText("Motion to suppress evidence"),
	)
	require.NoError(t, err)

	// Mock classifier response
	classifierResult := &classifier.ClassificationResult{
		DocumentType:  "motion",
		LegalCategory: "filing",
		Confidence:    0.95,
		LegalTags:     []string{"criminal", "suppression"},
		Summary:       "Motion to suppress evidence",
	}

	mockClassifier.On("IsConfigured").Return(true)
	mockClassifier.On("Classify", ctx, "Motion to suppress evidence", mock.AnythingOfType("*classifier.DocumentMetadata")).
		Return(classifierResult, nil)

	result, err := adapter.Classify(ctx, doc)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "motion", result.DocumentType().String())
	assert.Equal(t, "filing", result.Category().String())
	assert.Equal(t, 0.95, result.Confidence().Value())
	assert.Equal(t, []string{"criminal", "suppression"}, result.LegalTags())
	mockClassifier.AssertExpectations(t)
}

func TestClassificationAdapter_Classify_NotConfigured(t *testing.T) {
	mockClassifier := new(mockClassifier)
	adapter := NewClassificationAdapter(mockClassifier, "openai")

	ctx := context.Background()

	doc, err := document.NewDocument(
		document.WithID("doc_123"),
		document.WithFileName("motion.pdf"),
		document.WithStoragePath("documents/motion.pdf"),
		document.WithContentType("application/pdf"),
		document.WithHash("a1b2c3d4e5f60123456789abcdef0123456789abcdef0123456789abcdef0123", "SHA256"),
		document.WithFileSize(2048),
		document.WithText("Motion to suppress evidence"),
	)
	require.NoError(t, err)

	mockClassifier.On("IsConfigured").Return(false)

	result, err := adapter.Classify(ctx, doc)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not configured")
	mockClassifier.AssertExpectations(t)
}

func TestClassificationAdapter_Classify_NoText(t *testing.T) {
	mockClassifier := new(mockClassifier)
	adapter := NewClassificationAdapter(mockClassifier, "openai")

	ctx := context.Background()

	// Create document without text
	doc, err := document.NewDocument(
		document.WithID("doc_123"),
		document.WithFileName("motion.pdf"),
		document.WithStoragePath("documents/motion.pdf"),
		document.WithContentType("application/pdf"),
		document.WithHash("a1b2c3d4e5f60123456789abcdef0123456789abcdef0123456789abcdef0123", "SHA256"),
		document.WithFileSize(2048),
	)
	require.NoError(t, err)

	mockClassifier.On("IsConfigured").Return(true)

	result, err := adapter.Classify(ctx, doc)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no text to classify")
	mockClassifier.AssertExpectations(t)
}

func TestClassificationAdapter_Classify_ClassifierError(t *testing.T) {
	mockClassifier := new(mockClassifier)
	adapter := NewClassificationAdapter(mockClassifier, "openai")

	ctx := context.Background()

	doc, err := document.NewDocument(
		document.WithID("doc_123"),
		document.WithFileName("motion.pdf"),
		document.WithStoragePath("documents/motion.pdf"),
		document.WithContentType("application/pdf"),
		document.WithHash("a1b2c3d4e5f60123456789abcdef0123456789abcdef0123456789abcdef0123", "SHA256"),
		document.WithFileSize(2048),
		document.WithText("Motion to suppress evidence"),
	)
	require.NoError(t, err)

	expectedErr := errors.New("API rate limit exceeded")
	mockClassifier.On("IsConfigured").Return(true)
	mockClassifier.On("Classify", ctx, "Motion to suppress evidence", mock.AnythingOfType("*classifier.DocumentMetadata")).
		Return(nil, expectedErr)

	result, err := adapter.Classify(ctx, doc)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "AI provider")
	mockClassifier.AssertExpectations(t)
}

func TestClassificationAdapter_Classify_InvalidConfidence(t *testing.T) {
	mockClassifier := new(mockClassifier)
	adapter := NewClassificationAdapter(mockClassifier, "openai")

	ctx := context.Background()

	doc, err := document.NewDocument(
		document.WithID("doc_123"),
		document.WithFileName("motion.pdf"),
		document.WithStoragePath("documents/motion.pdf"),
		document.WithContentType("application/pdf"),
		document.WithHash("a1b2c3d4e5f60123456789abcdef0123456789abcdef0123456789abcdef0123", "SHA256"),
		document.WithFileSize(2048),
		document.WithText("Motion to suppress evidence"),
	)
	require.NoError(t, err)

	// Mock classifier response with invalid confidence
	classifierResult := &classifier.ClassificationResult{
		DocumentType:  "motion",
		LegalCategory: "filing",
		Confidence:    1.5, // Invalid: > 1.0
		LegalTags:     []string{"criminal"},
		Summary:       "Motion to suppress",
	}

	mockClassifier.On("IsConfigured").Return(true)
	mockClassifier.On("Classify", ctx, "Motion to suppress evidence", mock.AnythingOfType("*classifier.DocumentMetadata")).
		Return(classifierResult, nil)

	// Should succeed but fallback to safe confidence value
	result, err := adapter.Classify(ctx, doc)

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Confidence should be reset to 0.5 (fallback)
	assert.Equal(t, 0.5, result.Confidence().Value())
	mockClassifier.AssertExpectations(t)
}

func TestClassificationAdapter_SupportedDocumentTypes(t *testing.T) {
	mockClassifier := new(mockClassifier)
	adapter := NewClassificationAdapter(mockClassifier, "openai")

	expectedTypes := []string{"motion", "brief", "order", "complaint"}
	mockClassifier.On("GetSupportedCategories").Return(expectedTypes)

	types := adapter.SupportedDocumentTypes()

	assert.Equal(t, expectedTypes, types)
	mockClassifier.AssertExpectations(t)
}

func TestClassificationAdapter_ProviderName(t *testing.T) {
	mockClassifier := new(mockClassifier)
	adapter := NewClassificationAdapter(mockClassifier, "claude")

	name := adapter.ProviderName()

	assert.Equal(t, "claude", name)
}

func TestConvertToDomainResult_NilResult(t *testing.T) {
	result, err := convertToDomainResult(nil, "openai")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "classification result is nil")
}

func TestConvertToDomainResult_InvalidDocumentType(t *testing.T) {
	classifierResult := &classifier.ClassificationResult{
		DocumentType:  "", // Invalid
		LegalCategory: "filing",
		Confidence:    0.95,
		LegalTags:     []string{"criminal"},
		Summary:       "Test summary",
	}

	result, err := convertToDomainResult(classifierResult, "openai")

	// Should succeed with fallback to "other"
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "other", result.DocumentType().String())
}

func TestConvertToDomainResult_InvalidCategory(t *testing.T) {
	classifierResult := &classifier.ClassificationResult{
		DocumentType:  "motion",
		LegalCategory: "", // Invalid
		Confidence:    0.95,
		LegalTags:     []string{"criminal"},
		Summary:       "Test summary",
	}

	result, err := convertToDomainResult(classifierResult, "openai")

	// Should succeed with fallback to "other"
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "other", result.Category().String())
}

func TestConvertToDomainResult_Success(t *testing.T) {
	classifierResult := &classifier.ClassificationResult{
		DocumentType:  "motion",
		LegalCategory: "filing",
		Confidence:    0.95,
		LegalTags:     []string{"criminal", "suppression"},
		Summary:       "Motion to suppress evidence",
	}

	result, err := convertToDomainResult(classifierResult, "openai")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "motion", result.DocumentType().String())
	assert.Equal(t, "filing", result.Category().String())
	assert.Equal(t, 0.95, result.Confidence().Value())
	assert.Equal(t, []string{"criminal", "suppression"}, result.LegalTags())
	assert.Equal(t, "Motion to suppress evidence", result.Explanation())
	assert.Equal(t, "openai", result.Provider())
}
