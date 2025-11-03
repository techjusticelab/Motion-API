package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/infrastructure/http/presenter"
)

// Use case interfaces for dependency injection and testing
type ProcessDocumentExecutor interface {
	Execute(ctx context.Context, req *dto.ProcessDocumentRequest) (*dto.ProcessDocumentResponse, error)
}

type BatchProcessExecutor interface {
	Execute(ctx context.Context, req *dto.BatchProcessDocumentRequest) (*dto.BatchProcessDocumentResponse, error)
}

type UpdateMetadataExecutor interface {
	Execute(ctx context.Context, req *dto.UpdateMetadataRequest) (*dto.UpdateMetadataResponse, error)
}

type AnalyzeRedactionsExecutor interface {
	Execute(ctx context.Context, req *dto.AnalyzeRedactionsRequest) (*dto.AnalyzeRedactionsResponse, error)
}

type ApplyRedactionsExecutor interface {
	Execute(ctx context.Context, req *dto.ApplyRedactionsRequest) (*dto.ApplyRedactionsResponse, error)
}

// ProcessingHandlerRefactored handles document processing using DDD use cases.
type ProcessingHandlerRefactored struct {
	processUC          ProcessDocumentExecutor
	batchUC            BatchProcessExecutor
	updateMetadataUC   UpdateMetadataExecutor
	analyzeRedactionUC AnalyzeRedactionsExecutor
	applyRedactionUC   ApplyRedactionsExecutor
}

// NewProcessingHandlerRefactored creates a refactored processing handler with use case injection.
func NewProcessingHandlerRefactored(
	processUC ProcessDocumentExecutor,
	batchUC BatchProcessExecutor,
	updateMetadataUC UpdateMetadataExecutor,
	analyzeRedactionUC AnalyzeRedactionsExecutor,
	applyRedactionUC ApplyRedactionsExecutor,
) *ProcessingHandlerRefactored {
	return &ProcessingHandlerRefactored{
		processUC:          processUC,
		batchUC:            batchUC,
		updateMetadataUC:   updateMetadataUC,
		analyzeRedactionUC: analyzeRedactionUC,
		applyRedactionUC:   applyRedactionUC,
	}
}

// ProcessDocument handles POST /api/documents/process - Process a single document upload.
func (h *ProcessingHandlerRefactored) ProcessDocument(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return presenter.BadRequest(c, "No file provided", map[string]string{"error": err.Error()})
	}

	// Read file content and calculate hash
	content, hash, err := readAndHashFile(file)
	if err != nil {
		return presenter.InternalError(c, "Failed to read file")
	}

	// Parse processing options
	storeDoc := c.FormValue("store_document") != "false"
	classifyDoc := c.FormValue("classify_doc") != "false"
	indexDoc := c.FormValue("index_document") != "false"

	// Build DTO request
	req := &dto.ProcessDocumentRequest{
		ID:            generateDocumentID(file.Filename),
		FileName:      file.Filename,
		Content:       bytes.NewReader(content),
		ContentSize:   file.Size,
		ContentType:   file.Header.Get("Content-Type"),
		StoragePath:   fmt.Sprintf("documents/%s/%s", time.Now().Format("2006/01"), file.Filename),
		Text:          "", // Text extraction would happen in processor
		HashValue:     hash,
		HashAlgorithm: "SHA256",
		StoreBinary:   storeDoc,
		Classify:      classifyDoc,
		Index:         indexDoc,
		Metadata:      parseMetadata(c),
	}

	// Execute use case
	result, err := h.processUC.Execute(c.Context(), req)
	if err != nil {
		return presenter.InternalError(c, fmt.Sprintf("Processing failed: %v", err))
	}

	// Present response
	return presenter.Success(c, presenter.PresentDocument(result))
}

// BatchProcessDocuments handles POST /api/documents/batch - Process multiple documents.
func (h *ProcessingHandlerRefactored) BatchProcessDocuments(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return presenter.BadRequest(c, "Failed to parse form", nil)
	}

	files := form.File["files"]
	if len(files) == 0 {
		return presenter.BadRequest(c, "No files provided", nil)
	}

	// Build batch request
	docs := make([]*dto.ProcessDocumentRequest, 0, len(files))
	for _, file := range files {
		content, hash, err := readAndHashFile(file)
		if err != nil {
			continue // Skip failed files
		}

		docs = append(docs, &dto.ProcessDocumentRequest{
			ID:            generateDocumentID(file.Filename),
			FileName:      file.Filename,
			Content:       bytes.NewReader(content),
			ContentSize:   file.Size,
			ContentType:   file.Header.Get("Content-Type"),
			StoragePath:   fmt.Sprintf("documents/%s/%s", time.Now().Format("2006/01"), file.Filename),
			HashValue:     hash,
			HashAlgorithm: "SHA256",
			StoreBinary:   true,
			Classify:      true,
			Index:         true,
		})
	}

	batchReq := &dto.BatchProcessDocumentRequest{Documents: docs}

	// Execute batch use case
	result, err := h.batchUC.Execute(c.Context(), batchReq)
	if err != nil {
		return presenter.InternalError(c, fmt.Sprintf("Batch processing failed: %v", err))
	}

	// Present response
	return presenter.Success(c, presenter.PresentBatchDocument(result))
}

