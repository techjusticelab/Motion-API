package client

import (
	"context"

	"github.com/opensearch-project/opensearch-go/v2"
)

// SearchClient defines the interface for OpenSearch client operations
type SearchClient interface {
	// GetClient returns the underlying OpenSearch client
	GetClient() *opensearch.Client
	
	// GetIndex returns the current index name
	GetIndex() string
	
	// IsHealthy returns true if the client is healthy
	IsHealthy() bool
	
	// Health returns detailed health information
	Health(ctx context.Context) (*HealthStatus, error)
	
	// IndexExists checks if the index exists
	IndexExists(ctx context.Context) (bool, error)
	
	// CreateIndex creates a new index with mapping
	CreateIndex(ctx context.Context, mapping map[string]interface{}) error
	
	// DeleteIndex deletes the index
	DeleteIndex(ctx context.Context) error
	
	// RefreshIndex refreshes the index
	RefreshIndex(ctx context.Context) error
	
	// Close closes the client connection
	Close() error
}