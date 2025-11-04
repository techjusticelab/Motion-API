package opensearch

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"

	"motion-index-fiber/internal/config"
	internalmodels "motion-index-fiber/internal/models"
	pkgmodels "motion-index-fiber/pkg/models"
	pkgextractor "motion-index-fiber/pkg/processing/extractor"
	pkgsearch "motion-index-fiber/pkg/search"
	pkgstorage "motion-index-fiber/pkg/storage"
)

// DryClassificationHandler provides an endpoint to perform a dry classification run.
// It converts the uploaded document to PDF, uploads it to storage, extracts the text
// and pushes the document without classification metadata to OpenSearch.
type DryClassificationHandler struct {
	cfg              *config.Config
	storageService   pkgstorage.Service
	searchService    pkgsearch.Service
	extractorService pkgextractor.Service
}

// NewDryClassificationHandler creates a new dry classification handler.
func NewDryClassificationHandler(
	cfg *config.Config,
	storageService pkgstorage.Service,
	searchService pkgsearch.Service,
	extractorService pkgextractor.Service,
) *DryClassificationHandler {
	return &DryClassificationHandler{
		cfg:              cfg,
		storageService:   storageService,
		searchService:    searchService,
		extractorService: extractorService,
	}
}

// Run handles the dry classification workflow.
func (h *DryClassificationHandler) Run(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalmodels.NewErrorResponse(
			"file_required",
			"No file provided in request",
			map[string]interface{}{"field": "file"},
		))
	}

	if err := validateDryFileType(fileHeader.Filename); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalmodels.NewErrorResponse(
			"unsupported_file_type",
			err.Error(),
			map[string]interface{}{"file": fileHeader.Filename},
		))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalmodels.NewErrorResponse(
			"file_open_failed",
			"Failed to open uploaded file",
			map[string]interface{}{"error": err.Error()},
		))
	}
	defer file.Close()

	originalBytes, err := io.ReadAll(file)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalmodels.NewErrorResponse(
			"file_read_failed",
			"Failed to read uploaded file",
			map[string]interface{}{"error": err.Error()},
		))
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Minute)
	defer cancel()

	originalExt := strings.ToLower(filepath.Ext(fileHeader.Filename))
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = pkgstorage.GetContentTypeFromFilename(fileHeader.Filename)
	}

	extractionMetadata := &pkgextractor.DocumentMetadata{
		FileName: fileHeader.Filename,
		MimeType: contentType,
		Size:     fileHeader.Size,
		Format:   strings.TrimPrefix(originalExt, "."),
	}

	// Step 1: Extract text from original format (needed for PDF generation)
	intermediateExtraction, err := h.extractorService.ExtractText(ctx, bytes.NewReader(originalBytes), extractionMetadata)
	if err != nil || intermediateExtraction == nil {
		reason := "Failed to extract text from document"
		if err != nil {
			reason = err.Error()
		}
		return c.Status(fiber.StatusInternalServerError).JSON(internalmodels.NewErrorResponse(
			"extraction_failed",
			reason,
			map[string]interface{}{"file": fileHeader.Filename},
		))
	}

	// Step 2: Generate PDF from extracted text
	pdfBytes, err := h.ensurePDFBytes(originalBytes, originalExt, intermediateExtraction.Text)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalmodels.NewErrorResponse(
			"pdf_conversion_failed",
			err.Error(),
			nil,
		))
	}

	documentID := uuid.New().String()
	sanitizedName := sanitizeFilename(strings.TrimSuffix(fileHeader.Filename, originalExt))
	storagePath := fmt.Sprintf("dry-classification/%s_%s.pdf", documentID, sanitizedName)

	uploadMetadata := &pkgstorage.UploadMetadata{
		ContentType: "application/pdf",
		Size:        int64(len(pdfBytes)),
		FileName:    fmt.Sprintf("%s.pdf", sanitizedName),
		Tags: map[string]string{
			"status": "dry_classification",
		},
	}

	uploadResult, err := h.storageService.Upload(ctx, storagePath, bytes.NewReader(pdfBytes), uploadMetadata)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalmodels.NewErrorResponse(
			"upload_failed",
			"Failed to upload converted PDF to storage",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Step 3: Use appropriate text extraction based on original format
	var finalExtraction *pkgextractor.ExtractionResult

	if strings.EqualFold(originalExt, ".pdf") {
		// Original was PDF - extract from the PDF
		pdfMetadata := &pkgextractor.DocumentMetadata{
			FileName: fmt.Sprintf("%s.pdf", sanitizedName),
			MimeType: "application/pdf",
			Size:     int64(len(pdfBytes)),
			Format:   "pdf",
		}

		var err error
		finalExtraction, err = h.extractorService.ExtractText(ctx, bytes.NewReader(pdfBytes), pdfMetadata)
		if err != nil || finalExtraction == nil {
			reason := "Failed to extract text from PDF"
			if err != nil {
				reason = err.Error()
			}
			return c.Status(fiber.StatusInternalServerError).JSON(internalmodels.NewErrorResponse(
				"pdf_extraction_failed",
				reason,
				map[string]interface{}{"storage_path": storagePath},
			))
		}
	} else {
		// Generated PDF from DOC/DOCX/PPT/PPTX/TXT - use intermediate extraction
		// (gofpdf-generated PDFs don't extract properly, causing garbled text)
		finalExtraction = intermediateExtraction
	}

	docHash := hashBytes(pdfBytes)
	now := time.Now()
	metadata := pkgmodels.NewDocumentMetadata()
	metadata.DocumentName = fileHeader.Filename
	metadata.ProcessedAt = now
	metadata.AIClassified = false

	document := &pkgmodels.Document{
		ID:          documentID,
		FileName:    fileHeader.Filename,
		FilePath:    storagePath,
		FileURL:     h.storageService.GetURL(storagePath),
		S3URI:       buildS3URI(h.cfg, storagePath),
		Text:        finalExtraction.Text, // Use text extracted from PDF
		Hash:        docHash,
		CreatedAt:   now,
		UpdatedAt:   now,
		Metadata:    metadata,
		Size:        int64(len(pdfBytes)),
		ContentType: "application/pdf",
	}

	if _, err := h.searchService.IndexDocument(ctx, document); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalmodels.NewErrorResponse(
			"indexing_failed",
			"Failed to push document to OpenSearch",
			map[string]interface{}{"error": err.Error()},
		))
	}

	response := fiber.Map{
		"document_id":    documentID,
		"storage_path":   storagePath,
		"file_url":       uploadResult.URL,
		"cdn_url":        h.storageService.GetURL(storagePath),
		"original_name":  fileHeader.Filename,
		"text_chars":     len(finalExtraction.Text),
		"word_count":     finalExtraction.WordCount,
		"page_count":     finalExtraction.PageCount,
		"classification": fiber.Map{},
	}

	return c.Status(fiber.StatusCreated).JSON(internalmodels.NewSuccessResponse(response, "Document processed successfully"))
}

