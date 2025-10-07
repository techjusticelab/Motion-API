package digitalocean

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"motion-index-fiber/pkg/cloud/digitalocean/config"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/storage"
)

func TestNewServiceFactory(t *testing.T) {
	cfg := config.DefaultConfig()
	factory := NewServiceFactory(cfg)

	assert.NotNil(t, factory)
	assert.Equal(t, cfg, factory.config)
}

func TestServiceFactory_CreateStorageService(t *testing.T) {
	tests := []struct {
		name          string
		environment   config.Environment
		setupConfig   func(*config.Config)
		expectError   bool
		errorContains string
	}{
		{
			name:        "local environment returns mock service",
			environment: config.EnvLocal,
			setupConfig: func(cfg *config.Config) {
				// Local environment uses mock services by default
			},
			expectError:   false,
			errorContains: "",
		},
		{
			name:        "staging environment requires valid configuration",
			environment: config.EnvStaging,
			setupConfig: func(cfg *config.Config) {
				// Invalid/missing configuration
			},
			expectError:   true,
			errorContains: "invalid Spaces configuration",
		},
		{
			name:        "production environment requires valid configuration",
			environment: config.EnvProduction,
			setupConfig: func(cfg *config.Config) {
				// Invalid/missing configuration
			},
			expectError:   true,
			errorContains: "invalid Spaces configuration",
		},
		{
			name:        "staging environment with valid config creates spaces service",
			environment: config.EnvStaging,
			setupConfig: func(cfg *config.Config) {
				cfg.DigitalOcean.Spaces.AccessKey = "test-access-key"
				cfg.DigitalOcean.Spaces.SecretKey = "test-secret-key"
				cfg.DigitalOcean.Spaces.Bucket = "test-bucket"
				cfg.DigitalOcean.Spaces.Region = "nyc3"
			},
			expectError:   false,
			errorContains: "",
		},
		{
			name:        "production environment with valid config creates spaces service",
			environment: config.EnvProduction,
			setupConfig: func(cfg *config.Config) {
				cfg.DigitalOcean.Spaces.AccessKey = "test-access-key"
				cfg.DigitalOcean.Spaces.SecretKey = "test-secret-key"
				cfg.DigitalOcean.Spaces.Bucket = "test-bucket"
				cfg.DigitalOcean.Spaces.Region = "nyc3"
			},
			expectError:   false,
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Environment = tt.environment

			if tt.setupConfig != nil {
				tt.setupConfig(cfg)
			}

			factory := NewServiceFactory(cfg)

			service, err := factory.CreateStorageService()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, service)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)

				// For local environment, verify mock service is healthy
				if tt.environment == config.EnvLocal {
					assert.True(t, service.IsHealthy())
				}
				// For staging/production with test credentials, service may not be healthy
				// but should implement the interface correctly
				metrics := service.GetMetrics()
				assert.NotNil(t, metrics)
			}
		})
	}
}

func TestServiceFactory_CreateSearchService(t *testing.T) {
	tests := []struct {
		name          string
		environment   config.Environment
		setupConfig   func(*config.Config)
		expectError   bool
		errorContains string
	}{
		{
			name:        "local environment returns mock service",
			environment: config.EnvLocal,
			setupConfig: func(cfg *config.Config) {
				// Local environment uses mock services by default
			},
			expectError:   false,
			errorContains: "",
		},
		{
			name:        "staging environment with invalid config fails to create opensearch service",
			environment: config.EnvStaging,
			setupConfig: func(cfg *config.Config) {
				cfg.DigitalOcean.OpenSearch.Host = "invalid-host.com"
				cfg.DigitalOcean.OpenSearch.Port = 9200
				cfg.DigitalOcean.OpenSearch.Username = "invalid-user"
				cfg.DigitalOcean.OpenSearch.Password = "invalid-password"
				cfg.DigitalOcean.OpenSearch.UseSSL = false
				cfg.DigitalOcean.OpenSearch.Index = "test-index"
			},
			expectError:   true,
			errorContains: "failed to create OpenSearch client",
		},
		{
			name:        "production environment with invalid config fails to create opensearch service",
			environment: config.EnvProduction,
			setupConfig: func(cfg *config.Config) {
				cfg.DigitalOcean.OpenSearch.Host = "invalid-host.com"
				cfg.DigitalOcean.OpenSearch.Port = 9200
				cfg.DigitalOcean.OpenSearch.Username = "invalid-user"
				cfg.DigitalOcean.OpenSearch.Password = "invalid-password"
				cfg.DigitalOcean.OpenSearch.UseSSL = true
				cfg.DigitalOcean.OpenSearch.Index = "test-index"
			},
			expectError:   true,
			errorContains: "failed to create OpenSearch client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Environment = tt.environment

			if tt.setupConfig != nil {
				tt.setupConfig(cfg)
			}

			factory := NewServiceFactory(cfg)

			service, err := factory.CreateSearchService()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, service)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)

				// For local environment, verify mock service is healthy
				if tt.environment == config.EnvLocal {
					assert.True(t, service.IsHealthy())
				}
				// For staging/production with test credentials, service may not be healthy
				// but should implement the interface correctly
			}
		})
	}
}

