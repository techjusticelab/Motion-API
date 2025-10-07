package spaces

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"motion-index-fiber/pkg/cloud/digitalocean/config"
	"motion-index-fiber/pkg/storage"
)

// SpacesClient implements storage.Service for DigitalOcean Spaces
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

// RetryConfig contains retry configuration for Spaces operations
type RetryConfig struct {
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// SpacesMetrics contains operational metrics for the Spaces client
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

// CDNHealthState tracks the health and availability of the CDN
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

// NewSpacesClient creates a new DigitalOcean Spaces client
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

// Storage Service Interface Implementation

// Upload uploads a document to DigitalOcean Spaces
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

// Download downloads a document from DigitalOcean Spaces
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

// Delete deletes a document from DigitalOcean Spaces
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

// GetURL returns a public URL for the document with CDN failover
func (c *SpacesClient) GetURL(path string) string {
	path = sanitizePath(path)

	// Use CDN URL if available and healthy
	if c.cdnInfo != nil && c.isCDNHealthy() {
		return c.getCDNURL(path)
	}

	// Fall back to direct Spaces URL if CDN is unavailable
	return c.s3Client.GetPublicURL(c.bucket, path, true)
}

// GetSignedURL returns a signed URL for temporary access
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

// Exists checks if a document exists in DigitalOcean Spaces
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

// List lists documents in a directory
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

// IsHealthy returns true if the storage service is healthy
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

// GetMetrics returns storage-specific metrics
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

// Private helper methods

// initializeCDN discovers and initializes CDN information
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

// getCDNURL returns the CDN-optimized URL for a path with performance optimizations
func (c *SpacesClient) getCDNURL(path string) string {
	if c.cdnInfo == nil {
		return c.s3Client.GetPublicURL(c.bucket, path, true)
	}

	// Apply performance optimizations to the URL
	optimizedURL := c.optimizeCDNURL(path)
	return optimizedURL
}

// optimizeCDNURL applies various performance optimizations to CDN URLs
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

// getCDNOptimizationParams returns URL parameters for CDN optimization based on file type
func (c *SpacesClient) getCDNOptimizationParams(fileExt string) string {
	var params []string

	switch fileExt {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		// Image optimization parameters
		params = append(params, "auto=compress")
		params = append(params, "fm=auto") // Auto format selection

	case ".pdf":
		// PDF optimization
		params = append(params, "compress=true")

	case ".js", ".css":
		// Script/CSS optimization
		params = append(params, "minify=true")
		params = append(params, "gzip=true")

	case ".mp4", ".avi", ".mov":
		// Video optimization
		params = append(params, "quality=auto")

	default:
		// General optimization for all files
		params = append(params, "gzip=true")
	}

	// Add cache optimization
	params = append(params, "cache=max")

	// Join parameters
	result := ""
	for i, param := range params {
		if i > 0 {
			result += "&"
		}
		result += param
	}

	return result
}

// invalidateCDNCache invalidates CDN cache for specified files
func (c *SpacesClient) invalidateCDNCache(ctx context.Context, files []string) error {
	if c.cdnInfo == nil {
		return nil // No CDN to invalidate
	}

	return c.doAPIClient.FlushCDNCache(ctx, c.cdnInfo.ID, files)
}

// updateUploadMetrics updates upload-related metrics
func (c *SpacesClient) updateUploadMetrics(startTime time.Time, result *storage.UploadResult, err error) {
	duration := time.Since(startTime)

	c.metrics.UploadCount++
	if err != nil {
		c.metrics.ErrorCount++
	} else if result != nil {
		c.metrics.TotalBytesUploaded += result.Size
	}

	// Update average duration (simple moving average)
	if c.metrics.AvgUploadDuration == 0 {
		c.metrics.AvgUploadDuration = duration
	} else {
		c.metrics.AvgUploadDuration = (c.metrics.AvgUploadDuration + duration) / 2
	}
}

// updateDownloadMetrics updates download-related metrics
func (c *SpacesClient) updateDownloadMetrics(startTime time.Time, err error) {
	duration := time.Since(startTime)

	c.metrics.DownloadCount++
	if err != nil {
		c.metrics.ErrorCount++
	}

	// Update average duration (simple moving average)
	if c.metrics.AvgDownloadDuration == 0 {
		c.metrics.AvgDownloadDuration = duration
	} else {
		c.metrics.AvgDownloadDuration = (c.metrics.AvgDownloadDuration + duration) / 2
	}
}

// sanitizePath ensures the path is properly formatted for S3/Spaces
func sanitizePath(path string) string {
	// Remove leading slashes
	path = strings.TrimPrefix(path, "/")

	// TODO: Add more path sanitization as needed
	// - Remove double slashes
	// - Handle special characters
	// - Validate path length

	return path
}

// validateSpacesConfig validates the Spaces configuration
func validateSpacesConfig(cfg *config.Config) error {
	spaces := cfg.DigitalOcean.Spaces

	if !cfg.IsLocal() {
		if spaces.AccessKey == "" {
			return fmt.Errorf("access key is required for non-local environments")
		}
		if spaces.SecretKey == "" {
			return fmt.Errorf("secret key is required for non-local environments")
		}
		if spaces.Bucket == "" {
			return fmt.Errorf("bucket is required for non-local environments")
		}
		if spaces.Region == "" {
			return fmt.Errorf("region is required for non-local environments")
		}
	}

	return nil
}

// Performance optimization helper functions

// getFileExtension extracts the file extension from a path
func getFileExtension(path string) string {
	lastDot := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			lastDot = i
			break
		}
		if path[i] == '/' {
			break // Stop at directory separator
		}
	}

	if lastDot == -1 {
		return ""
	}

	return strings.ToLower(path[lastDot:])
}

