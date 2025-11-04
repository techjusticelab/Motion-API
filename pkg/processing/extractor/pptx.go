package extractor

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"sort"
	"strings"
)

// pptxExtractor extracts text from PowerPoint PPTX files
type pptxExtractor struct{}

// NewPPTXExtractor creates a new PPTX extractor instance
func NewPPTXExtractor() Extractor {
	return &pptxExtractor{}
}

// Extract extracts text from PPTX slides
func (e *pptxExtractor) Extract(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (*ExtractionResult, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, NewExtractionError("pptx", "failed to read PPTX file", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, NewExtractionError("pptx", "failed to parse PPTX archive", err)
	}

	text, err := e.extractTextFromSlides(zipReader)
	if err != nil {
		return nil, NewExtractionError("pptx", "failed to extract text from PPTX slides", err)
	}

	if strings.TrimSpace(text) == "" {
		result, err := extractTextFromConvertedPDF(ctx, content, metadata, "pptx")
		if err != nil {
			return nil, NewExtractionError("pptx", "failed to extract text via PDF conversion", err)
		}
		return result, nil
	}

	cleaner := NewTextCleaner(DefaultCleaningConfig())
	text = cleaner.CleanText(text)

	wordCount := countWords(text)
	charCount := len(text)

	return &ExtractionResult{
		Text:      text,
		WordCount: wordCount,
		CharCount: charCount,
		PageCount: 0,
		Success:   true,
		Metadata: map[string]interface{}{
			"format":    "pptx",
			"file_size": len(content),
		},
	}, nil
}

// SupportedFormats returns supported formats for PPTX extractor
func (e *pptxExtractor) SupportedFormats() []string {
	return []string{"pptx"}
}

// CanExtract checks if the format is supported
func (e *pptxExtractor) CanExtract(format string) bool {
	format = strings.ToLower(format)
	for _, supported := range e.SupportedFormats() {
		if format == supported {
			return true
		}
	}
	return false
}

func (e *pptxExtractor) extractTextFromSlides(zipReader *zip.Reader) (string, error) {
	slideFiles := make([]*zip.File, 0)

	for _, file := range zipReader.File {
		if strings.HasPrefix(file.Name, "ppt/slides/slide") && strings.HasSuffix(file.Name, ".xml") {
			slideFiles = append(slideFiles, file)
		}
	}

	sort.Slice(slideFiles, func(i, j int) bool {
		return slideFiles[i].Name < slideFiles[j].Name
	})

	var builder strings.Builder

	for _, slideFile := range slideFiles {
		rc, err := slideFile.Open()
		if err != nil {
			return "", err
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return "", err
		}

		text := e.extractTextFromSlideXML(content)
		if text != "" {
			builder.WriteString(text)
			builder.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(builder.String()), nil
}

func (e *pptxExtractor) extractTextFromSlideXML(content []byte) string {
	decoder := xml.NewDecoder(bytes.NewReader(content))
	var builder strings.Builder
	var inText bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ""
		}

		switch elem := token.(type) {
		case xml.StartElement:
			if (elem.Name.Space == "a" || elem.Name.Space == "") && elem.Name.Local == "t" {
				inText = true
			}
		case xml.EndElement:
			if (elem.Name.Space == "a" || elem.Name.Space == "") && elem.Name.Local == "t" {
				inText = false
			}
			if (elem.Name.Space == "a" || elem.Name.Space == "") && elem.Name.Local == "p" {
				builder.WriteString("\n")
			}
		case xml.CharData:
			if inText {
				builder.WriteString(strings.TrimSpace(string(elem)))
			}
		}
	}

	return strings.TrimSpace(builder.String())
}
