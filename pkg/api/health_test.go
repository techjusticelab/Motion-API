package api

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/storage"
)

func TestHealthService_GetHealthNoDetails(t *testing.T) {
	svc := NewHealthService(nil, nil)

	resp, err := svc.GetHealth(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, "ok", resp.Status)
	require.Nil(t, resp.Services)
	require.Nil(t, resp.System)
}

func TestHealthService_GetHealthWithHealthyDependencies(t *testing.T) {
	searchStub := &stubSearchService{
		healthy: true,
		status: &search.HealthStatus{
			Status:        "green",
			ClusterName:   "motion-cluster",
			NumberOfNodes: 3,
			ActiveShards:  5,
			IndexExists:   true,
			IndexHealth:   "green",
		},
	}

	storageStub := &stubStorageService{healthy: true}

	svc := NewHealthService(storageStub, searchStub)

	resp, err := svc.GetHealth(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, "ok", resp.Status)
	require.NotNil(t, resp.Services)

	searchInfo, ok := resp.Services["opensearch"]
	require.True(t, ok)
	require.Equal(t, "ok", searchInfo.Status)
	require.NotZero(t, searchInfo.ResponseTime)

	storageInfo, ok := resp.Services["storage"]
	require.True(t, ok)
	require.Equal(t, "ok", storageInfo.Status)
	require.NotZero(t, storageInfo.ResponseTime)

	require.NotNil(t, resp.System)
	require.NotZero(t, resp.System.MemoryTotal)
}

func TestHealthService_GetHealthDetectsDegradedDependencies(t *testing.T) {
	searchStub := &stubSearchService{
		healthy: true,
		status: &search.HealthStatus{
			Status:        "yellow",
			ClusterName:   "motion-cluster",
			NumberOfNodes: 3,
			ActiveShards:  5,
			IndexExists:   true,
			IndexHealth:   "yellow",
		},
	}

	storageStub := &stubStorageService{healthy: false}

	svc := NewHealthService(storageStub, searchStub)

	resp, err := svc.GetHealth(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, "degraded", resp.Status)

	searchInfo := resp.Services["opensearch"]
	require.Equal(t, "degraded", searchInfo.Status)

	storageInfo := resp.Services["storage"]
	require.Equal(t, "error", storageInfo.Status)
}

type stubSearchService struct {
	healthy bool
	status  *search.HealthStatus
	err     error
}

func (s *stubSearchService) SearchDocuments(context.Context, *models.SearchRequest) (*models.SearchResult, error) {
	panic("not implemented")
}

func (s *stubSearchService) IndexDocument(context.Context, *models.Document) (string, error) {
	panic("not implemented")
}

func (s *stubSearchService) BulkIndexDocuments(context.Context, []*models.Document) (*models.BulkResult, error) {
	panic("not implemented")
}

func (s *stubSearchService) UpdateDocumentMetadata(context.Context, string, map[string]interface{}) error {
	panic("not implemented")
}

func (s *stubSearchService) DeleteDocument(context.Context, string) error {
	panic("not implemented")
}

func (s *stubSearchService) GetDocument(context.Context, string) (*models.Document, error) {
	panic("not implemented")
}

func (s *stubSearchService) DocumentExists(context.Context, string) (bool, error) {
	panic("not implemented")
}

func (s *stubSearchService) GetLegalTags(context.Context) ([]*models.TagCount, error) {
	panic("not implemented")
}

func (s *stubSearchService) GetDocumentTypes(context.Context) ([]*models.TypeCount, error) {
	panic("not implemented")
}

func (s *stubSearchService) GetMetadataFieldValues(context.Context, string, string, int) ([]*models.FieldValue, error) {
	panic("not implemented")
}

func (s *stubSearchService) GetMetadataFieldValuesWithFilters(context.Context, *models.MetadataFieldValuesRequest) ([]*models.FieldValue, error) {
	panic("not implemented")
}

func (s *stubSearchService) GetDocumentStats(context.Context) (*models.DocumentStats, error) {
	panic("not implemented")
}

func (s *stubSearchService) GetAllFieldOptions(context.Context) (*models.FieldOptions, error) {
	panic("not implemented")
}

func (s *stubSearchService) IsHealthy() bool {
	return s.healthy
}

func (s *stubSearchService) Health(context.Context) (*search.HealthStatus, error) {
	return s.status, s.err
}

type stubStorageService struct {
	healthy bool
}

func (s *stubStorageService) Upload(context.Context, string, io.Reader, *storage.UploadMetadata) (*storage.UploadResult, error) {
	panic("not implemented")
}

func (s *stubStorageService) Download(context.Context, string) (io.ReadCloser, error) {
	panic("not implemented")
}

func (s *stubStorageService) Delete(context.Context, string) error {
	panic("not implemented")
}

func (s *stubStorageService) GetURL(string) string {
	return ""
}

func (s *stubStorageService) GetSignedURL(string, time.Duration) (string, error) {
	return "", nil
}

func (s *stubStorageService) Exists(context.Context, string) (bool, error) {
	panic("not implemented")
}

func (s *stubStorageService) List(context.Context, string) ([]*storage.StorageObject, error) {
	panic("not implemented")
}

func (s *stubStorageService) IsHealthy() bool {
	return s.healthy
}

func (s *stubStorageService) GetMetrics() map[string]interface{} {
	return nil
}
