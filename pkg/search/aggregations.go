package search

import (
	"context"
	"fmt"
	"time"

	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"

	"motion-index-fiber/pkg/models"
)

// GetLegalTags returns all legal tags with their document counts
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

// GetDocumentTypes returns all document types with their counts
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

// GetMetadataFieldValues returns unique values for a metadata field
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

// GetMetadataFieldValuesWithFilters returns unique values for a metadata field with custom filters
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

// GetDocumentStats returns overall document statistics
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

// GetAllFieldOptions returns all available filter options for the UI
func (s *service) GetAllFieldOptions(ctx context.Context) (*models.FieldOptions, error) {
	query := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"courts": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "metadata.court",
					"size":  100,
				},
			},
			"judges": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "metadata.judge",
					"size":  100,
				},
			},
			"doc_types": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "doc_type",
					"size":  50,
				},
			},
			"legal_tags": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "metadata.legal_tags",
					"size":  200,
				},
			},
			"statuses": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "metadata.status",
					"size":  20,
				},
			},
			"authors": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "metadata.author",
					"size":  100,
				},
			},
		},
	}

	res, err := s.executeAggregationQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	var response struct {
		Aggregations map[string]interface{} `json:"aggregations"`
	}

	if err := parseResponse(res, &response); err != nil {
		return nil, fmt.Errorf("failed to parse field options response: %w", err)
	}

	options := &models.FieldOptions{}

	// Extract all field options
	if courts, err := s.extractBucketsFromAgg(response.Aggregations, "courts"); err == nil {
		options.Courts = make([]*models.FieldValue, len(courts))
		for i, bucket := range courts {
			options.Courts[i] = &models.FieldValue{Value: bucket.Key, Count: bucket.DocCount}
		}
	}

	if judges, err := s.extractBucketsFromAgg(response.Aggregations, "judges"); err == nil {
		options.Judges = make([]*models.FieldValue, len(judges))
		for i, bucket := range judges {
			options.Judges[i] = &models.FieldValue{Value: bucket.Key, Count: bucket.DocCount}
		}
	}

	if docTypes, err := s.extractBucketsFromAgg(response.Aggregations, "doc_types"); err == nil {
		options.DocTypes = make([]*models.FieldValue, len(docTypes))
		for i, bucket := range docTypes {
			options.DocTypes[i] = &models.FieldValue{Value: bucket.Key, Count: bucket.DocCount}
		}
	}

	if legalTags, err := s.extractBucketsFromAgg(response.Aggregations, "legal_tags"); err == nil {
		options.LegalTags = make([]*models.FieldValue, len(legalTags))
		for i, bucket := range legalTags {
			options.LegalTags[i] = &models.FieldValue{Value: bucket.Key, Count: bucket.DocCount}
		}
	}

	if statuses, err := s.extractBucketsFromAgg(response.Aggregations, "statuses"); err == nil {
		options.Statuses = make([]*models.FieldValue, len(statuses))
		for i, bucket := range statuses {
			options.Statuses[i] = &models.FieldValue{Value: bucket.Key, Count: bucket.DocCount}
		}
	}

	if authors, err := s.extractBucketsFromAgg(response.Aggregations, "authors"); err == nil {
		options.Authors = make([]*models.FieldValue, len(authors))
		for i, bucket := range authors {
			options.Authors[i] = &models.FieldValue{Value: bucket.Key, Count: bucket.DocCount}
		}
	}

	return options, nil
}

// Helper functions for aggregations

type aggregationBucket struct {
	Key      string `json:"key"`
	DocCount int64  `json:"doc_count"`
}

func (s *service) executeAggregationQuery(ctx context.Context, query map[string]interface{}) (*opensearchapi.Response, error) {
	searchReq := opensearchapi.SearchRequest{
		Index: []string{s.client.GetIndex()},
		Body:  buildRequestBody(query),
	}

	res, err := searchReq.Do(ctx, s.client.GetClient())
	if err != nil {
		return nil, fmt.Errorf("aggregation request failed: %w", err)
	}

	if res.IsError() {
		res.Body.Close()
		return nil, fmt.Errorf("aggregation failed with status: %s", res.Status())
	}

	return res, nil
}

