package search

import (
	"context"
	"fmt"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"motion-index-fiber/pkg/models"
)

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
