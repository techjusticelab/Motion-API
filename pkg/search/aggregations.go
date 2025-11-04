package search

import (
	"context"
	"fmt"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"motion-index-fiber/pkg/models"
	"time"
)

func (s *service) GetLegalTags(ctx context.Context) ([]*models.TagCount, error) {
	query := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"legal_tags": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "metadata.legal_tags",
					"size":  100,
				},
			},
		},
	}

	res, err := s.executeAggregationQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	buckets, err := s.extractBuckets(res, "legal_tags")
	if err != nil {
		return nil, err
	}

	tags := make([]*models.TagCount, len(buckets))
	for i, bucket := range buckets {
		tags[i] = &models.TagCount{
			Tag:   bucket.Key,
			Count: bucket.DocCount,
		}
	}

	return tags, nil
}

func (s *service) GetDocumentTypes(ctx context.Context) ([]*models.TypeCount, error) {
	query := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"doc_types": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "doc_type",
					"size":  50,
				},
			},
		},
	}

	res, err := s.executeAggregationQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	buckets, err := s.extractBuckets(res, "doc_types")
	if err != nil {
		return nil, err
	}

	types := make([]*models.TypeCount, len(buckets))
	for i, bucket := range buckets {
		types[i] = &models.TypeCount{
			Type:  bucket.Key,
			Count: bucket.DocCount,
		}
	}

	return types, nil
}

func (s *service) GetMetadataFieldValues(ctx context.Context, field string, prefix string, size int) ([]*models.FieldValue, error) {
	if size <= 0 {
		size = 50
	}
	if size > 1000 {
		size = 1000
	}

	aggName := "field_values"
	agg := map[string]interface{}{
		"terms": map[string]interface{}{
			"field": field,
			"size":  size,
		},
	}

	// Add prefix filter if provided
	if prefix != "" {
		agg["terms"].(map[string]interface{})["include"] = fmt.Sprintf("%s.*", prefix)
	}

	query := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			aggName: agg,
		},
	}

	res, err := s.executeAggregationQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	buckets, err := s.extractBuckets(res, aggName)
	if err != nil {
		return nil, err
	}

	values := make([]*models.FieldValue, len(buckets))
	for i, bucket := range buckets {
		values[i] = &models.FieldValue{
			Value: bucket.Key,
			Count: bucket.DocCount,
		}
	}

	return values, nil
}

func (s *service) GetMetadataFieldValuesWithFilters(ctx context.Context, req *models.MetadataFieldValuesRequest) ([]*models.FieldValue, error) {
	// Validate and set defaults
	if req.Field == "" {
		return nil, fmt.Errorf("field is required")
	}

	size := req.Size
	if size <= 0 {
		size = 50
	}
	if size > 1000 {
		size = 1000
	}

	// Build aggregation
	aggName := "field_values"
	agg := map[string]interface{}{
		"terms": map[string]interface{}{
			"field": req.Field,
			"size":  size,
		},
	}

	// Add prefix filter if provided
	if req.Prefix != "" {
		agg["terms"].(map[string]interface{})["include"] = fmt.Sprintf("%s.*", req.Prefix)
	}

	// Add exclude filter if provided
	if len(req.ExcludeValues) > 0 {
		agg["terms"].(map[string]interface{})["exclude"] = req.ExcludeValues
	}

	// Build base query with custom filters
	query := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			aggName: agg,
		},
	}

	// Add custom filters if provided
	if len(req.Filters) > 0 {
		filterClauses := make([]map[string]interface{}, 0)

		for field, value := range req.Filters {
			switch v := value.(type) {
			case string:
				// Simple term filter
				filterClauses = append(filterClauses, map[string]interface{}{
					"term": map[string]interface{}{
						field: v,
					},
				})
			case []interface{}:
				// Terms filter for arrays
				if len(v) > 0 {
					filterClauses = append(filterClauses, map[string]interface{}{
						"terms": map[string]interface{}{
							field: v,
						},
					})
				}
			case []string:
				// Terms filter for string arrays
				if len(v) > 0 {
					values := make([]interface{}, len(v))
					for i, str := range v {
						values[i] = str
					}
					filterClauses = append(filterClauses, map[string]interface{}{
						"terms": map[string]interface{}{
							field: values,
						},
					})
				}
			case map[string]interface{}:
				// Handle complex filters like date_range
				if field == "date_range" {
					if from, ok := v["from"]; ok {
						if to, ok := v["to"]; ok {
							filterClauses = append(filterClauses, map[string]interface{}{
								"range": map[string]interface{}{
									"created_at": map[string]interface{}{
										"gte": from,
										"lte": to,
									},
								},
							})
						}
					}
				}
			}
		}

		// Add filters to query if any were created
		if len(filterClauses) > 0 {
			query["query"] = map[string]interface{}{
				"bool": map[string]interface{}{
					"filter": filterClauses,
				},
			}
		}
	}

	res, err := s.executeAggregationQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	buckets, err := s.extractBuckets(res, aggName)
	if err != nil {
		return nil, err
	}

	values := make([]*models.FieldValue, len(buckets))
	for i, bucket := range buckets {
		values[i] = &models.FieldValue{
			Value: bucket.Key,
			Count: bucket.DocCount,
		}
	}

	return values, nil
}

