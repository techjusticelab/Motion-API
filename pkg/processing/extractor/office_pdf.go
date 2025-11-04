package extractor

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
)

func extractTextFromConvertedPDF(ctx context.Context, content []byte, metadata *DocumentMetadata, extension string) (*ExtractionResult, error) {
	pdfBytes, err := convertWithLibreOfficeToPDF(ctx, content, extension)
	if err != nil {
		return nil, err
	}

	pdfExtractor := NewPDFExtractor()
	pdfMeta := &DocumentMetadata{
		FileName: buildConvertedPDFName(metadata),
		MimeType: "application/pdf",
		Size:     int64(len(pdfBytes)),
		Format:   "pdf",
	}

	result, err := pdfExtractor.Extract(ctx, bytes.NewReader(pdfBytes), pdfMeta)
	if err != nil {
		return nil, err
	}

	if result.Metadata == nil {
		result.Metadata = map[string]interface{}{}
	}
	result.Metadata["original_format"] = extension
	result.Metadata["conversion_tool"] = "libreoffice_pdf"
	result.Metadata["pdf_size"] = len(pdfBytes)

	return result, nil
}

func buildConvertedPDFName(metadata *DocumentMetadata) string {
	if metadata == nil || metadata.FileName == "" {
		return "document.pdf"
	}

	name := filepath.Base(metadata.FileName)
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "" {
		base = "document"
	}
	return base + ".pdf"
}
