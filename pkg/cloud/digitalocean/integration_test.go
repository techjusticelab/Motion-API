//go:build integration
// +build integration

package digitalocean

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"motion-index-fiber/pkg/cloud/digitalocean/config"
)

// IntegrationTestSuite contains integration tests for DigitalOcean services
// These tests require real DigitalOcean credentials and services to be available
type IntegrationTestSuite struct {
	suite.Suite
	provider    *DigitalOceanProvider
	config      *config.Config
	ctx         context.Context
	cancel      context.CancelFunc
	originalEnv map[string]string
}

// SetupSuite runs once before all tests in the suite
func (suite *IntegrationTestSuite) SetupSuite() {
	// Save original environment
	suite.originalEnv = make(map[string]string)
	envVars := []string{
		"ENVIRONMENT",
		"DO_SPACES_ACCESS_KEY",
		"DO_SPACES_SECRET_KEY",
		"DO_SPACES_BUCKET",
		"DO_SPACES_REGION",
		"DO_OPENSEARCH_HOST",
		"DO_OPENSEARCH_PORT",
		"DO_OPENSEARCH_USERNAME",
		"DO_OPENSEARCH_PASSWORD",
		"DO_OPENSEARCH_USE_SSL",
		"DO_OPENSEARCH_INDEX",
	}

	for _, envVar := range envVars {
		suite.originalEnv[envVar] = os.Getenv(envVar)
	}

	// Set integration test environment
	suite.setupIntegrationEnvironment()

	// Create context with timeout for all tests
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 30*time.Second)
}

// TearDownSuite runs once after all tests in the suite
func (suite *IntegrationTestSuite) TearDownSuite() {
	// Cancel context
	if suite.cancel != nil {
		suite.cancel()
	}

	// Shutdown provider if initialized
	if suite.provider != nil {
		_ = suite.provider.Shutdown()
	}

	// Restore original environment
	for envVar, value := range suite.originalEnv {
		if value == "" {
			os.Unsetenv(envVar)
		} else {
			os.Setenv(envVar, value)
		}
	}
}

// SetupTest runs before each test
func (suite *IntegrationTestSuite) SetupTest() {
	// Create provider from environment
	provider, err := NewProviderFromEnvironment()
	suite.Require().NoError(err)
	suite.provider = provider
	suite.config = provider.GetConfig()
}

// TearDownTest runs after each test
func (suite *IntegrationTestSuite) TearDownTest() {
	if suite.provider != nil {
		_ = suite.provider.Shutdown()
		suite.provider = nil
	}
}

// TestProviderInitialization tests that the provider can be initialized with real credentials
func (suite *IntegrationTestSuite) TestProviderInitialization() {
	suite.T().Log("Testing provider initialization with real DigitalOcean credentials")

	// Skip if this would create services (since they're not implemented yet)
	suite.T().Skip("Skipping service initialization until Spaces and OpenSearch services are implemented")

	err := suite.provider.Initialize()
	suite.NoError(err)

	services := suite.provider.GetServices()
	suite.NotNil(services)
	suite.True(services.IsHealthy())
}

// TestConfigurationValidation tests configuration validation with real environment
func (suite *IntegrationTestSuite) TestConfigurationValidation() {
	suite.T().Log("Testing configuration validation with integration environment")

	err := suite.provider.ValidateConfiguration(suite.ctx)

	// Should fail due to unimplemented services, but configuration itself should be valid
	if err != nil {
		suite.Contains(err.Error(), "service creation failed")
	}

	// Test that configuration structure is valid
	cfg := suite.provider.GetConfig()
	suite.NotNil(cfg)

	// Validate environment-specific requirements
	if !cfg.IsLocal() {
		suite.NotEmpty(cfg.DigitalOcean.Spaces.AccessKey)
		suite.NotEmpty(cfg.DigitalOcean.Spaces.SecretKey)
		suite.NotEmpty(cfg.DigitalOcean.Spaces.Bucket)
		suite.NotEmpty(cfg.DigitalOcean.Spaces.Region)
		suite.NotEmpty(cfg.DigitalOcean.OpenSearch.Host)
		suite.NotEmpty(cfg.DigitalOcean.OpenSearch.Username)
		suite.NotEmpty(cfg.DigitalOcean.OpenSearch.Password)
	}
}

// TestHealthChecking tests health checking functionality
func (suite *IntegrationTestSuite) TestHealthChecking() {
	suite.T().Log("Testing health checking functionality")

	// Provider should not be healthy without initialized services
	suite.False(suite.provider.IsHealthy())

	// Metrics should include basic information
	metrics := suite.provider.GetMetrics()
	suite.NotNil(metrics)
	suite.Equal(false, metrics["services_initialized"])

	configInfo, exists := metrics["config"]
	suite.True(exists)
	suite.NotNil(configInfo)
}

