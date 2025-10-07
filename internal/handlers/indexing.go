package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	internalModels "motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/processing/classifier"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/models"
)

// IndexingHandler handles direct document indexing operations
type IndexingHandler struct {
	search search.Service
}

// NewIndexingHandler creates a new indexing handler
func NewIndexingHandler(search search.Service) *IndexingHandler {
	return &IndexingHandler{
		search: search,
	}
}


// IndexDocument handles POST /api/v1/index/document - Direct document indexing
func (h *IndexingHandler) IndexDocument(c *fiber.Ctx) error {
	var request internalModels.IndexDocumentRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"parse_error",
			"Failed to parse request body",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Validate required fields
	if err := h.validateIndexRequest(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			err.Error(),
			nil,
		))
	}

	// Check search service health
	if !h.search.IsHealthy() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(internalModels.NewErrorResponse(
			"service_unavailable",
			"Search service is not healthy",
			nil,
		))
	}

	ctx := context.Background()

	// Index the document
	log.Printf("[INDEXING] Processing document: %s", request.DocumentID)
	indexID, err := h.indexDocument(ctx, &request)
	if err != nil {
		log.Printf("[INDEXING] ❌ Failed to index document %s: %v", request.DocumentID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"indexing_failed",
			fmt.Sprintf("Failed to index document: %v", err),
			map[string]interface{}{
				"document_id": request.DocumentID,
				"error":       err.Error(),
			},
		))
	}
	log.Printf("[INDEXING] ✅ Successfully indexed document %s with ID: %s", request.DocumentID, indexID)

	// Create response
	response := &internalModels.IndexDocumentResponse{
		DocumentID: request.DocumentID,
		IndexID:    indexID,
		Success:    true,
		Message:    "Document indexed successfully",
		IndexedAt:  time.Now(),
	}

	return c.JSON(internalModels.NewSuccessResponse(response, "Document indexed successfully"))
}

// validateIndexRequest validates the indexing request
func (h *IndexingHandler) validateIndexRequest(req *internalModels.IndexDocumentRequest) error {
	if req.DocumentID == "" {
		return fmt.Errorf("document_id is required")
	}
	if req.Text == "" {
		return fmt.Errorf("text is required")
	}
	if req.ClassificationResult == nil {
		return fmt.Errorf("classification_result is required")
	}
	if req.ClassificationResult.DocumentType == "" {
		return fmt.Errorf("classification_result.document_type is required")
	}
	if req.ClassificationResult.LegalCategory == "" {
		return fmt.Errorf("classification_result.legal_category is required")
	}

	return nil
}

// indexDocument performs the actual document indexing
func (h *IndexingHandler) indexDocument(ctx context.Context, req *internalModels.IndexDocumentRequest) (string, error) {
	// Prepare document for indexing
	now := time.Now()
	
	// Use provided values or generate defaults
	fileName := req.FileName
	if fileName == "" && req.DocumentPath != "" {
		fileName = filepath.Base(req.DocumentPath)
	}
	if fileName == "" {
		fileName = "unknown"
	}

	fileURL := req.FileURL
	if fileURL == "" && req.DocumentPath != "" {
		fileURL = fmt.Sprintf("/api/documents/%s", req.DocumentPath)
	}

	contentType := req.ContentType
	if contentType == "" {
		contentType = determineContentType(req.DocumentPath)
	}

	size := req.Size
	if size == 0 {
		size = int64(len(req.Text))
	}

	// Create search document
	searchDoc := &models.Document{
		ID:          req.DocumentID,
		FileName:    fileName,
		FilePath:    req.DocumentPath,
		FileURL:     fileURL,
		Text:        req.Text,
		DocType:     req.ClassificationResult.DocumentType,
		Category:    req.ClassificationResult.LegalCategory,
		Hash:        generateDocumentHash(req.Text),
		CreatedAt:   now,
		UpdatedAt:   now,
		ContentType: contentType,
		Size:        size,
		Metadata:    h.buildDocumentMetadata(req.ClassificationResult),
	}

	// Validate the document structure
	if err := h.validateDocumentForIndexing(searchDoc); err != nil {
		return "", fmt.Errorf("document validation failed: %w", err)
	}

	// Index the document
	log.Printf("[INDEXING] Calling OpenSearch IndexDocument for %s", req.DocumentID)
	indexID, err := h.search.IndexDocument(ctx, searchDoc)
	if err != nil {
		log.Printf("[INDEXING] ❌ OpenSearch IndexDocument failed for %s: %v", req.DocumentID, err)
		return "", fmt.Errorf("failed to index document: %w", err)
	}
	log.Printf("[INDEXING] ✅ OpenSearch IndexDocument succeeded for %s, got ID: %s", req.DocumentID, indexID)

	if indexID == "" {
		return "", fmt.Errorf("indexing succeeded but no document ID was returned")
	}

	return indexID, nil
}

// validateDocumentForIndexing validates that a document has all required fields
func (h *IndexingHandler) validateDocumentForIndexing(doc *models.Document) error {
	if doc.ID == "" {
		return fmt.Errorf("document ID cannot be empty")
	}
	if doc.Text == "" {
		return fmt.Errorf("document text cannot be empty")
	}
	if doc.FileName == "" {
		return fmt.Errorf("document filename cannot be empty")
	}
	if doc.DocType == "" {
		return fmt.Errorf("document type cannot be empty")
	}
	if doc.Hash == "" {
		return fmt.Errorf("document hash cannot be empty")
	}
	if doc.Metadata == nil {
		return fmt.Errorf("document metadata cannot be nil")
	}
	if doc.CreatedAt.IsZero() {
		return fmt.Errorf("document CreatedAt timestamp cannot be zero")
	}
	if doc.UpdatedAt.IsZero() {
		return fmt.Errorf("document UpdatedAt timestamp cannot be zero")
	}

	return nil
}

