package digitalocean

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"motion-index-fiber/pkg/cloud/digitalocean/config"
)

func TestNewProvider(t *testing.T) {
	cfg := config.DefaultConfig()

	provider, err := NewProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, provider)

	assert.Equal(t, cfg, provider.config)
	assert.NotNil(t, provider.factory)
	assert.Nil(t, provider.services) // Services not initialized yet
}

func TestNewProvider_NilConfig(t *testing.T) {
	provider, err := NewProvider(nil)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "configuration cannot be nil")
}

func TestNewProviderFromEnvironment(t *testing.T) {
	// Clean environment
	clearTestEnvironment()
	defer clearTestEnvironment()

	// Set minimal environment for local development
	os.Setenv("ENVIRONMENT", "local")

	provider, err := NewProviderFromEnvironment()
	require.NoError(t, err)
	require.NotNil(t, provider)

	assert.Equal(t, config.EnvLocal, provider.config.Environment)
}

func TestNewProviderFromEnvironment_InvalidConfig(t *testing.T) {
	// Clean environment
	clearTestEnvironment()
	defer clearTestEnvironment()

	// Set invalid environment
	os.Setenv("ENVIRONMENT", "invalid")

	provider, err := NewProviderFromEnvironment()
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "failed to load configuration")
}

func TestDefaultProvider(t *testing.T) {
	provider, err := DefaultProvider()
	require.NoError(t, err)
	require.NotNil(t, provider)

	assert.Equal(t, config.EnvLocal, provider.config.Environment)
	assert.True(t, provider.IsLocal())
}

func TestDigitalOceanProvider_Initialize(t *testing.T) {
	tests := []struct {
		name          string
		environment   config.Environment
		setupConfig   func(*config.Config)
		expectError   bool
		errorContains string
	}{
		{
			name:        "local environment initializes successfully with mock services",
			environment: config.EnvLocal,
			setupConfig: func(cfg *config.Config) {
				// Local environment uses mock services by default
			},
			expectError: false,
		},
		{
			name:        "staging environment with valid config initializes successfully",
			environment: config.EnvStaging,
			setupConfig: func(cfg *config.Config) {
				// Configure valid services
				cfg.DigitalOcean.Spaces.AccessKey = "test-access-key"
				cfg.DigitalOcean.Spaces.SecretKey = "test-secret-key"
				cfg.DigitalOcean.Spaces.Bucket = "test-bucket"
				cfg.DigitalOcean.Spaces.Region = "nyc3"

				cfg.DigitalOcean.OpenSearch.Host = "test-host.com"
				cfg.DigitalOcean.OpenSearch.Port = 9200
				cfg.DigitalOcean.OpenSearch.Username = "test-user"
				cfg.DigitalOcean.OpenSearch.Password = "test-password"
				cfg.DigitalOcean.OpenSearch.UseSSL = false
				cfg.DigitalOcean.OpenSearch.Index = "test-index"
			},
			expectError: false,
		},
		{
			name:        "staging environment with invalid config fails to initialize",
			environment: config.EnvStaging,
			setupConfig: func(cfg *config.Config) {
				// Invalid configuration (missing required fields)
			},
			expectError:   true,
			errorContains: "failed to create services",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Environment = tt.environment

			if tt.setupConfig != nil {
				tt.setupConfig(cfg)
			}

			provider, err := NewProvider(cfg)
			require.NoError(t, err)

			err = provider.Initialize()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				// Services should remain nil on error
				assert.Nil(t, provider.GetServices())
			} else {
				assert.NoError(t, err)
				// Services should be initialized
				services := provider.GetServices()
				require.NotNil(t, services)
				assert.NotNil(t, services.Storage)
				assert.NotNil(t, services.Search)
				assert.True(t, services.IsHealthy())
			}
		})
	}
}

func TestDigitalOceanProvider_GetConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	provider, err := NewProvider(cfg)
	require.NoError(t, err)

	returnedConfig := provider.GetConfig()
	assert.Equal(t, cfg, returnedConfig)
}

func TestDigitalOceanProvider_GetServices(t *testing.T) {
	cfg := config.DefaultConfig()
	provider, err := NewProvider(cfg)
	require.NoError(t, err)

	// Should return nil before initialization
	services := provider.GetServices()
	assert.Nil(t, services)

	// Initialize and check services are available
	err = provider.Initialize()
	require.NoError(t, err)

	services = provider.GetServices()
	require.NotNil(t, services)
	assert.NotNil(t, services.Storage)
	assert.NotNil(t, services.Search)
}

