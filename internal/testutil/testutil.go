package testutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"motion-index-fiber/internal/config"
	doConfig "motion-index-fiber/pkg/cloud/digitalocean/config"
)

// UNIX Principle: Do one thing and do it well
// Each function in this package has a single, clear responsibility

// TempDir creates a temporary directory for testing
// Returns the path and a cleanup function
func TempDir(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "motion-index-test-*")
	require.NoError(t, err, "failed to create temp directory")

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return dir, cleanup
}

// TempFile creates a temporary file with given content
// Returns the file path and a cleanup function
func TempFile(t *testing.T, name, content string) (string, func()) {
	t.Helper()

	dir, dirCleanup := TempDir(t)
	filePath := filepath.Join(dir, name)

	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err, "failed to write temp file")

	cleanup := func() {
		dirCleanup()
	}

	return filePath, cleanup
}

// SetEnv sets environment variables for testing and returns a cleanup function
// UNIX Principle: Composition - combine small tools to build larger functionality
func SetEnv(t *testing.T, envVars map[string]string) func() {
	t.Helper()

	originalValues := make(map[string]string)
	originalExists := make(map[string]bool)

	// Store original values
	for key := range envVars {
		if val, exists := os.LookupEnv(key); exists {
			originalValues[key] = val
			originalExists[key] = true
		} else {
			originalExists[key] = false
		}
	}

	// Set new values
	for key, value := range envVars {
		os.Setenv(key, value)
	}

	// Return cleanup function
	return func() {
		for key := range envVars {
			if originalExists[key] {
				os.Setenv(key, originalValues[key])
			} else {
				os.Unsetenv(key)
			}
		}
	}
}

// TestConfig returns a minimal test configuration
// UNIX Principle: Simplicity - provide sensible defaults for testing
func TestConfig() *config.Config {
	return &config.Config{
		Environment: "local",
		Server: config.ServerConfig{
			Port:           "8080",
			Production:     false,
			AllowedOrigins: "http://localhost:3000",
			MaxRequestSize: 10 * 1024 * 1024, // 10MB
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Username: "test",
			Password: "test",
			Database: "test_db",
			UseSSL:   false,
		},
		Storage: config.StorageConfig{
			Backend:   "local",
			AccessKey: "test-key",
			SecretKey: "test-secret",
			Bucket:    "test-bucket",
			Region:    "nyc3",
			CDNDomain: "",
		},
		Auth: config.AuthConfig{
			JWTSecret:       "test-secret",
			SupabaseURL:     "https://test.supabase.co",
			SupabaseAnonKey: "test-anon-key",
			SupabaseAPIKey:  "test-api-key",
		},
		Processing: config.ProcessingConfig{
			MaxFileSize:    50 * 1024 * 1024, // 50MB
			MaxWorkers:     2,
			BatchSize:      10,
			ProcessTimeout: 30 * time.Second,
		},
		OpenSearch: config.OpenSearchConfig{
			Host:     "localhost",
			Port:     9200,
			Username: "admin",
			Password: "admin",
			UseSSL:   false,
			Index:    "test_documents",
		},
		OpenAI: config.OpenAIConfig{
			APIKey: "test-openai-key",
			Model:  "gpt-4",
		},
		DigitalOcean: nil, // Will be set separately when needed
	}
}

// TestDigitalOceanConfig returns a test DigitalOcean configuration
func TestDigitalOceanConfig() *doConfig.Config {
	cfg := doConfig.DefaultConfig()

	// Override with test values
	cfg.DigitalOcean.Spaces.AccessKey = "test-spaces-key"
	cfg.DigitalOcean.Spaces.SecretKey = "test-spaces-secret"
	cfg.DigitalOcean.Spaces.Bucket = "test-bucket"
	cfg.DigitalOcean.Spaces.Region = "nyc3"

	cfg.DigitalOcean.OpenSearch.Host = "localhost"
	cfg.DigitalOcean.OpenSearch.Port = 9200
	cfg.DigitalOcean.OpenSearch.Username = "admin"
	cfg.DigitalOcean.OpenSearch.Password = "admin"
	cfg.DigitalOcean.OpenSearch.UseSSL = false
	cfg.DigitalOcean.OpenSearch.Index = "test_documents"

	return cfg
}

// ConfigWithDigitalOcean returns a test config with DigitalOcean configuration
func ConfigWithDigitalOcean() *config.Config {
	cfg := TestConfig()
	cfg.DigitalOcean = TestDigitalOceanConfig()
	return cfg
}

// AssertEventuallyTrue waits for a condition to become true within a timeout
// UNIX Principle: Robustness - handle timing issues in tests gracefully
func AssertEventuallyTrue(t *testing.T, condition func() bool, timeout time.Duration, msgAndArgs ...interface{}) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}

		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				require.Fail(t, "condition never became true", msgAndArgs...)
				return
			}
		}
	}
}

// SkipIfShort skips a test if running in short mode
// UNIX Principle: Modularity - allow tests to be run selectively
func SkipIfShort(t *testing.T, reason string) {
	t.Helper()
	if testing.Short() {
		t.Skipf("Skipping in short mode: %s", reason)
	}
}

// SkipIfCI skips a test if running in CI environment
func SkipIfCI(t *testing.T, reason string) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Skipf("Skipping in CI: %s", reason)
	}
}

// TestTimeout provides a standard timeout for tests
const TestTimeout = 30 * time.Second

// ShortTestTimeout provides a shorter timeout for unit tests
const ShortTestTimeout = 5 * time.Second
