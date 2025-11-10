package search

import "time"

// TagCount represents a legal tag with its document count
type TagCount struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

// TypeCount represents a document type with its count
type TypeCount struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

// FieldValue represents a metadata field value
type FieldValue struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

// DocumentStats represents overall document statistics
type DocumentStats struct {
	TotalDocuments int64                `json:"total_documents"`
	IndexSize      string               `json:"index_size"`
	TypeCounts     []*TypeCount         `json:"type_counts"`
	TagCounts      []*TagCount          `json:"tag_counts"`
	LastUpdated    time.Time            `json:"last_updated"`
	FieldStats     map[string]FieldStat `json:"field_stats"`
}

// FieldStat represents statistics for a specific field
type FieldStat struct {
	UniqueValues int64 `json:"unique_values"`
	TotalValues  int64 `json:"total_values"`
}

// FieldOptions represents all available filter options
type FieldOptions struct {
	Courts    []*FieldValue `json:"courts"`
	Judges    []*FieldValue `json:"judges"`
	DocTypes  []*FieldValue `json:"doc_types"`
	LegalTags []*FieldValue `json:"legal_tags"`
	Statuses  []*FieldValue `json:"statuses"`
	Authors   []*FieldValue `json:"authors"`
}

// BulkResult represents the result of a bulk operation
type BulkResult struct {
	Took       int64            `json:"took"`
	Errors     bool             `json:"errors"`
	Items      []BulkResultItem `json:"items"`
	Indexed    int              `json:"indexed"`
	Failed     int              `json:"failed"`
	FailedDocs []*BulkFailedDoc `json:"failed_docs,omitempty"`
}

// BulkResultItem represents a single item in bulk operation result
type BulkResultItem struct {
	Index  *BulkItemResult `json:"index,omitempty"`
	Create *BulkItemResult `json:"create,omitempty"`
	Update *BulkItemResult `json:"update,omitempty"`
	Delete *BulkItemResult `json:"delete,omitempty"`
}

// BulkItemResult represents the result of a single bulk item
type BulkItemResult struct {
	ID     string     `json:"_id"`
	Index  string     `json:"_index"`
	Type   string     `json:"_type"`
	Status int        `json:"status"`
	Error  *BulkError `json:"error,omitempty"`
}

// BulkError represents an error in bulk operation
type BulkError struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

// BulkFailedDoc represents a document that failed to be indexed
type BulkFailedDoc struct {
	ID     string `json:"id"`
	Error  string `json:"error"`
	Status int    `json:"status"`
}

// IndexStats represents statistics about the search index
type IndexStats struct {
	TotalDocuments int64             `json:"total_documents"`
	IndexSize      string            `json:"index_size"`
	IndexHealth    string            `json:"index_health"`
	ShardInfo      *ShardInfo        `json:"shard_info,omitempty"`
	FieldCounts    map[string]int64  `json:"field_counts"`
	LastUpdated    string            `json:"last_updated"`
	Performance    *PerformanceStats `json:"performance,omitempty"`
}

// ShardInfo represents shard information for the index
type ShardInfo struct {
	Primary  int `json:"primary"`
	Replicas int `json:"replicas"`
	Total    int `json:"total"`
	Active   int `json:"active"`
	Failed   int `json:"failed"`
}

// PerformanceStats represents performance metrics
type PerformanceStats struct {
	AvgQueryTime  string `json:"avg_query_time"`
	TotalQueries  int64  `json:"total_queries"`
	IndexingRate  string `json:"indexing_rate"`
	CacheHitRatio string `json:"cache_hit_ratio"`
}

// AggregationResponse represents the response from aggregation queries
type AggregationResponse struct {
	Success       bool                   `json:"success"`
	Message       string                 `json:"message,omitempty"`
	Aggregations  map[string]interface{} `json:"aggregations"`
	TotalHits     int64                  `json:"total_hits"`
	RequestID     string                 `json:"request_id,omitempty"`
	Timestamp     string                 `json:"timestamp"`
	Error         *SearchError           `json:"error,omitempty"`
	DocumentTypes []AggregationBucket    `json:"document_types,omitempty"`
	Categories    []AggregationBucket    `json:"categories,omitempty"`
	DateRanges    []AggregationBucket    `json:"date_ranges,omitempty"`
	Courts        []AggregationBucket    `json:"courts,omitempty"`
	Judges        []AggregationBucket    `json:"judges,omitempty"`
}

// AggregationBucket represents a single aggregation bucket
type AggregationBucket struct {
	Key      string `json:"key"`
	DocCount int    `json:"doc_count"`
}
