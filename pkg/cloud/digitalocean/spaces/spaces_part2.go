package spaces

import (
	"context"
	"fmt"
	"motion-index-fiber/pkg/cloud/digitalocean/config"
	"motion-index-fiber/pkg/storage"
	"strings"
	"time"
)

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

func (c *SpacesClient) invalidateCDNCache(ctx context.Context, files []string) error {
	if c.cdnInfo == nil {
		return nil // No CDN to invalidate
	}

	return c.doAPIClient.FlushCDNCache(ctx, c.cdnInfo.ID, files)
}

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

func sanitizePath(path string) string {
	// Remove leading slashes
	path = strings.TrimPrefix(path, "/")

	// TODO: Add more path sanitization as needed
	// - Remove double slashes
	// - Handle special characters
	// - Validate path length

	return path
}

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

type URLOptimizationOptions struct {
	Quality     string // auto, high, medium, low
	Format      string // auto, webp, jpg, png
	Resize      *ResizeOptions
	Compression bool
	UserAgent   string
	DeviceType  string // mobile, tablet, desktop
	Bandwidth   string // high, medium, low
}

type ResizeOptions struct {
	Width  int
	Height int
	Mode   string // fit, fill, crop
}

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

func (c *SpacesClient) shouldCheckCDNHealth() bool {
	if c.cdnHealthState.LastHealthCheck.IsZero() {
		return true // Never checked before
	}

	return time.Since(c.cdnHealthState.LastHealthCheck) >= c.cdnHealthState.HealthCheckInterval
}

func (c *SpacesClient) recordCDNSuccess() {
	c.cdnHealthState.IsHealthy = true
	c.cdnHealthState.ConsecutiveFailures = 0
	c.cdnHealthState.CircuitBreakerOpen = false
	c.cdnHealthState.LastHealthCheck = time.Now()
}

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
