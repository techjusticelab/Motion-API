package errors

import (
	"errors"
	"fmt"
	"strings"

	domainerrors "motion-index-fiber/internal/domain/errors"
)

// TranslateStorageError converts storage-layer errors to domain errors.
func TranslateStorageError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Check for not found errors
	if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "NoSuchKey") {
		return fmt.Errorf("%w: %v", domainerrors.ErrDocumentNotFound, err)
	}

	// Check for access denied / permission errors
	if strings.Contains(errMsg, "access denied") || strings.Contains(errMsg, "forbidden") {
		return fmt.Errorf("storage access denied: %w", err)
	}

	// Check for network/timeout errors
	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded") {
		return fmt.Errorf("storage timeout: %w", err)
	}

	// Default: wrap as generic storage error
	return fmt.Errorf("storage error: %w", err)
}

// TranslateSearchError converts search-layer errors to domain errors.
func TranslateSearchError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Check for not found errors
	if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "404") {
		return fmt.Errorf("%w: %v", domainerrors.ErrDocumentNotFound, err)
	}

	// Check for timeout errors
	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded") {
		return fmt.Errorf("search timeout: %w", err)
	}

	// Check for connection errors
	if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "no such host") {
		return fmt.Errorf("search service unavailable: %w", err)
	}

	// Default: wrap as generic search error
	return fmt.Errorf("search error: %w", err)
}

// TranslateAIError converts AI provider errors to domain errors.
func TranslateAIError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Check for authentication errors
	if strings.Contains(errMsg, "unauthorized") || strings.Contains(errMsg, "invalid api key") ||
		strings.Contains(errMsg, "authentication") {
		return fmt.Errorf("AI provider authentication failed: %w", err)
	}

	// Check for rate limit errors
	if strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "429") ||
		strings.Contains(errMsg, "too many requests") {
		return fmt.Errorf("AI provider rate limit exceeded: %w", err)
	}

	// Check for quota/billing errors
	if strings.Contains(errMsg, "quota") || strings.Contains(errMsg, "insufficient funds") ||
		strings.Contains(errMsg, "billing") {
		return fmt.Errorf("AI provider quota exceeded: %w", err)
	}

	// Check for timeout errors
	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded") {
		return fmt.Errorf("AI provider timeout: %w", err)
	}

	// Check for invalid request errors
	if strings.Contains(errMsg, "invalid request") || strings.Contains(errMsg, "400") {
		return fmt.Errorf("invalid AI provider request: %w", err)
	}

	// Default: wrap as generic AI error
	return fmt.Errorf("AI provider error: %w", err)
}

// IsNotFound checks if an error is a "not found" error.
func IsNotFound(err error) bool {
	return errors.Is(err, domainerrors.ErrDocumentNotFound)
}

// IsTimeout checks if an error is a timeout error.
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded")
}

// IsRateLimit checks if an error is a rate limit error.
func IsRateLimit(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "429")
}
