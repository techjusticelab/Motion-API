package spaces

import (
	"context"
	"fmt"
	"io"
	"motion-index-fiber/pkg/cloud/digitalocean/config"
	"motion-index-fiber/pkg/storage"
	"time"
)

type SpacesClient struct {
	config      *config.Config
	doAPIClient DOAPIClient
	s3Client    S3Client
	bucket      string
	cdnInfo     *CDNInfo

	// Performance and reliability settings
	maxConcurrentUploads   int
	maxConcurrentDownloads int
	retryConfig            *RetryConfig

	// CDN health and failover state
	cdnHealthState *CDNHealthState

	// Metrics and monitoring
	metrics *SpacesMetrics
}

type RetryConfig struct {
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

type SpacesMetrics struct {
	UploadCount          int64         `json:"upload_count"`
	DownloadCount        int64         `json:"download_count"`
	DeleteCount          int64         `json:"delete_count"`
	ErrorCount           int64         `json:"error_count"`
	TotalBytesUploaded   int64         `json:"total_bytes_uploaded"`
	TotalBytesDownloaded int64         `json:"total_bytes_downloaded"`
	AvgUploadDuration    time.Duration `json:"avg_upload_duration"`
	AvgDownloadDuration  time.Duration `json:"avg_download_duration"`
	LastHealthCheck      time.Time     `json:"last_health_check"`
	IsHealthy            bool          `json:"is_healthy"`
	CDNHitRate           float64       `json:"cdn_hit_rate"`
}

type CDNHealthState struct {
	IsHealthy              bool          `json:"is_healthy"`
	LastHealthCheck        time.Time     `json:"last_health_check"`
	LastFailure            time.Time     `json:"last_failure"`
	ConsecutiveFailures    int           `json:"consecutive_failures"`
	CircuitBreakerOpen     bool          `json:"circuit_breaker_open"`
	HealthCheckInterval    time.Duration `json:"health_check_interval"`
	MaxConsecutiveFailures int           `json:"max_consecutive_failures"`
	CircuitBreakerTimeout  time.Duration `json:"circuit_breaker_timeout"`
}

func NewSpacesClient(cfg *config.Config) (*SpacesClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Validate Spaces configuration
	if err := validateSpacesConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid Spaces configuration: %w", err)
	}

	// Create direct DigitalOcean API client
	doAPIClient := NewDOAPIClient(cfg.DigitalOcean.APIToken)

	// Create S3 client configuration
	s3Config := &S3Config{
		AccessKey:      cfg.DigitalOcean.Spaces.AccessKey,
		SecretKey:      cfg.DigitalOcean.Spaces.SecretKey,
		Endpoint:       cfg.GetSpacesEndpoint(),
		Region:         cfg.DigitalOcean.Spaces.Region,
		Bucket:         cfg.DigitalOcean.Spaces.Bucket,
		UseSSL:         true, // Always use SSL for DigitalOcean Spaces
		ForcePathStyle: true, // Use path-style for better AWS SDK compatibility
	}

	// Create S3 client
	s3Client, err := NewS3Client(s3Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Create Spaces client
	client := &SpacesClient{
		config:      cfg,
		doAPIClient: doAPIClient,
		s3Client:    s3Client,
		bucket:      cfg.DigitalOcean.Spaces.Bucket,

		// Performance settings from config
		maxConcurrentUploads:   cfg.Performance.MaxConcurrentUploads,
		maxConcurrentDownloads: cfg.Performance.MaxConcurrentDownloads,

		// Default retry configuration
		retryConfig: &RetryConfig{
			MaxRetries:    cfg.Health.MaxRetries,
			InitialDelay:  time.Duration(cfg.Health.TimeoutSeconds) * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		},

		// Initialize CDN health state
		cdnHealthState: &CDNHealthState{
			IsHealthy:              true, // Assume healthy until proven otherwise
			LastHealthCheck:        time.Time{},
			LastFailure:            time.Time{},
			ConsecutiveFailures:    0,
			CircuitBreakerOpen:     false,
			HealthCheckInterval:    5 * time.Minute,  // Check every 5 minutes
			MaxConsecutiveFailures: 3,                // Open circuit after 3 failures
			CircuitBreakerTimeout:  30 * time.Second, // Try again after 30 seconds
		},

		// Initialize metrics
		metrics: &SpacesMetrics{
			LastHealthCheck: time.Time{},
			IsHealthy:       false,
		},
	}

	// Initialize CDN information if available
	if err := client.initializeCDN(context.Background()); err != nil {
		// Log error but don't fail - CDN is optional for basic operations
		// TODO: Add proper logging here
	}

	return client, nil
}

func (c *SpacesClient) Upload(ctx context.Context, path string, content io.Reader, metadata *storage.UploadMetadata) (*storage.UploadResult, error) {
	startTime := time.Now()

	// Validate inputs
	if path == "" {
		return nil, storage.NewStorageError("validation", "path cannot be empty", path, nil)
	}
	if content == nil {
		return nil, storage.NewStorageError("validation", "content cannot be nil", path, nil)
	}

	// Sanitize path
	path = sanitizePath(path)

	// Perform upload using S3 client
	result, err := c.s3Client.Upload(ctx, c.bucket, path, content, metadata)

	// Update metrics
	c.updateUploadMetrics(startTime, result, err)

	if err != nil {
		return nil, storage.NewStorageError("upload", "failed to upload to Spaces", path, err)
	}

	// Optimize URL for CDN if available
	if c.cdnInfo != nil {
		result.URL = c.getCDNURL(path)
	}

	return result, nil
}

func (c *SpacesClient) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	startTime := time.Now()

	// Validate inputs
	if path == "" {
		return nil, storage.NewStorageError("validation", "path cannot be empty", path, nil)
	}

	// Sanitize path
	path = sanitizePath(path)

	// Perform download using S3 client
	reader, err := c.s3Client.Download(ctx, c.bucket, path)

	// Update metrics
	c.updateDownloadMetrics(startTime, err)

	if err != nil {
		return nil, storage.NewStorageError("download", "failed to download from Spaces", path, err)
	}

	return reader, nil
}

