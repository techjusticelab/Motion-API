package handlers

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"motion-index-fiber/internal/application/usecase/system"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/storage"
)

// Mock services for health testing

type mockHealthyStorageService struct{}

func (m *mockHealthyStorageService) IsHealthy() bool { return true }
func (m *mockHealthyStorageService) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_documents": 100,
		"storage_used":    "1.5GB",
	}
}

// Stub methods for storage.Service interface compliance
func (m *mockHealthyStorageService) Upload(ctx context.Context, path string, content io.Reader, metadata *storage.UploadMetadata) (*storage.UploadResult, error) {
	return &storage.UploadResult{}, nil
}
func (m *mockHealthyStorageService) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, nil
}
func (m *mockHealthyStorageService) Delete(ctx context.Context, path string) error {
	return nil
}
func (m *mockHealthyStorageService) GetURL(path string) string {
	return ""
}
func (m *mockHealthyStorageService) GetSignedURL(path string, expiration time.Duration) (string, error) {
	return "", nil
}
func (m *mockHealthyStorageService) Exists(ctx context.Context, path string) (bool, error) {
	return false, nil
}
func (m *mockHealthyStorageService) List(ctx context.Context, prefix string) ([]*storage.StorageObject, error) {
	return nil, nil
}

type mockHealthySearchService struct{}

func (m *mockHealthySearchService) IsHealthy() bool { return true }
func (m *mockHealthySearchService) Health(ctx context.Context) (*search.HealthStatus, error) {
	return &search.HealthStatus{
		Status:         "healthy",
		ClusterName:    "test-cluster",
		NumberOfNodes:  3,
		ActiveShards:   10,
		IndexExists:    true,
		IndexHealth:    "green",
	}, nil
}

// Stub methods for search.Service interface compliance
func (m *mockHealthySearchService) SearchDocuments(ctx context.Context, req *models.SearchRequest) (*models.SearchResult, error) {
	return nil, nil
}
func (m *mockHealthySearchService) GetDocument(ctx context.Context, id string) (*models.Document, error) {
	return nil, nil
}
func (m *mockHealthySearchService) IndexDocument(ctx context.Context, doc *models.Document) (string, error) {
	return "", nil
}
func (m *mockHealthySearchService) UpdateDocumentMetadata(ctx context.Context, docID string, metadata map[string]interface{}) error {
	return nil
}
func (m *mockHealthySearchService) DeleteDocument(ctx context.Context, id string) error {
	return nil
}
func (m *mockHealthySearchService) GetLegalTags(ctx context.Context) ([]*models.TagCount, error) {
	return nil, nil
}
func (m *mockHealthySearchService) GetDocumentTypes(ctx context.Context) ([]*models.TypeCount, error) {
	return nil, nil
}
func (m *mockHealthySearchService) GetDocumentStats(ctx context.Context) (*models.DocumentStats, error) {
	return nil, nil
}
func (m *mockHealthySearchService) GetAllFieldOptions(ctx context.Context) (*models.FieldOptions, error) {
	return nil, nil
}
func (m *mockHealthySearchService) GetMetadataFieldValues(ctx context.Context, field string, prefix string, size int) ([]*models.FieldValue, error) {
	return nil, nil
}
func (m *mockHealthySearchService) GetMetadataFieldValuesWithFilters(ctx context.Context, req *models.MetadataFieldValuesRequest) ([]*models.FieldValue, error) {
	return nil, nil
}
func (m *mockHealthySearchService) BulkIndexDocuments(ctx context.Context, docs []*models.Document) (*models.BulkResult, error) {
	return nil, nil
}
func (m *mockHealthySearchService) DocumentExists(ctx context.Context, id string) (bool, error) {
	return false, nil
}

// Unhealthy mock services

type mockUnhealthyStorageService struct {
	mockHealthyStorageService
}

func (m *mockUnhealthyStorageService) IsHealthy() bool { return false }

type mockUnhealthySearchService struct {
	mockHealthySearchService
}

func (m *mockUnhealthySearchService) IsHealthy() bool { return false }

// Ensure interface compliance
var _ search.Service = (*mockHealthySearchService)(nil)

// Tests for Root, Health, HealthCheck (basic health endpoints)

func TestHealthHandlerRefactored_Root_Success(t *testing.T) {
	// Given: A handler with healthy services
	app := fiber.New()
	healthService := system.NewHealthService(&mockHealthyStorageService{}, &mockHealthySearchService{})
	handler := NewHealthHandlerRefactored(healthService)
	app.Get("/", handler.Root)

	// When: Root endpoint is requested
	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, 200, resp.StatusCode)
}

func TestHealthHandlerRefactored_Health_Success(t *testing.T) {
	// Given: A handler with healthy services
	app := fiber.New()
	healthService := system.NewHealthService(&mockHealthyStorageService{}, &mockHealthySearchService{})
	handler := NewHealthHandlerRefactored(healthService)
	app.Get("/health", handler.Health)

	// When: Health endpoint is requested
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, 200, resp.StatusCode)
}

func TestHealthHandlerRefactored_HealthCheck_Success(t *testing.T) {
	// Given: A handler with healthy services
	app := fiber.New()
	healthService := system.NewHealthService(&mockHealthyStorageService{}, &mockHealthySearchService{})
	handler := NewHealthHandlerRefactored(healthService)
	app.Get("/healthcheck", handler.HealthCheck)

	// When: HealthCheck endpoint is requested
	req := httptest.NewRequest("GET", "/healthcheck", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, 200, resp.StatusCode)
}

