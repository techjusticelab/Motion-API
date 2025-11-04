package extract

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jung-kurt/gofpdf"

	"motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/processing/extractor"
)

const (
	requestTimeout = 2 * time.Minute
)

// Handler provides endpoints for text extraction and document conversion
type Handler struct {
	extractor extractor.Service
}

// NewHandler creates a new extract handler
func NewHandler(extractorSvc extractor.Service) *Handler {
	return &Handler{
		extractor: extractorSvc,
	}
}

// PDFToText extracts text from a PDF document and returns plain text response
func (h *Handler) PDFToText(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"file_required",
			"No file provided in request",
			map[string]interface{}{"field": "file"},
		))
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".pdf" {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"invalid_file_type",
			"Only PDF files are supported for text extraction",
			map[string]interface{}{"provided_extension": ext},
		))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"file_open_failed",
			"Failed to open uploaded file",
			map[string]interface{}{"error": err.Error()},
		))
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(c.Context(), requestTimeout)
	defer cancel()

	metadata := &extractor.DocumentMetadata{
		FileName: fileHeader.Filename,
		MimeType: fileHeader.Header.Get("Content-Type"),
		Size:     fileHeader.Size,
		Format:   "pdf",
	}

	result, err := h.extractor.ExtractText(ctx, file, metadata)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"extraction_failed",
			"Failed to extract text from PDF",
			map[string]interface{}{"error": err.Error()},
		))
	}

	c.Set(fiber.HeaderContentType, "text/plain; charset=utf-8")
	return c.SendString(result.Text)
}

// ConvertToPDF converts supported document formats to PDF and returns the PDF file
func (h *Handler) ConvertToPDF(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"file_required",
			"No file provided in request",
			map[string]interface{}{"field": "file"},
		))
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == "" {
		ext = inferExtensionFromMime(fileHeader.Header.Get("Content-Type"))
	}

	if !isConvertibleFormat(ext) {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"unsupported_format",
			"Only pptx, doc, docx, pdf, ppt, and txt files can be converted to PDF",
			map[string]interface{}{"provided_extension": ext},
		))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"file_open_failed",
			"Failed to open uploaded file",
			map[string]interface{}{"error": err.Error()},
		))
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"file_read_failed",
			"Failed to read uploaded file",
			map[string]interface{}{"error": err.Error()},
		))
	}

	format := strings.TrimPrefix(ext, ".")
	metadata := &extractor.DocumentMetadata{
		FileName: fileHeader.Filename,
		MimeType: fileHeader.Header.Get("Content-Type"),
		Size:     fileHeader.Size,
		Format:   format,
	}

	var pdfBytes []byte

	if format == "pdf" {
		pdfBytes = content
	} else {
		ctx, cancel := context.WithTimeout(c.Context(), requestTimeout)
		defer cancel()

		result, err := h.extractor.ExtractText(ctx, bytes.NewReader(content), metadata)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
				"conversion_failed",
				"Failed to extract text for PDF conversion",
				map[string]interface{}{"error": err.Error()},
			))
		}

		pdfBytes, err = generatePDFDocument(result.Text)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
				"pdf_generation_failed",
				"Failed to generate PDF document",
				map[string]interface{}{"error": err.Error()},
			))
		}
	}

	outputName := buildPDFFileName(fileHeader.Filename)
	c.Set(fiber.HeaderContentType, "application/pdf")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf("attachment; filename=\"%s\"", outputName))

	return c.Send(pdfBytes)
}

func isConvertibleFormat(ext string) bool {
	allowed := map[string]bool{
		".pptx": true,
		".doc":  true,
		".docx": true,
		".pdf":  true,
		".ppt":  true,
		".txt":  true,
	}
	return allowed[strings.ToLower(ext)]
}

func inferExtensionFromMime(mime string) string {
	switch strings.ToLower(mime) {
	case "application/pdf":
		return ".pdf"
	case "application/msword":
		return ".doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "application/vnd.ms-powerpoint":
		return ".ppt"
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return ".pptx"
	case "text/plain":
		return ".txt"
	default:
		return ""
	}
}

func buildPDFFileName(original string) string {
	if original == "" {
		return "document.pdf"
	}

	ext := filepath.Ext(original)
	base := strings.TrimSuffix(original, ext)
	if base == "" {
		base = "document"
	}
	return base + ".pdf"
}

func generatePDFDocument(text string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 12)

	lines := strings.Split(text, "\n")
	for _, line := range lines {
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
