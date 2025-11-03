package dto

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactionOptionsNormalize(t *testing.T) {
	opts := (*RedactionOptions)(nil).Normalize()
	assert.True(t, opts.CaliforniaLaws)
	assert.Equal(t, "■", opts.ReplacementChar)

	opts = (&RedactionOptions{ReplacementChar: " "}).Normalize()
	assert.Equal(t, "■", opts.ReplacementChar)
}

func TestAnalyzeRedactionsRequestValidate(t *testing.T) {
	req := &AnalyzeRedactionsRequest{
		DocumentID: "doc123",
	}
	assert.NoError(t, req.Validate())

	data := base64.StdEncoding.EncodeToString([]byte("pdf"))
	req = &AnalyzeRedactionsRequest{
		PDFBase64: data,
	}
	assert.NoError(t, req.Validate())

	req = &AnalyzeRedactionsRequest{}
	assert.Error(t, req.Validate())

	req = &AnalyzeRedactionsRequest{PDFBase64: "not-base64"}
	assert.Error(t, req.Validate())
}

func TestApplyRedactionsRequestValidate(t *testing.T) {
	req := &ApplyRedactionsRequest{
		DocumentID: "doc123",
	}
	assert.NoError(t, req.Validate())

	data := base64.StdEncoding.EncodeToString([]byte("pdf"))
	req = &ApplyRedactionsRequest{
		PDFBase64: data,
	}
	assert.NoError(t, req.Validate())

	req = &ApplyRedactionsRequest{}
	assert.Error(t, req.Validate())

	req = &ApplyRedactionsRequest{PDFBase64: "bad"}
	assert.Error(t, req.Validate())
}