func (s *service) GetDocumentStats(ctx context.Context) (*models.DocumentStats, error) {
	query := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"doc_types": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "doc_type",
					"size":  50,
				},
			},
			"legal_tags": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "metadata.legal_tags",
					"size":  100,
				},
			},
			"unique_courts": map[string]interface{}{
				"cardinality": map[string]interface{}{
					"field": "metadata.court",
				},
			},
			"unique_judges": map[string]interface{}{
				"cardinality": map[string]interface{}{
					"field": "metadata.judge",
				},
			},
		},
	}

	searchReq := opensearchapi.SearchRequest{
		Index: []string{s.client.GetIndex()},
		Body:  buildRequestBody(query),
	}

	res, err := searchReq.Do(ctx, s.client.GetClient())
	if err != nil {
		return nil, fmt.Errorf("stats aggregation request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("stats aggregation failed with status: %s", res.Status())
	}

	var response struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
		} `json:"hits"`
		Aggregations map[string]interface{} `json:"aggregations"`
	}

	if err := parseResponse(res, &response); err != nil {
		return nil, fmt.Errorf("failed to parse stats response: %w", err)
	}

	// Extract document type counts
	docTypeBuckets, _ := s.extractBucketsFromAgg(response.Aggregations, "doc_types")
	typeCounts := make([]*models.TypeCount, len(docTypeBuckets))
	for i, bucket := range docTypeBuckets {
		typeCounts[i] = &models.TypeCount{
			Type:  bucket.Key,
			Count: bucket.DocCount,
		}
	}

	// Extract legal tag counts
	legalTagBuckets, _ := s.extractBucketsFromAgg(response.Aggregations, "legal_tags")
	tagCounts := make([]*models.TagCount, len(legalTagBuckets))
	for i, bucket := range legalTagBuckets {
		tagCounts[i] = &models.TagCount{
			Tag:   bucket.Key,
			Count: bucket.DocCount,
		}
	}

	// Extract cardinality values
	fieldStats := make(map[string]models.FieldStat)
	if courtCard, ok := response.Aggregations["unique_courts"].(map[string]interface{}); ok {
		if value, ok := courtCard["value"].(float64); ok {
			fieldStats["court"] = models.FieldStat{
				UniqueValues: int64(value),
				TotalValues:  response.Hits.Total.Value,
			}
		}
	}

	if judgeCard, ok := response.Aggregations["unique_judges"].(map[string]interface{}); ok {
		if value, ok := judgeCard["value"].(float64); ok {
			fieldStats["judge"] = models.FieldStat{
				UniqueValues: int64(value),
				TotalValues:  response.Hits.Total.Value,
			}
		}
	}

	return &models.DocumentStats{
		TotalDocuments: response.Hits.Total.Value,
		IndexSize:      "Unknown", // Would need index stats API for this
		TypeCounts:     typeCounts,
		TagCounts:      tagCounts,
		LastUpdated:    time.Now(),
		FieldStats:     fieldStats,
	}, nil
}