func TestServiceFactory_CreateAllServices(t *testing.T) {
	tests := []struct {
		name          string
		environment   config.Environment
		setupConfig   func(*config.Config)
		expectError   bool
		errorContains string
	}{
		{
			name:        "local environment creates mock services",
			environment: config.EnvLocal,
			setupConfig: func(cfg *config.Config) {
				// Local environment uses mock services by default
			},
			expectError:   false,
			errorContains: "",
		},
		{
			name:        "staging environment with valid config creates real services",
			environment: config.EnvStaging,
			setupConfig: func(cfg *config.Config) {
				// Configure valid Spaces
				cfg.DigitalOcean.Spaces.AccessKey = "test-access-key"
				cfg.DigitalOcean.Spaces.SecretKey = "test-secret-key"
				cfg.DigitalOcean.Spaces.Bucket = "test-bucket"
				cfg.DigitalOcean.Spaces.Region = "nyc3"

				// Configure valid OpenSearch
				cfg.DigitalOcean.OpenSearch.Host = "test-host.com"
				cfg.DigitalOcean.OpenSearch.Port = 9200
				cfg.DigitalOcean.OpenSearch.Username = "test-user"
				cfg.DigitalOcean.OpenSearch.Password = "test-password"
				cfg.DigitalOcean.OpenSearch.UseSSL = false
				cfg.DigitalOcean.OpenSearch.Index = "test-index"
			},
			expectError:   false,
			errorContains: "",
		},
		{
			name:        "staging environment with invalid config fails",
			environment: config.EnvStaging,
			setupConfig: func(cfg *config.Config) {
				// Invalid configuration (missing required fields)
			},
			expectError:   true,
			errorContains: "failed to create storage service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Environment = tt.environment

			if tt.setupConfig != nil {
				tt.setupConfig(cfg)
			}

			factory := NewServiceFactory(cfg)

			services, err := factory.CreateAllServices()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, services)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				require.NotNil(t, services)
				assert.NotNil(t, services.Storage)
				assert.NotNil(t, services.Search)
				assert.Equal(t, cfg, services.Config)

				// Verify services are healthy
				assert.True(t, services.IsHealthy())

				// Verify metrics collection
				metrics := services.GetMetrics()
				assert.NotNil(t, metrics)
				assert.Contains(t, metrics, "config")
			}
		})
	}
}

