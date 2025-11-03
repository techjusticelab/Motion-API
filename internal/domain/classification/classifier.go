package classification

import (
	"context"
)

// Service defines the behaviour required from a document classification provider.
type Service interface {
	// Classify analyses the document content and returns a Result with domain classifications.
	Classify(ctx context.Context, input Input) (*Result, error)

	// Health verifies the readiness of the underlying classifier.
	Health(ctx context.Context) error
}

// Input packages the data required by the classifier to operate.
type Input struct {
	DocumentID string
	Text       string
	Hash       string
	Metadata   map[string]string
}

// HasContent reports whether the classifier input includes substantive text.
func (i Input) HasContent() bool {
	return len(i.Text) > 0
}