// UpdateMetadata handles PUT /api/documents/:id/metadata - Update document metadata.
func (h *ProcessingHandlerRefactored) UpdateMetadata(c *fiber.Ctx) error {
	var body struct {
		Language     string   `json:"language"`
		LegalTags    []string `json:"legal_tags"`
		AIClassified bool     `json:"ai_classified"`
	}

	if err := c.BodyParser(&body); err != nil {
		return presenter.BadRequest(c, "Invalid request body", nil)
	}

	req := &dto.UpdateMetadataRequest{
		DocumentID:   c.Params("id"),
		Language:     body.Language,
		LegalTags:    body.LegalTags,
		AIClassified: body.AIClassified,
	}

	result, err := h.updateMetadataUC.Execute(c.Context(), req)
	if err != nil {
		return presenter.InternalError(c, fmt.Sprintf("Metadata update failed: %v", err))
	}

	return presenter.Success(c, presenter.PresentMetadata(result))
}

// AnalyzeRedactions handles POST /api/documents/redactions/analyze - Analyze document for redactions.
func (h *ProcessingHandlerRefactored) AnalyzeRedactions(c *fiber.Ctx) error {
	contentType := c.Get("Content-Type")

	var req *dto.AnalyzeRedactionsRequest

	if strings.Contains(contentType, "multipart/form-data") {
		req = h.parseRedactionFromMultipart(c)
	} else {
		req = h.parseRedactionFromJSON(c)
	}

	if req == nil {
		return presenter.BadRequest(c, "Invalid request", nil)
	}

	result, err := h.analyzeRedactionUC.Execute(c.Context(), req)
	if err != nil {
		return presenter.InternalError(c, fmt.Sprintf("Redaction analysis failed: %v", err))
	}

	return presenter.Success(c, result)
}

// ApplyRedactions handles POST /api/documents/redactions/apply - Apply redactions to document.
func (h *ProcessingHandlerRefactored) ApplyRedactions(c *fiber.Ctx) error {
	var body struct {
		DocumentID       string             `json:"document_id"`
		PDFBase64        string             `json:"pdf_base64"`
		CustomRedactions []dto.RedactionItem `json:"custom_redactions"`
		Options          *dto.RedactionOptions `json:"options"`
	}

	if err := c.BodyParser(&body); err != nil {
		return presenter.BadRequest(c, "Invalid request body", nil)
	}

	req := &dto.ApplyRedactionsRequest{
		DocumentID:       body.DocumentID,
		PDFBase64:        body.PDFBase64,
		Options:          body.Options,
		CustomRedactions: body.CustomRedactions,
		ReturnBase64:     true,
	}

	result, err := h.applyRedactionUC.Execute(c.Context(), req)
	if err != nil {
		return presenter.InternalError(c, fmt.Sprintf("Redaction application failed: %v", err))
	}

	return presenter.Success(c, result)
}

// Helper functions

func readAndHashFile(file *multipart.FileHeader) ([]byte, string, error) {
	f, err := file.Open()
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return nil, "", err
	}

	hash := sha256.Sum256(content)
	return content, hex.EncodeToString(hash[:]), nil
}

func parseMetadata(c *fiber.Ctx) map[string]string {
	metadata := make(map[string]string)
	if caseName := c.FormValue("case_name"); caseName != "" {
		metadata["case_name"] = caseName
	}
	if caseNumber := c.FormValue("case_number"); caseNumber != "" {
		metadata["case_number"] = caseNumber
	}
	if author := c.FormValue("author"); author != "" {
		metadata["author"] = author
	}
	return metadata
}

func (h *ProcessingHandlerRefactored) parseRedactionFromMultipart(c *fiber.Ctx) *dto.AnalyzeRedactionsRequest {
	file, err := c.FormFile("file")
	if err != nil {
		return nil
	}

	content, _, err := readAndHashFile(file)
	if err != nil {
		return nil
	}

	return &dto.AnalyzeRedactionsRequest{
		PDFBase64: base64.StdEncoding.EncodeToString(content),
		Options: &dto.RedactionOptions{
			UseAI:          c.FormValue("use_ai") == "true",
			CaliforniaLaws: c.FormValue("california_laws") != "false",
		},
	}
}

func (h *ProcessingHandlerRefactored) parseRedactionFromJSON(c *fiber.Ctx) *dto.AnalyzeRedactionsRequest {
	var body struct {
		DocumentID string                `json:"document_id"`
		PDFBase64  string                `json:"pdf_base64"`
		Options    *dto.RedactionOptions `json:"options"`
	}

	if err := c.BodyParser(&body); err != nil {
		return nil
	}

	return &dto.AnalyzeRedactionsRequest{
		DocumentID: body.DocumentID,
		PDFBase64:  body.PDFBase64,
		Options:    body.Options,
	}
}
