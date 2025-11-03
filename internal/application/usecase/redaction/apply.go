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

// ApplyRedactionsUseCase orchestrates applying redactions to a PDF.
type ApplyRedactionsUseCase struct {
	repo      ports.DocumentRepository
	storage   ports.StorageService
	redaction ports.RedactionService
}

// NewApplyRedactionsUseCase wires dependencies required for redaction application.
func NewApplyRedactionsUseCase(
	repo ports.DocumentRepository,
	storage ports.StorageService,
	redaction ports.RedactionService,
) *ApplyRedactionsUseCase {
	return &ApplyRedactionsUseCase{
		repo:      repo,
		storage:   storage,
		redaction: redaction,
	}
}

// Execute applies redactions, optionally using custom regions supplied by the caller.
func (uc *ApplyRedactionsUseCase) Execute(ctx context.Context, req *dto.ApplyRedactionsRequest) (*dto.ApplyRedactionsResponse, error) {
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
	custom := fromDTOItems(req.CustomRedactions)

	var result *ports.RedactionResult
	if len(custom) > 0 {
		result, err = uc.redaction.ApplyCustom(ctx, reader, options, custom)
	} else {
		result, err = uc.redaction.Redact(ctx, reader, options)
	}
	if err != nil {
		return nil, err
	}

	response := &dto.ApplyRedactionsResponse{
		DocumentID:      docID,
		FileName:        fileName,
		Redactions:      toDTOItems(result.Redactions),
		TotalRedactions: result.TotalCount,
	}

	if req.ReturnBase64 {
		response.PDFBase64 = result.PDFBase64
	}

	return response, nil
}

func (uc *ApplyRedactionsUseCase) resolveSource(ctx context.Context, req *dto.ApplyRedactionsRequest) (io.Reader, string, string, func() error, error) {
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

func fromDTOItems(items []dto.RedactionItem) []ports.RedactionItem {
	if len(items) == 0 {
		return nil
	}
	result := make([]ports.RedactionItem, len(items))
	for i, item := range items {
		result[i] = ports.RedactionItem{
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
