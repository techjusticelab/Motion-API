package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"io"
	"log"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/search/client"
	"motion-index-fiber/pkg/search/query"
	"strings"
	"time"
)

type service struct {
	client  client.SearchClient
	builder *query.Builder
}

func NewService(searchClient client.SearchClient) Service {
	return &service{
		client:  searchClient,
		builder: query.NewBuilder(),
	}
}

func (s *service) SearchDocuments(ctx context.Context, req *models.SearchRequest) (*models.SearchResult, error) {
	// Validate request
	if req.Size <= 0 {
		req.Size = models.DefaultSearchSize
	}
	if req.Size > models.MaxSearchSize {
		req.Size = models.MaxSearchSize
	}

	// Build OpenSearch query
	searchQuery, err := s.builder.BuildQuery(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build search query: %w", err)
	}

	// Execute search
	searchReq := opensearchapi.SearchRequest{
		Index: []string{s.client.GetIndex()},
		Body:  buildRequestBody(searchQuery),
	}

	res, err := searchReq.Do(ctx, s.client.GetClient())
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search failed with status: %s", res.Status())
	}

	// Parse response
	var searchResponse struct {
		Took     int64 `json:"took"`
		TimedOut bool  `json:"timed_out"`
		Hits     struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			MaxScore float64 `json:"max_score"`
			Hits     []struct {
				ID        string                 `json:"_id"`
				Score     float64                `json:"_score"`
				Source    map[string]interface{} `json:"_source"`
				Highlight map[string][]string    `json:"highlight,omitempty"`
			} `json:"hits"`
		} `json:"hits"`
		Aggregations map[string]interface{} `json:"aggregations,omitempty"`
	}

	if err := parseResponse(res, &searchResponse); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	// Convert to result format
	result := &models.SearchResult{
		TotalHits:    searchResponse.Hits.Total.Value,
		MaxScore:     searchResponse.Hits.MaxScore,
		Documents:    make([]*models.SearchDocument, len(searchResponse.Hits.Hits)),
		Aggregations: searchResponse.Aggregations,
		Took:         searchResponse.Took,
		TimedOut:     searchResponse.TimedOut,
	}

	for i, hit := range searchResponse.Hits.Hits {
		result.Documents[i] = &models.SearchDocument{
			ID:         hit.ID,
			Score:      hit.Score,
			Document:   hit.Source,
			Highlights: hit.Highlight,
		}
	}

	return result, nil
}

func (s *service) IndexDocument(ctx context.Context, doc *models.Document) (string, error) {
	if doc.ID == "" {
		return "", fmt.Errorf("document ID is required")
	}

	// Sanitize document ID by replacing forward slashes with underscores
	// OpenSearch can't handle document IDs with forward slashes in the URL path
	sanitizedID := strings.ReplaceAll(doc.ID, "/", "_")
	sanitizedID = strings.ReplaceAll(sanitizedID, "\\", "_")

	log.Printf("[OPENSEARCH] Indexing document: original ID='%s', sanitized ID='%s'", doc.ID, sanitizedID)

	// Prepare document for indexing
	docData, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal document: %w", err)
	}

	// Index document with sanitized ID
	indexReq := opensearchapi.IndexRequest{
		Index:      s.client.GetIndex(),
		DocumentID: sanitizedID,
		Body:       strings.NewReader(string(docData)),
	}

	res, err := indexReq.Do(ctx, s.client.GetClient())
	if err != nil {
		return "", fmt.Errorf("index request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		// Read error response body for detailed error information
		bodyBytes, _ := io.ReadAll(res.Body)
		res.Body.Close()
		errorBody := string(bodyBytes)
		log.Printf("[OPENSEARCH] Index error response: %s", errorBody)
		return "", fmt.Errorf("indexing failed with status: %s, body: %s", res.Status(), errorBody)
	}

	// Parse response to get document ID
	var indexResponse struct {
		ID string `json:"_id"`
	}

	if err := parseResponse(res, &indexResponse); err != nil {
		return "", fmt.Errorf("failed to parse index response: %w", err)
	}

	return indexResponse.ID, nil
}

