package spaces

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"motion-index-fiber/pkg/cloud/digitalocean/config"
	"motion-index-fiber/pkg/storage"
)

// Mock implementations for testing

type MockDOAPIClient struct {
	mock.Mock
}

func (m *MockDOAPIClient) CreateCDN(ctx context.Context, origin string, ttl int) (*CDNInfo, error) {
	args := m.Called(ctx, origin, ttl)
	return args.Get(0).(*CDNInfo), args.Error(1)
}

func (m *MockDOAPIClient) GetCDN(ctx context.Context, cdnID string) (*CDNInfo, error) {
	args := m.Called(ctx, cdnID)
	return args.Get(0).(*CDNInfo), args.Error(1)
}

func (m *MockDOAPIClient) ListCDNs(ctx context.Context) ([]*CDNInfo, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*CDNInfo), args.Error(1)
}

func (m *MockDOAPIClient) DeleteCDN(ctx context.Context, cdnID string) error {
	args := m.Called(ctx, cdnID)
	return args.Error(0)
}

func (m *MockDOAPIClient) FlushCDNCache(ctx context.Context, cdnID string, files []string) error {
	args := m.Called(ctx, cdnID, files)
	return args.Error(0)
}

func (m *MockDOAPIClient) CreateSpacesKey(ctx context.Context, name string) (*SpacesKey, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*SpacesKey), args.Error(1)
}

func (m *MockDOAPIClient) GetSpacesKey(ctx context.Context, accessKey string) (*SpacesKey, error) {
	args := m.Called(ctx, accessKey)
	return args.Get(0).(*SpacesKey), args.Error(1)
}

func (m *MockDOAPIClient) ListSpacesKeys(ctx context.Context) ([]*SpacesKey, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*SpacesKey), args.Error(1)
}

func (m *MockDOAPIClient) UpdateSpacesKey(ctx context.Context, accessKey, name string) (*SpacesKey, error) {
	args := m.Called(ctx, accessKey, name)
	return args.Get(0).(*SpacesKey), args.Error(1)
}

func (m *MockDOAPIClient) DeleteSpacesKey(ctx context.Context, accessKey string) error {
	args := m.Called(ctx, accessKey)
	return args.Error(0)
}

type MockS3Client struct {
	mock.Mock
}

func (m *MockS3Client) Upload(ctx context.Context, bucket, key string, content io.Reader, metadata *storage.UploadMetadata) (*storage.UploadResult, error) {
	args := m.Called(ctx, bucket, key, content, metadata)
	return args.Get(0).(*storage.UploadResult), args.Error(1)
}

func (m *MockS3Client) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	args := m.Called(ctx, bucket, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockS3Client) Delete(ctx context.Context, bucket, key string) error {
	args := m.Called(ctx, bucket, key)
	return args.Error(0)
}

func (m *MockS3Client) Exists(ctx context.Context, bucket, key string) (bool, error) {
	args := m.Called(ctx, bucket, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockS3Client) List(ctx context.Context, bucket, prefix string, maxKeys int) ([]*storage.StorageObject, error) {
	args := m.Called(ctx, bucket, prefix, maxKeys)
	return args.Get(0).([]*storage.StorageObject), args.Error(1)
}

func (m *MockS3Client) GetPublicURL(bucket, key string, useSSL bool) string {
	args := m.Called(bucket, key, useSSL)
	return args.String(0)
}

func (m *MockS3Client) GetSignedURL(ctx context.Context, bucket, key string, expiration time.Duration) (string, error) {
	args := m.Called(ctx, bucket, key, expiration)
	return args.String(0), args.Error(1)
}

func (m *MockS3Client) BatchUpload(ctx context.Context, bucket string, uploads []*BatchUploadItem) ([]*storage.UploadResult, error) {
	args := m.Called(ctx, bucket, uploads)
	return args.Get(0).([]*storage.UploadResult), args.Error(1)
}

func (m *MockS3Client) BatchDelete(ctx context.Context, bucket string, keys []string) error {
	args := m.Called(ctx, bucket, keys)
	return args.Error(0)
}

func (m *MockS3Client) IsHealthy(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockS3Client) GetConnectionInfo() *ConnectionInfo {
	args := m.Called()
	return args.Get(0).(*ConnectionInfo)
}

type MockReadCloser struct {
	*strings.Reader
}

func (m *MockReadCloser) Close() error {
	return nil
}

// Test helper functions

func createTestConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.Environment = config.EnvStaging
	cfg.DigitalOcean.Spaces.AccessKey = "test-access-key"
	cfg.DigitalOcean.Spaces.SecretKey = "test-secret-key"
	cfg.DigitalOcean.Spaces.Bucket = "test-bucket"
	cfg.DigitalOcean.Spaces.Region = "nyc3"
	cfg.DigitalOcean.OpenSearch.Port = 25060
	cfg.DigitalOcean.OpenSearch.Index = "documents"
	return cfg
}