func TestServiceFactory_ValidateServices(t *testing.T) {
	tests := []struct {
		name          string
		environment   config.Environment
		setupConfig   func(*config.Config)
		expectError   bool
		errorContains string
	}{
		{
			name:        "local environment validates successfully with mock services",
			environment: config.EnvLocal,
			setupConfig: func(cfg *config.Config) {
				// Local environment uses mock services
			},
			expectError: false,
		},
		{
			name:        "staging environment with valid config validates successfully",
			environment: config.EnvStaging,
			setupConfig: func(cfg *config.Config) {
				// Configure valid services
				cfg.DigitalOcean.Spaces.AccessKey = "test-access-key"
				cfg.DigitalOcean.Spaces.SecretKey = "test-secret-key"
				cfg.DigitalOcean.Spaces.Bucket = "test-bucket"
				cfg.DigitalOcean.Spaces.Region = "nyc3"

				cfg.DigitalOcean.OpenSearch.Host = "test-host.com"
				cfg.DigitalOcean.OpenSearch.Port = 9200
				cfg.DigitalOcean.OpenSearch.Username = "test-user"
				cfg.DigitalOcean.OpenSearch.Password = "test-password"
				cfg.DigitalOcean.OpenSearch.UseSSL = false
				cfg.DigitalOcean.OpenSearch.Index = "test-index"
			},
			expectError: false,
		},
		{
			name:        "staging environment with invalid config fails validation",
			environment: config.EnvStaging,
			setupConfig: func(cfg *config.Config) {
				// Invalid configuration
			},
			expectError:   true,
			errorContains: "service creation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Environment = tt.environment

			if tt.setupConfig != nil {
				tt.setupConfig(cfg)
			}

			factory := NewServiceFactory(cfg)

			ctx := context.Background()
			err := factory.ValidateServices(ctx)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServices_Close(t *testing.T) {
	cfg := config.DefaultConfig()
	services := &Services{
		Storage: nil, // Mock services would go here
		Search:  nil,
		Config:  cfg,
	}

	err := services.Close()
	assert.NoError(t, err) // Should not error with nil services
}

func TestServices_IsHealthy(t *testing.T) {
	cfg := config.DefaultConfig()

	t.Run("with nil storage returns false", func(t *testing.T) {
		services := &Services{
			Storage: nil,
			Search:  nil,
			Config:  cfg,
		}

		// Should return false due to nil storage
		healthy := services.IsHealthy()
		assert.False(t, healthy)
	})

	t.Run("with unhealthy storage returns false", func(t *testing.T) {
		mockStorage := &MockStorageService{healthy: false}
		services := &Services{
			Storage: mockStorage,
			Search:  nil,
			Config:  cfg,
		}

		healthy := services.IsHealthy()
		assert.False(t, healthy)
	})

	t.Run("with nil search returns false", func(t *testing.T) {
		mockStorage := &MockStorageService{healthy: true}
		services := &Services{
			Storage: mockStorage,
			Search:  nil,
			Config:  cfg,
		}

		healthy := services.IsHealthy()
		assert.False(t, healthy)
	})

	t.Run("with healthy storage and search returns true", func(t *testing.T) {
		mockStorage := &MockStorageService{healthy: true}
		mockSearch := &MockSearchService{healthy: true}
		services := &Services{
			Storage: mockStorage,
			Search:  mockSearch,
			Config:  cfg,
		}

		healthy := services.IsHealthy()
		assert.True(t, healthy)
	})
}

func TestServices_GetMetrics(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Environment = config.EnvStaging
	cfg.DigitalOcean.Spaces.Region = "nyc3"
	cfg.DigitalOcean.OpenSearch.Host = "test-host"

	t.Run("with storage metrics", func(t *testing.T) {
		// Create mock storage service that implements GetMetrics
		mockStorage := &MockStorageService{
			healthy: true,
			metrics: map[string]interface{}{
				"total_files": 100,
				"total_size":  1024000,
			},
		}

		services := &Services{
			Storage: mockStorage,
			Search:  nil, // Mock search service would go here
			Config:  cfg,
		}

		metrics := services.GetMetrics()

		require.NotNil(t, metrics)

		// Check storage metrics
		storageMetrics, exists := metrics["storage"]
		assert.True(t, exists)
		assert.Equal(t, mockStorage.metrics, storageMetrics)

		// Check config metrics
		configMetrics, exists := metrics["config"]
		require.True(t, exists)

		configMap, ok := configMetrics.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, config.EnvStaging, configMap["environment"])
		assert.Equal(t, false, configMap["use_mock_services"])
		assert.Equal(t, "nyc3", configMap["spaces_region"])
		assert.Equal(t, "test-host", configMap["opensearch_host"])
	})

	t.Run("with empty storage metrics", func(t *testing.T) {
		mockStorage := &MockStorageService{
			healthy: true,
			metrics: map[string]interface{}{}, // Empty metrics
		}

		services := &Services{
			Storage: mockStorage,
			Search:  nil,
			Config:  cfg,
		}

		metrics := services.GetMetrics()

		require.NotNil(t, metrics)

		// Empty storage metrics should not be included
		_, exists := metrics["storage"]
		assert.False(t, exists)

		// Config metrics should still be present
		_, exists = metrics["config"]
		assert.True(t, exists)
	})
}

func TestServiceFactory_BridgeConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DigitalOcean.OpenSearch.Host = "test-host.com"
	cfg.DigitalOcean.OpenSearch.Port = 9200
	cfg.DigitalOcean.OpenSearch.Username = "test-user"
	cfg.DigitalOcean.OpenSearch.Password = "test-password"
	cfg.DigitalOcean.OpenSearch.UseSSL = true
	cfg.DigitalOcean.OpenSearch.Index = "test-index"

	factory := NewServiceFactory(cfg)

	// Test the bridge config method by calling it indirectly
	bridgedConfig := factory.bridgeOpenSearchConfig()

	assert.Equal(t, "test-host.com", bridgedConfig.Host)
	assert.Equal(t, 9200, bridgedConfig.Port)
	assert.Equal(t, "test-user", bridgedConfig.Username)
	assert.Equal(t, "test-password", bridgedConfig.Password)
	assert.True(t, bridgedConfig.UseSSL)
	assert.Equal(t, "test-index", bridgedConfig.Index)
}

func TestServiceFactory_EdgeCases(t *testing.T) {
	t.Run("nil config panics", func(t *testing.T) {
		assert.Panics(t, func() {
			factory := &ServiceFactory{config: nil}
			factory.CreateStorageService()
		})
	})

	t.Run("unknown environment defaults to production behavior", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Environment = "unknown" // This should be treated as non-local

		factory := NewServiceFactory(cfg)

		// Should try to create real services (which will fail due to missing config)
		_, err := factory.CreateStorageService()
		assert.Error(t, err)
	})
}