func TestDigitalOceanProvider_ValidateConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		environment   config.Environment
		setupConfig   func(*config.Config)
		initialize    bool
		expectError   bool
		errorContains string
	}{
		{
			name:        "local environment validates successfully without initialization",
			environment: config.EnvLocal,
			setupConfig: func(cfg *config.Config) {
				// Local environment is valid by default
			},
			initialize:  false,
			expectError: false,
		},
		{
			name:        "local environment validates successfully with initialization",
			environment: config.EnvLocal,
			setupConfig: func(cfg *config.Config) {
				// Local environment is valid by default
			},
			initialize:  true,
			expectError: false,
		},
		{
			name:        "staging environment with invalid config fails validation",
			environment: config.EnvStaging,
			setupConfig: func(cfg *config.Config) {
				// Invalid configuration
			},
			initialize:    false,
			expectError:   true,
			errorContains: "service creation failed",
		},
		{
			name:        "staging environment with valid config validates successfully",
			environment: config.EnvStaging,
			setupConfig: func(cfg *config.Config) {
				// Configure valid services
				cfg.DigitalOcean.Spaces.AccessKey = "test-access-key"
				cfg.DigitalOcean.Spaces.SecretKey = "test-secret-key"
				cfg.DigitalOcean.Spaces.Bucket = "test-bucket"
				cfg.DigitalOcean.Spaces.Region = "nyc3"

				cfg.DigitalOcean.OpenSearch.Host = "test-host.com"
				cfg.DigitalOcean.OpenSearch.Port = 9200
				cfg.DigitalOcean.OpenSearch.Username = "test-user"
				cfg.DigitalOcean.OpenSearch.Password = "test-password"
				cfg.DigitalOcean.OpenSearch.UseSSL = false
				cfg.DigitalOcean.OpenSearch.Index = "test-index"
			},
			initialize:  false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Environment = tt.environment

			if tt.setupConfig != nil {
				tt.setupConfig(cfg)
			}

			provider, err := NewProvider(cfg)
			require.NoError(t, err)

			if tt.initialize {
				err = provider.Initialize()
				require.NoError(t, err)
			}

			ctx := context.Background()
			err = provider.ValidateConfiguration(ctx)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDigitalOceanProvider_Shutdown(t *testing.T) {
	cfg := config.DefaultConfig()
	provider, err := NewProvider(cfg)
	require.NoError(t, err)

	// Shutdown before initialization should not error
	err = provider.Shutdown()
	assert.NoError(t, err)

	// Initialize and then shutdown
	err = provider.Initialize()
	require.NoError(t, err)

	err = provider.Shutdown()
	assert.NoError(t, err)
}

func TestDigitalOceanProvider_IsHealthy(t *testing.T) {
	cfg := config.DefaultConfig()
	provider, err := NewProvider(cfg)
	require.NoError(t, err)

	// Should be unhealthy before initialization
	assert.False(t, provider.IsHealthy())

	// Initialize and check health
	err = provider.Initialize()
	require.NoError(t, err)

	assert.True(t, provider.IsHealthy())
}

func TestDigitalOceanProvider_GetMetrics(t *testing.T) {
	cfg := config.DefaultConfig()
	provider, err := NewProvider(cfg)
	require.NoError(t, err)

	// Metrics before initialization
	metrics := provider.GetMetrics()
	require.NotNil(t, metrics)
	assert.Equal(t, false, metrics["services_initialized"])
	assert.Contains(t, metrics, "config")

	// Initialize and check metrics
	err = provider.Initialize()
	require.NoError(t, err)

	metrics = provider.GetMetrics()
	require.NotNil(t, metrics)
	assert.Equal(t, true, metrics["services_initialized"])
	assert.Contains(t, metrics, "config")
}

func TestDigitalOceanProvider_EnvironmentMethods(t *testing.T) {
	tests := []struct {
		name          string
		environment   config.Environment
		expectLocal   bool
		expectStaging bool
		expectProd    bool
	}{
		{
			name:          "local environment",
			environment:   config.EnvLocal,
			expectLocal:   true,
			expectStaging: false,
			expectProd:    false,
		},
		{
			name:          "staging environment",
			environment:   config.EnvStaging,
			expectLocal:   false,
			expectStaging: true,
			expectProd:    false,
		},
		{
			name:          "production environment",
			environment:   config.EnvProduction,
			expectLocal:   false,
			expectStaging: false,
			expectProd:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Environment = tt.environment

			provider, err := NewProvider(cfg)
			require.NoError(t, err)

			assert.Equal(t, tt.environment, provider.GetEnvironment())
			assert.Equal(t, tt.expectLocal, provider.IsLocal())
			assert.Equal(t, tt.expectStaging, provider.IsStaging())
			assert.Equal(t, tt.expectProd, provider.IsProduction())
		})
	}
}

// Helper function to clear test environment variables
func clearTestEnvironment() {
	envVars := []string{
		"ENVIRONMENT",
		"DO_SPACES_ACCESS_KEY",
		"DO_SPACES_SECRET_KEY",
		"DO_SPACES_BUCKET",
		"DO_SPACES_REGION",
		"DO_SPACES_CDN_ENDPOINT",
		"DO_SPACES_ENDPOINT",
		"DO_OPENSEARCH_HOST",
		"DO_OPENSEARCH_PORT",
		"DO_OPENSEARCH_USERNAME",
		"DO_OPENSEARCH_PASSWORD",
		"DO_OPENSEARCH_USE_SSL",
		"DO_OPENSEARCH_INDEX",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}
