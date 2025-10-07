package search

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"motion-index-fiber/internal/config"
	"motion-index-fiber/pkg/search/client"
)

func TestNewClient_ValidConfig(t *testing.T) {
	cfg := &config.OpenSearchConfig{
		Host:     "localhost",
		Port:     9200,
		Username: "admin",
		Password: "admin",
		UseSSL:   false,
		Index:    "test-documents",
	}

	// This test will fail without a running OpenSearch instance
	// In CI/CD, we would use a test container
	client, err := client.NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping test - OpenSearch not available: %v", err)
		return
	}

	assert.NotNil(t, client)
	assert.True(t, client.IsHealthy())
	assert.Equal(t, "test-documents", client.GetIndex())

	// Cleanup
	client.Close()
}

func TestNewClient_InvalidConfig(t *testing.T) {
	testCases := []struct {
		name   string
		config *config.OpenSearchConfig
	}{
		{
			name: "empty host",
			config: &config.OpenSearchConfig{
				Host:  "",
				Port:  9200,
				Index: "test",
			},
		},
		{
			name: "invalid host",
			config: &config.OpenSearchConfig{
				Host:  "nonexistent-host-12345",
				Port:  9200,
				Index: "test",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := client.NewClient(tc.config)
			assert.Error(t, err)
			assert.Nil(t, client)
		})
	}
}

func TestClient_Health(t *testing.T) {
	cfg := &config.OpenSearchConfig{
		Host:     "localhost",
		Port:     9200,
		Username: "admin",
		Password: "admin",
		UseSSL:   false,
		Index:    "test-documents",
	}

	client, err := client.NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping test - OpenSearch not available: %v", err)
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	require.NoError(t, err)
	assert.NotNil(t, health)
	assert.NotEmpty(t, health.ClusterName)
	assert.Contains(t, []string{"green", "yellow", "red"}, health.Status)
}

func TestClient_IndexOperations(t *testing.T) {
	cfg := &config.OpenSearchConfig{
		Host:     "localhost",
		Port:     9200,
		Username: "admin",
		Password: "admin",
		UseSSL:   false,
		Index:    "test-documents-ops",
	}

	client, err := client.NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping test - OpenSearch not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// Clean up any existing index
	client.DeleteIndex(ctx)

	// Test index doesn't exist initially
	exists, err := client.IndexExists(ctx)
	require.NoError(t, err)
	assert.False(t, exists)

	// Create index with mapping
	mapping := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type": "text",
				},
			},
		},
	}

	err = client.CreateIndex(ctx, mapping)
	require.NoError(t, err)

	// Test index exists now
	exists, err = client.IndexExists(ctx)
	require.NoError(t, err)
	assert.True(t, exists)

	// Test refresh index
	err = client.RefreshIndex(ctx)
	require.NoError(t, err)

	// Clean up
	err = client.DeleteIndex(ctx)
	require.NoError(t, err)
}