func createSpacesClientWithMocks(t *testing.T) (*SpacesClient, *MockDOAPIClient, *MockS3Client) {
	cfg := createTestConfig()

	mockDOAPI := &MockDOAPIClient{}
	mockS3 := &MockS3Client{}

	client := &SpacesClient{
		config:      cfg,
		doAPIClient: mockDOAPI,
		s3Client:    mockS3,
		bucket:      cfg.DigitalOcean.Spaces.Bucket,

		maxConcurrentUploads:   cfg.Performance.MaxConcurrentUploads,
		maxConcurrentDownloads: cfg.Performance.MaxConcurrentDownloads,

		retryConfig: &RetryConfig{
			MaxRetries:    cfg.Health.MaxRetries,
			InitialDelay:  time.Duration(cfg.Health.TimeoutSeconds) * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		},

		// Initialize CDN health state for tests
		cdnHealthState: &CDNHealthState{
			IsHealthy:              true, // Assume healthy in tests
			LastHealthCheck:        time.Time{},
			LastFailure:            time.Time{},
			ConsecutiveFailures:    0,
			CircuitBreakerOpen:     false,
			HealthCheckInterval:    5 * time.Minute,
			MaxConsecutiveFailures: 3,
			CircuitBreakerTimeout:  30 * time.Second,
		},

		metrics: &SpacesMetrics{
			LastHealthCheck: time.Time{},
			IsHealthy:       false,
		},
	}

	return client, mockDOAPI, mockS3
}

// Unit Tests

func TestNewSpacesClient(t *testing.T) {
	t.Run("valid configuration creates client", func(t *testing.T) {
		cfg := createTestConfig()

		client, err := NewSpacesClient(cfg)

		// The client should be created successfully with placeholder implementations
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, cfg.DigitalOcean.Spaces.Bucket, client.bucket)
		assert.NotNil(t, client.metrics)
		assert.NotNil(t, client.retryConfig)
	})

	t.Run("nil configuration returns error", func(t *testing.T) {
		client, err := NewSpacesClient(nil)

		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "config cannot be nil")
	})

	t.Run("invalid configuration returns error", func(t *testing.T) {
		cfg := createTestConfig()
		cfg.DigitalOcean.Spaces.AccessKey = "" // Missing required field

		client, err := NewSpacesClient(cfg)

		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "invalid Spaces configuration")
	})
}

func TestSpacesClient_Upload(t *testing.T) {
	client, _, mockS3 := createSpacesClientWithMocks(t)
	ctx := context.Background()

	t.Run("successful upload", func(t *testing.T) {
		content := strings.NewReader("test content")
		metadata := &storage.UploadMetadata{
			ContentType: "text/plain",
			Size:        12,
			FileName:    "test.txt",
		}

		expectedResult := &storage.UploadResult{
			Path:       "documents/test.txt",
			URL:        "https://test-bucket.nyc3.digitaloceanspaces.com/documents/test.txt",
			Size:       12,
			Success:    true,
			UploadedAt: time.Now(),
		}

		mockS3.On("Upload", ctx, "test-bucket", "documents/test.txt", content, metadata).Return(expectedResult, nil)

		result, err := client.Upload(ctx, "documents/test.txt", content, metadata)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedResult.Path, result.Path)
		assert.Equal(t, expectedResult.Size, result.Size)
		assert.True(t, result.Success)

		mockS3.AssertExpectations(t)
	})

	t.Run("upload with CDN optimization", func(t *testing.T) {
		// Set up CDN info
		client.cdnInfo = &CDNInfo{
			ID:       "test-cdn-id",
			Origin:   "test-bucket.nyc3.digitaloceanspaces.com",
			Endpoint: "test-bucket.nyc3.cdn.digitaloceanspaces.com",
			TTL:      86400,
		}

		content := strings.NewReader("test content")
		metadata := &storage.UploadMetadata{
			ContentType: "text/plain",
			Size:        12,
			FileName:    "test.txt",
		}

		s3Result := &storage.UploadResult{
			Path:       "documents/test.txt",
			URL:        "https://test-bucket.nyc3.digitaloceanspaces.com/documents/test.txt",
			Size:       12,
			Success:    true,
			UploadedAt: time.Now(),
		}

		mockS3.On("Upload", ctx, "test-bucket", "documents/test.txt", content, metadata).Return(s3Result, nil)

		result, err := client.Upload(ctx, "documents/test.txt", content, metadata)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "https://test-bucket.nyc3.cdn.digitaloceanspaces.com/documents/test.txt?gzip=true&cache=max", result.URL)

		mockS3.AssertExpectations(t)
	})

	t.Run("upload with empty path returns error", func(t *testing.T) {
		content := strings.NewReader("test content")
		metadata := &storage.UploadMetadata{
			ContentType: "text/plain",
			Size:        12,
		}

		result, err := client.Upload(ctx, "", content, metadata)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "path cannot be empty")
	})

	t.Run("upload with nil content returns error", func(t *testing.T) {
		metadata := &storage.UploadMetadata{
			ContentType: "text/plain",
			Size:        12,
		}

		result, err := client.Upload(ctx, "test.txt", nil, metadata)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "content cannot be nil")
	})

	t.Run("S3 upload error propagates", func(t *testing.T) {
		content := strings.NewReader("test content")
		metadata := &storage.UploadMetadata{
			ContentType: "text/plain",
			Size:        12,
		}

		mockS3.On("Upload", ctx, "test-bucket", "test.txt", content, metadata).Return((*storage.UploadResult)(nil), fmt.Errorf("S3 error"))

		result, err := client.Upload(ctx, "test.txt", content, metadata)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to upload to Spaces")

		mockS3.AssertExpectations(t)
	})
}

