package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyDocumentRequestValidate(t *testing.T) {
	req := &ClassifyDocumentRequest{DocumentID: "doc_123"}
	assert.NoError(t, req.Validate())

	req.DocumentID = ""
	assert.Error(t, req.Validate())
}

func TestUpdateClassificationRequestValidate(t *testing.T) {
	req := &UpdateClassificationRequest{
		DocumentID: "doc_123",
		LegalTags:  []string{"motion"},
	}
	assert.NoError(t, req.Validate())

	req.LegalTags = nil
	assert.Error(t, req.Validate())
}