// buildDocumentMetadata creates document metadata from classification results
func (h *IndexingHandler) buildDocumentMetadata(classResult *classifier.ClassificationResult) *models.DocumentMetadata {
	metadata := &models.DocumentMetadata{
		DocumentName:  classResult.Subject,
		Subject:       classResult.Subject,
		Summary:       classResult.Summary,
		DocumentType:  models.DocumentType(classResult.DocumentType),
		Status:        classResult.Status,
		Language:      "en", // Default to English
		ProcessedAt:   time.Now(),
		Confidence:    classResult.Confidence,
		AIClassified:  true,
		LegalTags:     classResult.LegalTags,
	}

	// Convert case information
	if classResult.CaseInfo != nil {
		metadata.Case = &models.CaseInfo{
			CaseNumber:   classResult.CaseInfo.CaseNumber,
			CaseName:     classResult.CaseInfo.CaseName,
			CaseType:     classResult.CaseInfo.CaseType,
			Chapter:      classResult.CaseInfo.Chapter,
			Docket:       classResult.CaseInfo.Docket,
			NatureOfSuit: classResult.CaseInfo.NatureOfSuit,
		}
	}

	// Convert court information
	if classResult.CourtInfo != nil {
		metadata.Court = &models.CourtInfo{
			CourtID:      classResult.CourtInfo.CourtID,
			CourtName:    classResult.CourtInfo.CourtName,
			Jurisdiction: classResult.CourtInfo.Jurisdiction,
			Level:        classResult.CourtInfo.Level,
			District:     classResult.CourtInfo.District,
			Division:     classResult.CourtInfo.Division,
			County:       classResult.CourtInfo.County,
		}
	}

	// Convert parties
	if len(classResult.Parties) > 0 {
		metadata.Parties = make([]models.Party, len(classResult.Parties))
		for i, party := range classResult.Parties {
			metadata.Parties[i] = models.Party{
				Name:      party.Name,
				Role:      party.Role,
				PartyType: party.PartyType,
			}
		}
	}

	// Convert attorneys
	if len(classResult.Attorneys) > 0 {
		metadata.Attorneys = make([]models.Attorney, len(classResult.Attorneys))
		for i, attorney := range classResult.Attorneys {
			metadata.Attorneys[i] = models.Attorney{
				Name:         attorney.Name,
				BarNumber:    attorney.BarNumber,
				Role:         attorney.Role,
				Organization: attorney.Organization,
			}
		}
	}

	// Convert judge
	if classResult.Judge != nil {
		metadata.Judge = &models.Judge{
			Name:    classResult.Judge.Name,
			Title:   classResult.Judge.Title,
			JudgeID: classResult.Judge.JudgeID,
		}
	}

	// Convert charges
	if len(classResult.Charges) > 0 {
		metadata.Charges = make([]models.Charge, len(classResult.Charges))
		for i, charge := range classResult.Charges {
			metadata.Charges[i] = models.Charge{
				Statute:     charge.Statute,
				Description: charge.Description,
				Grade:       charge.Grade,
				Class:       charge.Class,
				Count:       charge.Count,
			}
		}
	}

	// Convert authorities
	if len(classResult.Authorities) > 0 {
		metadata.Authorities = make([]models.Authority, len(classResult.Authorities))
		for i, authority := range classResult.Authorities {
			metadata.Authorities[i] = models.Authority{
				Citation:  authority.Citation,
				CaseTitle: authority.CaseTitle,
				Type:      authority.Type,
				Precedent: authority.Precedent,
				Page:      authority.Page,
			}
		}
	}

	// Parse dates if available (all 5 enhanced date fields)
	if classResult.FilingDate != nil {
		if filingTime, err := time.Parse("2006-01-02", *classResult.FilingDate); err == nil {
			metadata.FilingDate = &filingTime
		}
	}

	if classResult.EventDate != nil {
		if eventTime, err := time.Parse("2006-01-02", *classResult.EventDate); err == nil {
			metadata.EventDate = &eventTime
		}
	}

	if classResult.HearingDate != nil {
		if hearingTime, err := time.Parse("2006-01-02", *classResult.HearingDate); err == nil {
			metadata.HearingDate = &hearingTime
		}
	}

	if classResult.DecisionDate != nil {
		if decisionTime, err := time.Parse("2006-01-02", *classResult.DecisionDate); err == nil {
			metadata.DecisionDate = &decisionTime
		}
	}

	if classResult.ServedDate != nil {
		if servedTime, err := time.Parse("2006-01-02", *classResult.ServedDate); err == nil {
			metadata.ServedDate = &servedTime
		}
	}

	return metadata
}

// generateDocumentHash creates a SHA-256 hash for the document content
func generateDocumentHash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// determineContentType determines the content type from file path
func determineContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".doc":
		return "application/msword"
	case ".txt":
		return "text/plain"
	case ".rtf":
		return "application/rtf"
	default:
		return "application/octet-stream"
	}
}