// GetOptimizedURL returns a URL optimized for the given parameters
func (c *SpacesClient) GetOptimizedURL(path string, opts *URLOptimizationOptions) string {
	if opts == nil {
		return c.GetURL(path)
	}

	baseURL := c.GetURL(path)

	// If it's not a CDN URL, return the base URL
	if c.cdnInfo == nil {
		return baseURL
	}

	// Apply additional optimizations based on options
	return c.applyURLOptimizations(baseURL, opts)
}

// URLOptimizationOptions contains options for URL optimization
type URLOptimizationOptions struct {
	Quality     string // auto, high, medium, low
	Format      string // auto, webp, jpg, png
	Resize      *ResizeOptions
	Compression bool
	UserAgent   string
	DeviceType  string // mobile, tablet, desktop
	Bandwidth   string // high, medium, low
}

// ResizeOptions contains image resizing options
type ResizeOptions struct {
	Width  int
	Height int
	Mode   string // fit, fill, crop
}

// applyURLOptimizations applies additional URL optimizations
func (c *SpacesClient) applyURLOptimizations(baseURL string, opts *URLOptimizationOptions) string {
	if opts == nil {
		return baseURL
	}

	// Parse existing URL to add new parameters
	params := make(map[string]string)

	// Quality optimization
	if opts.Quality != "" {
		params["q"] = opts.Quality
	}

	// Format optimization
	if opts.Format != "" {
		params["fm"] = opts.Format
	}

	// Resize optimization
	if opts.Resize != nil {
		if opts.Resize.Width > 0 {
			params["w"] = fmt.Sprintf("%d", opts.Resize.Width)
		}
		if opts.Resize.Height > 0 {
			params["h"] = fmt.Sprintf("%d", opts.Resize.Height)
		}
		if opts.Resize.Mode != "" {
			params["fit"] = opts.Resize.Mode
		}
	}

	// Device-specific optimization
	if opts.DeviceType != "" {
		switch opts.DeviceType {
		case "mobile":
			params["dpr"] = "2" // Device pixel ratio
			params["auto"] = "compress,format"
		case "tablet":
			params["dpr"] = "2"
		case "desktop":
			params["dpr"] = "1"
		}
	}

	// Bandwidth optimization
	if opts.Bandwidth != "" {
		switch opts.Bandwidth {
		case "low":
			params["q"] = "60"
			params["compress"] = "true"
		case "medium":
			params["q"] = "80"
		case "high":
			params["q"] = "95"
		}
	}

	// Compression
	if opts.Compression {
		params["compress"] = "true"
	}

	// Build final URL with parameters
	if len(params) == 0 {
		return baseURL
	}

	// Check if URL already has parameters
	separator := "?"
	if strings.Contains(baseURL, "?") {
		separator = "&"
	}

	var paramStrings []string
	for key, value := range params {
		paramStrings = append(paramStrings, fmt.Sprintf("%s=%s", key, value))
	}

	return baseURL + separator + strings.Join(paramStrings, "&")
}

// CDN Health and Failover Implementation