func (c *SpacesClient) Delete(ctx context.Context, path string) error {
	// Validate inputs
	if path == "" {
		return storage.NewStorageError("validation", "path cannot be empty", path, nil)
	}

	// Sanitize path
	path = sanitizePath(path)

	// Perform delete using S3 client
	err := c.s3Client.Delete(ctx, c.bucket, path)

	// Update metrics
	c.metrics.DeleteCount++
	if err != nil {
		c.metrics.ErrorCount++
		return storage.NewStorageError("delete", "failed to delete from Spaces", path, err)
	}

	// Invalidate CDN cache if available
	if c.cdnInfo != nil {
		if cacheErr := c.invalidateCDNCache(ctx, []string{path}); cacheErr != nil {
			// Log error but don't fail the delete operation
			// TODO: Add proper logging here
		}
	}

	return nil
}

func (c *SpacesClient) GetURL(path string) string {
	path = sanitizePath(path)

	// Use CDN URL if available and healthy
	if c.cdnInfo != nil && c.isCDNHealthy() {
		return c.getCDNURL(path)
	}

	// Fall back to direct Spaces URL if CDN is unavailable
	return c.s3Client.GetPublicURL(c.bucket, path, true)
}

func (c *SpacesClient) GetSignedURL(path string, expiration time.Duration) (string, error) {
	if path == "" {
		return "", storage.NewStorageError("validation", "path cannot be empty", path, nil)
	}

	path = sanitizePath(path)

	ctx := context.Background()
	url, err := c.s3Client.GetSignedURL(ctx, c.bucket, path, expiration)
	if err != nil {
		return "", storage.NewStorageError("signed_url", "failed to generate signed URL", path, err)
	}

	return url, nil
}

