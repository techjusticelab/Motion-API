package search

import (
	"motion-index-fiber/pkg/models"
)

func ParseAggregationResponse(rawResponse map[string]interface{}) (*models.AggregationResponse, error) {
	response := &models.AggregationResponse{}

	// Extract aggregations if present
	if aggs, ok := rawResponse["aggregations"].(map[string]interface{}); ok {
		// Parse document types
		if docTypes, ok := aggs["document_types"].(map[string]interface{}); ok {
			if buckets, ok := docTypes["buckets"].([]interface{}); ok {
				for _, bucket := range buckets {
					if b, ok := bucket.(map[string]interface{}); ok {
						if key, ok := b["key"].(string); ok {
							if docCount, ok := b["doc_count"].(float64); ok {
								response.DocumentTypes = append(response.DocumentTypes, models.AggregationBucket{
									Key:      key,
									DocCount: int(docCount),
								})
							}
						}
					}
				}
			}
		}

		// Parse categories
		if categories, ok := aggs["categories"].(map[string]interface{}); ok {
			if buckets, ok := categories["buckets"].([]interface{}); ok {
				for _, bucket := range buckets {
					if b, ok := bucket.(map[string]interface{}); ok {
						if key, ok := b["key"].(string); ok {
							if docCount, ok := b["doc_count"].(float64); ok {
								response.Categories = append(response.Categories, models.AggregationBucket{
									Key:      key,
									DocCount: int(docCount),
								})
							}
						}
					}
				}
			}
		}

		// Parse date ranges
		if dateRanges, ok := aggs["date_ranges"].(map[string]interface{}); ok {
			if buckets, ok := dateRanges["buckets"].([]interface{}); ok {
				for _, bucket := range buckets {
					if b, ok := bucket.(map[string]interface{}); ok {
						if key, ok := b["key"].(string); ok {
							if docCount, ok := b["doc_count"].(float64); ok {
								response.DateRanges = append(response.DateRanges, models.AggregationBucket{
									Key:      key,
									DocCount: int(docCount),
								})
							}
						}
					}
				}
			}
		}
	}

	return response, nil
}

func GetAvailableAggregations() []string {
	return []string{
		"document_types",
		"categories",
		"date_ranges",
		"courts",
		"judges",
	}
}

func ValidateAggregations(aggregations []string) ([]string, error) {
	if aggregations == nil {
		return []string{}, nil
	}

	available := GetAvailableAggregations()
	availableMap := make(map[string]bool)
	for _, agg := range available {
		availableMap[agg] = true
	}

	var valid []string
	for _, agg := range aggregations {
		if availableMap[agg] {
			valid = append(valid, agg)
		}
	}

	return valid, nil
}
