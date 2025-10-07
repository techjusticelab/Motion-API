// Package digitalocean provides DigitalOcean cloud service integrations for Motion-Index
//
// This package follows the UNIX philosophy by providing small, focused interfaces
// that can be composed together. It supports multiple environments (local, staging, production)
// and automatically chooses between mock services and real DigitalOcean services based on configuration.
//
// Key components:
//   - Configuration management with environment-specific validation
//   - Service factory for creating storage and search services
//   - Health monitoring and metrics collection
//   - MCP (Model Context Protocol) integration for DigitalOcean services
//
// Usage:
//
//	// Load configuration from environment
//	cfg, err := config.LoadFromEnvironment()
//	if err != nil {
//	    return err
//	}
//
//	// Create service factory
//	factory := digitalocean.NewServiceFactory(cfg)
//
//	// Create all services
//	services, err := factory.CreateAllServices()
//	if err != nil {
//	    return err
//	}
//	defer services.Close()
//
//	// Use services...
//	storageService := services.Storage
//	searchService := services.Search
package digitalocean

import (
	"context"
	"fmt"

	"motion-index-fiber/pkg/cloud/digitalocean/config"
)

// Provider is the main interface for DigitalOcean cloud services
type Provider interface {
	// GetConfig returns the current configuration
	GetConfig() *config.Config

	// GetServices returns the initialized services
	GetServices() *Services

	// ValidateConfiguration validates the current configuration
	ValidateConfiguration(ctx context.Context) error

	// Shutdown gracefully shuts down all services
	Shutdown() error

	// IsHealthy returns true if all services are healthy
	IsHealthy() bool

	// GetMetrics returns metrics from all services
	GetMetrics() map[string]interface{}
}

// DigitalOceanProvider implements the Provider interface
type DigitalOceanProvider struct {
	config   *config.Config
	factory  *ServiceFactory
	services *Services
}

// NewProvider creates a new DigitalOcean provider with the given configuration
func NewProvider(cfg *config.Config) (*DigitalOceanProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	factory := NewServiceFactory(cfg)

	return &DigitalOceanProvider{
		config:  cfg,
		factory: factory,
	}, nil
}

// NewProviderFromEnvironment creates a provider by loading configuration from environment variables
func NewProviderFromEnvironment() (*DigitalOceanProvider, error) {
	cfg, err := config.LoadFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return NewProvider(cfg)
}

// Initialize initializes all services
func (p *DigitalOceanProvider) Initialize() error {
	services, err := p.factory.CreateAllServices()
	if err != nil {
		return fmt.Errorf("failed to create services: %w", err)
	}

	p.services = services
	return nil
}

// GetConfig returns the current configuration
func (p *DigitalOceanProvider) GetConfig() *config.Config {
	return p.config
}

// GetServices returns the initialized services
func (p *DigitalOceanProvider) GetServices() *Services {
	return p.services
}

// ValidateConfiguration validates the current configuration and service health
func (p *DigitalOceanProvider) ValidateConfiguration(ctx context.Context) error {
	// Validate configuration
	if err := p.config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate services if initialized
	if p.services != nil {
		if !p.services.IsHealthy() {
			return fmt.Errorf("one or more services are unhealthy")
		}
	} else {
		// Validate that services can be created
		return p.factory.ValidateServices(ctx)
	}

	return nil
}

// Shutdown gracefully shuts down all services
func (p *DigitalOceanProvider) Shutdown() error {
	if p.services != nil {
		return p.services.Close()
	}
	return nil
}

// IsHealthy returns true if all services are healthy
func (p *DigitalOceanProvider) IsHealthy() bool {
	if p.services == nil {
		return false
	}
	return p.services.IsHealthy()
}

// GetMetrics returns metrics from all services
func (p *DigitalOceanProvider) GetMetrics() map[string]interface{} {
	if p.services == nil {
		return map[string]interface{}{
			"services_initialized": false,
			"config": map[string]interface{}{
				"environment": p.config.Environment,
			},
		}
	}

	metrics := p.services.GetMetrics()
	metrics["services_initialized"] = true
	return metrics
}

// GetEnvironment returns the current environment
func (p *DigitalOceanProvider) GetEnvironment() config.Environment {
	return p.config.Environment
}

// IsLocal returns true if running in local development environment
func (p *DigitalOceanProvider) IsLocal() bool {
	return p.config.IsLocal()
}

// IsStaging returns true if running in staging environment
func (p *DigitalOceanProvider) IsStaging() bool {
	return p.config.IsStaging()
}

// IsProduction returns true if running in production environment
func (p *DigitalOceanProvider) IsProduction() bool {
	return p.config.IsProduction()
}

// DefaultProvider creates a provider with default local configuration
func DefaultProvider() (*DigitalOceanProvider, error) {
	cfg := config.DefaultConfig()
	return NewProvider(cfg)
}
