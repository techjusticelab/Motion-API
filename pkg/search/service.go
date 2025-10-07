package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"

	"motion-index-fiber/pkg/search/client"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/search/query"
)

// service implements the Service interface
type service struct {
	client  client.SearchClient
	builder *query.Builder
}

// NewService creates a new search service
func NewService(searchClient client.SearchClient) Service {
	return &service{
		client:  searchClient,
		builder: query.NewBuilder(),
	}
}

// SearchDocuments performs a search query and returns results
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

// IndexDocument indexes a single document
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

// BulkIndexDocuments indexes multiple documents in a single operation
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

// UpdateDocumentMetadata updates metadata for an existing document
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

// DeleteDocument removes a document from the index
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

// GetDocument retrieves a document by ID
func (s *service) GetDocument(ctx context.Context, docID string) (*models.Document, error) {
	getReq := opensearchapi.GetRequest{
		Index:      s.client.GetIndex(),
		DocumentID: docID,
	}

	res, err := getReq.Do(ctx, s.client.GetClient())
	if err != nil {
		return nil, fmt.Errorf("get request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			return nil, fmt.Errorf("document not found")
		}
		return nil, fmt.Errorf("get failed with status: %s", res.Status())
	}

	var getResponse struct {
		Source models.Document `json:"_source"`
		Found  bool            `json:"found"`
	}

	if err := parseResponse(res, &getResponse); err != nil {
		return nil, fmt.Errorf("failed to parse get response: %w", err)
	}

	if !getResponse.Found {
		return nil, fmt.Errorf("document not found")
	}

	return &getResponse.Source, nil
}

// DocumentExists checks if a document exists in the index
func (s *service) DocumentExists(ctx context.Context, docID string) (bool, error) {
	existsReq := opensearchapi.ExistsRequest{
		Index:      s.client.GetIndex(),
		DocumentID: docID,
	}

	res, err := existsReq.Do(ctx, s.client.GetClient())
	if err != nil {
		return false, fmt.Errorf("exists request failed: %w", err)
	}
	defer res.Body.Close()

	return res.StatusCode == 200, nil
}

// IsHealthy returns true if the search service is healthy
func (s *service) IsHealthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.Health(ctx)
	return err == nil
}

// Health returns detailed health information
func (s *service) Health(ctx context.Context) (*HealthStatus, error) {
	healthReq := opensearchapi.ClusterHealthRequest{
		Index: []string{s.client.GetIndex()},
	}

	res, err := healthReq.Do(ctx, s.client.GetClient())
	if err != nil {
		return nil, fmt.Errorf("health check request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("health check failed with status: %s", res.Status())
	}

	var healthResponse struct {
		ClusterName                 string  `json:"cluster_name"`
		Status                      string  `json:"status"`
		NumberOfNodes               int     `json:"number_of_nodes"`
		NumberOfDataNodes           int     `json:"number_of_data_nodes"`
		ActivePrimaryShards         int     `json:"active_primary_shards"`
		ActiveShards                int     `json:"active_shards"`
		RelocatingShards            int     `json:"relocating_shards"`
		InitializingShards          int     `json:"initializing_shards"`
		UnassignedShards            int     `json:"unassigned_shards"`
		DelayedUnassignedShards     int     `json:"delayed_unassigned_shards"`
		NumberOfPendingTasks        int     `json:"number_of_pending_tasks"`
		NumberOfInFlightFetch       int     `json:"number_of_in_flight_fetch"`
		TaskMaxWaitingInQueueMillis int     `json:"task_max_waiting_in_queue_millis"`
		ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
	}

	if err := parseResponse(res, &healthResponse); err != nil {
		return nil, fmt.Errorf("failed to parse health response: %w", err)
	}

	// Check if index exists
	indexExists := true
	existsReq := opensearchapi.IndicesExistsRequest{
		Index: []string{s.client.GetIndex()},
	}

	existsRes, err := existsReq.Do(ctx, s.client.GetClient())
	if err == nil {
		indexExists = existsRes.StatusCode == 200
		existsRes.Body.Close()
	}

	return &HealthStatus{
		Status:        healthResponse.Status,
		ClusterName:   healthResponse.ClusterName,
		NumberOfNodes: healthResponse.NumberOfNodes,
		ActiveShards:  healthResponse.ActiveShards,
		IndexExists:   indexExists,
		IndexHealth:   healthResponse.Status,
	}, nil
}

// Helper functions
func buildRequestBody(data interface{}) *strings.Reader {
	if data == nil {
		return nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil
	}

	return strings.NewReader(string(jsonData))
}

func parseResponse(res *opensearchapi.Response, target interface{}) error {
	return json.NewDecoder(res.Body).Decode(target)
}

// Index management methods for setup-index command

// IndexExists checks if an index exists
func (s *service) IndexExists(ctx context.Context, name string) (bool, error) {
	// If name matches our configured index, use client method
	if name == s.client.GetIndex() {
		return s.client.IndexExists(ctx)
	}
	
	// For other indices, use direct API call
	req := opensearchapi.IndicesExistsRequest{
		Index: []string{name},
	}
	res, err := req.Do(ctx, s.client.GetClient())
	if err != nil {
		return false, fmt.Errorf("index exists check failed: %w", err)
	}
	defer res.Body.Close()
	
	return res.StatusCode == 200, nil
}

// CreateIndex creates an index with the given mapping
func (s *service) CreateIndex(ctx context.Context, name string, mapping map[string]interface{}) error {
	// If name matches our configured index, use client method
	if name == s.client.GetIndex() {
		return s.client.CreateIndex(ctx, mapping)
	}
	
	// For other indices, use direct API call
	mappingJSON, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}
	
	req := opensearchapi.IndicesCreateRequest{
		Index: name,
		Body:  strings.NewReader(string(mappingJSON)),
	}
	
	res, err := req.Do(ctx, s.client.GetClient())
	if err != nil {
		return fmt.Errorf("create index request failed: %w", err)
	}
	defer res.Body.Close()
	
	if res.IsError() {
		return fmt.Errorf("create index failed with status: %s", res.Status())
	}
	
	return nil
}

// DeleteIndex deletes an index
func (s *service) DeleteIndex(ctx context.Context, name string) error {
	// If name matches our configured index, use client method
	if name == s.client.GetIndex() {
		return s.client.DeleteIndex(ctx)
	}
	
	// For other indices, use direct API call
	req := opensearchapi.IndicesDeleteRequest{
		Index: []string{name},
	}
	
	res, err := req.Do(ctx, s.client.GetClient())
	if err != nil {
		return fmt.Errorf("delete index request failed: %w", err)
	}
	defer res.Body.Close()
	
	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("delete index failed with status: %s", res.Status())
	}
	
	return nil
}
