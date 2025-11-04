package processing

import (
	"bytes"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/infrastructure/http/presenter"
)

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