// Tests for DetailedStatus

func TestHealthHandlerRefactored_DetailedStatus_AllHealthy(t *testing.T) {
	// Given: A handler with all services healthy
	app := fiber.New()
	healthService := system.NewHealthService(&mockHealthyStorageService{}, &mockHealthySearchService{})
	handler := NewHealthHandlerRefactored(healthService)
	app.Get("/status", handler.DetailedStatus)

	// When: Detailed status is requested
	req := httptest.NewRequest("GET", "/status", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 200 OK
	assert.Equal(t, 200, resp.StatusCode)
}

func TestHealthHandlerRefactored_DetailedStatus_StorageUnhealthy(t *testing.T) {
	// Given: A handler with unhealthy storage
	app := fiber.New()
	healthService := system.NewHealthService(&mockUnhealthyStorageService{}, &mockHealthySearchService{})
	handler := NewHealthHandlerRefactored(healthService)
	app.Get("/status", handler.DetailedStatus)

	// When: Detailed status is requested
	req := httptest.NewRequest("GET", "/status", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 503 Service Unavailable
	assert.Equal(t, 503, resp.StatusCode)
}

func TestHealthHandlerRefactored_DetailedStatus_SearchUnhealthy(t *testing.T) {
	// Given: A handler with unhealthy search
	app := fiber.New()
	healthService := system.NewHealthService(&mockHealthyStorageService{}, &mockUnhealthySearchService{})
	handler := NewHealthHandlerRefactored(healthService)
	app.Get("/status", handler.DetailedStatus)

	// When: Detailed status is requested
	req := httptest.NewRequest("GET", "/status", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 503 Service Unavailable
	assert.Equal(t, 503, resp.StatusCode)
}

// Tests for ReadinessCheck

func TestHealthHandlerRefactored_ReadinessCheck_Ready(t *testing.T) {
	// Given: A handler with all services ready
	app := fiber.New()
	healthService := system.NewHealthService(&mockHealthyStorageService{}, &mockHealthySearchService{})
	handler := NewHealthHandlerRefactored(healthService)
	app.Get("/readiness", handler.ReadinessCheck)

	// When: Readiness check is requested
	req := httptest.NewRequest("GET", "/readiness", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 200 OK
	assert.Equal(t, 200, resp.StatusCode)
}

func TestHealthHandlerRefactored_ReadinessCheck_NotReady(t *testing.T) {
	// Given: A handler with unhealthy services
	app := fiber.New()
	healthService := system.NewHealthService(&mockUnhealthyStorageService{}, &mockUnhealthySearchService{})
	handler := NewHealthHandlerRefactored(healthService)
	app.Get("/readiness", handler.ReadinessCheck)

	// When: Readiness check is requested
	req := httptest.NewRequest("GET", "/readiness", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 503 Service Unavailable
	assert.Equal(t, 503, resp.StatusCode)
}

// Tests for LivenessCheck

func TestHealthHandlerRefactored_LivenessCheck_Success(t *testing.T) {
	// Given: A handler
	app := fiber.New()
	healthService := system.NewHealthService(&mockHealthyStorageService{}, &mockHealthySearchService{})
	handler := NewHealthHandlerRefactored(healthService)
	app.Get("/liveness", handler.LivenessCheck)

	// When: Liveness check is requested
	req := httptest.NewRequest("GET", "/liveness", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is always 200 OK (as long as server responds)
	assert.Equal(t, 200, resp.StatusCode)
}

// Tests for Metrics

func TestHealthHandlerRefactored_Metrics_Success(t *testing.T) {
	// Given: A handler with healthy services
	app := fiber.New()
	healthService := system.NewHealthService(&mockHealthyStorageService{}, &mockHealthySearchService{})
	handler := NewHealthHandlerRefactored(healthService)
	app.Get("/metrics", handler.Metrics)

	// When: Metrics are requested
	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, 200, resp.StatusCode)
}

func TestHealthHandlerRefactored_Metrics_WithUnhealthyServices(t *testing.T) {
	// Given: A handler with unhealthy services (metrics still work)
	app := fiber.New()
	healthService := system.NewHealthService(&mockUnhealthyStorageService{}, &mockUnhealthySearchService{})
	handler := NewHealthHandlerRefactored(healthService)
	app.Get("/metrics", handler.Metrics)

	// When: Metrics are requested
	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is still successful (metrics don't fail on unhealthy services)
	assert.Equal(t, 200, resp.StatusCode)
}

// Integration test

func TestHealthHandlerRefactored_Integration(t *testing.T) {
	// Given: A fully configured handler
	app := fiber.New()
	healthService := system.NewHealthService(&mockHealthyStorageService{}, &mockHealthySearchService{})
	handler := NewHealthHandlerRefactored(healthService)

	// Register all routes
	app.Get("/", handler.Root)
	app.Get("/health", handler.Health)
	app.Get("/healthcheck", handler.HealthCheck)
	app.Get("/status", handler.DetailedStatus)
	app.Get("/readiness", handler.ReadinessCheck)
	app.Get("/liveness", handler.LivenessCheck)
	app.Get("/metrics", handler.Metrics)

	t.Run("root endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("health endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("healthcheck endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/healthcheck", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("detailed status endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/status", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("readiness endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/readiness", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("liveness endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/liveness", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("metrics endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}
