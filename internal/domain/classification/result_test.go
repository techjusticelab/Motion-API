package classification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"motion-index-fiber/internal/domain/document"
	domainerrors "motion-index-fiber/internal/domain/errors"
)

func TestInputHasContent(t *testing.T) {
	input := Input{Text: "content"}
	assert.True(t, input.HasContent())

	input = Input{}
	assert.False(t, input.HasContent())
}

func TestNewResultSuccess(t *testing.T) {
	docType, err := document.NewDocumentType("motion")
	require.NoError(t, err)
	category, err := document.NewCategory("filing")
	require.NoError(t, err)
	confidence, err := document.NewConfidence(0.9)
	require.NoError(t, err)

	result, err := NewResult(docType, category, confidence, "classifier", []string{"Motion", "  Filing "}, "  example ")
	require.NoError(t, err)

	assert.Equal(t, docType, result.DocumentType())
	assert.Equal(t, category, result.Category())
	assert.Equal(t, confidence, result.Confidence())
	assert.ElementsMatch(t, []string{"Motion", "Filing"}, result.LegalTags())
	assert.Equal(t, "classifier", result.Provider())
	assert.Equal(t, "example", result.Explanation())
	assert.False(t, result.ClassifiedAt().IsZero())

	classification, err := result.ToDocumentClassification()
	require.NoError(t, err)
	assert.Equal(t, docType, classification.DocumentType())
	assert.Equal(t, category, classification.Category())
	assert.Equal(t, confidence, classification.Confidence())
}

func TestNewResultValidationErrors(t *testing.T) {
	category, _ := document.NewCategory("filing")
	confidence, _ := document.NewConfidence(0.9)

	_, err := NewResult(document.DocumentType{}, category, confidence, "classifier", nil, "")
	assert.ErrorIs(t, err, domainerrors.ErrInvalidClassification)

	docType, _ := document.NewDocumentType("motion")
	_, err = NewResult(docType, document.Category{}, confidence, "classifier", nil, "")
	assert.ErrorIs(t, err, domainerrors.ErrInvalidClassification)

	invalidConfidence := document.Confidence(1.5)
	_, err = NewResult(docType, category, invalidConfidence, "classifier", nil, "")
	assert.ErrorIs(t, err, domainerrors.ErrInvalidConfidence)
}

// compile-time check ensuring Service can be implemented without extra methods.
type serviceMock struct{}

func (serviceMock) Classify(ctx context.Context, input Input) (*Result, error) { return nil, nil }
func (serviceMock) Health(ctx context.Context) error                           { return nil }

func TestServiceInterface(t *testing.T) {
	var svc Service = serviceMock{}
	assert.NotNil(t, svc)
}

func TestResultLegalTagsEmpty(t *testing.T) {
	docType, _ := document.NewDocumentType("motion")
	category, _ := document.NewCategory("filing")
	confidence, _ := document.NewConfidence(0.9)

	result, err := NewResult(docType, category, confidence, "classifier", nil, "")
	require.NoError(t, err)
	assert.Nil(t, result.LegalTags())
}

func TestSanitizeTagsHelper(t *testing.T) {
	assert.Nil(t, sanitizeTags(nil))
	assert.Nil(t, sanitizeTags([]string{"  "}))

	result := sanitizeTags([]string{"Motion", " motion ", "Order"})
	assert.ElementsMatch(t, []string{"Motion", "Order"}, result)
}
