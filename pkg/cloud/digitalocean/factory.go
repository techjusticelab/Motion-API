package digitalocean

import (
	"context"
	"fmt"

	internalConfig "motion-index-fiber/internal/config"
	"motion-index-fiber/pkg/cloud/digitalocean/config"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/search/client"
	"motion-index-fiber/pkg/storage"
)

// ServiceFactory creates DigitalOcean services based on configuration
type ServiceFactory struct {
	config *config.Config
}

// NewServiceFactory creates a new service factory with the given configuration
func NewServiceFactory(cfg *config.Config) *ServiceFactory {
	return &ServiceFactory{
		config: cfg,
	}
}

// CreateStorageService creates a storage service using DigitalOcean Spaces
func (f *ServiceFactory) CreateStorageService() (storage.Service, error) {
	return f.createSpacesStorageService()
}

// CreateSearchService creates a search service using DigitalOcean OpenSearch
func (f *ServiceFactory) CreateSearchService() (search.Service, error) {
	return f.createOpenSearchService()
}

// CreateAllServices creates both storage and search services
func (f *ServiceFactory) CreateAllServices() (*Services, error) {
	storageService, err := f.CreateStorageService()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage service: %w", err)
	}

	searchService, err := f.CreateSearchService()
	if err != nil {
		return nil, fmt.Errorf("failed to create search service: %w", err)
	}

	return &Services{
		Storage: storageService,
		Search:  searchService,
		Config:  f.config,
	}, nil
}

// ValidateServices validates that all required services can be created
func (f *ServiceFactory) ValidateServices(ctx context.Context) error {
	services, err := f.CreateAllServices()
	if err != nil {
		return fmt.Errorf("service creation failed: %w", err)
	}

	// Test storage health
	if !services.Storage.IsHealthy() {
		return fmt.Errorf("storage service is not healthy")
	}

	// Test search health
	healthChecker, ok := services.Search.(search.HealthChecker)
	if ok {
		if !healthChecker.IsHealthy() {
			return fmt.Errorf("search service is not healthy")
		}
	}

	return nil
}


// createSpacesStorageService creates a DigitalOcean Spaces storage service
func (f *ServiceFactory) createSpacesStorageService() (storage.Service, error) {
	// Convert DigitalOcean config to internal config format needed by SpacesService
	internalCfg := &internalConfig.Config{
		Storage: internalConfig.StorageConfig{
			AccessKey:   f.config.DigitalOcean.Spaces.AccessKey,
			SecretKey:   f.config.DigitalOcean.Spaces.SecretKey,
			Bucket:      f.config.DigitalOcean.Spaces.Bucket,
			Region:      f.config.DigitalOcean.Spaces.Region,
			CDNDomain:   f.config.DigitalOcean.Spaces.CDNEndpoint,
		},
	}
	
	// Use the properly implemented SpacesService
	return storage.NewSpacesService(internalCfg)
}


// createOpenSearchService creates a DigitalOcean OpenSearch service
func (f *ServiceFactory) createOpenSearchService() (search.Service, error) {
	// Create internal config from DigitalOcean config
	internalConfig := f.bridgeOpenSearchConfig()

	// Create OpenSearch client using the existing client package
	client, err := f.createOpenSearchClient(internalConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenSearch client: %w", err)
	}

	// Create and return the search service
	return search.NewService(client), nil
}

// Services holds all DigitalOcean services
type Services struct {
	Storage storage.Service
	Search  search.Service
	Config  *config.Config
}

// Close closes all services and cleans up resources
func (s *Services) Close() error {
	var errors []error

	// Close services that implement io.Closer if needed
	// For now, services don't implement Close() so this is a no-op

	if len(errors) > 0 {
		return fmt.Errorf("errors closing services: %v", errors)
	}

	return nil
}

// IsHealthy returns true if all services are healthy
func (s *Services) IsHealthy() bool {
	if s.Storage == nil || !s.Storage.IsHealthy() {
		return false
	}

	if s.Search == nil {
		return false
	}

	if healthChecker, ok := s.Search.(search.HealthChecker); ok {
		return healthChecker.IsHealthy()
	}

	return true
}

// GetMetrics returns combined metrics from all services
func (s *Services) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Add storage metrics
	storageMetrics := s.Storage.GetMetrics()
	if len(storageMetrics) > 0 {
		metrics["storage"] = storageMetrics
	}

	// Add configuration info
	metrics["config"] = map[string]interface{}{
		"environment":       s.Config.Environment,
		"use_mock_services": s.Config.ShouldUseMockServices(),
		"spaces_region":     s.Config.DigitalOcean.Spaces.Region,
		"opensearch_host":   s.Config.DigitalOcean.OpenSearch.Host,
	}

	return metrics
}

// bridgeOpenSearchConfig creates an internal OpenSearch config from DigitalOcean config
func (f *ServiceFactory) bridgeOpenSearchConfig() *internalConfig.OpenSearchConfig {
	doOpenSearch := &f.config.DigitalOcean.OpenSearch

	return &internalConfig.OpenSearchConfig{
		Host:     doOpenSearch.Host,
		Port:     doOpenSearch.Port,
		Username: doOpenSearch.Username,
		Password: doOpenSearch.Password,
		UseSSL:   doOpenSearch.UseSSL,
		Index:    doOpenSearch.Index,
	}
}

// createOpenSearchClient creates an OpenSearch client using the internal client package
func (f *ServiceFactory) createOpenSearchClient(cfg *internalConfig.OpenSearchConfig) (*client.Client, error) {
	return client.NewClient(cfg)
}
