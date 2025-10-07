//go:build enhanced
// +build enhanced

package extractor

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/dslipak/pdf"
)

// DocumentType represents the type of document detected
type DocumentType int

const (
	DocumentTypeUnknown DocumentType = iota
	DocumentTypeTextPDF              // PDF with extractable text
	DocumentTypeScannedPDF           // PDF with images/scanned content
	DocumentTypeHybridPDF            // PDF with both text and images
	DocumentTypeImage                // Image file (PNG, JPG, etc.)
)

// DocumentAnalysis contains the analysis results
type DocumentAnalysis struct {
	Type                DocumentType
	HasExtractableText  bool
	HasImages          bool
	EstimatedPages     int
	Confidence         float64
	RecommendedMethod  string
	Fallbacks          []string
}

// DocumentAnalyzer analyzes documents to determine the best extraction strategy
// This follows UNIX philosophy: analyze one thing (document type) and do it well
type DocumentAnalyzer struct{}

// NewDocumentAnalyzer creates a new document analyzer
func NewDocumentAnalyzer() *DocumentAnalyzer {
	return &DocumentAnalyzer{}
}

// AnalyzeDocument analyzes a document and returns the best extraction strategy
func (a *DocumentAnalyzer) AnalyzeDocument(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (*DocumentAnalysis, error) {
	// Read content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Detect file type
	if a.isPDF(content) {
		return a.analyzePDF(content, metadata)
	}

	if a.isImage(content, metadata) {
		return a.analyzeImage(content, metadata)
	}

	// Default to unknown
	return &DocumentAnalysis{
		Type:               DocumentTypeUnknown,
		HasExtractableText: false,
		HasImages:         false,
		EstimatedPages:    1,
		Confidence:        0.0,
		RecommendedMethod: "text",
		Fallbacks:         []string{"ocr"},
	}, nil
}

// isPDF checks if content is a PDF
func (a *DocumentAnalyzer) isPDF(content []byte) bool {
	if len(content) < 4 {
		return false
	}

	// Check for PDF header within first 1024 bytes
	searchLimit := min(1024, len(content))
	for i := 0; i <= searchLimit-4; i++ {
		if string(content[i:i+4]) == "%PDF" {
			return true
		}
	}
	return false
}

// isImage checks if content is an image based on metadata or content
func (a *DocumentAnalyzer) isImage(content []byte, metadata *DocumentMetadata) bool {
	// Check file extension
	if metadata != nil && metadata.FileName != "" {
		ext := strings.ToLower(metadata.FileName)
		imageExts := []string{".png", ".jpg", ".jpeg", ".tiff", ".bmp", ".gif"}
		for _, imgExt := range imageExts {
			if strings.HasSuffix(ext, imgExt) {
				return true
			}
		}
	}

	// Check MIME type
	if metadata != nil && metadata.MimeType != "" {
		return strings.HasPrefix(metadata.MimeType, "image/")
	}

	// Check magic bytes for common image formats
	if len(content) >= 8 {
		// PNG
		if bytes.Equal(content[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
			return true
		}
	}

	if len(content) >= 3 {
		// JPEG
		if bytes.Equal(content[:3], []byte{0xFF, 0xD8, 0xFF}) {
			return true
		}
	}

	if len(content) >= 6 {
		// GIF
		if bytes.Equal(content[:6], []byte{'G', 'I', 'F', '8', '7', 'a'}) ||
			bytes.Equal(content[:6], []byte{'G', 'I', 'F', '8', '9', 'a'}) {
			return true
		}
	}

	return false
}

// analyzePDF analyzes a PDF document to determine its characteristics
func (a *DocumentAnalyzer) analyzePDF(content []byte, metadata *DocumentMetadata) (*DocumentAnalysis, error) {
	// Try to open with dslipak/pdf to check for extractable text
	hasText, pageCount := a.checkPDFTextContent(content)
	
	// Analyze content structure
	hasImages := a.checkPDFImageContent(content)
	
	// Determine document type and recommendations
	var docType DocumentType
	var recommended string
	var fallbacks []string
	var confidence float64

	if hasText && !hasImages {
		// Pure text PDF
		docType = DocumentTypeTextPDF
		recommended = "text"
		fallbacks = []string{"dslipak", "ocr"}
		confidence = 0.9
	} else if !hasText && hasImages {
		// Pure scanned/image PDF
		docType = DocumentTypeScannedPDF
		recommended = "ocr"
		fallbacks = []string{"text", "dslipak"}
		confidence = 0.8
	} else if hasText && hasImages {
		// Hybrid PDF
		docType = DocumentTypeHybridPDF
		recommended = "text"
		fallbacks = []string{"dslipak", "ocr"}
		confidence = 0.7
	} else {
		// Unknown or problematic PDF
		docType = DocumentTypeUnknown
		recommended = "text"
		fallbacks = []string{"dslipak", "ocr"}
		confidence = 0.3
	}

	return &DocumentAnalysis{
		Type:               docType,
		HasExtractableText: hasText,
		HasImages:         hasImages,
		EstimatedPages:    pageCount,
		Confidence:        confidence,
		RecommendedMethod: recommended,
		Fallbacks:         fallbacks,
	}, nil
}

// analyzeImage analyzes an image file
func (a *DocumentAnalyzer) analyzeImage(content []byte, metadata *DocumentMetadata) (*DocumentAnalysis, error) {
	return &DocumentAnalysis{
		Type:               DocumentTypeImage,
		HasExtractableText: false,
		HasImages:         true,
		EstimatedPages:    1,
		Confidence:        0.9,
		RecommendedMethod: "ocr",
		Fallbacks:         []string{},
	}, nil
}

// checkPDFTextContent attempts to extract text to see if PDF has extractable text
func (a *DocumentAnalyzer) checkPDFTextContent(content []byte) (bool, int) {
	// Try to open with dslipak/pdf
	contentReader := bytes.NewReader(content)
	r, err := pdf.NewReader(contentReader, int64(len(content)))
	if err != nil {
		return false, 0
	}

	pageCount := r.NumPage()
	if pageCount == 0 {
		return false, 0
	}

	// Check first few pages for extractable text
	maxPagesToCheck := min(3, pageCount)
	totalTextLength := 0

	for pageIndex := 1; pageIndex <= maxPagesToCheck; pageIndex++ {
		page := r.Page(pageIndex)
		if page.V.IsNull() {
			continue
		}

		// Try to extract text from this page
		rows, err := page.GetTextByRow()
		if err != nil {
			continue
		}

		pageTextLength := 0
		for _, row := range rows {
			for _, word := range row.Content {
				if word.S != "" {
					pageTextLength += len(word.S)
				}
			}
		}
		totalTextLength += pageTextLength
	}

	// Consider it has text if we found reasonable amount of text
	// (more than 50 characters across checked pages)
	hasText := totalTextLength > 50
	return hasText, pageCount
}

// checkPDFImageContent checks if PDF likely contains images
func (a *DocumentAnalyzer) checkPDFImageContent(content []byte) bool {
	contentStr := string(content)
	
	// Look for image-related keywords in PDF structure
	imageIndicators := []string{
		"/Image",
		"/DCTDecode",
		"/FlateDecode",
		"/JPXDecode",
		"/JBIG2Decode",
		"/CCITTFaxDecode",
	}

	indicatorCount := 0
	for _, indicator := range imageIndicators {
		if strings.Contains(contentStr, indicator) {
			indicatorCount++
		}
	}

	// If we find multiple image indicators, likely has images
	return indicatorCount >= 2
}

// GetAnalysisDescription returns a human-readable description of the analysis
func (a *DocumentAnalysis) GetDescription() string {
	switch a.Type {
	case DocumentTypeTextPDF:
		return "Text-based PDF with extractable content"
	case DocumentTypeScannedPDF:
		return "Scanned PDF requiring OCR"
	case DocumentTypeHybridPDF:
		return "Hybrid PDF with both text and images"
	case DocumentTypeImage:
		return "Image file requiring OCR"
	default:
		return "Unknown document type"
	}
}

// GetRecommendedStrategy returns the recommended extraction strategy
func (a *DocumentAnalysis) GetRecommendedStrategy() map[string]interface{} {
	return map[string]interface{}{
		"primary_method":   a.RecommendedMethod,
		"fallback_methods": a.Fallbacks,
		"confidence":      a.Confidence,
		"description":     a.GetDescription(),
		"estimated_pages": a.EstimatedPages,
		"requires_ocr":    a.Type == DocumentTypeScannedPDF || a.Type == DocumentTypeImage,
	}
}