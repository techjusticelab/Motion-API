package spaces

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"motion-index-fiber/pkg/storage"
)

// S3Client handles S3-compatible storage operations for DigitalOcean Spaces
type S3Client interface {
	// Core file operations
	Upload(ctx context.Context, bucket, key string, content io.Reader, metadata *storage.UploadMetadata) (*storage.UploadResult, error)
	Download(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, bucket, key string) error

	// File management
	Exists(ctx context.Context, bucket, key string) (bool, error)
	List(ctx context.Context, bucket, prefix string, maxKeys int) ([]*storage.StorageObject, error)

	// URL generation
	GetPublicURL(bucket, key string, useSSL bool) string
	GetSignedURL(ctx context.Context, bucket, key string, expiration time.Duration) (string, error)

	// Batch operations
	BatchUpload(ctx context.Context, bucket string, uploads []*BatchUploadItem) ([]*storage.UploadResult, error)
	BatchDelete(ctx context.Context, bucket string, keys []string) error

	// Health and connectivity
	IsHealthy(ctx context.Context) bool
	GetConnectionInfo() *ConnectionInfo
}

// BatchUploadItem represents a single item in a batch upload operation
type BatchUploadItem struct {
	Key      string                  `json:"key"`
	Content  io.Reader               `json:"-"`
	Metadata *storage.UploadMetadata `json:"metadata"`
}

// ConnectionInfo provides information about the S3 connection
type ConnectionInfo struct {
	Endpoint        string    `json:"endpoint"`
	Region          string    `json:"region"`
	Bucket          string    `json:"bucket"`
	UseSSL          bool      `json:"use_ssl"`
	ConnectedAt     time.Time `json:"connected_at"`
	LastHealthCheck time.Time `json:"last_health_check"`
	IsHealthy       bool      `json:"is_healthy"`
}

// S3Config contains configuration for S3-compatible client
type S3Config struct {
	AccessKey      string `json:"access_key"`
	SecretKey      string `json:"secret_key"`
	Endpoint       string `json:"endpoint"`
	Region         string `json:"region"`
	Bucket         string `json:"bucket"`
	UseSSL         bool   `json:"use_ssl"`
	ForcePathStyle bool   `json:"force_path_style"` // Required for DigitalOcean Spaces
}

// s3ClientImpl implements S3Client using AWS SDK v2
type s3ClientImpl struct {
	config      *S3Config
	connInfo    *ConnectionInfo
	client      *s3.Client
	initialized bool
}

// NewS3Client creates a new S3-compatible client for DigitalOcean Spaces
func NewS3Client(config *S3Config) (S3Client, error) {
	if err := validateS3Config(config); err != nil {
		return nil, fmt.Errorf("invalid S3 config: %w", err)
	}

	client := &s3ClientImpl{
		config: config,
		connInfo: &ConnectionInfo{
			Endpoint:        config.Endpoint,
			Region:          config.Region,
			Bucket:          config.Bucket,
			UseSSL:          config.UseSSL,
			ConnectedAt:     time.Now(),
			LastHealthCheck: time.Time{},
			IsHealthy:       false,
		},
	}

	// Initialize actual AWS S3 client for DigitalOcean Spaces
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Use a custom HTTP client with reasonable timeouts for DigitalOcean Spaces
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			ResponseHeaderTimeout: 15 * time.Second,
			IdleConnTimeout: 30 * time.Second,
			MaxIdleConns: 10,
			MaxIdleConnsPerHost: 10,
		},
	}
	
	awsConfig, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(client.config.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			client.config.AccessKey, client.config.SecretKey, "")),
		awsconfig.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               client.config.Endpoint,
					SigningRegion:     client.config.Region,
					HostnameImmutable: true,
				}, nil
			})),
		awsconfig.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with DigitalOcean Spaces configuration
	client.client = s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.UsePathStyle = client.config.ForcePathStyle
		// Disable retries entirely to avoid rate limiting issues with DigitalOcean Spaces
		o.RetryMaxAttempts = 0
		o.RetryMode = aws.RetryModeStandard
		// Disable signature version 4a for better compatibility with DigitalOcean Spaces
		o.DisableMultiRegionAccessPoints = true
	})

	client.initialized = true
	return client, nil
}

// Core file operations

func (c *s3ClientImpl) Upload(ctx context.Context, bucket, key string, content io.Reader, metadata *storage.UploadMetadata) (*storage.UploadResult, error) {
	if !c.initialized {
		return nil, fmt.Errorf("S3 client not initialized")
	}

	// TODO: Implement actual S3 upload using AWS SDK
	// This would involve:
	// 1. Converting metadata to S3 metadata format
	// 2. Handling multipart uploads for large files
	// 3. Progress tracking if needed
	// 4. Error handling and retries

	return &storage.UploadResult{
		Path:       key,
		URL:        c.GetPublicURL(bucket, key, c.config.UseSSL),
		Size:       metadata.Size,
		Success:    false,
		Error:      "S3 upload not yet implemented",
		UploadedAt: time.Now(),
	}, fmt.Errorf("S3 upload not yet implemented")
}

func (c *s3ClientImpl) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	if !c.initialized {
		return nil, fmt.Errorf("S3 client not initialized")
	}

	// TODO: Implement actual S3 download using AWS SDK
	// This would involve:
	// 1. GetObject request
	// 2. Streaming response handling
	// 3. Error handling for not found, access denied, etc.

	return nil, fmt.Errorf("S3 download not yet implemented")
}

func (c *s3ClientImpl) Delete(ctx context.Context, bucket, key string) error {
	if !c.initialized {
		return fmt.Errorf("S3 client not initialized")
	}

	// TODO: Implement actual S3 delete using AWS SDK
	// This would involve:
	// 1. DeleteObject request
	// 2. Error handling for not found cases

	return fmt.Errorf("S3 delete not yet implemented")
}