func TestSpacesClient_Download(t *testing.T) {
	client, _, mockS3 := createSpacesClientWithMocks(t)
	ctx := context.Background()

	t.Run("successful download", func(t *testing.T) {
		expectedReader := &MockReadCloser{strings.NewReader("test content")}

		mockS3.On("Download", ctx, "test-bucket", "documents/test.txt").Return(expectedReader, nil)

		reader, err := client.Download(ctx, "documents/test.txt")

		assert.NoError(t, err)
		assert.NotNil(t, reader)
		assert.Equal(t, expectedReader, reader)

		mockS3.AssertExpectations(t)
	})

	t.Run("download with empty path returns error", func(t *testing.T) {
		reader, err := client.Download(ctx, "")

		assert.Error(t, err)
		assert.Nil(t, reader)
		assert.Contains(t, err.Error(), "path cannot be empty")
	})

	t.Run("S3 download error propagates", func(t *testing.T) {
		mockS3.On("Download", ctx, "test-bucket", "test.txt").Return(nil, fmt.Errorf("S3 error"))

		reader, err := client.Download(ctx, "test.txt")

		assert.Error(t, err)
		assert.Nil(t, reader)
		assert.Contains(t, err.Error(), "failed to download from Spaces")

		mockS3.AssertExpectations(t)
	})
}

func TestSpacesClient_Delete(t *testing.T) {
	client, mockMCP, mockS3 := createSpacesClientWithMocks(t)
	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		mockS3.On("Delete", ctx, "test-bucket", "documents/test.txt").Return(nil)

		err := client.Delete(ctx, "documents/test.txt")

		assert.NoError(t, err)
		mockS3.AssertExpectations(t)
	})

	t.Run("delete with CDN cache invalidation", func(t *testing.T) {
		// Set up CDN info
		client.cdnInfo = &CDNInfo{
			ID:       "test-cdn-id",
			Origin:   "test-bucket.nyc3.digitaloceanspaces.com",
			Endpoint: "test-bucket.nyc3.cdn.digitaloceanspaces.com",
			TTL:      86400,
		}

		mockS3.On("Delete", ctx, "test-bucket", "documents/test.txt").Return(nil)
		mockMCP.On("FlushCDNCache", ctx, "test-cdn-id", []string{"documents/test.txt"}).Return(nil)

		err := client.Delete(ctx, "documents/test.txt")

		assert.NoError(t, err)
		mockS3.AssertExpectations(t)
		mockMCP.AssertExpectations(t)
	})

	t.Run("delete with empty path returns error", func(t *testing.T) {
		err := client.Delete(ctx, "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path cannot be empty")
	})
}