func (h *DryClassificationHandler) ensurePDFBytes(original []byte, ext string, text string) ([]byte, error) {
	if strings.EqualFold(ext, ".pdf") {
		return original, nil
	}

	if text == "" {
		return nil, fmt.Errorf("no text extracted for PDF conversion")
	}

	return generatePDFDocument(text)
}

func validateDryFileType(filename string) error {
	allowed := map[string]bool{
		".pdf":  true,
		".doc":  true,
		".docx": true,
		".ppt":  true,
		".pptx": true,
		".txt":  true,
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !allowed[ext] {
		return fmt.Errorf("file type %s is not supported", ext)
	}
	return nil
}

func sanitizeFilename(filename string) string {
	reg := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	sanitized := reg.ReplaceAllString(filename, "_")
	sanitized = regexp.MustCompile(`_+`).ReplaceAllString(sanitized, "_")
	sanitized = strings.Trim(sanitized, "_")
	if sanitized == "" {
		sanitized = "document"
	}
	return sanitized
}

func generatePDFDocument(text string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 12)

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimRight(line, " ")
		if trimmed == "" {
			pdf.Ln(6)
			continue
		}
		pdf.MultiCell(0, 6, trimmed, "", "L", false)
	}

	var buffer bytes.Buffer
	if err := pdf.Output(&buffer); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func hashBytes(data []byte) string {
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}

func buildS3URI(cfg *config.Config, key string) string {
	bucket := strings.TrimSpace(cfg.Storage.Bucket)
	if bucket == "" {
		return fmt.Sprintf("s3://%s", key)
	}
	return fmt.Sprintf("s3://%s/%s", bucket, key)
}