// File management

func (c *s3ClientImpl) Exists(ctx context.Context, bucket, key string) (bool, error) {
	if !c.initialized {
		return false, fmt.Errorf("S3 client not initialized")
	}

	// TODO: Implement actual S3 head object using AWS SDK
	// This would involve:
	// 1. HeadObject request
	// 2. Handling not found vs other errors

	return false, fmt.Errorf("S3 exists check not yet implemented")
}

func (c *s3ClientImpl) List(ctx context.Context, bucket, prefix string, maxKeys int) ([]*storage.StorageObject, error) {
	if !c.initialized {
		return nil, fmt.Errorf("S3 client not initialized")
	}

	// Create ListObjectsV2 input
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(int32(maxKeys)),
	}

	// Execute the list operation
	result, err := c.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	// Convert S3 objects to StorageObject format
	var objects []*storage.StorageObject
	for _, obj := range result.Contents {
		if obj.Key == nil {
			continue
		}

		storageObj := &storage.StorageObject{
			Path:         aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: aws.ToTime(obj.LastModified),
		}
		objects = append(objects, storageObj)
	}

	return objects, nil
}

// URL generation

func (c *s3ClientImpl) GetPublicURL(bucket, key string, useSSL bool) string {
	protocol := "https"
	if !useSSL {
		protocol = "http"
	}

	// For DigitalOcean Spaces, the URL format is:
	// https://bucket.region.digitaloceanspaces.com/key
	// or for custom endpoints: https://endpoint/bucket/key (if path-style)

	if c.config.ForcePathStyle {
		return fmt.Sprintf("%s://%s/%s/%s", protocol, c.config.Endpoint, bucket, key)
	} else {
		// Extract hostname from endpoint for subdomain style
		// This is a simplified implementation - would need proper URL parsing
		return fmt.Sprintf("%s://%s.%s/%s", protocol, bucket, c.config.Endpoint, key)
	}
}

func (c *s3ClientImpl) GetSignedURL(ctx context.Context, bucket, key string, expiration time.Duration) (string, error) {
	if !c.initialized {
		return "", fmt.Errorf("S3 client not initialized")
	}

	// TODO: Implement actual S3 presigned URL using AWS SDK
	// This would involve:
	// 1. S3 presign client
	// 2. PresignGetObject request
	// 3. Expiration handling

	return "", fmt.Errorf("S3 signed URL generation not yet implemented")
}

// Batch operations

func (c *s3ClientImpl) BatchUpload(ctx context.Context, bucket string, uploads []*BatchUploadItem) ([]*storage.UploadResult, error) {
	if !c.initialized {
		return nil, fmt.Errorf("S3 client not initialized")
	}

	// TODO: Implement efficient batch upload
	// This could involve:
	// 1. Concurrent uploads with worker pool
	// 2. Progress tracking
	// 3. Error aggregation

	results := make([]*storage.UploadResult, len(uploads))
	for i, upload := range uploads {
		result, err := c.Upload(ctx, bucket, upload.Key, upload.Content, upload.Metadata)
		if err != nil {
			result = &storage.UploadResult{
				Path:    upload.Key,
				Success: false,
				Error:   err.Error(),
			}
		}
		results[i] = result
	}

	return results, nil
}

func (c *s3ClientImpl) BatchDelete(ctx context.Context, bucket string, keys []string) error {
	if !c.initialized {
		return fmt.Errorf("S3 client not initialized")
	}

	// TODO: Implement efficient batch delete using S3 DeleteObjects
	// This would involve:
	// 1. DeleteObjects request with multiple keys
	// 2. Handling partial failures
	// 3. Retry logic for failed deletes

	return fmt.Errorf("S3 batch delete not yet implemented")
}

// Health and connectivity

func (c *s3ClientImpl) IsHealthy(ctx context.Context) bool {
	if !c.initialized {
		return false
	}

	// Create a context with timeout for health check
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Perform a simple HeadBucket request to verify connectivity
	_, err := c.client.HeadBucket(healthCtx, &s3.HeadBucketInput{
		Bucket: aws.String(c.config.Bucket),
	})

	c.connInfo.LastHealthCheck = time.Now()
	c.connInfo.IsHealthy = err == nil
	return c.connInfo.IsHealthy
}

func (c *s3ClientImpl) GetConnectionInfo() *ConnectionInfo {
	return c.connInfo
}

// Validation and helper functions

func validateS3Config(config *S3Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	if config.AccessKey == "" {
		return fmt.Errorf("access key cannot be empty")
	}
	if config.SecretKey == "" {
		return fmt.Errorf("secret key cannot be empty")
	}
	if config.Endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}
	if config.Region == "" {
		return fmt.Errorf("region cannot be empty")
	}
	if config.Bucket == "" {
		return fmt.Errorf("bucket cannot be empty")
	}
	return nil
}

// S3Error represents errors from S3 operations
type S3Error struct {
	Operation string
	Bucket    string
	Key       string
	Message   string
	Cause     error
}

func (e *S3Error) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("S3 %s error for %s/%s: %s", e.Operation, e.Bucket, e.Key, e.Message)
	}
	return fmt.Sprintf("S3 %s error for bucket %s: %s", e.Operation, e.Bucket, e.Message)
}

func (e *S3Error) Unwrap() error {
	return e.Cause
}

// NewS3Error creates a new S3 error
func NewS3Error(operation, bucket, key, message string, cause error) *S3Error {
	return &S3Error{
		Operation: operation,
		Bucket:    bucket,
		Key:       key,
		Message:   message,
		Cause:     cause,
	}
}
