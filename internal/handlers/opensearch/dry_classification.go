package opensearch

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
	"io"
	"motion-index-fiber/internal/config"
	internalmodels "motion-index-fiber/internal/models"
	pkgmodels "motion-index-fiber/pkg/models"
	pkgextractor "motion-index-fiber/pkg/processing/extractor"
	pkgsearch "motion-index-fiber/pkg/search"
	pkgstorage "motion-index-fiber/pkg/storage"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type DryClassificationHandler struct {
	cfg              *config.Config
	storageService   pkgstorage.Service
	searchService    pkgsearch.Service
	extractorService pkgextractor.Service
	jobQueue         *dryJobQueue
}

func NewDryClassificationHandler(
	cfg *config.Config,
	storageService pkgstorage.Service,
	searchService pkgsearch.Service,
	extractorService pkgextractor.Service,
) *DryClassificationHandler {
	h := &DryClassificationHandler{
		cfg:              cfg,
		storageService:   storageService,
		searchService:    searchService,
		extractorService: extractorService,
	}
	h.jobQueue = newDryJobQueue(h.processDryJob)
	return h
}

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

	src, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalmodels.NewErrorResponse(
			"file_open_failed",
			"Failed to open uploaded file",
			map[string]interface{}{"error": err.Error()},
		))
	}
	defer src.Close()

	tempFile, err := os.CreateTemp("", "dry-upload-*")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalmodels.NewErrorResponse(
			"tempfile_creation_failed",
			"Failed to create temporary storage for upload",
			map[string]interface{}{"error": err.Error()},
		))
	}
	tempPath := tempFile.Name()

	written, err := io.Copy(tempFile, src)
	if err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return c.Status(fiber.StatusBadRequest).JSON(internalmodels.NewErrorResponse(
			"file_read_failed",
			"Failed to read uploaded file",
			map[string]interface{}{"error": err.Error()},
		))
	}
	tempFile.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = pkgstorage.GetContentTypeFromFilename(fileHeader.Filename)
	}

	payload := &dryJobPayload{
		FileName:    fileHeader.Filename,
		ContentType: contentType,
		Size:        written,
		TempPath:    tempPath,
	}

	result, submitErr := h.jobQueue.Submit(c.Context(), payload)
	if submitErr != nil {
		os.Remove(tempPath)
		if errors.Is(submitErr, context.Canceled) || errors.Is(submitErr, context.DeadlineExceeded) {
			return c.Status(fiber.StatusRequestTimeout).JSON(internalmodels.NewErrorResponse(
				"request_cancelled",
				"Request cancelled while waiting in processing queue",
				nil,
			))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(internalmodels.NewErrorResponse(
			"queue_error",
			"Failed to enqueue processing job",
			map[string]interface{}{"error": submitErr.Error()},
		))
	}

	return c.Status(result.status).JSON(result.body)
}

