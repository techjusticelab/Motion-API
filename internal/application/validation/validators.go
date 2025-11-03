package validation

import (
	"errors"
	"strings"
	"time"

	"motion-index-fiber/internal/domain/document"
)

var (
	errInvalidExtension = errors.New("invalid file extension")
	errEmptyFileName    = errors.New("file name cannot be empty")
)

// ValidateFileName uses the document value object to enforce rules.
func ValidateFileName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errEmptyFileName
	}
	if _, err := document.NewFileName(name); err != nil {
		return err
	}
	return nil
}

// ValidateStoragePath ensures a storage path is non-empty and valid.
func ValidateStoragePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("storage path cannot be empty")
	}
	if _, err := document.NewFilePath(path); err != nil {
		return err
	}
	return nil
}

// ValidateConfidenceRange ensures the provided confidence lies within [0,1] when non-zero.
func ValidateConfidenceRange(conf float64) error {
	if conf == 0 {
		return nil
	}
	if _, err := document.NewConfidence(conf); err != nil {
		return err
	}
	return nil
}

// ValidatePagination checks that page and size are positive.
func ValidatePagination(page, size int) error {
	if page == 0 && size == 0 {
		return nil
	}
	if page <= 0 {
		return errors.New("page must be greater than zero")
	}
	if size <= 0 {
		return errors.New("page size must be greater than zero")
	}
	return nil
}

// ValidateDateRange ensures the from date is not after the to date.
func ValidateDateRange(from, to *time.Time) error {
	if from == nil || to == nil {
		return nil
	}
	if from.After(*to) {
		return errors.New("from date cannot be after to date")
	}
	return nil
}

// ValidateSort checks sort parameters.
func ValidateSort(field, order string) error {
	if field == "" && order == "" {
		return nil
	}
	if field == "" {
		return errors.New("sort field is required when sort order is specified")
	}
	switch strings.ToLower(order) {
	case "", "asc", "desc":
		return nil
	default:
		return errors.New("sort order must be 'asc' or 'desc'")
	}
}
