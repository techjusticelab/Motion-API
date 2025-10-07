package search

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"motion-index-fiber/pkg/models"
)

func TestBuildDocumentTypeAggregation(t *testing.T) {
	tests := []struct {
		name     string
		expected map[string]interface{}
	}{
		{
			name: "document type aggregation",
			expected: map[string]interface{}{
				"aggs": map[string]interface{}{
					"document_types": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": "document_type.keyword",
							"size":  20,
						},
					},
				},
				"size": 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildDocumentTypeAggregation()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildCategoryAggregation(t *testing.T) {
	tests := []struct {
		name     string
		expected map[string]interface{}
	}{
		{
			name: "category aggregation",
			expected: map[string]interface{}{
				"aggs": map[string]interface{}{
					"categories": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": "category.keyword",
							"size":  15,
						},
					},
				},
				"size": 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildCategoryAggregation()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildDateRangeAggregation(t *testing.T) {
	tests := []struct {
		name     string
		expected map[string]interface{}
	}{
		{
			name: "date range aggregation",
			expected: map[string]interface{}{
				"aggs": map[string]interface{}{
					"date_ranges": map[string]interface{}{
						"date_range": map[string]interface{}{
							"field": "created_at",
							"ranges": []map[string]interface{}{
								{
									"key":  "last_7_days",
									"from": "now-7d/d",
									"to":   "now/d",
								},
								{
									"key":  "last_30_days",
									"from": "now-30d/d",
									"to":   "now/d",
								},
								{
									"key":  "last_90_days",
									"from": "now-90d/d",
									"to":   "now/d",
								},
								{
									"key":  "last_year",
									"from": "now-1y/d",
									"to":   "now/d",
								},
							},
						},
					},
				},
				"size": 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildDateRangeAggregation()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildCourtAggregation(t *testing.T) {
	tests := []struct {
		name     string
		expected map[string]interface{}
	}{
		{
			name: "court aggregation",
			expected: map[string]interface{}{
				"aggs": map[string]interface{}{
					"courts": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": "court.keyword",
							"size":  25,
						},
					},
				},
				"size": 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildCourtAggregation()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildJudgeAggregation(t *testing.T) {
	tests := []struct {
		name     string
		expected map[string]interface{}
	}{
		{
			name: "judge aggregation",
			expected: map[string]interface{}{
				"aggs": map[string]interface{}{
					"judges": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": "judge.keyword",
							"size":  30,
						},
					},
				},
				"size": 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildJudgeAggregation()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildCombinedAggregation(t *testing.T) {
	tests := []struct {
		name         string
		aggregations []string
		expected     map[string]interface{}
	}{
		{
			name:         "single aggregation",
			aggregations: []string{"document_types"},
			expected: map[string]interface{}{
				"aggs": map[string]interface{}{
					"document_types": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": "document_type.keyword",
							"size":  20,
						},
					},
				},
				"size": 0,
			},
		},
		{
			name:         "multiple aggregations",
			aggregations: []string{"document_types", "categories"},
			expected: map[string]interface{}{
				"aggs": map[string]interface{}{
					"document_types": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": "document_type.keyword",
							"size":  20,
						},
					},
					"categories": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": "category.keyword",
							"size":  15,
						},
					},
				},
				"size": 0,
			},
		},
		{
			name:         "all aggregations",
			aggregations: []string{"document_types", "categories", "date_ranges", "courts", "judges"},
			expected: map[string]interface{}{
				"aggs": map[string]interface{}{
					"document_types": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": "document_type.keyword",
							"size":  20,
						},
					},
					"categories": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": "category.keyword",
							"size":  15,
						},
					},
					"date_ranges": map[string]interface{}{
						"date_range": map[string]interface{}{
							"field": "created_at",
							"ranges": []map[string]interface{}{
								{
									"key":  "last_7_days",
									"from": "now-7d/d",
									"to":   "now/d",
								},
								{
									"key":  "last_30_days",
									"from": "now-30d/d",
									"to":   "now/d",
								},
								{
									"key":  "last_90_days",
									"from": "now-90d/d",
									"to":   "now/d",
								},
								{
									"key":  "last_year",
									"from": "now-1y/d",
									"to":   "now/d",
								},
							},
						},
					},
					"courts": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": "court.keyword",
							"size":  25,
						},
					},
					"judges": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": "judge.keyword",
							"size":  30,
						},
					},
				},
				"size": 0,
			},
		},
		{
			name:         "empty aggregations",
			aggregations: []string{},
			expected: map[string]interface{}{
				"aggs": map[string]interface{}{},
				"size": 0,
			},
		},
		{
			name:         "unknown aggregation",
			aggregations: []string{"unknown"},
			expected: map[string]interface{}{
				"aggs": map[string]interface{}{},
				"size": 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildCombinedAggregation(tt.aggregations)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseAggregationResponse(t *testing.T) {
	tests := []struct {
		name             string
		rawResponse      map[string]interface{}
		expectedResponse *models.AggregationResponse
		expectedError    bool
	}{
		{
			name: "valid aggregation response",
			rawResponse: map[string]interface{}{
				"aggregations": map[string]interface{}{
					"document_types": map[string]interface{}{
						"buckets": []interface{}{
							map[string]interface{}{
								"key":       "motion",
								"doc_count": 50,
							},
							map[string]interface{}{
								"key":       "brief",
								"doc_count": 30,
							},
						},
					},
					"categories": map[string]interface{}{
						"buckets": []interface{}{
							map[string]interface{}{
								"key":       "civil",
								"doc_count": 40,
							},
							map[string]interface{}{
								"key":       "criminal",
								"doc_count": 40,
							},
						},
					},
				},
			},
			expectedResponse: &models.AggregationResponse{
				DocumentTypes: []models.AggregationBucket{
					{Key: "motion", DocCount: 50},
					{Key: "brief", DocCount: 30},
				},
				Categories: []models.AggregationBucket{
					{Key: "civil", DocCount: 40},
					{Key: "criminal", DocCount: 40},
				},
			},
			expectedError: false,
		},
		{
			name: "response with date ranges",
			rawResponse: map[string]interface{}{
				"aggregations": map[string]interface{}{
					"date_ranges": map[string]interface{}{
						"buckets": []interface{}{
							map[string]interface{}{
								"key":       "last_7_days",
								"doc_count": 15,
							},
							map[string]interface{}{
								"key":       "last_30_days",
								"doc_count": 45,
							},
						},
					},
				},
			},
			expectedResponse: &models.AggregationResponse{
				DateRanges: []models.AggregationBucket{
					{Key: "last_7_days", DocCount: 15},
					{Key: "last_30_days", DocCount: 45},
				},
			},
			expectedError: false,
		},
		{
			name:             "empty response",
			rawResponse:      map[string]interface{}{},
			expectedResponse: &models.AggregationResponse{},
			expectedError:    false,
		},
		{
			name: "invalid bucket format",
			rawResponse: map[string]interface{}{
				"aggregations": map[string]interface{}{
					"document_types": map[string]interface{}{
						"buckets": "invalid",
					},
				},
			},
			expectedResponse: &models.AggregationResponse{},
			expectedError:    false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseAggregationResponse(tt.rawResponse)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResponse, result)
			}
		})
	}
}

func TestGetAvailableAggregations(t *testing.T) {
	expected := []string{
		"document_types",
		"categories",
		"date_ranges",
		"courts",
		"judges",
	}

	result := GetAvailableAggregations()
	assert.Equal(t, expected, result)
}

func TestValidateAggregations(t *testing.T) {
	tests := []struct {
		name          string
		aggregations  []string
		expectedValid []string
		expectedError bool
	}{
		{
			name:          "all valid aggregations",
			aggregations:  []string{"document_types", "categories", "courts"},
			expectedValid: []string{"document_types", "categories", "courts"},
			expectedError: false,
		},
		{
			name:          "some invalid aggregations",
			aggregations:  []string{"document_types", "invalid", "categories"},
			expectedValid: []string{"document_types", "categories"},
			expectedError: false,
		},
		{
			name:          "all invalid aggregations",
			aggregations:  []string{"invalid1", "invalid2"},
			expectedValid: []string{},
			expectedError: false,
		},
		{
			name:          "empty input",
			aggregations:  []string{},
			expectedValid: []string{},
			expectedError: false,
		},
		{
			name:          "nil input",
			aggregations:  nil,
			expectedValid: []string{},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateAggregations(tt.aggregations)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedValid, result)
			}
		})
	}
}