func TestSpacesClient_GetURL(t *testing.T) {
	client, _, mockS3 := createSpacesClientWithMocks(t)

	t.Run("returns direct URL without CDN", func(t *testing.T) {
		expectedURL := "https://test-bucket.nyc3.digitaloceanspaces.com/documents/test.txt"

		mockS3.On("GetPublicURL", "test-bucket", "documents/test.txt", true).Return(expectedURL)

		url := client.GetURL("documents/test.txt")

		assert.Equal(t, expectedURL, url)
		mockS3.AssertExpectations(t)
	})

	t.Run("returns CDN URL when CDN is available", func(t *testing.T) {
		client, mockMCP, _ := createSpacesClientWithMocks(t)

		client.cdnInfo = &CDNInfo{
			ID:       "test-cdn-id",
			Endpoint: "test-bucket.nyc3.cdn.digitaloceanspaces.com",
		}

		// Mock the CDN health check to return success
		mockMCP.On("GetCDN", mock.AnythingOfType("*context.timerCtx"), "test-cdn-id").Return(client.cdnInfo, nil)

		expectedURL := "https://test-bucket.nyc3.cdn.digitaloceanspaces.com/documents/test.txt?gzip=true&cache=max"

		url := client.GetURL("documents/test.txt")

		assert.Equal(t, expectedURL, url)
		mockMCP.AssertExpectations(t)
	})
}

func TestSpacesClient_IsHealthy(t *testing.T) {
	t.Run("returns healthy when S3 is healthy", func(t *testing.T) {
		client, _, mockS3 := createSpacesClientWithMocks(t)
		mockS3.On("IsHealthy", mock.AnythingOfType("*context.timerCtx")).Return(true)

		healthy := client.IsHealthy()

		assert.True(t, healthy)
		assert.True(t, client.metrics.IsHealthy)
		assert.False(t, client.metrics.LastHealthCheck.IsZero())

		mockS3.AssertExpectations(t)
	})

	t.Run("returns unhealthy when S3 is unhealthy", func(t *testing.T) {
		client, _, mockS3 := createSpacesClientWithMocks(t)
		mockS3.On("IsHealthy", mock.AnythingOfType("*context.timerCtx")).Return(false)

		healthy := client.IsHealthy()

		assert.False(t, healthy)
		assert.False(t, client.metrics.IsHealthy)

		mockS3.AssertExpectations(t)
	})
}

func TestSpacesClient_GetMetrics(t *testing.T) {
	client, _, _ := createSpacesClientWithMocks(t)

	// Set some test metrics
	client.metrics.UploadCount = 10
	client.metrics.DownloadCount = 20
	client.metrics.DeleteCount = 5
	client.metrics.ErrorCount = 2
	client.metrics.TotalBytesUploaded = 1024000
	client.metrics.TotalBytesDownloaded = 2048000
	client.metrics.IsHealthy = true

	metrics := client.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, int64(10), metrics["upload_count"])
	assert.Equal(t, int64(20), metrics["download_count"])
	assert.Equal(t, int64(5), metrics["delete_count"])
	assert.Equal(t, int64(2), metrics["error_count"])
	assert.Equal(t, int64(1024000), metrics["total_bytes_uploaded"])
	assert.Equal(t, int64(2048000), metrics["total_bytes_downloaded"])
	assert.Equal(t, true, metrics["is_healthy"])
	assert.Equal(t, "test-bucket", metrics["bucket"])
	assert.Equal(t, "nyc3", metrics["region"])
	assert.Equal(t, false, metrics["cdn_enabled"])
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes leading slash",
			input:    "/documents/test.txt",
			expected: "documents/test.txt",
		},
		{
			name:     "no change for clean path",
			input:    "documents/test.txt",
			expected: "documents/test.txt",
		},
		{
			name:     "handles empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "handles just slash",
			input:    "/",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateSpacesConfig(t *testing.T) {
	t.Run("valid staging config passes", func(t *testing.T) {
		cfg := createTestConfig()
		err := validateSpacesConfig(cfg)
		assert.NoError(t, err)
	})

	t.Run("local config passes without credentials", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Environment = config.EnvLocal
		err := validateSpacesConfig(cfg)
		assert.NoError(t, err)
	})

	t.Run("staging config without access key fails", func(t *testing.T) {
		cfg := createTestConfig()
		cfg.DigitalOcean.Spaces.AccessKey = ""
		err := validateSpacesConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access key is required")
	})

	t.Run("staging config without secret key fails", func(t *testing.T) {
		cfg := createTestConfig()
		cfg.DigitalOcean.Spaces.SecretKey = ""
		err := validateSpacesConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secret key is required")
	})

	t.Run("staging config without bucket fails", func(t *testing.T) {
		cfg := createTestConfig()
		cfg.DigitalOcean.Spaces.Bucket = ""
		err := validateSpacesConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket is required")
	})

	t.Run("staging config without region fails", func(t *testing.T) {
		cfg := createTestConfig()
		cfg.DigitalOcean.Spaces.Region = ""
		err := validateSpacesConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "region is required")
	})
}
