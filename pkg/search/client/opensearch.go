package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"

	"motion-index-fiber/internal/config"
)

// Client wraps the OpenSearch client with additional functionality
type Client struct {
	client    *opensearch.Client
	index     string
	config    *config.OpenSearchConfig
	isHealthy bool
}

// HealthStatus represents the health status of the OpenSearch cluster
type HealthStatus struct {
	ClusterName   string `json:"cluster_name"`
	Status        string `json:"status"`
	TimedOut      bool   `json:"timed_out"`
	NumberOfNodes int    `json:"number_of_nodes"`
	ActiveShards  int    `json:"active_primary_shards"`
}

// NewClient creates a new OpenSearch client with connection pooling and health checks
func NewClient(cfg *config.OpenSearchConfig) (*Client, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("OpenSearch host is required")
	}

	// Build the OpenSearch URL
	protocol := "http"
	if cfg.UseSSL {
		protocol = "https"
	}
	url := fmt.Sprintf("%s://%s:%d", protocol, cfg.Host, cfg.Port)

	// Configure OpenSearch client
	opensearchConfig := opensearch.Config{
		Addresses: []string{url},
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: 120 * time.Second, // Increased from 30s to 120s for large documents
			IdleConnTimeout:       90 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // For DigitalOcean managed OpenSearch
			},
		},
	}

	// Add authentication if provided
	if cfg.Username != "" && cfg.Password != "" {
		opensearchConfig.Username = cfg.Username
		opensearchConfig.Password = cfg.Password
	}

	// Create OpenSearch client
	client, err := opensearch.NewClient(opensearchConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenSearch client: %w", err)
	}

	c := &Client{
		client: client,
		index:  cfg.Index,
		config: cfg,
	}

	// Test connection
	if err := c.ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to connect to OpenSearch: %w", err)
	}

	c.isHealthy = true
	return c, nil
}

// GetClient returns the underlying OpenSearch client
func (c *Client) GetClient() *opensearch.Client {
	return c.client
}

// GetIndex returns the configured index name
func (c *Client) GetIndex() string {
	return c.index
}

// IsHealthy returns the current health status
func (c *Client) IsHealthy() bool {
	return c.isHealthy
}

// ping tests the connection to OpenSearch
func (c *Client) ping(ctx context.Context) error {
	req := opensearchapi.InfoRequest{}
	res, err := req.Do(ctx, c.client)
	if err != nil {
		return fmt.Errorf("ping request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ping failed with status: %s", res.Status())
	}

	return nil
}

// Health checks the health of the OpenSearch cluster
func (c *Client) Health(ctx context.Context) (*HealthStatus, error) {
	req := opensearchapi.ClusterHealthRequest{
		Timeout: time.Second * 10,
	}

	res, err := req.Do(ctx, c.client)
	if err != nil {
		c.isHealthy = false
		return nil, fmt.Errorf("health check request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		c.isHealthy = false
		return nil, fmt.Errorf("health check failed with status: %s", res.Status())
	}

	var health HealthStatus
	if err := parseResponse(res, &health); err != nil {
		c.isHealthy = false
		return nil, fmt.Errorf("failed to parse health response: %w", err)
	}

	// Update health status based on cluster status
	c.isHealthy = health.Status == "green" || health.Status == "yellow"

	return &health, nil
}

// IndexExists checks if the configured index exists
func (c *Client) IndexExists(ctx context.Context) (bool, error) {
	req := opensearchapi.IndicesExistsRequest{
		Index: []string{c.index},
	}

	res, err := req.Do(ctx, c.client)
	if err != nil {
		return false, fmt.Errorf("index exists check failed: %w", err)
	}
	defer res.Body.Close()

	// 200 means index exists, 404 means it doesn't
	if res.StatusCode == 200 {
		return true, nil
	} else if res.StatusCode == 404 {
		return false, nil
	}

	return false, fmt.Errorf("unexpected status code: %d", res.StatusCode)
}

// CreateIndex creates the index with the provided mapping
func (c *Client) CreateIndex(ctx context.Context, mapping map[string]interface{}) error {
	exists, err := c.IndexExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if index exists: %w", err)
	}

	if exists {
		return nil // Index already exists
	}

	req := opensearchapi.IndicesCreateRequest{
		Index: c.index,
		Body:  buildRequestBody(mapping),
	}

	res, err := req.Do(ctx, c.client)
	if err != nil {
		return fmt.Errorf("create index request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("create index failed with status: %s", res.Status())
	}

	return nil
}

// DeleteIndex deletes the configured index
func (c *Client) DeleteIndex(ctx context.Context) error {
	req := opensearchapi.IndicesDeleteRequest{
		Index: []string{c.index},
	}

	res, err := req.Do(ctx, c.client)
	if err != nil {
		return fmt.Errorf("delete index request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("delete index failed with status: %s", res.Status())
	}

	return nil
}

// RefreshIndex refreshes the index to make recent changes searchable
func (c *Client) RefreshIndex(ctx context.Context) error {
	req := opensearchapi.IndicesRefreshRequest{
		Index: []string{c.index},
	}

	res, err := req.Do(ctx, c.client)
	if err != nil {
		return fmt.Errorf("refresh index request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("refresh index failed with status: %s", res.Status())
	}

	return nil
}

// Close closes the client connections
func (c *Client) Close() error {
	// OpenSearch Go client doesn't require explicit cleanup
	c.isHealthy = false
	return nil
}
