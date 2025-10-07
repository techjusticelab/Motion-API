package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	appconfig "motion-index-fiber/internal/config"
)

type SpacesService struct {
	client    *s3.Client
	bucket    string
	region    string
	cdnDomain string
	config    *SpacesConfig
}

// SpacesUploadResult contains specific spaces upload result
type SpacesUploadResult struct {
	Key     string `json:"key"`
	URL     string `json:"url"`
	CDN_URL string `json:"cdn_url"`
	Size    int64  `json:"size"`
	ETag    string `json:"etag"`
}

func NewSpacesService(cfg *appconfig.Config) (*SpacesService, error) {
	// Create custom endpoint resolver for DigitalOcean Spaces
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:           fmt.Sprintf("https://%s.digitaloceanspaces.com", cfg.Storage.Region),
				SigningRegion: cfg.Storage.Region,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	// Configure AWS SDK for DigitalOcean Spaces
	awsConfig, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.Storage.AccessKey,
			cfg.Storage.SecretKey,
			"",
		)),
		config.WithRegion(cfg.Storage.Region),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.UsePathStyle = false
	})

	// Create SpacesConfig from app config
	spacesConfig := &SpacesConfig{
		AccessKey:   cfg.Storage.AccessKey,
		SecretKey:   cfg.Storage.SecretKey,
		Bucket:      cfg.Storage.Bucket,
		Region:      cfg.Storage.Region,
		CDNDomain:   cfg.Storage.CDNDomain,
	}

	return &SpacesService{
		client:    client,
		bucket:    cfg.Storage.Bucket,
		region:    cfg.Storage.Region,
		cdnDomain: cfg.Storage.CDNDomain,
		config:    spacesConfig,
	}, nil
}

func (s *SpacesService) Upload(ctx context.Context, path string, content io.Reader, metadata *UploadMetadata) (*UploadResult, error) {
	// Ensure path starts with documents/ prefix for organization
	if !strings.HasPrefix(path, "documents/") {
		path = "documents/" + path
	}

	// Determine content type
	contentType := "application/octet-stream"
	if metadata != nil && metadata.ContentType != "" {
		contentType = metadata.ContentType
	}

	// Upload object to Spaces
	putResult, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(path),
		Body:        content,
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload to Spaces: %w", err)
	}

	// Generate URLs
	directURL := fmt.Sprintf("https://%s.%s.digitaloceanspaces.com/%s", s.bucket, s.region, path)

	size := int64(0)
	if metadata != nil {
		size = metadata.Size
	}

	return &UploadResult{
		Path:       path,
		URL:        directURL,
		Size:       size,
		ETag:       aws.ToString(putResult.ETag),
		Success:    true,
		UploadedAt: time.Now(),
	}, nil
}

// Download downloads a document from storage
func (s *SpacesService) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download from Spaces: %w", err)
	}
	return result.Body, nil
}

// Delete deletes a document from storage
func (s *SpacesService) Delete(ctx context.Context, path string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from Spaces: %w", err)
	}
	return nil
}

// GetURL returns a public URL for the document
func (s *SpacesService) GetURL(path string) string {
	return fmt.Sprintf("https://%s.%s.digitaloceanspaces.com/%s", s.bucket, s.region, path)
}

// GetSignedURL returns a signed URL for temporary access
func (s *SpacesService) GetSignedURL(path string, expiration time.Duration) (string, error) {
	// For Spaces, we can use the same as S3 signed URLs
	presignClient := s3.NewPresignClient(s.client)
	presignResult, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}
	return presignResult.URL, nil
}

// List lists documents in a directory with pagination support
func (s *SpacesService) List(ctx context.Context, prefix string) ([]*StorageObject, error) {
	var objects []*StorageObject
	var continuationToken *string
	
	for {
		input := &s3.ListObjectsV2Input{
			Bucket: aws.String(s.bucket),
			Prefix: aws.String(prefix),
		}
		
		if continuationToken != nil {
			input.ContinuationToken = continuationToken
		}
		
		result, err := s.client.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		// Add objects from this page
		for _, obj := range result.Contents {
			objects = append(objects, &StorageObject{
				Path:         aws.ToString(obj.Key),
				Size:         aws.ToInt64(obj.Size),
				LastModified: aws.ToTime(obj.LastModified),
				ETag:         aws.ToString(obj.ETag),
			})
		}
		
		// Check if there are more pages
		if !aws.ToBool(result.IsTruncated) {
			break
		}
		
		continuationToken = result.NextContinuationToken
	}
	
	return objects, nil
}

// IsHealthy returns true if the storage service is healthy
func (s *SpacesService) IsHealthy() bool {
	// Simple health check - try to list objects with empty prefix
	_, err := s.client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucket),
		MaxKeys: aws.Int32(1),
	})
	return err == nil
}

func (s *SpacesService) GetObject(ctx context.Context, key string) (*s3.GetObjectOutput, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from Spaces: %w", err)
	}
	return result, nil
}

func (s *SpacesService) DeleteObject(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from Spaces: %w", err)
	}
	return nil
}

func (s *SpacesService) GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return request.URL, nil
}

func (s *SpacesService) GenerateCDNURL(key string) string {
	// Use CDN domain if configured, otherwise use default CDN endpoint
	if s.cdnDomain != "" {
		return fmt.Sprintf("https://%s/%s", s.cdnDomain, key)
	}
	return fmt.Sprintf("https://%s.%s.cdn.digitaloceanspaces.com/%s", s.bucket, s.region, key)
}

func (s *SpacesService) ListObjects(ctx context.Context, prefix string, maxKeys int32) ([]types.Object, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
	}

	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	if maxKeys > 0 {
		input.MaxKeys = aws.Int32(maxKeys)
	}

	result, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	return result.Contents, nil
}

// Exists checks if a document exists in storage (interface method)
func (s *SpacesService) Exists(ctx context.Context, path string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		// Check if it's a "not found" error
		var notFound *types.NotFound
		if errors.As(err, &notFound) || strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// ObjectExists is an alias for Exists for backward compatibility
func (s *SpacesService) ObjectExists(ctx context.Context, key string) (bool, error) {
	return s.Exists(ctx, key)
}

func (s *SpacesService) GetObjectURL(key string) string {
	return fmt.Sprintf("https://%s.%s.digitaloceanspaces.com/%s", s.bucket, s.region, key)
}

// GetMetrics returns storage-specific metrics
func (s *SpacesService) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})
	metrics["storage_type"] = "digitalocean_spaces"
	metrics["bucket"] = s.bucket
	metrics["region"] = s.region
	metrics["healthy"] = s.IsHealthy()
	if s.cdnDomain != "" {
		metrics["cdn_enabled"] = true
		metrics["cdn_domain"] = s.cdnDomain
	} else {
		metrics["cdn_enabled"] = false
	}
	return metrics
}
