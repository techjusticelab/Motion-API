package search

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/search/client"
	"strings"
	"time"
)

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

func (s *service) IsHealthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.Health(ctx)
	return err == nil
}

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