func (h *DryClassificationHandler) processDryJob(job *dryJob) dryJobResult {
	defer os.Remove(job.payload.TempPath)

	// Monitor memory usage at start
	h.logMemoryUsage("Job start", job.payload.FileName)

	if job.ctx.Err() != nil {
		return dryJobResult{
			status: fiber.StatusRequestTimeout,
			body: internalmodels.NewErrorResponse(
				"request_cancelled",
				"Request context cancelled before processing",
				nil,
			),
			err: job.ctx.Err(),
		}
	}

	tempFile, err := os.Open(job.payload.TempPath)
	if err != nil {
		return dryJobResult{
			status: fiber.StatusInternalServerError,
			body: internalmodels.NewErrorResponse(
				"file_open_failed",
				"Failed to open temporary file for processing",
				map[string]interface{}{"error": err.Error()},
			),
			err: err,
		}
	}
	defer tempFile.Close()

	originalExt := strings.ToLower(filepath.Ext(job.payload.FileName))
	ctx, cancel := context.WithTimeout(job.ctx, 5*time.Minute)
	defer cancel()

	extractionMetadata := &pkgextractor.DocumentMetadata{
		FileName: job.payload.FileName,
		MimeType: job.payload.ContentType,
		Size:     job.payload.Size,
		Format:   strings.TrimPrefix(originalExt, "."),
	}

	intermediateExtraction, err := h.extractorService.ExtractText(ctx, tempFile, extractionMetadata)
	if err != nil || intermediateExtraction == nil {
		reason := "Failed to extract text from document"
		if err != nil {
			reason = err.Error()
		}
		return dryJobResult{
			status: fiber.StatusInternalServerError,
			body: internalmodels.NewErrorResponse(
				"extraction_failed",
				reason,
				map[string]interface{}{"file": job.payload.FileName},
			),
			err: err,
		}
	}

	// Force garbage collection after text extraction to release memory from large string operations
	runtime.GC()
	h.logMemoryUsage("After extraction & GC", job.payload.FileName)

	pdfBytes, err := h.ensurePDFBytes(tempFile, originalExt, intermediateExtraction.Text)
	if err != nil {
		return dryJobResult{
			status: fiber.StatusInternalServerError,
			body: internalmodels.NewErrorResponse(
				"pdf_conversion_failed",
				err.Error(),
				nil,
			),
			err: err,
		}
	}

	documentID := uuid.New().String()
	sanitizedName := sanitizeFilename(strings.TrimSuffix(job.payload.FileName, originalExt))
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
		return dryJobResult{
			status: fiber.StatusInternalServerError,
			body: internalmodels.NewErrorResponse(
				"upload_failed",
				"Failed to upload converted PDF to storage",
				map[string]interface{}{"error": err.Error()},
			),
			err: err,
		}
	}

	var finalExtraction *pkgextractor.ExtractionResult

	if strings.EqualFold(originalExt, ".pdf") {
		finalExtraction = intermediateExtraction
	} else {
		finalExtraction = intermediateExtraction
	}

	docHash := hashBytes(pdfBytes)
	now := time.Now()
	metadata := pkgmodels.NewDocumentMetadata()
	metadata.DocumentName = job.payload.FileName
	metadata.ProcessedAt = now
	metadata.AIClassified = false

	document := &pkgmodels.Document{
		ID:          documentID,
		FileName:    job.payload.FileName,
		FilePath:    storagePath,
		FileURL:     h.storageService.GetURL(storagePath),
		S3URI:       buildS3URI(h.cfg, storagePath),
		Text:        finalExtraction.Text,
		Hash:        docHash,
		CreatedAt:   now,
		UpdatedAt:   now,
		Metadata:    metadata,
		Size:        int64(len(pdfBytes)),
		ContentType: "application/pdf",
	}

	if _, err := h.searchService.IndexDocument(ctx, document); err != nil {
		return dryJobResult{
			status: fiber.StatusInternalServerError,
			body: internalmodels.NewErrorResponse(
				"indexing_failed",
				"Failed to push document to OpenSearch",
				map[string]interface{}{"error": err.Error()},
			),
			err: err,
		}
	}

	// Force garbage collection after indexing to release accumulated document memory
	runtime.GC()
	h.logMemoryUsage("After indexing & GC", job.payload.FileName)

	response := fiber.Map{
		"document_id":    documentID,
		"storage_path":   storagePath,
		"file_url":       uploadResult.URL,
		"cdn_url":        h.storageService.GetURL(storagePath),
		"original_name":  job.payload.FileName,
		"text_chars":     len(finalExtraction.Text),
		"word_count":     finalExtraction.WordCount,
		"page_count":     finalExtraction.PageCount,
		"classification": fiber.Map{},
	}

	return dryJobResult{
		status: fiber.StatusCreated,
		body:   internalmodels.NewSuccessResponse(response, "Document processed successfully"),
		err:    nil,
	}
}

func (h *DryClassificationHandler) ensurePDFBytes(file *os.File, ext string, text string) ([]byte, error) {
	if strings.EqualFold(ext, ".pdf") {
		// For PDF files, read the original file content
		_, err := file.Seek(0, 0) // Reset to beginning
		if err != nil {
			return nil, fmt.Errorf("failed to seek to beginning of file: %w", err)
		}
		return io.ReadAll(file)
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
