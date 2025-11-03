package system

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/storage"
)

// Mock services for testing

type mockHealthyStorage struct{}

func (m *mockHealthyStorage) IsHealthy() bool { return true }
func (m *mockHealthyStorage) GetMetrics() map[string]interface{} {
	return map[string]interface{}{"test_metric": "value"}
}
func (m *mockHealthyStorage) Upload(ctx context.Context, path string, content io.Reader, metadata *storage.UploadMetadata) (*storage.UploadResult, error) {
	return nil, nil
}
func (m *mockHealthyStorage) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, nil
}
func (m *mockHealthyStorage) Delete(ctx context.Context, path string) error {
	return nil
}
func (m *mockHealthyStorage) GetURL(path string) string {
	return ""
}
func (m *mockHealthyStorage) GetSignedURL(path string, expiration time.Duration) (string, error) {
	return "", nil
}
func (m *mockHealthyStorage) Exists(ctx context.Context, path string) (bool, error) {
	return false, nil
}
func (m *mockHealthyStorage) List(ctx context.Context, prefix string) ([]*storage.StorageObject, error) {
	return nil, nil
}

type mockUnhealthyStorage struct {
	mockHealthyStorage
}

func (m *mockUnhealthyStorage) IsHealthy() bool { return false }

type mockHealthySearch struct{}

func (m *mockHealthySearch) IsHealthy() bool { return true }
func (m *mockHealthySearch) Health(ctx context.Context) (*search.HealthStatus, error) {
	return &search.HealthStatus{
		ClusterName:    "test-cluster",
		NumberOfNodes:  3,
		ActiveShards:   10,
		IndexExists:    true,
		IndexHealth:    "green",
	}, nil
}

// Stub methods for search.Service
func (m *mockHealthySearch) SearchDocuments(ctx context.Context, req *models.SearchRequest) (*models.SearchResult, error) {
	return nil, nil
}
func (m *mockHealthySearch) GetDocument(ctx context.Context, id string) (*models.Document, error) {
	return nil, nil
}
func (m *mockHealthySearch) IndexDocument(ctx context.Context, doc *models.Document) (string, error) {
	return "", nil
}
func (m *mockHealthySearch) UpdateDocumentMetadata(ctx context.Context, docID string, metadata map[string]interface{}) error {
	return nil
}
func (m *mockHealthySearch) DeleteDocument(ctx context.Context, id string) error {
	return nil
}
func (m *mockHealthySearch) GetLegalTags(ctx context.Context) ([]*models.TagCount, error) {
	return nil, nil
}
func (m *mockHealthySearch) GetDocumentTypes(ctx context.Context) ([]*models.TypeCount, error) {
	return nil, nil
}
func (m *mockHealthySearch) GetDocumentStats(ctx context.Context) (*models.DocumentStats, error) {
	return nil, nil
}
func (m *mockHealthySearch) GetAllFieldOptions(ctx context.Context) (*models.FieldOptions, error) {
	return nil, nil
}
func (m *mockHealthySearch) GetMetadataFieldValues(ctx context.Context, field string, prefix string, size int) ([]*models.FieldValue, error) {
	return nil, nil
}
func (m *mockHealthySearch) GetMetadataFieldValuesWithFilters(ctx context.Context, req *models.MetadataFieldValuesRequest) ([]*models.FieldValue, error) {
	return nil, nil
}
func (m *mockHealthySearch) BulkIndexDocuments(ctx context.Context, docs []*models.Document) (*models.BulkResult, error) {
	return nil, nil
}
func (m *mockHealthySearch) DocumentExists(ctx context.Context, id string) (bool, error) {
	return false, nil
}

type mockUnhealthySearch struct {
	mockHealthySearch
}

func (m *mockUnhealthySearch) IsHealthy() bool { return false }

// Tests

func TestHealthService_GetBasicHealth(t *testing.T) {
	// Given: A health service
	service := NewHealthService(&mockHealthyStorage{}, &mockHealthySearch{})

	// When: Getting basic health
	result := service.GetBasicHealth()

	// Then: Returns health response
	assert.NotNil(t, result)
	assert.Equal(t, "healthy", result.Status)
	assert.Equal(t, "1.0.0", result.Version)
	assert.Equal(t, "motion-index-fiber", result.Service)
}

func TestHealthService_GetDetailedStatus_AllHealthy(t *testing.T) {
	// Given: A service with healthy components
	service := NewHealthService(&mockHealthyStorage{}, &mockHealthySearch{})

	// When: Getting detailed status
	result := service.GetDetailedStatus()

	// Then: Status is healthy
	assert.NotNil(t, result)
	assert.Equal(t, "healthy", result.Status)
	assert.Equal(t, "healthy", result.Storage.Status)
	assert.Equal(t, "healthy", result.Indexer.Status)
}

