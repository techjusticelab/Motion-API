package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromEnvironment(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		expectedEnv   Environment
		expectError   bool
		errorContains string
	}{
		{
			name: "local environment with minimal config",
			envVars: map[string]string{
				"ENVIRONMENT": "local",
			},
			expectedEnv: EnvLocal,
			expectError: false,
		},
		{
			name: "staging environment with full config",
			envVars: map[string]string{
				"ENVIRONMENT":            "staging",
				"DO_API_TOKEN":           "dop_v1_test_token_for_staging",
				"DO_SPACES_ACCESS_KEY":   "test-access-key",
				"DO_SPACES_SECRET_KEY":   "test-secret-key",
				"DO_SPACES_BUCKET":       "test-bucket",
				"DO_SPACES_REGION":       "nyc3",
				"DO_OPENSEARCH_HOST":     "test-host.db.ondigitalocean.com",
				"DO_OPENSEARCH_PORT":     "25060",
				"DO_OPENSEARCH_USERNAME": "doadmin",
				"DO_OPENSEARCH_PASSWORD": "test-password",
				"DO_OPENSEARCH_USE_SSL":  "true",
				"DO_OPENSEARCH_INDEX":    "documents",
			},
			expectedEnv: EnvStaging,
			expectError: false,
		},
		{
			name: "production environment with full config",
			envVars: map[string]string{
				"ENVIRONMENT":            "production",
				"DO_API_TOKEN":           "dop_v1_production_token_for_api",
				"DO_SPACES_ACCESS_KEY":   "prod-access-key",
				"DO_SPACES_SECRET_KEY":   "prod-secret-key",
				"DO_SPACES_BUCKET":       "motion-index-docs",
				"DO_SPACES_REGION":       "nyc3",
				"DO_SPACES_CDN_ENDPOINT": "https://motion-index-docs.nyc3.cdn.digitaloceanspaces.com",
				"DO_OPENSEARCH_HOST":     "test-host.db.ondigitalocean.com",
				"DO_OPENSEARCH_PORT":     "25060",
				"DO_OPENSEARCH_USERNAME": "doadmin",
				"DO_OPENSEARCH_PASSWORD": "production-password",
				"DO_OPENSEARCH_USE_SSL":  "true",
				"DO_OPENSEARCH_INDEX":    "documents",
			},
			expectedEnv: EnvProduction,
			expectError: false,
		},
		{
			name: "invalid environment",
			envVars: map[string]string{
				"ENVIRONMENT": "invalid",
			},
			expectError:   true,
			errorContains: "invalid environment",
		},
		{
			name: "staging environment missing required fields",
			envVars: map[string]string{
				"ENVIRONMENT": "staging",
				// Missing required DO_SPACES_ACCESS_KEY
			},
			expectError:   true,
			errorContains: "configuration error",
		},
		{
			name: "invalid port number",
			envVars: map[string]string{
				"ENVIRONMENT":        "local",
				"DO_OPENSEARCH_PORT": "invalid",
			},
			expectError:   true,
			errorContains: "invalid port number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment
			clearEnvironment()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer clearEnvironment()

			config, err := LoadFromEnvironment()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)
				assert.Equal(t, tt.expectedEnv, config.Environment)

				// Verify configuration methods work
				switch tt.expectedEnv {
				case EnvLocal:
					assert.True(t, config.IsLocal())
					assert.False(t, config.IsStaging())
					assert.False(t, config.IsProduction())
					assert.True(t, config.ShouldUseMockServices())
				case EnvStaging:
					assert.False(t, config.IsLocal())
					assert.True(t, config.IsStaging())
					assert.False(t, config.IsProduction())
					assert.False(t, config.ShouldUseMockServices())
				case EnvProduction:
					assert.False(t, config.IsLocal())
					assert.False(t, config.IsStaging())
					assert.True(t, config.IsProduction())
					assert.False(t, config.ShouldUseMockServices())
				}
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorField  string
	}{
		{
			name: "valid local config",
			config: &Config{
				Environment: EnvLocal,
				Health: struct {
					CheckInterval  int  `json:"check_interval_seconds" validate:"min=1,max=3600"`
					TimeoutSeconds int  `json:"timeout_seconds" validate:"min=1,max=300"`
					MaxRetries     int  `json:"max_retries" validate:"min=0,max=10"`
					CircuitBreaker bool `json:"circuit_breaker"`
				}{
					CheckInterval:  30,
					TimeoutSeconds: 10,
					MaxRetries:     3,
					CircuitBreaker: true,
				},
				Performance: struct {
					MaxConcurrentUploads   int  `json:"max_concurrent_uploads" validate:"min=1,max=100"`
					MaxConcurrentDownloads int  `json:"max_concurrent_downloads" validate:"min=1,max=100"`
					ChunkSizeBytes         int  `json:"chunk_size_bytes" validate:"min=1024"`
					EnableCaching          bool `json:"enable_caching"`
					CacheTTLSeconds        int  `json:"cache_ttl_seconds" validate:"min=60"`
				}{
					MaxConcurrentUploads:   10,
					MaxConcurrentDownloads: 20,
					ChunkSizeBytes:         8 * 1024 * 1024,
					EnableCaching:          true,
					CacheTTLSeconds:        3600,
				},
			},
			expectError: false,
		},
		{
			name: "invalid health check interval",
			config: &Config{
				Environment: EnvLocal,
				Health: struct {
					CheckInterval  int  `json:"check_interval_seconds" validate:"min=1,max=3600"`
					TimeoutSeconds int  `json:"timeout_seconds" validate:"min=1,max=300"`
					MaxRetries     int  `json:"max_retries" validate:"min=0,max=10"`
					CircuitBreaker bool `json:"circuit_breaker"`
				}{
					CheckInterval:  0, // Invalid: too low
					TimeoutSeconds: 10,
					MaxRetries:     3,
					CircuitBreaker: true,
				},
				Performance: struct {
					MaxConcurrentUploads   int  `json:"max_concurrent_uploads" validate:"min=1,max=100"`
					MaxConcurrentDownloads int  `json:"max_concurrent_downloads" validate:"min=1,max=100"`
					ChunkSizeBytes         int  `json:"chunk_size_bytes" validate:"min=1024"`
					EnableCaching          bool `json:"enable_caching"`
					CacheTTLSeconds        int  `json:"cache_ttl_seconds" validate:"min=60"`
				}{
					MaxConcurrentUploads:   10,
					MaxConcurrentDownloads: 20,
					ChunkSizeBytes:         8 * 1024 * 1024,
					EnableCaching:          true,
					CacheTTLSeconds:        3600,
				},
			},
			expectError: true,
			errorField:  "CheckInterval",
		},
		{
			name: "invalid chunk size",
			config: &Config{
				Environment: EnvLocal,
				Health: struct {
					CheckInterval  int  `json:"check_interval_seconds" validate:"min=1,max=3600"`
					TimeoutSeconds int  `json:"timeout_seconds" validate:"min=1,max=300"`
					MaxRetries     int  `json:"max_retries" validate:"min=0,max=10"`
					CircuitBreaker bool `json:"circuit_breaker"`
				}{
					CheckInterval:  30,
					TimeoutSeconds: 10,
					MaxRetries:     3,
					CircuitBreaker: true,
				},
				Performance: struct {
					MaxConcurrentUploads   int  `json:"max_concurrent_uploads" validate:"min=1,max=100"`
					MaxConcurrentDownloads int  `json:"max_concurrent_downloads" validate:"min=1,max=100"`
					ChunkSizeBytes         int  `json:"chunk_size_bytes" validate:"min=1024"`
					EnableCaching          bool `json:"enable_caching"`
					CacheTTLSeconds        int  `json:"cache_ttl_seconds" validate:"min=60"`
				}{
					MaxConcurrentUploads:   10,
					MaxConcurrentDownloads: 20,
					ChunkSizeBytes:         512, // Invalid: too small
					EnableCaching:          true,
					CacheTTLSeconds:        3600,
				},
			},
			expectError: true,
			errorField:  "ChunkSizeBytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add required OpenSearch index for validation
			tt.config.DigitalOcean.OpenSearch.Index = "documents"
			tt.config.DigitalOcean.OpenSearch.Port = 25060

			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorField != "" {
					configErr, ok := err.(*ConfigError)
					require.True(t, ok, "Expected ConfigError")
					// Since we simplified error handling, check that error contains the field name
					assert.Contains(t, configErr.Message, tt.errorField)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigEndpointMethods(t *testing.T) {
	config := &Config{
		Environment: EnvProduction,
	}

	config.DigitalOcean.Spaces.Region = "nyc3"
	config.DigitalOcean.Spaces.Bucket = "motion-index-docs"
	config.DigitalOcean.Spaces.CDNEndpoint = "https://custom-cdn.example.com"

	config.DigitalOcean.OpenSearch.Host = "test-host.db.ondigitalocean.com"
	config.DigitalOcean.OpenSearch.Port = 25060
	config.DigitalOcean.OpenSearch.UseSSL = true

	t.Run("GetSpacesEndpoint", func(t *testing.T) {
		expected := "https://nyc3.digitaloceanspaces.com"
		assert.Equal(t, expected, config.GetSpacesEndpoint())

		// Test with custom endpoint
		config.DigitalOcean.Spaces.Endpoint = "https://custom.endpoint.com"
		assert.Equal(t, "https://custom.endpoint.com", config.GetSpacesEndpoint())
	})

	t.Run("GetSpacesCDNEndpoint", func(t *testing.T) {
		expected := "https://custom-cdn.example.com"
		assert.Equal(t, expected, config.GetSpacesCDNEndpoint())

		// Test with default CDN endpoint
		config.DigitalOcean.Spaces.CDNEndpoint = ""
		expected = "https://motion-index-docs.nyc3.cdn.digitaloceanspaces.com"
		assert.Equal(t, expected, config.GetSpacesCDNEndpoint())
	})

	t.Run("GetOpenSearchEndpoint", func(t *testing.T) {
		expected := "https://test-host.db.ondigitalocean.com:25060"
		assert.Equal(t, expected, config.GetOpenSearchEndpoint())

		// Test with HTTP
		config.DigitalOcean.OpenSearch.UseSSL = false
		expected = "http://test-host.db.ondigitalocean.com:25060"
		assert.Equal(t, expected, config.GetOpenSearchEndpoint())
	})
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, EnvLocal, config.Environment)
	assert.True(t, config.IsLocal())
	assert.True(t, config.ShouldUseMockServices())

	// Verify sensible defaults
	assert.Equal(t, 30, config.Health.CheckInterval)
	assert.Equal(t, 10, config.Health.TimeoutSeconds)
	assert.Equal(t, 3, config.Health.MaxRetries)
	assert.True(t, config.Health.CircuitBreaker)

	assert.Equal(t, 10, config.Performance.MaxConcurrentUploads)
	assert.Equal(t, 20, config.Performance.MaxConcurrentDownloads)
	assert.Equal(t, 8*1024*1024, config.Performance.ChunkSizeBytes)
	assert.True(t, config.Performance.EnableCaching)
	assert.Equal(t, 3600, config.Performance.CacheTTLSeconds)
}

func TestEnvironmentHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		testFunc func() interface{}
		expected interface{}
	}{
		{
			name: "getEnvWithDefault with value",
			envVars: map[string]string{
				"TEST_STRING": "test-value",
			},
			testFunc: func() interface{} {
				return getEnvWithDefault("TEST_STRING", "default")
			},
			expected: "test-value",
		},
		{
			name:    "getEnvWithDefault with default",
			envVars: map[string]string{},
			testFunc: func() interface{} {
				return getEnvWithDefault("TEST_STRING", "default")
			},
			expected: "default",
		},
		{
			name: "getEnvIntWithDefault with value",
			envVars: map[string]string{
				"TEST_INT": "42",
			},
			testFunc: func() interface{} {
				return getEnvIntWithDefault("TEST_INT", 100)
			},
			expected: 42,
		},
		{
			name:    "getEnvIntWithDefault with default",
			envVars: map[string]string{},
			testFunc: func() interface{} {
				return getEnvIntWithDefault("TEST_INT", 100)
			},
			expected: 100,
		},
		{
			name: "getEnvIntWithDefault with invalid value",
			envVars: map[string]string{
				"TEST_INT": "invalid",
			},
			testFunc: func() interface{} {
				return getEnvIntWithDefault("TEST_INT", 100)
			},
			expected: 100,
		},
		{
			name: "getEnvBoolWithDefault with true",
			envVars: map[string]string{
				"TEST_BOOL": "true",
			},
			testFunc: func() interface{} {
				return getEnvBoolWithDefault("TEST_BOOL", false)
			},
			expected: true,
		},
		{
			name: "getEnvBoolWithDefault with false",
			envVars: map[string]string{
				"TEST_BOOL": "false",
			},
			testFunc: func() interface{} {
				return getEnvBoolWithDefault("TEST_BOOL", true)
			},
			expected: false,
		},
		{
			name:    "getEnvBoolWithDefault with default",
			envVars: map[string]string{},
			testFunc: func() interface{} {
				return getEnvBoolWithDefault("TEST_BOOL", true)
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnvironment()

			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer clearEnvironment()

			result := tt.testFunc()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to clear test environment variables
func clearEnvironment() {
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
		"HEALTH_CHECK_INTERVAL",
		"HEALTH_TIMEOUT_SECONDS",
		"HEALTH_MAX_RETRIES",
		"HEALTH_CIRCUIT_BREAKER",
		"PERF_MAX_CONCURRENT_UPLOADS",
		"PERF_MAX_CONCURRENT_DOWNLOADS",
		"PERF_CHUNK_SIZE_BYTES",
		"PERF_ENABLE_CACHING",
		"PERF_CACHE_TTL_SECONDS",
		"TEST_STRING",
		"TEST_INT",
		"TEST_BOOL",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}
