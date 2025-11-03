package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"motion-index-fiber/internal/infrastructure/http/presenter"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/storage"
)

// mockStorageService is a mock implementation of storage.Service
type mockStorageService struct {
	mock.Mock
}

func (m *mockStorageService) Upload(ctx context.Context, key string, reader io.Reader, metadata *storage.UploadMetadata) (*storage.UploadResult, error) {
	args := m.Called(ctx, key, reader, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.UploadResult), args.Error(1)
}

func (m *mockStorageService) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *mockStorageService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockStorageService) GetURL(key string) string {
	args := m.Called(key)
	return args.String(0)
}

func (m *mockStorageService) IsHealthy() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockStorageService) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *mockStorageService) GetMetrics() map[string]interface{} {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[string]interface{})
}

func (m *mockStorageService) GetSignedURL(key string, expiresIn time.Duration) (string, error) {
	args := m.Called(key, expiresIn)
	return args.String(0), args.Error(1)
}

func (m *mockStorageService) List(ctx context.Context, prefix string) ([]*storage.StorageObject, error) {
	args := m.Called(ctx, prefix)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.StorageObject), args.Error(1)
}

// mockSearchService is a mock implementation of search.Service
type mockSearchService struct {
	mock.Mock
}

func (m *mockSearchService) IsHealthy() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockSearchService) Health(ctx context.Context) (*search.HealthStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*search.HealthStatus), args.Error(1)
}

// SearchService methods (stubs for interface compliance)
func (m *mockSearchService) SearchDocuments(ctx context.Context, req *models.SearchRequest) (*models.SearchResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SearchResult), args.Error(1)
}

func (m *mockSearchService) IndexDocument(ctx context.Context, doc *models.Document) (string, error) {
	args := m.Called(ctx, doc)
	return args.String(0), args.Error(1)
}

func (m *mockSearchService) BulkIndexDocuments(ctx context.Context, docs []*models.Document) (*models.BulkResult, error) {
	args := m.Called(ctx, docs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BulkResult), args.Error(1)
}

func (m *mockSearchService) UpdateDocumentMetadata(ctx context.Context, docID string, metadata map[string]interface{}) error {
	args := m.Called(ctx, docID, metadata)
	return args.Error(0)
}

func (m *mockSearchService) DeleteDocument(ctx context.Context, docID string) error {
	args := m.Called(ctx, docID)
	return args.Error(0)
}

func (m *mockSearchService) GetDocument(ctx context.Context, docID string) (*models.Document, error) {
	args := m.Called(ctx, docID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Document), args.Error(1)
}

func (m *mockSearchService) DocumentExists(ctx context.Context, docID string) (bool, error) {
	args := m.Called(ctx, docID)
	return args.Bool(0), args.Error(1)
}

// AggregationService methods (stubs for interface compliance)
func (m *mockSearchService) GetLegalTags(ctx context.Context) ([]*models.TagCount, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.TagCount), args.Error(1)
}

func (m *mockSearchService) GetDocumentTypes(ctx context.Context) ([]*models.TypeCount, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.TypeCount), args.Error(1)
}

func (m *mockSearchService) GetMetadataFieldValues(ctx context.Context, field string, prefix string, size int) ([]*models.FieldValue, error) {
	args := m.Called(ctx, field, prefix, size)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FieldValue), args.Error(1)
}

func (m *mockSearchService) GetMetadataFieldValuesWithFilters(ctx context.Context, req *models.MetadataFieldValuesRequest) ([]*models.FieldValue, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FieldValue), args.Error(1)
}

func (m *mockSearchService) GetDocumentStats(ctx context.Context) (*models.DocumentStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DocumentStats), args.Error(1)
}

func (m *mockSearchService) GetAllFieldOptions(ctx context.Context) (*models.FieldOptions, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FieldOptions), args.Error(1)
}

func TestNewHealthHandler(t *testing.T) {
	// Given: Mock services
	mockStorage := new(mockStorageService)
	mockSearch := new(mockSearchService)

	// When: Handler is created
	handler := NewHealthHandler(mockStorage, mockSearch)

	// Then: Handler is properly initialized
	assert.NotNil(t, handler)
	assert.Equal(t, mockStorage, handler.storage)
	assert.Equal(t, mockSearch, handler.searchSvc)
	assert.False(t, handler.startTime.IsZero())
}