func TestHealthService_GetDetailedStatus_StorageUnhealthy(t *testing.T) {
	// Given: A service with unhealthy storage
	service := NewHealthService(&mockUnhealthyStorage{}, &mockHealthySearch{})

	// When: Getting detailed status
	result := service.GetDetailedStatus()

	// Then: Status is degraded
	assert.NotNil(t, result)
	assert.Equal(t, "degraded", result.Status)
	assert.Equal(t, "unhealthy", result.Storage.Status)
	assert.Contains(t, result.Storage.Error, "storage service is not healthy")
}

func TestHealthService_GetDetailedStatus_SearchUnhealthy(t *testing.T) {
	// Given: A service with unhealthy search
	service := NewHealthService(&mockHealthyStorage{}, &mockUnhealthySearch{})

	// When: Getting detailed status
	result := service.GetDetailedStatus()

	// Then: Status is degraded
	assert.NotNil(t, result)
	assert.Equal(t, "degraded", result.Status)
	assert.Equal(t, "unhealthy", result.Indexer.Status)
	assert.Contains(t, result.Indexer.Error, "search service is not healthy")
}

func TestHealthService_GetDetailedStatus_NilServices(t *testing.T) {
	// Given: A service with nil services
	service := NewHealthService(nil, nil)

	// When: Getting detailed status
	result := service.GetDetailedStatus()

	// Then: Status is degraded with initialization errors
	assert.NotNil(t, result)
	assert.Equal(t, "degraded", result.Status)
	assert.Equal(t, "unhealthy", result.Storage.Status)
	assert.Contains(t, result.Storage.Error, "not initialized")
	assert.Equal(t, "unhealthy", result.Indexer.Status)
	assert.Contains(t, result.Indexer.Error, "not initialized")
}

func TestHealthService_CheckReadiness_Ready(t *testing.T) {
	// Given: A service with all healthy components
	service := NewHealthService(&mockHealthyStorage{}, &mockHealthySearch{})

	// When: Checking readiness
	result := service.CheckReadiness()

	// Then: System is ready
	assert.NotNil(t, result)
	assert.True(t, result.Ready)
	assert.True(t, result.Checks["storage"])
	assert.True(t, result.Checks["search"])
}

func TestHealthService_CheckReadiness_NotReady(t *testing.T) {
	// Given: A service with unhealthy components
	service := NewHealthService(&mockUnhealthyStorage{}, &mockUnhealthySearch{})

	// When: Checking readiness
	result := service.CheckReadiness()

	// Then: System is not ready
	assert.NotNil(t, result)
	assert.False(t, result.Ready)
	assert.False(t, result.Checks["storage"])
	assert.False(t, result.Checks["search"])
}

func TestHealthService_CheckLiveness(t *testing.T) {
	// Given: A health service
	service := NewHealthService(&mockHealthyStorage{}, &mockHealthySearch{})

	// When: Checking liveness
	result := service.CheckLiveness()

	// Then: Always returns alive
	assert.NotNil(t, result)
	assert.True(t, result.Alive)
	assert.Greater(t, result.PID, 0)
}

func TestHealthService_CollectMetrics(t *testing.T) {
	// Given: A service with healthy components
	service := NewHealthService(&mockHealthyStorage{}, &mockHealthySearch{})

	// When: Collecting metrics
	result := service.CollectMetrics()

	// Then: Returns metrics
	assert.NotNil(t, result)
	assert.NotNil(t, result.Memory)
	assert.Greater(t, result.Goroutines, 0)
	assert.NotNil(t, result.GC)
	assert.NotNil(t, result.Storage)
	assert.NotNil(t, result.Indexer)
}

func TestHealthService_CollectMetrics_WithNilServices(t *testing.T) {
	// Given: A service with nil services
	service := NewHealthService(nil, nil)

	// When: Collecting metrics
	result := service.CollectMetrics()

	// Then: Returns metrics with empty service metrics maps
	assert.NotNil(t, result)
	assert.NotNil(t, result.Memory)
	assert.NotNil(t, result.Storage)
	assert.NotNil(t, result.Indexer)
	assert.Empty(t, result.Storage)   // Empty when storage is nil
	assert.Empty(t, result.Indexer)   // Empty when search is nil
}

func TestNewHealthService(t *testing.T) {
	// Given: Services
	storageService := &mockHealthyStorage{}
	searchService := &mockHealthySearch{}

	// When: Creating a new health service
	service := NewHealthService(storageService, searchService)

	// Then: Service is properly initialized
	assert.NotNil(t, service)
	assert.NotNil(t, service.storage)
	assert.NotNil(t, service.searchSvc)
	assert.False(t, service.startTime.IsZero())
}
