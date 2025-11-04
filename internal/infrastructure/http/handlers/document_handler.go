package handlers

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/infrastructure/http/presenter"
)

// ProcessDocumentUseCase defines the interface for document processing
type ProcessDocumentUseCase interface {
	Execute(ctx context.Context, req *dto.ProcessDocumentRequest) (*dto.ProcessDocumentResponse, error)
}

// BatchProcessDocumentUseCase defines the interface for batch document processing
type BatchProcessDocumentUseCase interface {
	Execute(ctx context.Context, req *dto.BatchProcessDocumentRequest) (*dto.BatchProcessDocumentResponse, error)
}

// IndexDocumentUseCase defines the interface for document indexing
type IndexDocumentUseCase interface {
	Execute(ctx context.Context, req *dto.IndexDocumentRequest) (*dto.IndexDocumentResponse, error)
}

// UpdateMetadataUseCase defines the interface for metadata updates
type UpdateMetadataUseCase interface {
	Execute(ctx context.Context, req *dto.UpdateMetadataRequest) (*dto.UpdateMetadataResponse, error)
}

// DocumentHandler handles document-related HTTP requests
type DocumentHandler struct {
	processUseCase        ProcessDocumentUseCase
	batchProcessUseCase   BatchProcessDocumentUseCase
	indexUseCase          IndexDocumentUseCase
	updateMetadataUseCase UpdateMetadataUseCase
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(
	processUseCase ProcessDocumentUseCase,
	batchProcessUseCase BatchProcessDocumentUseCase,
	indexUseCase IndexDocumentUseCase,
	updateMetadataUseCase UpdateMetadataUseCase,
) *DocumentHandler {
	return &DocumentHandler{
		processUseCase:        processUseCase,
		batchProcessUseCase:   batchProcessUseCase,
		indexUseCase:          indexUseCase,
		updateMetadataUseCase: updateMetadataUseCase,
	}
}

// ProcessDocument handles POST /documents
func (h *DocumentHandler) ProcessDocument(c *fiber.Ctx) error {
	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "File is required")
	}

	// Open file
	fileReader, err := file.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to open file")
	}
	defer fileReader.Close()

	// Read file content
	content, err := io.ReadAll(fileReader)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to read file content")
	}

	// Build use case request
	req, err := h.buildProcessRequest(c, file, content)
	if err != nil {
		return err
	}

	// Execute use case
	result, err := h.processUseCase.Execute(c.Context(), req)
	if err != nil {
		return err // Middleware will translate
	}

	// Present results
	return presenter.Success(c, presenter.PresentDocument(result))
}

// BatchProcessDocuments handles POST /documents/batch
func (h *DocumentHandler) BatchProcessDocuments(c *fiber.Ctx) error {
	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Failed to parse multipart form")
	}

	files := form.File["files"]
	if len(files) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "At least one file is required")
	}

	// Build batch request
	batchReq := &dto.BatchProcessDocumentRequest{
		Documents: make([]*dto.ProcessDocumentRequest, 0, len(files)),
	}

	for _, file := range files {
		fileReader, err := file.Open()
		if err != nil {
			continue // Skip files that can't be opened
		}
		defer fileReader.Close()

		content, err := io.ReadAll(fileReader)
		if err != nil {
			continue // Skip files that can't be read
		}

		req, err := h.buildProcessRequest(c, file, content)
		if err != nil {
			continue // Skip invalid files
		}

		batchReq.Documents = append(batchReq.Documents, req)
	}

	// Execute batch use case
	result, err := h.batchProcessUseCase.Execute(c.Context(), batchReq)
	if err != nil {
		return err // Middleware will translate
	}

	// Present results
	return presenter.Success(c, presenter.PresentBatchDocument(result))
}

// IndexDocument handles POST /documents/:id/index
func (h *DocumentHandler) IndexDocument(c *fiber.Ctx) error {
	documentID := c.Params("id")
	if documentID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Document ID is required")
	}

	// Parse request body for optional force parameter
	var body struct {
		Force bool `json:"force"`
	}
	_ = c.BodyParser(&body) // Ignore parsing errors, use defaults

	// Build use case request
	req := &dto.IndexDocumentRequest{
		DocumentID: documentID,
		Force:      body.Force,
	}

	// Execute use case
	result, err := h.indexUseCase.Execute(c.Context(), req)
	if err != nil {
		return err // Middleware will translate
	}

	// Present results
	return presenter.Success(c, presenter.PresentIndex(result))
}

// UpdateMetadata handles PUT /documents/:id/metadata
func (h *DocumentHandler) UpdateMetadata(c *fiber.Ctx) error {
	documentID := c.Params("id")
	if documentID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Document ID is required")
	}

	// Parse request body
	var body dto.UpdateMetadataRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// Set document ID from URL
	body.DocumentID = documentID

	// Execute use case
	result, err := h.updateMetadataUseCase.Execute(c.Context(), &body)
	if err != nil {
		return err // Middleware will translate
	}

	// Present results
	return presenter.Success(c, presenter.PresentMetadata(result))
}

// buildProcessRequest builds a ProcessDocumentRequest from HTTP request
func (h *DocumentHandler) buildProcessRequest(
	c *fiber.Ctx,
	file *multipart.FileHeader,
	content []byte,
) (*dto.ProcessDocumentRequest, error) {
	// Generate document ID from filename and timestamp
	// This is a simplified version - in production you'd use a proper ID generator
	documentID := generateDocumentID(file.Filename)

	return &dto.ProcessDocumentRequest{
		ID:            documentID,
		FileName:      file.Filename,
		Content:       nil, // Content already read for hash calculation
		ContentSize:   file.Size,
		ContentType:   file.Header.Get("Content-Type"),
		StoragePath:   generateStoragePath(documentID, file.Filename),
		Text:          "", // Will be extracted by use case
		HashValue:     calculateHash(content),
		HashAlgorithm: "SHA256",
		StoreBinary:   c.FormValue("store_document") != "false",
		Classify:      c.FormValue("classify_doc") != "false",
		Index:         c.FormValue("index_document") != "false",
		Metadata: map[string]string{
			"case_name":   c.FormValue("case_name"),
			"case_number": c.FormValue("case_number"),
			"author":      c.FormValue("author"),
			"court":       c.FormValue("court"),
		},
	}, nil
}

// Helper functions (simplified versions)
func generateDocumentID(filename string) string {
	return "doc_" + filename // Simplified - use proper UUID in production
}

func generateStoragePath(id, filename string) string {
	return "documents/" + id + "/" + filename
}

func calculateHash(content []byte) string {
	return "hash_placeholder" // Simplified - calculate actual SHA256 hash in production
}