// MockStorageService implements storage.Service for testing
type MockStorageService struct {
	healthy bool
	metrics map[string]interface{}
}

func (m *MockStorageService) Upload(ctx context.Context, path string, content io.Reader, metadata *storage.UploadMetadata) (*storage.UploadResult, error) {
	return nil, nil
}

func (m *MockStorageService) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, nil
}

func (m *MockStorageService) Delete(ctx context.Context, path string) error {
	return nil
}

func (m *MockStorageService) GetURL(path string) string {
	return ""
}

func (m *MockStorageService) GetSignedURL(path string, expiration time.Duration) (string, error) {
	return "", nil
}

func (m *MockStorageService) Exists(ctx context.Context, path string) (bool, error) {
	return false, nil
}

func (m *MockStorageService) List(ctx context.Context, prefix string) ([]*storage.StorageObject, error) {
	return nil, nil
}

func (m *MockStorageService) IsHealthy() bool {
	return m.healthy
}

func (m *MockStorageService) GetMetrics() map[string]interface{} {
	return m.metrics
}

// MockSearchService implements search.Service for testing
type MockSearchService struct {
	healthy bool
}

// SearchService methods
func (m *MockSearchService) SearchDocuments(ctx context.Context, req *models.SearchRequest) (*models.SearchResult, error) {
	return &models.SearchResult{
		TotalHits: 0,
		MaxScore:  0.0,
		Documents: []*models.SearchDocument{},
		Took:      10,
		TimedOut:  false,
	}, nil
}

func (m *MockSearchService) IndexDocument(ctx context.Context, doc *models.Document) (string, error) {
	if doc.ID == "" {
		return "", fmt.Errorf("document ID is required")
	}
	return doc.ID, nil
}

func (m *MockSearchService) BulkIndexDocuments(ctx context.Context, docs []*models.Document) (*models.BulkResult, error) {
	return &models.BulkResult{
		Indexed:    len(docs),
		Failed:     0,
		Took:       50,
		Errors:     false,
		Items:      []models.BulkResultItem{},
		FailedDocs: []*models.BulkFailedDoc{},
	}, nil
}

func (m *MockSearchService) UpdateDocumentMetadata(ctx context.Context, docID string, metadata map[string]interface{}) error {
	return nil
}

func (m *MockSearchService) DeleteDocument(ctx context.Context, docID string) error {
	return nil
}

func (m *MockSearchService) GetDocument(ctx context.Context, docID string) (*models.Document, error) {
	return nil, fmt.Errorf("document not found")
}

func (m *MockSearchService) DocumentExists(ctx context.Context, docID string) (bool, error) {
	return false, nil
}

// AggregationService methods
func (m *MockSearchService) GetLegalTags(ctx context.Context) ([]*models.TagCount, error) {
	return []*models.TagCount{}, nil
}

func (m *MockSearchService) GetDocumentTypes(ctx context.Context) ([]*models.TypeCount, error) {
	return []*models.TypeCount{}, nil
}

func (m *MockSearchService) GetMetadataFieldValues(ctx context.Context, field string, prefix string, size int) ([]*models.FieldValue, error) {
	return []*models.FieldValue{}, nil
}

func (m *MockSearchService) GetMetadataFieldValuesWithFilters(ctx context.Context, req *models.MetadataFieldValuesRequest) ([]*models.FieldValue, error) {
	return []*models.FieldValue{}, nil
}

func (m *MockSearchService) GetDocumentStats(ctx context.Context) (*models.DocumentStats, error) {
	return &models.DocumentStats{
		TotalDocuments: 0,
		IndexSize:      "0 bytes",
		TypeCounts:     []*models.TypeCount{},
		TagCounts:      []*models.TagCount{},
		LastUpdated:    time.Now(),
		FieldStats:     make(map[string]models.FieldStat),
	}, nil
}

func (m *MockSearchService) GetAllFieldOptions(ctx context.Context) (*models.FieldOptions, error) {
	return &models.FieldOptions{}, nil
}

// HealthChecker methods
func (m *MockSearchService) IsHealthy() bool {
	return m.healthy
}

func (m *MockSearchService) Health(ctx context.Context) (*search.HealthStatus, error) {
	return &search.HealthStatus{
		Status:        "green",
		ClusterName:   "mock-cluster",
		NumberOfNodes: 1,
		ActiveShards:  1,
		IndexExists:   true,
		IndexHealth:   "green",
	}, nil
}
