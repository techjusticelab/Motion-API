package storage

import (
	"context"
	"fmt"
	"strings"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/pkg/storage"
)

// GetStorageObjectURLUseCase handles getting URLs for storage objects.
type GetStorageObjectURLUseCase struct {
	storage storage.Service
}

// NewGetStorageObjectURLUseCase creates a new use case instance.
func NewGetStorageObjectURLUseCase(storage storage.Service) *GetStorageObjectURLUseCase {
	return &GetStorageObjectURLUseCase{storage: storage}
}

// Execute validates the path, checks existence, and returns the appropriate URL.
func (uc *GetStorageObjectURLUseCase) Execute(ctx context.Context, req *dto.GetStorageObjectURLRequest) (*dto.GetStorageObjectURLResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Normalize path - ensure it starts with documents/
	documentPath := req.Path
	if !strings.HasPrefix(documentPath, "documents/") {
		documentPath = "documents/" + documentPath
	}

	// Check if document exists
	exists, err := uc.storage.Exists(ctx, documentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check document existence: %w", err)
	}

	if !exists {
		// Try to recover path by searching
		if recoveredPath, found := uc.resolveDocumentPath(ctx, documentPath); found {
			documentPath = recoveredPath
		} else {
			return nil, fmt.Errorf("document not found: %s", documentPath)
		}
	}

	// Generate URL based on type
	var documentURL string
	if req.UseSignedURL {
		documentURL, err = uc.storage.GetSignedURL(documentPath, req.Expiration)
		if err != nil {
			return nil, fmt.Errorf("failed to generate signed URL: %w", err)
		}
	} else {
		documentURL = uc.storage.GetURL(documentPath)
		if documentURL == "" {
			return nil, fmt.Errorf("failed to generate document URL")
		}
	}

	// Get content type
	ext := getFileExtension(documentPath)
	contentType := getContentTypeFromExtension(ext)

	return &dto.GetStorageObjectURLResponse{
		URL:          documentURL,
		Path:         documentPath,
		ContentType:  contentType,
		IsSignedURL:  req.UseSignedURL,
		ExpiresIn:    req.Expiration,
	}, nil
}

// resolveDocumentPath attempts to recover the correct storage path when exact path is unknown.
func (uc *GetStorageObjectURLUseCase) resolveDocumentPath(ctx context.Context, requestedPath string) (string, bool) {
	// Ensure prefix for consistency
	path := requestedPath
	if !strings.HasPrefix(path, "documents/") {
		path = "documents/" + path
	}

	base := getFilename(path)
	if base == "" || base == "documents" {
		return "", false
	}

	// Build candidate prefixes
	var prefixes []string
	parts := strings.Split(base, "_")

	if len(parts) >= 3 && strings.HasPrefix(parts[0], "doc") {
		// Example: doc_1759855215756364200_2015-Dependency-...pdf
		docIDPrefix := strings.Join(parts[0:2], "_") // doc_<number>
		name := strings.Join(parts[2:], "_")         // filename
		nameNoExt := strings.TrimSuffix(name, getFileExtension(name))

		prefixes = append(prefixes,
			"documents/"+docIDPrefix,
			"documents/"+name,
		)
		if nameNoExt != "" {
			prefixes = append(prefixes, "documents/"+nameNoExt)
		}
	} else {
		// Fall back to filename-based prefixes
		name := base
		nameNoExt := strings.TrimSuffix(name, getFileExtension(name))
		prefixes = append(prefixes, "documents/"+name)
		if nameNoExt != "" {
			prefixes = append(prefixes, "documents/"+nameNoExt)
		}
	}

	// Try each prefix and pick the best candidate
	for _, prefix := range prefixes {
		objects, err := uc.storage.List(ctx, prefix)
		if err != nil || len(objects) == 0 {
			continue
		}

		// 1) Exact basename match wins
		for _, obj := range objects {
			if strings.EqualFold(getFilename(obj.Path), base) {
				return obj.Path, true
			}
		}

		// 2) Any object containing the basename
		for _, obj := range objects {
			if strings.Contains(strings.ToLower(obj.Path), strings.ToLower(base)) {
				return obj.Path, true
			}
		}

		// 3) Single object under the prefix is a reasonable guess
		if len(objects) == 1 {
			return objects[0].Path, true
		}
	}

	return "", false
}
