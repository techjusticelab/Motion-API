package classifier

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid mock config",
			config: &Config{
				Provider: "mock",
			},
			wantErr: false,
		},
		{
			name: "valid openai config",
			config: &Config{
				Provider: "openai",
				APIKey:   "test-key",
				Model:    "gpt-4",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "unsupported provider",
			config: &Config{
				Provider: "unsupported",
			},
			wantErr: true,
		},
		{
			name: "openai without API key",
			config: &Config{
				Provider: "openai",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewService(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestService_ClassifyDocument(t *testing.T) {
	service, err := NewService(&Config{Provider: "mock"})
	assert.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name     string
		text     string
		metadata *DocumentMetadata
		wantErr  bool
	}{
		{
			name: "valid legal document",
			text: "This is a motion to dismiss the case filed by plaintiff against defendant.",
			metadata: &DocumentMetadata{
				FileName:  "motion.pdf",
				FileType:  "pdf",
				WordCount: 12,
				PageCount: 1,
			},
			wantErr: false,
		},
		{
			name: "contract document",
			text: "This agreement is entered into between the parties for the sale of goods.",
			metadata: &DocumentMetadata{
				FileName:  "contract.docx",
				FileType:  "docx",
				WordCount: 13,
				PageCount: 2,
			},
			wantErr: false,
		},
		{
			name:     "empty text",
			text:     "",
			metadata: &DocumentMetadata{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ClassifyDocument(ctx, tt.text, tt.metadata)

			if tt.wantErr {
				assert.Error(t, err)
				assert.False(t, result.Success)
				assert.NotEmpty(t, result.Error)
			} else {
				assert.NoError(t, err)
				assert.True(t, result.Success)
				assert.NotEmpty(t, result.DocumentType)
				assert.NotEmpty(t, result.LegalCategory)
				assert.GreaterOrEqual(t, result.Confidence, 0.0)
				assert.LessOrEqual(t, result.Confidence, 1.0)
				assert.GreaterOrEqual(t, result.ProcessingTime, int64(0))
			}
		})
	}
}

func TestService_GetAvailableCategories(t *testing.T) {
	service, err := NewService(&Config{Provider: "mock"})
	assert.NoError(t, err)

	categories := service.GetAvailableCategories()
	assert.NotEmpty(t, categories)
	assert.Contains(t, categories, LegalCategoryCriminal)
	assert.Contains(t, categories, LegalCategoryCivil)
	assert.Contains(t, categories, LegalCategoryContract)
}

func TestService_IsHealthy(t *testing.T) {
	service, err := NewService(&Config{Provider: "mock"})
	assert.NoError(t, err)

	healthy := service.IsHealthy()
	assert.True(t, healthy)
}

func TestService_ValidateResult(t *testing.T) {
	service, err := NewService(&Config{Provider: "mock"})
	assert.NoError(t, err)

	tests := []struct {
		name    string
		result  *ClassificationResult
		wantErr bool
	}{
		{
			name: "valid result",
			result: &ClassificationResult{
				DocumentType:  DocumentTypeMotionToSuppress,
				LegalCategory: LegalCategoryCivil,
				Confidence:    0.8,
			},
			wantErr: false,
		},
		{
			name:    "nil result",
			result:  nil,
			wantErr: true,
		},
		{
			name: "invalid confidence too low",
			result: &ClassificationResult{
				DocumentType:  DocumentTypeMotionToSuppress,
				LegalCategory: LegalCategoryCivil,
				Confidence:    -0.1,
			},
			wantErr: true,
		},
		{
			name: "invalid confidence too high",
			result: &ClassificationResult{
				DocumentType:  DocumentTypeMotionToSuppress,
				LegalCategory: LegalCategoryCivil,
				Confidence:    1.1,
			},
			wantErr: true,
		},
		{
			name: "missing document type",
			result: &ClassificationResult{
				LegalCategory: LegalCategoryCivil,
				Confidence:    0.8,
			},
			wantErr: true,
		},
		{
			name: "missing legal category",
			result: &ClassificationResult{
				DocumentType: DocumentTypeMotionToSuppress,
				Confidence:   0.8,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateResult(tt.result)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetDefaultCategories(t *testing.T) {
	categories := GetDefaultCategories()
	assert.NotEmpty(t, categories)
	assert.Contains(t, categories, LegalCategoryCriminal)
	assert.Contains(t, categories, LegalCategoryCivil)
	assert.Contains(t, categories, LegalCategoryContract)
	assert.Contains(t, categories, LegalCategoryFamily)
}

func TestGetDefaultDocumentTypes(t *testing.T) {
	types := GetDefaultDocumentTypes()
	assert.NotEmpty(t, types)
	assert.Contains(t, types, DocumentTypeMotionToSuppress)
	assert.Contains(t, types, DocumentTypeOrder)
	assert.Contains(t, types, DocumentTypeBrief)
	assert.Contains(t, types, DocumentTypeComplaint)
}

func TestClassificationError(t *testing.T) {
	baseErr := assert.AnError
	classErr := NewClassificationError("test_type", "test message", baseErr)

	assert.Equal(t, "test_type", classErr.Type)
	assert.Equal(t, "test message", classErr.Message)
	assert.Equal(t, baseErr, classErr.Cause)
	assert.Contains(t, classErr.Error(), "test message")
	assert.Contains(t, classErr.Error(), baseErr.Error())
	assert.Equal(t, baseErr, classErr.Unwrap())
}

func TestClassificationError_WithoutCause(t *testing.T) {
	classErr := NewClassificationError("test_type", "simple error", nil)

	assert.Equal(t, "simple error", classErr.Error())
	assert.Nil(t, classErr.Unwrap())
}
