package dto

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProcessDocumentRequestValidate(t *testing.T) {
	req := &ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		Content:       bytes.NewReader([]byte("content")),
		ContentSize:   7,
		ContentType:   "application/pdf",
		StoragePath:   "documents/2024/05/example.pdf",
		Text:          "content",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
		StoreBinary:   true,
		Metadata:      map[string]string{"origin": "test"},
	}
	assert.NoError(t, req.Validate())

	req.FileName = ""
	assert.Error(t, req.Validate())
}

func TestBatchProcessDocumentRequestValidate(t *testing.T) {
	req := &BatchProcessDocumentRequest{}
	assert.Error(t, req.Validate())

	req.Documents = []*ProcessDocumentRequest{
		{
			ID:            "doc_1",
			FileName:      "a.pdf",
			Content:       bytes.NewReader([]byte("a")),
			ContentSize:   1,
			ContentType:   "application/pdf",
			StoragePath:   "documents/a.pdf",
			HashValue:     "b3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
			HashAlgorithm: "SHA256",
			StoreBinary:   true,
		},
	}
	assert.NoError(t, req.Validate())
}

func TestProcessDocumentRequestValidateSkipStorage(t *testing.T) {
	req := &ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
		ContentType:   "application/pdf",
		Text:          "content",
		StoreBinary:   false,
	}
	assert.NoError(t, req.Validate())
}

func TestBatchProcessDocumentRequestNilEntry(t *testing.T) {
	req := &BatchProcessDocumentRequest{
		Documents: []*ProcessDocumentRequest{
			nil,
		},
	}
	assert.Error(t, req.Validate())
}

func TestProcessDocumentResponseFields(t *testing.T) {
	now := time.Now()
	resp := ProcessDocumentResponse{
		DocumentID:  "doc",
		Stored:      true,
		StorageURL:  "https://example.com/doc.pdf",
		Classified:  true,
		Indexed:     true,
		CreatedAt:   now,
		ProcessedAt: &now,
	}
	assert.Equal(t, "doc", resp.DocumentID)
	assert.True(t, resp.Stored)
}

func TestProcessDocumentRequestValidateClassifyRequiresText(t *testing.T) {
	req := &ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		ContentType:   "application/pdf",
		StoragePath:   "documents/example.pdf",
		ContentSize:   1,
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
		Classify:      true,
	}
	assert.Error(t, req.Validate())
}

func TestIndexDocumentRequestValidate(t *testing.T) {
	req := &IndexDocumentRequest{
		DocumentID: "doc_123",
	}
	assert.NoError(t, req.Validate())

	req.DocumentID = ""
	assert.Error(t, req.Validate())
}

func TestUpdateMetadataRequestValidate(t *testing.T) {
	now := time.Now()
	req := &UpdateMetadataRequest{
		DocumentID:  "doc_456",
		Language:    "en",
		LegalTags:   []string{"motion", "suppression"},
		ProcessedAt: &now,
	}
	assert.NoError(t, req.Validate())

	req.Language = " "
	assert.Error(t, req.Validate())

	nowZero := time.Time{}
	req.Language = "es"
	req.ProcessedAt = &nowZero
	assert.Error(t, req.Validate())
}

func TestProcessDocumentRequestValidateInvalidID(t *testing.T) {
	req := &ProcessDocumentRequest{
		ID:            "",
		FileName:      "example.pdf",
		ContentType:   "application/pdf",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
	}
	assert.Error(t, req.Validate())
}

func TestProcessDocumentRequestValidateStoreBinaryMissingContent(t *testing.T) {
	req := &ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		ContentType:   "application/pdf",
		StoragePath:   "documents/example.pdf",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
		StoreBinary:   true,
		Content:       nil,
		ContentSize:   100,
	}
	assert.Error(t, req.Validate())
}

func TestProcessDocumentRequestValidateStoreBinaryInvalidSize(t *testing.T) {
	req := &ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		ContentType:   "application/pdf",
		StoragePath:   "documents/example.pdf",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
		StoreBinary:   true,
		Content:       bytes.NewReader([]byte("test")),
		ContentSize:   0,
	}
	assert.Error(t, req.Validate())
}

func TestProcessDocumentRequestValidateStoreBinaryInvalidPath(t *testing.T) {
	req := &ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		ContentType:   "application/pdf",
		StoragePath:   "",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
		StoreBinary:   true,
		Content:       bytes.NewReader([]byte("test")),
		ContentSize:   4,
	}
	assert.Error(t, req.Validate())
}

func TestProcessDocumentRequestValidateInvalidContentType(t *testing.T) {
	req := &ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		ContentType:   "",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
	}
	assert.Error(t, req.Validate())
}

func TestProcessDocumentRequestValidateInvalidHash(t *testing.T) {
	req := &ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		ContentType:   "application/pdf",
		HashValue:     "invalid",
		HashAlgorithm: "SHA256",
	}
	assert.Error(t, req.Validate())
}