func (c *SpacesClient) Exists(ctx context.Context, path string) (bool, error) {
	if path == "" {
		return false, storage.NewStorageError("validation", "path cannot be empty", path, nil)
	}

	path = sanitizePath(path)

	exists, err := c.s3Client.Exists(ctx, c.bucket, path)
	if err != nil {
		return false, storage.NewStorageError("exists", "failed to check existence in Spaces", path, err)
	}

	return exists, nil
}

func (c *SpacesClient) List(ctx context.Context, prefix string) ([]*storage.StorageObject, error) {
	prefix = sanitizePath(prefix)

	// Use reasonable default for max keys
	maxKeys := 1000

	objects, err := c.s3Client.List(ctx, c.bucket, prefix, maxKeys)
	if err != nil {
		return nil, storage.NewStorageError("list", "failed to list objects in Spaces", prefix, err)
	}

	return objects, nil
}

func (c *SpacesClient) IsHealthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.config.Health.TimeoutSeconds)*time.Second)
	defer cancel()

	// Check S3 client health
	s3Healthy := c.s3Client.IsHealthy(ctx)

	// Update metrics
	c.metrics.LastHealthCheck = time.Now()
	c.metrics.IsHealthy = s3Healthy

	return s3Healthy
}

func (c *SpacesClient) GetMetrics() map[string]interface{} {
	metrics := map[string]interface{}{
		"upload_count":             c.metrics.UploadCount,
		"download_count":           c.metrics.DownloadCount,
		"delete_count":             c.metrics.DeleteCount,
		"error_count":              c.metrics.ErrorCount,
		"total_bytes_uploaded":     c.metrics.TotalBytesUploaded,
		"total_bytes_downloaded":   c.metrics.TotalBytesDownloaded,
		"avg_upload_duration_ms":   c.metrics.AvgUploadDuration.Milliseconds(),
		"avg_download_duration_ms": c.metrics.AvgDownloadDuration.Milliseconds(),
		"last_health_check":        c.metrics.LastHealthCheck,
		"is_healthy":               c.metrics.IsHealthy,
		"cdn_hit_rate":             c.metrics.CDNHitRate,
		"bucket":                   c.bucket,
		"region":                   c.config.DigitalOcean.Spaces.Region,
		"cdn_enabled":              c.cdnInfo != nil,
	}

	// Add CDN health metrics
	cdnHealth := c.GetCDNHealthStatus()
	for key, value := range cdnHealth {
		metrics["cdn_"+key] = value
	}

	return metrics
}

func (c *SpacesClient) initializeCDN(ctx context.Context) error {
	// List CDNs to find the one for our bucket
	cdns, err := c.doAPIClient.ListCDNs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list CDNs: %w", err)
	}

	// Find CDN for our bucket
	expectedOrigin := fmt.Sprintf("%s.%s.digitaloceanspaces.com", c.bucket, c.config.DigitalOcean.Spaces.Region)
	for _, cdn := range cdns {
		if cdn.Origin == expectedOrigin {
			c.cdnInfo = cdn
			return nil
		}
	}

	// No CDN found - this is okay, we'll use direct URLs
	return nil
}

func (c *SpacesClient) getCDNURL(path string) string {
	if c.cdnInfo == nil {
		return c.s3Client.GetPublicURL(c.bucket, path, true)
	}

	// Apply performance optimizations to the URL
	optimizedURL := c.optimizeCDNURL(path)
	return optimizedURL
}

func (c *SpacesClient) optimizeCDNURL(path string) string {
	baseURL := fmt.Sprintf("https://%s/%s", c.cdnInfo.Endpoint, path)

	// Add query parameters for optimization based on file type
	fileExt := getFileExtension(path)
	params := c.getCDNOptimizationParams(fileExt)

	if len(params) > 0 {
		baseURL += "?" + params
	}

	return baseURL
}
