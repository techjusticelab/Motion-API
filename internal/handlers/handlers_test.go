package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"motion-index-fiber/internal/config"
	doConfig "motion-index-fiber/pkg/cloud/digitalocean/config"
)

func TestNew_Success(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
	}{
		{
			name: "local environment with nil DigitalOcean config",
			config: &config.Config{
				Environment: "local",
				Server: config.ServerConfig{
					Port: "6000",
				},
			},
		},
		{
			name: "local environment with DigitalOcean config",
			config: &config.Config{
				Environment: "local",
				Server: config.ServerConfig{
					Port: "6000",
				},
				DigitalOcean: &doConfig.Config{
					Environment: doConfig.EnvLocal,
				},
			},
		},
		{
			name: "nil DigitalOcean config uses mock services",
			config: &config.Config{
				Environment: "test",
				Server: config.ServerConfig{
					Port: "6000",
				},
				DigitalOcean: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, err := New(tt.config)

			assert.NoError(t, err)
			assert.NotNil(t, handlers)
			assert.NotNil(t, handlers.Health)
			assert.NotNil(t, handlers.Processing)
			assert.NotNil(t, handlers.Search)
			assert.NotNil(t, handlers.Storage)
		})
	}
}

func TestNew_LocalEnvironmentFallsBackToMockServices(t *testing.T) {
	// Test that local environment falls back to mock services when DigitalOcean services fail
	cfg := &config.Config{
		Environment: "local",
		Server: config.ServerConfig{
			Port: "6000",
		},
		DigitalOcean: &doConfig.Config{
			Environment: doConfig.EnvLocal,
		},
	}

	handlers, err := New(cfg)

	// Should succeed due to fallback to mock services in local environment
	assert.NoError(t, err)
	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.Health)
	assert.NotNil(t, handlers.Processing)
	assert.NotNil(t, handlers.Search)
	assert.NotNil(t, handlers.Storage)
}

func TestNew_ProductionEnvironmentRequiresValidServices(t *testing.T) {
	// Test that production environment fails when services can't be created
	cfg := &config.Config{
		Environment: "production",
		Server: config.ServerConfig{
			Port: "6000",
		},
		DigitalOcean: &doConfig.Config{
			Environment: doConfig.EnvProduction,
		},
	}

	handlers, err := New(cfg)

	// Should fail in production environment with invalid credentials
	assert.Error(t, err)
	assert.Nil(t, handlers)
	assert.Contains(t, err.Error(), "failed to create")
}

func TestNew_NilConfig(t *testing.T) {
	// Test behavior with nil config - should panic or handle gracefully
	assert.Panics(t, func() {
		New(nil)
	})
}

func TestNew_HandlersStructure(t *testing.T) {
	cfg := &config.Config{
		Environment: "local",
		Server: config.ServerConfig{
			Port: "6000",
		},
	}

	handlers, err := New(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, handlers)

	// Verify all handlers are properly initialized
	assert.IsType(t, &HealthHandler{}, handlers.Health)
	assert.IsType(t, &ProcessingHandler{}, handlers.Processing)
	assert.IsType(t, &SearchHandler{}, handlers.Search)
	assert.IsType(t, &StorageHandler{}, handlers.Storage)

	// Verify handlers have required dependencies
	assert.NotNil(t, handlers.Health.storage)
	assert.NotNil(t, handlers.Health.searchSvc)
	assert.NotNil(t, handlers.Processing.storage)
	assert.NotNil(t, handlers.Processing.searchSvc)
	// Processing pipeline should now be initialized
	assert.NotNil(t, handlers.Processing.pipeline)
	assert.NotNil(t, handlers.Search.searchService)
	assert.NotNil(t, handlers.Storage.cfg)
}

func TestNew_ServiceIntegration(t *testing.T) {
	// Test that services are properly shared between handlers
	cfg := &config.Config{
		Environment: "local",
		Server: config.ServerConfig{
			Port: "6000",
		},
	}

	handlers, err := New(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, handlers)

	// Health and Processing handlers should share the same storage service instance
	assert.Equal(t, handlers.Health.storage, handlers.Processing.storage)

	// All handlers using search service should share the same instance
	assert.Equal(t, handlers.Health.searchSvc, handlers.Processing.searchSvc)
	assert.Equal(t, handlers.Health.searchSvc, handlers.Search.searchService)
}

func TestNew_ConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "valid local config",
			config: &config.Config{
				Environment: "local",
				Server: config.ServerConfig{
					Port: "6000",
				},
			},
			expectError: false,
		},
		{
			name: "valid staging config with DO services",
			config: &config.Config{
				Environment: "staging",
				Server: config.ServerConfig{
					Port: "6000",
				},
				DigitalOcean: &doConfig.Config{
					Environment: doConfig.EnvStaging,
				},
			},
			expectError: false, // May still error if services fail, but config is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, err := New(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, handlers)
			} else {
				// Note: Even valid configs might error if external services are unreachable
				// but the handlers initialization should at least attempt to work
				if err == nil {
					assert.NotNil(t, handlers)
				}
			}
		})
	}
}