// isCDNHealthy returns true if the CDN is available and healthy
func (c *SpacesClient) isCDNHealthy() bool {
	if c.cdnInfo == nil || c.cdnHealthState == nil {
		return false
	}

	// Check circuit breaker state
	if c.cdnHealthState.CircuitBreakerOpen {
		// Check if enough time has passed to try again
		if time.Since(c.cdnHealthState.LastFailure) >= c.cdnHealthState.CircuitBreakerTimeout {
			// Try a health check to see if we can close the circuit
			if c.checkCDNHealth(context.Background()) {
				c.recordCDNSuccess()
				return true
			} else {
				c.recordCDNFailure()
				return false
			}
		}
		return false
	}

	// Check if we need to perform a health check
	if c.shouldCheckCDNHealth() {
		healthy := c.checkCDNHealth(context.Background())
		if healthy {
			c.recordCDNSuccess()
		} else {
			c.recordCDNFailure()
		}
		return healthy
	}

	// Use cached health status
	return c.cdnHealthState.IsHealthy
}

// checkCDNHealth performs an actual health check against the CDN
func (c *SpacesClient) checkCDNHealth(ctx context.Context) bool {
	if c.cdnInfo == nil {
		return false
	}

	// Create a context with timeout for health check
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Try to get CDN info via DigitalOcean API to verify it's still available
	_, err := c.doAPIClient.GetCDN(healthCtx, c.cdnInfo.ID)
	if err != nil {
		return false
	}

	// TODO: In a production implementation, you might also want to:
	// 1. Make an HTTP HEAD request to a known CDN URL
	// 2. Check CDN latency/response time
	// 3. Verify CDN cache hit rates
	// 4. Test CDN geographic distribution

	c.cdnHealthState.LastHealthCheck = time.Now()
	return true
}

// shouldCheckCDNHealth determines if it's time to perform a health check
func (c *SpacesClient) shouldCheckCDNHealth() bool {
	if c.cdnHealthState.LastHealthCheck.IsZero() {
		return true // Never checked before
	}

	return time.Since(c.cdnHealthState.LastHealthCheck) >= c.cdnHealthState.HealthCheckInterval
}

// recordCDNSuccess records a successful CDN operation and resets failure counters
func (c *SpacesClient) recordCDNSuccess() {
	c.cdnHealthState.IsHealthy = true
	c.cdnHealthState.ConsecutiveFailures = 0
	c.cdnHealthState.CircuitBreakerOpen = false
	c.cdnHealthState.LastHealthCheck = time.Now()
}

// recordCDNFailure records a CDN failure and updates circuit breaker state
func (c *SpacesClient) recordCDNFailure() {
	c.cdnHealthState.IsHealthy = false
	c.cdnHealthState.ConsecutiveFailures++
	c.cdnHealthState.LastFailure = time.Now()
	c.cdnHealthState.LastHealthCheck = time.Now()

	// Open circuit breaker if we've exceeded the failure threshold
	if c.cdnHealthState.ConsecutiveFailures >= c.cdnHealthState.MaxConsecutiveFailures {
		c.cdnHealthState.CircuitBreakerOpen = true
	}
}

// GetCDNHealthStatus returns the current CDN health status
func (c *SpacesClient) GetCDNHealthStatus() map[string]interface{} {
	if c.cdnHealthState == nil {
		return map[string]interface{}{
			"cdn_configured": false,
		}
	}

	return map[string]interface{}{
		"cdn_configured":        c.cdnInfo != nil,
		"is_healthy":            c.cdnHealthState.IsHealthy,
		"last_health_check":     c.cdnHealthState.LastHealthCheck,
		"last_failure":          c.cdnHealthState.LastFailure,
		"consecutive_failures":  c.cdnHealthState.ConsecutiveFailures,
		"circuit_breaker_open":  c.cdnHealthState.CircuitBreakerOpen,
		"health_check_interval": c.cdnHealthState.HealthCheckInterval.String(),
		"max_failures":          c.cdnHealthState.MaxConsecutiveFailures,
		"circuit_timeout":       c.cdnHealthState.CircuitBreakerTimeout.String(),
	}
}

// ForceRefreshCDNHealth forces a CDN health check and updates the status
func (c *SpacesClient) ForceRefreshCDNHealth(ctx context.Context) bool {
	if c.cdnInfo == nil {
		return false
	}

	healthy := c.checkCDNHealth(ctx)
	if healthy {
		c.recordCDNSuccess()
	} else {
		c.recordCDNFailure()
	}

	return healthy
}

// GetURLWithFallback returns a URL with explicit fallback handling
func (c *SpacesClient) GetURLWithFallback(path string, forceFallback bool) (url string, usedCDN bool) {
	path = sanitizePath(path)

	// Force fallback if requested
	if forceFallback {
		return c.s3Client.GetPublicURL(c.bucket, path, true), false
	}

	// Try CDN first if available and healthy
	if c.cdnInfo != nil && c.isCDNHealthy() {
		return c.getCDNURL(path), true
	}

	// Fall back to direct Spaces URL
	return c.s3Client.GetPublicURL(c.bucket, path, true), false
}