func (s *service) extractBuckets(res *opensearchapi.Response, aggName string) ([]aggregationBucket, error) {
	defer res.Body.Close()

	var response struct {
		Aggregations map[string]interface{} `json:"aggregations"`
	}

	if err := parseResponse(res, &response); err != nil {
		return nil, fmt.Errorf("failed to parse aggregation response: %w", err)
	}

	return s.extractBucketsFromAgg(response.Aggregations, aggName)
}

func (s *service) extractBucketsFromAgg(aggregations map[string]interface{}, aggName string) ([]aggregationBucket, error) {
	agg, ok := aggregations[aggName].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("aggregation %s not found", aggName)
	}

	bucketsInterface, ok := agg["buckets"]
	if !ok {
		return nil, fmt.Errorf("buckets not found in aggregation %s", aggName)
	}

	bucketsSlice, ok := bucketsInterface.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid buckets format in aggregation %s", aggName)
	}

	buckets := make([]aggregationBucket, len(bucketsSlice))
	for i, bucketInterface := range bucketsSlice {
		bucketMap, ok := bucketInterface.(map[string]interface{})
		if !ok {
			continue
		}

		bucket := aggregationBucket{}
		if key, ok := bucketMap["key"].(string); ok {
			bucket.Key = key
		}
		if docCount, ok := bucketMap["doc_count"].(float64); ok {
			bucket.DocCount = int64(docCount)
		}

		buckets[i] = bucket
	}

	return buckets, nil
}

// BuildDocumentTypeAggregation builds an aggregation query for document types
func BuildDocumentTypeAggregation() map[string]interface{} {
	return map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"document_types": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "document_type.keyword",
					"size":  20,
				},
			},
		},
	}
}

// BuildCategoryAggregation builds an aggregation query for document categories
func BuildCategoryAggregation() map[string]interface{} {
	return map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"categories": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "category.keyword",
					"size":  15,
				},
			},
		},
	}
}

// BuildDateRangeAggregation builds an aggregation query for date ranges
func BuildDateRangeAggregation() map[string]interface{} {
	return map[string]interface{}{
		"size": 0,
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
	}
}

// BuildCourtAggregation builds an aggregation query for courts
func BuildCourtAggregation() map[string]interface{} {
	return map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"courts": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "court.keyword",
					"size":  25,
				},
			},
		},
	}
}

// BuildJudgeAggregation builds an aggregation query for judges
func BuildJudgeAggregation() map[string]interface{} {
	return map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"judges": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "judge.keyword",
					"size":  30,
				},
			},
		},
	}
}

// BuildCombinedAggregation builds a combined aggregation query
func BuildCombinedAggregation(aggregations []string) map[string]interface{} {
	result := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{},
	}
	
	aggs := result["aggs"].(map[string]interface{})
	
	for _, aggType := range aggregations {
		switch aggType {
		case "document_types":
			aggs["document_types"] = map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "document_type.keyword",
					"size":  20,
				},
			}
		case "categories":
			aggs["categories"] = map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "category.keyword",
					"size":  15,
				},
			}
		case "date_ranges":
			aggs["date_ranges"] = map[string]interface{}{
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
			}
		case "courts":
			aggs["courts"] = map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "court.keyword",
					"size":  25,
				},
			}
		case "judges":
			aggs["judges"] = map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "judge.keyword",
					"size":  30,
				},
			}
		}
	}
	
	return result
}

// ParseAggregationResponse parses raw aggregation response into structured format
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

// GetAvailableAggregations returns list of available aggregation types
func GetAvailableAggregations() []string {
	return []string{
		"document_types",
		"categories",
		"date_ranges",
		"courts",
		"judges",
	}
}

// ValidateAggregations validates and filters aggregation types
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
