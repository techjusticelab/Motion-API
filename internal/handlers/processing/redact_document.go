package processing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	internalModels "motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/processing/redaction"
)

// RedactDocument creates a redacted version of a document
func (h *Handler) RedactDocument(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Minute)
	defer cancel()

	// Handle multipart form for file upload or JSON for existing document
	contentType := c.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {
		return h.redactUploadedFile(c, ctx)
	} else {
		return h.redactExistingDocument(c, ctx)
	}
}

// redactUploadedFile handles redaction of an uploaded PDF file
func (h *Handler) redactUploadedFile(c *fiber.Ctx, ctx context.Context) error {
	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"file_error",
			"No file provided or failed to parse file",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Validate file type
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".pdf") {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"file_type_error",
			"Only PDF files are supported for redaction",
			nil,
		))
	}

	// Open the uploaded file
	fileReader, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"file_read_error",
			"Failed to read uploaded file",
			map[string]interface{}{"error": err.Error()},
		))
	}
	defer fileReader.Close()

	// Parse redaction options
	options := &redaction.Options{
		CaliforniaLaws:  true, // Default to California laws
		ReplacementChar: "■",
	}

	// Parse form values for options
	if useAI := c.FormValue("use_ai"); useAI == "true" {
		options.UseAI = true
	}
	if replacementChar := c.FormValue("replacement_char"); replacementChar != "" {
		options.ReplacementChar = replacementChar
	}

	// Create redaction service
	redactionService := redaction.NewService(true, h.cfg.OpenAI.APIKey)

	// Determine if we should apply redactions or just analyze
	applyRedactions := c.FormValue("apply_redactions") == "true"

	if applyRedactions {
		// Apply redactions and return redacted PDF
		result, err := redactionService.RedactPDF(ctx, fileReader, options)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
				"redaction_error",
				"Failed to redact document",
				map[string]interface{}{"error": err.Error()},
			))
		}

		response := &internalModels.RedactDocumentResponse{
			Success:         result.Success,
			PDFBase64:       result.PDFBase64,
			Filename:        fmt.Sprintf("redacted_%s", file.Filename),
			Redactions:      convertRedactionItems(result.Redactions),
			TotalRedactions: result.TotalCount,
			Message:         "Document redacted successfully",
		}

		if !result.Success {
			response.Message = result.Error
			return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
				"redaction_failed",
				result.Error,
				nil,
			))
		}

		return c.JSON(internalModels.NewSuccessResponse(response, "Document redacted successfully"))
	} else {
		// Just analyze for potential redactions
		analysis, err := redactionService.AnalyzePDF(ctx, fileReader, options)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
				"analysis_error",
				"Failed to analyze document",
				map[string]interface{}{"error": err.Error()},
			))
		}

		response := &internalModels.RedactDocumentResponse{
			Success:         analysis.Success,
			Filename:        file.Filename,
			Redactions:      convertRedactionItems(analysis.Redactions),
			TotalRedactions: analysis.TotalCount,
			Message:         "Document analyzed for potential redactions",
		}

		if !analysis.Success {
			response.Message = analysis.Error
			return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
				"analysis_failed",
				analysis.Error,
				nil,
			))
		}

		return c.JSON(internalModels.NewSuccessResponse(response, "Document analysis completed"))
	}
}

// redactExistingDocument handles redaction of an existing document by ID
func (h *Handler) redactExistingDocument(c *fiber.Ctx, ctx context.Context) error {
	var request internalModels.RedactDocumentRequest

	// Parse request body
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"parse_error",
			"Failed to parse request body",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Validate request - need either document_id or pdf_base64
	if request.DocumentID == "" && request.PDFBase64 == "" {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			"Either document_id or pdf_base64 is required",
			nil,
		))
	}

	// Placeholder response
	response := &internalModels.RedactDocumentResponse{
		Success:         false,
		DocumentID:      request.DocumentID,
		Redactions:      request.CustomRedactions,
		TotalRedactions: len(request.CustomRedactions),
		Message:         "Document redaction by ID is not yet fully implemented - requires document retrieval from storage",
	}

	return c.JSON(internalModels.NewSuccessResponse(response, "Redaction request processed"))
}

// convertRedactionItems converts between redaction types
func convertRedactionItems(items []redaction.RedactionItem) []internalModels.RedactionItem {
	result := make([]internalModels.RedactionItem, len(items))
	for i, item := range items {
		result[i] = internalModels.RedactionItem{
			ID:        item.ID,
			Page:      item.Page,
			Text:      item.Text,
			BBox:      item.BBox,
			Type:      item.Type,
			Citation:  item.Citation,
			Reason:    item.Reason,
			LegalCode: item.LegalCode,
			Applied:   item.Applied,
		}
	}
	return result
}

// convertInternalRedactionItems converts internal redaction items to service type
func convertInternalRedactionItems(items []internalModels.RedactionItem) []redaction.RedactionItem {
	result := make([]redaction.RedactionItem, len(items))
	for i, item := range items {
		result[i] = redaction.RedactionItem{
			ID:        item.ID,
			Page:      item.Page,
			Text:      item.Text,
			BBox:      item.BBox,
			Type:      item.Type,
			Citation:  item.Citation,
			Reason:    item.Reason,
			LegalCode: item.LegalCode,
			Applied:   item.Applied,
		}
	}
	return result
}
