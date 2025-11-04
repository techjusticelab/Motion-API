package processing

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"mime/multipart"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/application/dto"
)

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
