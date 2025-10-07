package search

import (
	"context"
	"time"

	"motion-index-fiber/pkg/models"
)

// SearchService defines the interface for document search operations
type SearchService interface {
	// SearchDocuments performs a search query and returns results
	SearchDocuments(ctx context.Context, req *models.SearchRequest) (*models.SearchResult, error)

	// IndexDocument indexes a single document
	IndexDocument(ctx context.Context, doc *models.Document) (string, error)

	// BulkIndexDocuments indexes multiple documents in a single operation
	BulkIndexDocuments(ctx context.Context, docs []*models.Document) (*models.BulkResult, error)

	// UpdateDocumentMetadata updates metadata for an existing document
	UpdateDocumentMetadata(ctx context.Context, docID string, metadata map[string]interface{}) error

	// DeleteDocument removes a document from the index
	DeleteDocument(ctx context.Context, docID string) error

	// GetDocument retrieves a document by ID
	GetDocument(ctx context.Context, docID string) (*models.Document, error)

	// DocumentExists checks if a document exists in the index
	DocumentExists(ctx context.Context, docID string) (bool, error)
}

// AggregationService defines the interface for metadata aggregations
type AggregationService interface {
	// GetLegalTags returns all legal tags with their document counts
	GetLegalTags(ctx context.Context) ([]*models.TagCount, error)

	// GetDocumentTypes returns all document types with their counts
	GetDocumentTypes(ctx context.Context) ([]*models.TypeCount, error)

	// GetMetadataFieldValues returns unique values for a metadata field
	GetMetadataFieldValues(ctx context.Context, field string, prefix string, size int) ([]*models.FieldValue, error)

	// GetMetadataFieldValuesWithFilters returns unique values for a metadata field with custom filters
	GetMetadataFieldValuesWithFilters(ctx context.Context, req *models.MetadataFieldValuesRequest) ([]*models.FieldValue, error)

	// GetDocumentStats returns overall document statistics
	GetDocumentStats(ctx context.Context) (*models.DocumentStats, error)

	// GetAllFieldOptions returns all available filter options for the UI
	GetAllFieldOptions(ctx context.Context) (*models.FieldOptions, error)
}

// QueryBuilder defines the interface for search query construction
type QueryBuilder interface {
	// BuildQuery constructs an OpenSearch query from a search request
	BuildQuery(req *models.SearchRequest) (map[string]interface{}, error)

	// AddTextQuery adds a text search query
	AddTextQuery(query string, fuzzy bool) QueryBuilder

	// AddMetadataFilters adds metadata filtering
	AddMetadataFilters(filters map[string]interface{}, matchAll bool) QueryBuilder

	// AddDateRange adds date range filtering
	AddDateRange(field string, from, to *time.Time) QueryBuilder

	// AddSorting adds sorting to the query
	AddSorting(field string, order models.SortOrder) QueryBuilder

	// AddPagination adds pagination parameters
	AddPagination(from, size int) QueryBuilder

	// AddHighlighting adds highlighting for search terms
	AddHighlighting(fields []string) QueryBuilder

	// Reset clears the current query builder state
	Reset() QueryBuilder

	// Build returns the final query as a map
	Build() map[string]interface{}
}

// HealthChecker defines the interface for health checking
type HealthChecker interface {
	// IsHealthy returns true if the search service is healthy
	IsHealthy() bool

	// Health returns detailed health information
	Health(ctx context.Context) (*HealthStatus, error)
}

// HealthStatus represents the health status of the search service
type HealthStatus struct {
	Status        string `json:"status"`
	ClusterName   string `json:"cluster_name"`
	NumberOfNodes int    `json:"number_of_nodes"`
	ActiveShards  int    `json:"active_shards"`
	IndexExists   bool   `json:"index_exists"`
	IndexHealth   string `json:"index_health"`
}

// Service combines all search-related interfaces
type Service interface {
	SearchService
	AggregationService
	HealthChecker
}