func (s *service) IndexRawDocument(ctx context.Context, docID string, body []byte) (string, error) {
	if len(body) == 0 {
		return "", fmt.Errorf("document body is empty")
	}

	indexReq := opensearchapi.IndexRequest{
		Index: s.client.GetIndex(),
		Body:  bytes.NewReader(body),
	}

	if docID != "" {
		sanitizedID := strings.ReplaceAll(docID, "/", "_")
		sanitizedID = strings.ReplaceAll(sanitizedID, "\\", "_")
		indexReq.DocumentID = sanitizedID
	}

	res, err := indexReq.Do(ctx, s.client.GetClient())
	if err != nil {
		return "", fmt.Errorf("index request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("indexing failed with status: %s, body: %s", res.Status(), string(bodyBytes))
	}

	var indexResponse struct {
		ID string `json:"_id"`
	}

	if err := parseResponse(res, &indexResponse); err != nil {
		return "", fmt.Errorf("failed to parse index response: %w", err)
	}

	return indexResponse.ID, nil
}

func (s *service) BulkIndexDocuments(ctx context.Context, docs []*models.Document) (*models.BulkResult, error) {
	if len(docs) == 0 {
		return &models.BulkResult{}, nil
	}

	// Build bulk request body
	var bulkBody strings.Builder
	for _, doc := range docs {
		if doc.ID == "" {
			continue
		}

		// Add index action
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": s.client.GetIndex(),
				"_id":    doc.ID,
			},
		}
		actionJSON, _ := json.Marshal(action)
		bulkBody.Write(actionJSON)
		bulkBody.WriteString("\n")

		// Add document data
		docJSON, _ := json.Marshal(doc)
		bulkBody.Write(docJSON)
		bulkBody.WriteString("\n")
	}

	// Execute bulk request
	bulkReq := opensearchapi.BulkRequest{
		Body: strings.NewReader(bulkBody.String()),
	}

	res, err := bulkReq.Do(ctx, s.client.GetClient())
	if err != nil {
		return nil, fmt.Errorf("bulk request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("bulk indexing failed with status: %s", res.Status())
	}

	// Parse bulk response
	var bulkResponse struct {
		Took   int64                               `json:"took"`
		Errors bool                                `json:"errors"`
		Items  []map[string]map[string]interface{} `json:"items"`
	}

	if err := parseResponse(res, &bulkResponse); err != nil {
		return nil, fmt.Errorf("failed to parse bulk response: %w", err)
	}

	// Convert to result format
	result := &models.BulkResult{
		Took:   bulkResponse.Took,
		Errors: bulkResponse.Errors,
		Items:  make([]models.BulkResultItem, len(bulkResponse.Items)),
	}

	indexed := 0
	failed := 0
	var failedDocs []*models.BulkFailedDoc

	for i, item := range bulkResponse.Items {
		// Extract the operation result (index, create, update, or delete)
		var opResult map[string]interface{}
		var opType string

		for k, v := range item {
			opType = k
			opResult = v
			break
		}

		resultItem := models.BulkResultItem{}
		status := int(opResult["status"].(float64))

		switch opType {
		case "index":
			resultItem.Index = &models.BulkItemResult{
				ID:     opResult["_id"].(string),
				Index:  opResult["_index"].(string),
				Status: status,
			}
		}

		if status >= 200 && status < 300 {
			indexed++
		} else {
			failed++
			if errorInfo, exists := opResult["error"]; exists {
				errorMap := errorInfo.(map[string]interface{})
				failedDoc := &models.BulkFailedDoc{
					ID:     opResult["_id"].(string),
					Error:  errorMap["reason"].(string),
					Status: status,
				}
				failedDocs = append(failedDocs, failedDoc)
			}
		}

		result.Items[i] = resultItem
	}

	result.Indexed = indexed
	result.Failed = failed
	result.FailedDocs = failedDocs

	return result, nil
}

func (s *service) UpdateDocumentMetadata(ctx context.Context, docID string, metadata map[string]interface{}) error {
	updateDoc := map[string]interface{}{
		"doc": map[string]interface{}{
			"metadata":   metadata,
			"updated_at": time.Now(),
		},
	}

	updateReq := opensearchapi.UpdateRequest{
		Index:      s.client.GetIndex(),
		DocumentID: docID,
		Body:       buildRequestBody(updateDoc),
	}

	res, err := updateReq.Do(ctx, s.client.GetClient())
	if err != nil {
		return fmt.Errorf("update request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("update failed with status: %s", res.Status())
	}

	return nil
}

func (s *service) DeleteDocument(ctx context.Context, docID string) error {
	deleteReq := opensearchapi.DeleteRequest{
		Index:      s.client.GetIndex(),
		DocumentID: docID,
	}

	res, err := deleteReq.Do(ctx, s.client.GetClient())
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("delete failed with status: %s", res.Status())
	}

	return nil
}