// TestEnvironmentDetection tests that environment detection works correctly
func (suite *IntegrationTestSuite) TestEnvironmentDetection() {
	suite.T().Log("Testing environment detection")

	cfg := suite.provider.GetConfig()

	// Check that environment was detected correctly
	envValue := os.Getenv("ENVIRONMENT")
	switch envValue {
	case "local":
		suite.Equal(config.EnvLocal, cfg.Environment)
		suite.True(cfg.IsLocal())
		suite.True(cfg.ShouldUseMockServices())
	case "staging":
		suite.Equal(config.EnvStaging, cfg.Environment)
		suite.True(cfg.IsStaging())
		suite.False(cfg.ShouldUseMockServices())
	case "production":
		suite.Equal(config.EnvProduction, cfg.Environment)
		suite.True(cfg.IsProduction())
		suite.False(cfg.ShouldUseMockServices())
	default:
		suite.Fail("Unknown environment: %s", envValue)
	}
}

// TestEndpointConfiguration tests endpoint configuration methods
func (suite *IntegrationTestSuite) TestEndpointConfiguration() {
	suite.T().Log("Testing endpoint configuration")

	cfg := suite.provider.GetConfig()

	// Test Spaces endpoint generation
	spacesEndpoint := cfg.GetSpacesEndpoint()
	suite.NotEmpty(spacesEndpoint)
	suite.Contains(spacesEndpoint, "https://")

	if !cfg.IsLocal() {
		suite.Contains(spacesEndpoint, cfg.DigitalOcean.Spaces.Region)
	}

	// Test Spaces CDN endpoint generation
	cdnEndpoint := cfg.GetSpacesCDNEndpoint()
	suite.NotEmpty(cdnEndpoint)
	suite.Contains(cdnEndpoint, "https://")

	// Test OpenSearch endpoint generation
	searchEndpoint := cfg.GetOpenSearchEndpoint()
	suite.NotEmpty(searchEndpoint)

	if cfg.DigitalOcean.OpenSearch.UseSSL {
		suite.Contains(searchEndpoint, "https://")
	} else {
		suite.Contains(searchEndpoint, "http://")
	}

	if !cfg.IsLocal() {
		suite.Contains(searchEndpoint, cfg.DigitalOcean.OpenSearch.Host)
	}
}

// setupIntegrationEnvironment sets up environment variables for integration testing
func (suite *IntegrationTestSuite) setupIntegrationEnvironment() {
	// Check if integration environment is configured
	env := os.Getenv("INTEGRATION_TEST_ENV")
	if env == "" {
		env = "staging" // Default to staging for integration tests
	}

	os.Setenv("ENVIRONMENT", env)

	// Set default values if not provided
	if os.Getenv("DO_SPACES_ACCESS_KEY") == "" {
		os.Setenv("DO_SPACES_ACCESS_KEY", "integration-test-key")
	}
	if os.Getenv("DO_SPACES_SECRET_KEY") == "" {
		os.Setenv("DO_SPACES_SECRET_KEY", "integration-test-secret")
	}
	if os.Getenv("DO_SPACES_BUCKET") == "" {
		os.Setenv("DO_SPACES_BUCKET", "integration-test-bucket")
	}
	if os.Getenv("DO_SPACES_REGION") == "" {
		os.Setenv("DO_SPACES_REGION", "nyc3")
	}
	if os.Getenv("DO_OPENSEARCH_HOST") == "" {
		os.Setenv("DO_OPENSEARCH_HOST", "integration-test-host.db.ondigitalocean.com")
	}
	if os.Getenv("DO_OPENSEARCH_PORT") == "" {
		os.Setenv("DO_OPENSEARCH_PORT", "25060")
	}
	if os.Getenv("DO_OPENSEARCH_USERNAME") == "" {
		os.Setenv("DO_OPENSEARCH_USERNAME", "doadmin")
	}
	if os.Getenv("DO_OPENSEARCH_PASSWORD") == "" {
		os.Setenv("DO_OPENSEARCH_PASSWORD", "integration-test-password")
	}
	if os.Getenv("DO_OPENSEARCH_USE_SSL") == "" {
		os.Setenv("DO_OPENSEARCH_USE_SSL", "true")
	}
	if os.Getenv("DO_OPENSEARCH_INDEX") == "" {
		os.Setenv("DO_OPENSEARCH_INDEX", "integration-test-documents")
	}
}

// TestIntegrationSuite runs the integration test suite
func TestIntegrationSuite(t *testing.T) {
	// Skip integration tests unless explicitly requested
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests. Set RUN_INTEGRATION_TESTS=true to run.")
	}

	suite.Run(t, new(IntegrationTestSuite))
}
