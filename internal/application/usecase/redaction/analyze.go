package redaction

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/application/ports"
	"motion-index-fiber/internal/domain/document"
	domainerrors "motion-index-fiber/internal/domain/errors"
)

// AnalyzeRedactionsUseCase orchestrates a redaction analysis run.
type AnalyzeRedactionsUseCase struct {
	repo      ports.DocumentRepository
	storage   ports.StorageService
	redaction ports.RedactionService
}

// NewAnalyzeRedactionsUseCase wires dependencies required for analysis.
func NewAnalyzeRedactionsUseCase(
	repo ports.DocumentRepository,
	storage ports.StorageService,
	redaction ports.RedactionService,
) *AnalyzeRedactionsUseCase {
	return &AnalyzeRedactionsUseCase{
		repo:      repo,
		storage:   storage,
		redaction: redaction,
	}
}

// Execute analyses a document or uploaded PDF for redaction candidates.
func (uc *AnalyzeRedactionsUseCase) Execute(ctx context.Context, req *dto.AnalyzeRedactionsRequest) (*dto.AnalyzeRedactionsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if uc.redaction == nil {
		return nil, fmt.Errorf("redaction service is not configured")
	}

	reader, fileName, docID, cleanup, err := uc.resolveSource(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cleanup != nil {
			_ = cleanup()
		}
	}()

	options := toPortOptions(req.Options)
	analysis, err := uc.redaction.Analyze(ctx, reader, options)
	if err != nil {
		return nil, err
	}

	return &dto.AnalyzeRedactionsResponse{
		DocumentID: docID,
		FileName:   fileName,
		Redactions: toDTOItems(analysis.Redactions),
		TotalCount: analysis.TotalCount,
	}, nil
}

func (uc *AnalyzeRedactionsUseCase) resolveSource(ctx context.Context, req *dto.AnalyzeRedactionsRequest) (io.Reader, string, string, func() error, error) {
	if req.PDFBase64 != "" {
		data, err := base64.StdEncoding.DecodeString(req.PDFBase64)
		if err != nil {
			return nil, "", "", nil, fmt.Errorf("failed to decode pdf: %w", err)
		}
		return bytes.NewReader(data), "", req.DocumentID, nil, nil
	}

	if uc.repo == nil {
		return nil, "", "", nil, fmt.Errorf("document repository is not configured")
	}
	if uc.storage == nil {
		return nil, "", "", nil, fmt.Errorf("storage service is not configured")
	}

	docID, err := document.NewDocumentID(req.DocumentID)
	if err != nil {
		return nil, "", "", nil, err
	}

	doc, err := uc.repo.FindByID(ctx, docID)
	if err != nil {
		return nil, "", "", nil, err
	}
	if doc == nil {
		return nil, "", "", nil, domainerrors.ErrDocumentNotFound
	}

	filePath := doc.FilePath().String()
	if filePath == "" {
		return nil, "", "", nil, fmt.Errorf("document %s has no storage path", doc.ID().String())
	}

	reader, err := uc.storage.Retrieve(ctx, filePath)
	if err != nil {
		return nil, "", "", nil, err
	}

	cleanup := func() error {
		return reader.Close()
	}

	return reader, doc.FileName().String(), doc.ID().String(), cleanup, nil
}

func toPortOptions(opts *dto.RedactionOptions) *ports.RedactionOptions {
	normalized := opts.Normalize()
	return &ports.RedactionOptions{
		UseAI:           normalized.UseAI,
		CaliforniaLaws:  normalized.CaliforniaLaws,
		IncludePatterns: append([]string(nil), normalized.IncludePatterns...),
		ExcludePatterns: append([]string(nil), normalized.ExcludePatterns...),
		ReplacementChar: normalized.ReplacementChar,
	}
}

func toDTOItems(items []ports.RedactionItem) []dto.RedactionItem {
	if len(items) == 0 {
		return nil
	}
	result := make([]dto.RedactionItem, len(items))
	for i, item := range items {
		result[i] = dto.RedactionItem{
			ID:        item.ID,
			Page:      item.Page,
			Text:      item.Text,
			BBox:      append([]float64(nil), item.BBox...),
			Type:      item.Type,
			Citation:  item.Citation,
			Reason:    item.Reason,
			LegalCode: item.LegalCode,
			Applied:   item.Applied,
		}
	}
	return result
}
