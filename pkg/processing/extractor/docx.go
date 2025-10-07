package extractor

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"strings"
)

// docxExtractor handles DOCX files
type docxExtractor struct{}

// NewDOCXExtractor creates a new DOCX extractor
func NewDOCXExtractor() Extractor {
	return &docxExtractor{}
}

// Extract extracts text from DOCX files
func (e *docxExtractor) Extract(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (*ExtractionResult, error) {
	// Read all content into memory
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, NewExtractionError("docx", "failed to read DOCX file", err)
	}

	// Parse DOCX (which is a ZIP file)
	zipReader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, NewExtractionError("docx", "failed to parse DOCX file", err)
	}

	// Extract text from document.xml
	text, err := e.extractTextFromDocx(zipReader)
	if err != nil {
		return nil, NewExtractionError("docx", "failed to extract text from DOCX", err)
	}

	// Clean up the text using enhanced cleaner
	cleaner := NewTextCleaner(DefaultCleaningConfig())
	text = cleaner.CleanText(text)

	// Count words and characters
	wordCount := countWords(text)
	charCount := len(text)

	// Get document properties
	props := e.getDocumentProperties(zipReader)

	return &ExtractionResult{
		Text:      text,
		WordCount: wordCount,
		CharCount: charCount,
		PageCount: 1, // DOCX doesn't have a clear page count concept
		Metadata: map[string]interface{}{
			"format":     "docx",
			"file_size":  len(content),
			"properties": props,
		},
	}, nil
}

// SupportedFormats returns the formats this extractor supports
func (e *docxExtractor) SupportedFormats() []string {
	return []string{"docx", "docm"}
}

// CanExtract checks if this extractor can handle the given format
func (e *docxExtractor) CanExtract(format string) bool {
	format = strings.ToLower(format)
	for _, supported := range e.SupportedFormats() {
		if format == supported {
			return true
		}
	}
	return false
}

// extractTextFromDocx extracts text from the DOCX document.xml file
func (e *docxExtractor) extractTextFromDocx(zipReader *zip.Reader) (string, error) {
	var documentFile *zip.File

	// Find the document.xml file
	for _, file := range zipReader.File {
		if file.Name == "word/document.xml" {
			documentFile = file
			break
		}
	}

	if documentFile == nil {
		return "", NewExtractionError("docx", "document.xml not found in DOCX file", nil)
	}

	// Open and read the document.xml file
	rc, err := documentFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	// Parse XML and extract text
	return e.parseDocumentXML(content)
}

// parseDocumentXML parses the document.xml and extracts text content
func (e *docxExtractor) parseDocumentXML(xmlContent []byte) (string, error) {
	// Simple XML parser to extract text nodes
	decoder := xml.NewDecoder(bytes.NewReader(xmlContent))
	var textBuilder strings.Builder
	var inText bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			// Check for text elements (w:t)
			if elem.Name.Local == "t" {
				inText = true
			}
		case xml.EndElement:
			if elem.Name.Local == "t" {
				inText = false
			}
			// Add line break for paragraph ends
			if elem.Name.Local == "p" {
				textBuilder.WriteString("\n")
			}
		case xml.CharData:
			if inText {
				textBuilder.Write(elem)
			}
		}
	}

	return textBuilder.String(), nil
}

// getDocumentProperties extracts document properties from core.xml
func (e *docxExtractor) getDocumentProperties(zipReader *zip.Reader) map[string]string {
	props := make(map[string]string)

	// Look for core properties
	for _, file := range zipReader.File {
		if file.Name == "docProps/core.xml" {
			rc, err := file.Open()
			if err != nil {
				continue
			}

			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}

			// Parse core properties XML
			e.parseCoreProperties(content, props)
			break
		}
	}

	return props
}

// parseCoreProperties parses the core.xml file to extract document properties
func (e *docxExtractor) parseCoreProperties(xmlContent []byte, props map[string]string) {
	decoder := xml.NewDecoder(bytes.NewReader(xmlContent))
	var currentElement string

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return
		}

		switch elem := token.(type) {
		case xml.StartElement:
			currentElement = elem.Name.Local
		case xml.CharData:
			if currentElement != "" {
				value := strings.TrimSpace(string(elem))
				if value != "" {
					switch currentElement {
					case "title":
						props["title"] = value
					case "creator":
						props["author"] = value
					case "subject":
						props["subject"] = value
					case "description":
						props["description"] = value
					case "created":
						props["created"] = value
					case "modified":
						props["modified"] = value
					}
				}
			}
		case xml.EndElement:
			currentElement = ""
		}
	}
}