func TestHealthHandler_Health(t *testing.T) {
	// Given: Health handler
	handler := NewHealthHandler(nil, nil)
	app := fiber.New()
	app.Get("/health", handler.Health)

	// When: Request is made
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Basic health response is returned
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	data := response.Data.(map[string]interface{})
	assert.Equal(t, "healthy", data["status"])
	assert.Equal(t, "motion-index-fiber", data["service"])
	assert.Equal(t, "1.0.0", data["version"])
}

func TestHealthHandler_DetailedStatus_AllHealthy(t *testing.T) {
	// Given: Healthy services
	mockStorage := new(mockStorageService)
	mockStorage.On("IsHealthy").Return(true)
	mockSearch := new(mockSearchService)
	mockSearch.On("IsHealthy").Return(true)

	handler := NewHealthHandler(mockStorage, mockSearch)
	app := fiber.New()
	app.Get("/status", handler.DetailedStatus)

	// When: Request is made
	req := httptest.NewRequest("GET", "/status", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Detailed status shows healthy
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	data := response.Data.(map[string]interface{})
	assert.Equal(t, "healthy", data["status"])
	assert.NotNil(t, data["storage"])
	assert.NotNil(t, data["search"])
	assert.NotNil(t, data["system"])

	mockStorage.AssertExpectations(t)
	mockSearch.AssertExpectations(t)
}

func TestHealthHandler_DetailedStatus_Degraded(t *testing.T) {
	// Given: Unhealthy storage
	mockStorage := new(mockStorageService)
	mockStorage.On("IsHealthy").Return(false)
	mockSearch := new(mockSearchService)
	mockSearch.On("IsHealthy").Return(true)

	handler := NewHealthHandler(mockStorage, mockSearch)
	app := fiber.New()
	app.Get("/status", handler.DetailedStatus)

	// When: Request is made
	req := httptest.NewRequest("GET", "/status", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Detailed status shows degraded
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success) // Still success response format, but 503 status
	data := response.Data.(map[string]interface{})
	assert.Equal(t, "degraded", data["status"])

	mockStorage.AssertExpectations(t)
	mockSearch.AssertExpectations(t)
}

func TestHealthHandler_ReadinessCheck_Ready(t *testing.T) {
	// Given: Ready services
	mockStorage := new(mockStorageService)
	mockStorage.On("IsHealthy").Return(true)
	mockSearch := new(mockSearchService)
	mockSearch.On("IsHealthy").Return(true)

	handler := NewHealthHandler(mockStorage, mockSearch)
	app := fiber.New()
	app.Get("/readiness", handler.ReadinessCheck)

	// When: Request is made
	req := httptest.NewRequest("GET", "/readiness", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Readiness check passes
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	data := response.Data.(map[string]interface{})
	assert.True(t, data["ready"].(bool))

	mockStorage.AssertExpectations(t)
	mockSearch.AssertExpectations(t)
}

func TestHealthHandler_ReadinessCheck_NotReady(t *testing.T) {
	// Given: Storage not ready
	mockStorage := new(mockStorageService)
	mockStorage.On("IsHealthy").Return(false)
	mockSearch := new(mockSearchService)
	mockSearch.On("IsHealthy").Return(true)

	handler := NewHealthHandler(mockStorage, mockSearch)
	app := fiber.New()
	app.Get("/readiness", handler.ReadinessCheck)

	// When: Request is made
	req := httptest.NewRequest("GET", "/readiness", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Readiness check fails
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success) // Still success response format, but 503 status
	data := response.Data.(map[string]interface{})
	assert.False(t, data["ready"].(bool))

	mockStorage.AssertExpectations(t)
	mockSearch.AssertExpectations(t)
}

func TestHealthHandler_LivenessCheck(t *testing.T) {
	// Given: Health handler
	handler := NewHealthHandler(nil, nil)
	app := fiber.New()
	app.Get("/liveness", handler.LivenessCheck)

	// When: Request is made
	req := httptest.NewRequest("GET", "/liveness", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Liveness check passes
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	data := response.Data.(map[string]interface{})
	assert.True(t, data["alive"].(bool))
	assert.Equal(t, float64(os.Getpid()), data["pid"].(float64))
}

func TestHealthHandler_SystemInfo(t *testing.T) {
	// Given: Health handler
	handler := NewHealthHandler(nil, nil)

	// When: System info is retrieved
	info := handler.systemInfo()

	// Then: System info is populated
	assert.Equal(t, runtime.GOOS, info["os"])
	assert.Equal(t, runtime.GOARCH, info["architecture"])
	assert.Equal(t, runtime.Version(), info["go_version"])
	assert.Equal(t, runtime.NumCPU(), info["num_cpu"])
	assert.Greater(t, info["goroutines"].(int), 0)
	assert.NotNil(t, info["memory"])
}
