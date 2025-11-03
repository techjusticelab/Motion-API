package handlers

import (
	"context"
	"io"
	"strings"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/infrastructure/http/presenter"
)

// AnalyzeRedactionsUseCase defines the interface for redaction analysis
type AnalyzeRedactionsUseCase interface {
	Execute(ctx context.Context, req *dto.AnalyzeRedactionsRequest) (*dto.AnalyzeRedactionsResponse, error)
}

// ApplyRedactionsUseCase defines the interface for applying redactions
type ApplyRedactionsUseCase interface {
	Execute(ctx context.Context, req *dto.ApplyRedactionsRequest) (*dto.ApplyRedactionsResponse, error)
}

// RedactionHandler handles document redaction HTTP requests
type RedactionHandler struct {
	analyzeUseCase AnalyzeRedactionsUseCase
	applyUseCase   ApplyRedactionsUseCase
}

// NewRedactionHandler creates a new redaction handler
func NewRedactionHandler(analyzeUseCase AnalyzeRedactionsUseCase, applyUseCase ApplyRedactionsUseCase) *RedactionHandler {
	return &RedactionHandler{
		analyzeUseCase: analyzeUseCase,
		applyUseCase:   applyUseCase,
	}
}

// AnalyzeRedactions handles POST /redactions/analyze
func (h *RedactionHandler) AnalyzeRedactions(c *fiber.Ctx) error {
	// Parse multipart form for file upload
	file, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "File is required")
	}

	// Read file content
	fileReader, err := file.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to open file")
	}
	defer fileReader.Close()

	content, err := io.ReadAll(fileReader)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to read file content")
	}

	// Build use case request
	req := &dto.AnalyzeRedactionsRequest{
		PDFBase64: string(content),
		Options:   h.buildOptions(c),
	}

	// Execute use case
	result, err := h.analyzeUseCase.Execute(c.Context(), req)
	if err != nil {
		return err // Middleware will translate
	}

	// Present results
	return presenter.Success(c, presenter.PresentRedactionAnalysis(result))
}

// ApplyRedactions handles POST /redactions/apply
func (h *RedactionHandler) ApplyRedactions(c *fiber.Ctx) error {
	contentType := c.Get("Content-Type")

	// Handle multipart form or JSON
	if strings.Contains(contentType, "multipart/form-data") {
		return h.applyFromFile(c)
	}
	return h.applyFromJSON(c)
}

// applyFromFile handles redaction from uploaded file
func (h *RedactionHandler) applyFromFile(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "File is required")
	}

	fileReader, err := file.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to open file")
	}
	defer fileReader.Close()

	content, err := io.ReadAll(fileReader)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to read file content")
	}

	req := &dto.ApplyRedactionsRequest{
		PDFBase64:    string(content),
		Options:      h.buildOptions(c),
		ReturnBase64: c.FormValue("return_base64") == "true",
	}

	result, err := h.applyUseCase.Execute(c.Context(), req)
	if err != nil {
		return err
	}

	return presenter.Success(c, presenter.PresentRedactionResult(result))
}

// applyFromJSON handles redaction from JSON request
func (h *RedactionHandler) applyFromJSON(c *fiber.Ctx) error {
	var req dto.ApplyRedactionsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	result, err := h.applyUseCase.Execute(c.Context(), &req)
	if err != nil {
		return err
	}

	return presenter.Success(c, presenter.PresentRedactionResult(result))
}

// buildOptions builds redaction options from form values
func (h *RedactionHandler) buildOptions(c *fiber.Ctx) *dto.RedactionOptions {
	return &dto.RedactionOptions{
		UseAI:           c.FormValue("use_ai") == "true",
		CaliforniaLaws:  c.FormValue("california_laws") != "false",
		ReplacementChar: c.FormValue("replacement_char"),
	}
}
