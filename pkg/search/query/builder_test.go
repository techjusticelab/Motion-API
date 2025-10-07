package query

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"motion-index-fiber/pkg/models"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder()
	assert.NotNil(t, builder)
	assert.NotNil(t, builder.query)
}

func TestBuilder_WithQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected map[string]interface{}
	}{
		{
			name:  "simple query",
			query: "test query",
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"multi_match": map[string]interface{}{
						"query": "test query",
						"fields": []string{
							"title^3",
							"content^2",
							"extracted_text",
							"summary",
							"case_name^2",
							"parties",
							"attorneys",
						},
						"type":     "best_fields",
						"operator": "and",
					},
				},
			},
		},
		{
			name:  "empty query",
			query: "",
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			result := builder.WithQuery(tt.query).Build()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuilder_WithFilters(t *testing.T) {
	tests := []struct {
		name     string
		filters  map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "single filter",
			filters: map[string]interface{}{
				"document_type": "motion",
			},
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"must": []interface{}{
							map[string]interface{}{
								"match_all": map[string]interface{}{},
							},
						},
						"filter": []interface{}{
							map[string]interface{}{
								"term": map[string]interface{}{
									"document_type.keyword": "motion",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple filters",
			filters: map[string]interface{}{
				"document_type": "motion",
				"category":      "civil",
				"court":         "Superior Court",
			},
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"must": []interface{}{
							map[string]interface{}{
								"match_all": map[string]interface{}{},
							},
						},
						"filter": []interface{}{
							map[string]interface{}{
								"term": map[string]interface{}{
									"document_type.keyword": "motion",
								},
							},
							map[string]interface{}{
								"term": map[string]interface{}{
									"category.keyword": "civil",
								},
							},
							map[string]interface{}{
								"term": map[string]interface{}{
									"court.keyword": "Superior Court",
								},
							},
						},
					},
				},
			},
		},
		{
			name:    "empty filters",
			filters: map[string]interface{}{},
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			result := builder.WithFilters(tt.filters).Build()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuilder_WithDateRange(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		from      string
		to        string
		expected  map[string]interface{}
		wantError bool
	}{
		{
			name:  "valid date range",
			field: "created_at",
			from:  "2024-01-01",
			to:    "2024-12-31",
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"must": []interface{}{
							map[string]interface{}{
								"match_all": map[string]interface{}{},
							},
						},
						"filter": []interface{}{
							map[string]interface{}{
								"range": map[string]interface{}{
									"created_at": map[string]interface{}{
										"gte": "2024-01-01",
										"lte": "2024-12-31",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "only from date",
			field: "created_at",
			from:  "2024-01-01",
			to:    "",
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"must": []interface{}{
							map[string]interface{}{
								"match_all": map[string]interface{}{},
							},
						},
						"filter": []interface{}{
							map[string]interface{}{
								"range": map[string]interface{}{
									"created_at": map[string]interface{}{
										"gte": "2024-01-01",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "only to date",
			field: "created_at",
			from:  "",
			to:    "2024-12-31",
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"must": []interface{}{
							map[string]interface{}{
								"match_all": map[string]interface{}{},
							},
						},
						"filter": []interface{}{
							map[string]interface{}{
								"range": map[string]interface{}{
									"created_at": map[string]interface{}{
										"lte": "2024-12-31",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "empty field",
			field: "",
			from:  "2024-01-01",
			to:    "2024-12-31",
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			result := builder.WithDateRange(tt.field, tt.from, tt.to).Build()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuilder_WithSort(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		ascending bool
		expected  map[string]interface{}
	}{
		{
			name:      "ascending sort",
			field:     "created_at",
			ascending: true,
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
				"sort": []interface{}{
					map[string]interface{}{
						"created_at": map[string]interface{}{
							"order": "asc",
						},
					},
				},
			},
		},
		{
			name:      "descending sort",
			field:     "relevance_score",
			ascending: false,
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
				"sort": []interface{}{
					map[string]interface{}{
						"relevance_score": map[string]interface{}{
							"order": "desc",
						},
					},
				},
			},
		},
		{
			name:  "empty field",
			field: "",
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			result := builder.WithSort(tt.field, tt.ascending).Build()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuilder_WithPagination(t *testing.T) {
	tests := []struct {
		name     string
		from     int
		size     int
		expected map[string]interface{}
	}{
		{
			name: "valid pagination",
			from: 20,
			size: 10,
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
				"from": 20,
				"size": 10,
			},
		},
		{
			name: "zero values",
			from: 0,
			size: 0,
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
				"from": 0,
				"size": 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			result := builder.WithPagination(tt.from, tt.size).Build()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuilder_WithHighlighting(t *testing.T) {
	tests := []struct {
		name     string
		fields   []string
		expected map[string]interface{}
	}{
		{
			name:   "single field",
			fields: []string{"content"},
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
				"highlight": map[string]interface{}{
					"pre_tags":  []string{"<mark>"},
					"post_tags": []string{"</mark>"},
					"fields": map[string]interface{}{
						"content": map[string]interface{}{
							"fragment_size":       150,
							"number_of_fragments": 3,
						},
					},
				},
			},
		},
		{
			name:   "multiple fields",
			fields: []string{"title", "content", "summary"},
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
				"highlight": map[string]interface{}{
					"pre_tags":  []string{"<mark>"},
					"post_tags": []string{"</mark>"},
					"fields": map[string]interface{}{
						"title": map[string]interface{}{
							"fragment_size":       150,
							"number_of_fragments": 3,
						},
						"content": map[string]interface{}{
							"fragment_size":       150,
							"number_of_fragments": 3,
						},
						"summary": map[string]interface{}{
							"fragment_size":       150,
							"number_of_fragments": 3,
						},
					},
				},
			},
		},
		{
			name:   "empty fields",
			fields: []string{},
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			result := builder.WithHighlighting(tt.fields).Build()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuilder_ComplexQuery(t *testing.T) {
	// Test a complex query that combines multiple features
	builder := NewBuilder()
	result := builder.
		WithQuery("motion to dismiss").
		WithFilters(map[string]interface{}{
			"document_type": "motion",
			"court":         "Superior Court",
		}).
		WithDateRange("created_at", "2024-01-01", "2024-12-31").
		WithSort("created_at", false).
		WithPagination(10, 20).
		WithHighlighting([]string{"title", "content"}).
		Build()

	expected := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"multi_match": map[string]interface{}{
							"query": "motion to dismiss",
							"fields": []string{
								"title^3",
								"content^2",
								"extracted_text",
								"summary",
								"case_name^2",
								"parties",
								"attorneys",
							},
							"type":     "best_fields",
							"operator": "and",
						},
					},
				},
				"filter": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"document_type.keyword": "motion",
						},
					},
					map[string]interface{}{
						"term": map[string]interface{}{
							"court.keyword": "Superior Court",
						},
					},
					map[string]interface{}{
						"range": map[string]interface{}{
							"created_at": map[string]interface{}{
								"gte": "2024-01-01",
								"lte": "2024-12-31",
							},
						},
					},
				},
			},
		},
		"sort": []interface{}{
			map[string]interface{}{
				"created_at": map[string]interface{}{
					"order": "desc",
				},
			},
		},
		"from": 10,
		"size": 20,
		"highlight": map[string]interface{}{
			"pre_tags":  []string{"<mark>"},
			"post_tags": []string{"</mark>"},
			"fields": map[string]interface{}{
				"title": map[string]interface{}{
					"fragment_size":       150,
					"number_of_fragments": 3,
				},
				"content": map[string]interface{}{
					"fragment_size":       150,
					"number_of_fragments": 3,
				},
			},
		},
	}

	assert.Equal(t, expected, result)
}

func TestBuildFromSearchRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  *models.SearchRequest
		expected map[string]interface{}
	}{
		{
			name: "complete search request",
			request: &models.SearchRequest{
				Query:   "test query",
				Filters: map[string]interface{}{"document_type": "motion"},
				Sort: &models.SortOptions{
					Field:     "created_at",
					Ascending: false,
				},
				Pagination: &models.PaginationOptions{
					Offset: 10,
					Limit:  20,
				},
				Highlight: &models.HighlightOptions{
					Fields: []string{"title", "content"},
				},
			},
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"must": []interface{}{
							map[string]interface{}{
								"multi_match": map[string]interface{}{
									"query": "test query",
									"fields": []string{
										"title^3",
										"content^2",
										"extracted_text",
										"summary",
										"case_name^2",
										"parties",
										"attorneys",
									},
									"type":     "best_fields",
									"operator": "and",
								},
							},
						},
						"filter": []interface{}{
							map[string]interface{}{
								"term": map[string]interface{}{
									"document_type.keyword": "motion",
								},
							},
						},
					},
				},
				"sort": []interface{}{
					map[string]interface{}{
						"created_at": map[string]interface{}{
							"order": "desc",
						},
					},
				},
				"from": 10,
				"size": 20,
				"highlight": map[string]interface{}{
					"pre_tags":  []string{"<mark>"},
					"post_tags": []string{"</mark>"},
					"fields": map[string]interface{}{
						"title": map[string]interface{}{
							"fragment_size":       150,
							"number_of_fragments": 3,
						},
						"content": map[string]interface{}{
							"fragment_size":       150,
							"number_of_fragments": 3,
						},
					},
				},
			},
		},
		{
			name: "minimal search request",
			request: &models.SearchRequest{
				Query: "test",
			},
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"must": []map[string]interface{}{
							{
								"multi_match": map[string]interface{}{
									"query": "test",
									"fields": []string{
										"title^3",
										"content^2",
										"extracted_text",
										"summary",
										"case_name^2",
										"parties",
										"attorneys",
									},
									"type":     "best_fields",
									"operator": "and",
								},
							},
						},
					},
				},
				"size": 20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildFromSearchRequest(tt.request)
			assert.Equal(t, tt.expected, result)
		})
	}
}
