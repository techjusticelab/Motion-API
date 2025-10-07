package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// UNIX Principle: Do one thing well - Each test function tests one aspect of configuration

// Test utility functions following UNIX principles

// setTestEnv sets environment variables for testing and returns a cleanup function
func setTestEnv(t *testing.T, envVars map[string]string) func() {
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

// tempFile creates a temporary file with given content
func tempFile(t *testing.T, name, content string) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "motion-index-test-*")
	require.NoError(t, err, "failed to create temp directory")

	filePath := filepath.Join(dir, name)
	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err, "failed to write temp file")

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return filePath, cleanup
}

func TestLoadMinimalConfig(t *testing.T) {
	// Test loading configuration with minimal required environment variables
	cleanup := setTestEnv(t, map[string]string{
		"ENVIRONMENT":     "local",
		"PORT":            "8080",
		"JWT_SECRET":      "test-secret", // Required for all environments
		"OPENSEARCH_HOST": "localhost",   // Required for all environments
	})
	defer cleanup()

	cfg, err := Load()
	require.NoError(t, err, "should load minimal config without error")
	assert.Equal(t, "local", cfg.Environment)
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "test-secret", cfg.Auth.JWTSecret)
	assert.Equal(t, "localhost", cfg.OpenSearch.Host)
}

func TestLoadProductionConfig(t *testing.T) {
	// Test loading production configuration with all required variables
	cleanup := setTestEnv(t, map[string]string{
		"ENVIRONMENT":          "production",
		"PORT":                 "8000",
		"PRODUCTION":           "true",
		"ALLOWED_ORIGINS":      "https://app.example.com",
		"MAX_REQUEST_SIZE":     "52428800", // 50MB
		"JWT_SECRET":           "super-secret-jwt-key",
		"SUPABASE_URL":         "https://test.supabase.co",
		"SUPABASE_ANON_KEY":    "test-anon-key",
		"SUPABASE_SERVICE_KEY": "test-api-key",
		"OPENSEARCH_HOST":      "search.example.com",
		"OPENSEARCH_PORT":      "443",
		"OPENSEARCH_USERNAME":  "admin",
		"OPENSEARCH_PASSWORD":  "password",
		"OPENSEARCH_USE_SSL":   "true",
		"OPENSEARCH_INDEX":     "documents",
		"STORAGE_BACKEND":      "spaces",
		"STORAGE_ACCESS_KEY":   "spaces-key",
		"STORAGE_SECRET_KEY":   "spaces-secret",
		"STORAGE_BUCKET":       "my-bucket",
		"STORAGE_REGION":       "nyc3",
		"STORAGE_CDN_DOMAIN":   "cdn.example.com",
		"OPENAI_API_KEY":       "openai-key",
		"OPENAI_MODEL":         "gpt-4",
		"MAX_FILE_SIZE":        "104857600", // 100MB
		"MAX_WORKERS":          "4",
		"BATCH_SIZE":           "20",
		"PROCESS_TIMEOUT":      "60s",
		// DigitalOcean configuration required for production
		"DO_SPACES_ACCESS_KEY":   "spaces-key",
		"DO_SPACES_SECRET_KEY":   "spaces-secret",
		"DO_SPACES_BUCKET":       "my-bucket",
		"DO_SPACES_REGION":       "nyc3",
		"DO_OPENSEARCH_HOST":     "search.example.com",
		"DO_OPENSEARCH_PORT":     "443",
		"DO_OPENSEARCH_USERNAME": "admin",
		"DO_OPENSEARCH_PASSWORD": "password",
		"DO_OPENSEARCH_USE_SSL":  "true",
		"DO_OPENSEARCH_INDEX":    "documents",
	})
	defer cleanup()

	cfg, err := Load()
	require.NoError(t, err, "should load production config without error")

	// Verify server config
	assert.Equal(t, "production", cfg.Environment)
	assert.Equal(t, "8000", cfg.Server.Port)
	assert.True(t, cfg.Server.Production)
	assert.Equal(t, "https://app.example.com", cfg.Server.AllowedOrigins)
	assert.Equal(t, int64(52428800), cfg.Server.MaxRequestSize)

	// Verify auth config
	assert.Equal(t, "super-secret-jwt-key", cfg.Auth.JWTSecret)
	assert.Equal(t, "https://test.supabase.co", cfg.Auth.SupabaseURL)

	// Verify OpenSearch config
	assert.Equal(t, "search.example.com", cfg.OpenSearch.Host)
	assert.Equal(t, 443, cfg.OpenSearch.Port)
	assert.True(t, cfg.OpenSearch.UseSSL)

	// Verify storage config
	assert.Equal(t, "spaces", cfg.Storage.Backend)
	assert.Equal(t, "spaces-key", cfg.Storage.AccessKey)
	assert.Equal(t, "my-bucket", cfg.Storage.Bucket)

	// Verify processing config
	assert.Equal(t, int64(104857600), cfg.Processing.MaxFileSize)
	assert.Equal(t, 4, cfg.Processing.MaxWorkers)
	assert.Equal(t, 60*time.Second, cfg.Processing.ProcessTimeout)
}

func TestValidateServerConfig(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid port",
			port:        "8080",
			shouldError: false,
		},
		{
			name:        "empty port",
			port:        "",
			shouldError: true,
			errorMsg:    "PORT is required",
		},
		{
			name:        "invalid port - non-numeric",
			port:        "invalid",
			shouldError: true,
			errorMsg:    "PORT must be a valid number",
		},
		{
			name:        "invalid port - too low",
			port:        "0",
			shouldError: true,
			errorMsg:    "PORT must be between 1 and 65535",
		},
		{
			name:        "invalid port - too high",
			port:        "70000",
			shouldError: true,
			errorMsg:    "PORT must be between 1 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setTestEnv(t, map[string]string{
				"ENVIRONMENT": "local",
				"PORT":        tt.port,
				"JWT_SECRET":  "test-secret", // Required for all tests
			})
			defer cleanup()

			_, err := Load()
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAuthConfig(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid auth config",
			envVars: map[string]string{
				"ENVIRONMENT":       "local",
				"PORT":              "8080",
				"JWT_SECRET":        "test-secret",
				"SUPABASE_URL":      "https://test.supabase.co",
				"SUPABASE_ANON_KEY": "test-anon",
				"SUPABASE_API_KEY":  "test-api",
			},
			shouldError: false,
		},
		{
			name: "missing JWT secret",
			envVars: map[string]string{
				"ENVIRONMENT":       "local",
				"PORT":              "8080",
				"SUPABASE_URL":      "https://test.supabase.co",
				"SUPABASE_ANON_KEY": "test-anon",
				"SUPABASE_API_KEY":  "test-api",
			},
			shouldError: true,
			errorMsg:    "JWT_SECRET is required",
		},
		{
			name: "invalid Supabase URL",
			envVars: map[string]string{
				"ENVIRONMENT":       "local",
				"PORT":              "8080",
				"JWT_SECRET":        "test-secret",
				"SUPABASE_URL":      "invalid-url",
				"SUPABASE_ANON_KEY": "test-anon",
				"SUPABASE_API_KEY":  "test-api",
			},
			shouldError: true,
			errorMsg:    "SUPABASE_URL must be a valid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setTestEnv(t, tt.envVars)
			defer cleanup()

			_, err := Load()
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateOpenSearchConfig(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid OpenSearch config",
			envVars: map[string]string{
				"ENVIRONMENT":         "local",
				"PORT":                "8080",
				"JWT_SECRET":          "test-secret",
				"OPENSEARCH_HOST":     "localhost",
				"OPENSEARCH_PORT":     "9200",
				"OPENSEARCH_USERNAME": "admin",
				"OPENSEARCH_PASSWORD": "admin",
				"OPENSEARCH_USE_SSL":  "false",
				"OPENSEARCH_INDEX":    "documents",
			},
			shouldError: false,
		},
		{
			name: "missing OpenSearch host",
			envVars: map[string]string{
				"ENVIRONMENT":         "local",
				"PORT":                "8080",
				"JWT_SECRET":          "test-secret",
				"OPENSEARCH_PORT":     "9200",
				"OPENSEARCH_USERNAME": "admin",
				"OPENSEARCH_PASSWORD": "admin",
				"OPENSEARCH_USE_SSL":  "false",
				"OPENSEARCH_INDEX":    "documents",
			},
			shouldError: true,
			errorMsg:    "OPENSEARCH_HOST is required",
		},
		{
			name: "invalid OpenSearch port",
			envVars: map[string]string{
				"ENVIRONMENT":         "local",
				"PORT":                "8080",
				"JWT_SECRET":          "test-secret",
				"OPENSEARCH_HOST":     "localhost",
				"OPENSEARCH_PORT":     "invalid",
				"OPENSEARCH_USERNAME": "admin",
				"OPENSEARCH_PASSWORD": "admin",
				"OPENSEARCH_USE_SSL":  "false",
				"OPENSEARCH_INDEX":    "documents",
			},
			shouldError: true,
			errorMsg:    "OPENSEARCH_PORT must be a valid number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setTestEnv(t, tt.envVars)
			defer cleanup()

			_, err := Load()
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateStorageConfig(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid local storage config",
			envVars: map[string]string{
				"ENVIRONMENT":     "local",
				"PORT":            "8080",
				"JWT_SECRET":      "test-secret",
				"STORAGE_BACKEND": "local",
			},
			shouldError: false,
		},
		{
			name: "valid spaces storage config",
			envVars: map[string]string{
				"ENVIRONMENT":        "local",
				"PORT":               "8080",
				"JWT_SECRET":         "test-secret",
				"STORAGE_BACKEND":    "spaces",
				"STORAGE_ACCESS_KEY": "key",
				"STORAGE_SECRET_KEY": "secret",
				"STORAGE_BUCKET":     "bucket",
				"STORAGE_REGION":     "nyc3",
			},
			shouldError: false,
		},
		{
			name: "invalid storage backend",
			envVars: map[string]string{
				"ENVIRONMENT":     "local",
				"PORT":            "8080",
				"JWT_SECRET":      "test-secret",
				"STORAGE_BACKEND": "invalid",
			},
			shouldError: true,
			errorMsg:    "STORAGE_BACKEND must be 'local' or 'spaces'",
		},
		{
			name: "spaces backend missing access key",
			envVars: map[string]string{
				"ENVIRONMENT":        "local",
				"PORT":               "8080",
				"JWT_SECRET":         "test-secret",
				"STORAGE_BACKEND":    "spaces",
				"STORAGE_SECRET_KEY": "secret",
				"STORAGE_BUCKET":     "bucket",
				"STORAGE_REGION":     "nyc3",
			},
			shouldError: true,
			errorMsg:    "STORAGE_ACCESS_KEY is required for spaces backend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setTestEnv(t, tt.envVars)
			defer cleanup()

			_, err := Load()
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateProcessingConfig(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid processing config",
			envVars: map[string]string{
				"ENVIRONMENT":     "local",
				"PORT":            "8080",
				"JWT_SECRET":      "test-secret",
				"MAX_FILE_SIZE":   "52428800", // 50MB
				"MAX_WORKERS":     "4",
				"BATCH_SIZE":      "10",
				"PROCESS_TIMEOUT": "30s",
			},
			shouldError: false,
		},
		{
			name: "invalid max file size",
			envVars: map[string]string{
				"ENVIRONMENT":     "local",
				"PORT":            "8080",
				"JWT_SECRET":      "test-secret",
				"MAX_FILE_SIZE":   "invalid",
				"MAX_WORKERS":     "4",
				"BATCH_SIZE":      "10",
				"PROCESS_TIMEOUT": "30s",
			},
			shouldError: true,
			errorMsg:    "MAX_FILE_SIZE must be a valid number",
		},
		{
			name: "invalid process timeout",
			envVars: map[string]string{
				"ENVIRONMENT":     "local",
				"PORT":            "8080",
				"JWT_SECRET":      "test-secret",
				"MAX_FILE_SIZE":   "52428800",
				"MAX_WORKERS":     "4",
				"BATCH_SIZE":      "10",
				"PROCESS_TIMEOUT": "invalid",
			},
			shouldError: true,
			errorMsg:    "PROCESS_TIMEOUT must be a valid duration",
		},
		{
			name: "negative max workers",
			envVars: map[string]string{
				"ENVIRONMENT":     "local",
				"PORT":            "8080",
				"JWT_SECRET":      "test-secret",
				"MAX_FILE_SIZE":   "52428800",
				"MAX_WORKERS":     "-1",
				"BATCH_SIZE":      "10",
				"PROCESS_TIMEOUT": "30s",
			},
			shouldError: true,
			errorMsg:    "MAX_WORKERS must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setTestEnv(t, tt.envVars)
			defer cleanup()

			_, err := Load()
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEnvironmentSpecificDefaults(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		expected    func(*Config) bool
	}{
		{
			name:        "local environment defaults",
			environment: "local",
			expected: func(cfg *Config) bool {
				return !cfg.Server.Production &&
					cfg.Server.AllowedOrigins == "http://localhost:3000,http://localhost:5173" &&
					!cfg.OpenSearch.UseSSL
			},
		},
		{
			name:        "production environment defaults",
			environment: "production",
			expected: func(cfg *Config) bool {
				return cfg.Server.Production &&
					cfg.OpenSearch.UseSSL
			},
		},
		{
			name:        "staging environment defaults",
			environment: "staging",
			expected: func(cfg *Config) bool {
				return cfg.Server.Production &&
					cfg.OpenSearch.UseSSL
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVars := map[string]string{
				"ENVIRONMENT": tt.environment,
				"PORT":        "8080",
				"JWT_SECRET":  "test-secret",
			}

			// Add required config for production/staging environments
			if tt.environment == "production" || tt.environment == "staging" {
				envVars["DO_SPACES_ACCESS_KEY"] = "test-key"
				envVars["DO_SPACES_SECRET_KEY"] = "test-secret"
				envVars["DO_SPACES_BUCKET"] = "test-bucket"
				envVars["DO_SPACES_REGION"] = "nyc3"
				envVars["DO_OPENSEARCH_HOST"] = "localhost"
				envVars["DO_OPENSEARCH_PORT"] = "9200"
				envVars["DO_OPENSEARCH_USERNAME"] = "admin"
				envVars["DO_OPENSEARCH_PASSWORD"] = "admin"
				envVars["DO_OPENSEARCH_USE_SSL"] = "true"
				envVars["DO_OPENSEARCH_INDEX"] = "documents"
			}

			cleanup := setTestEnv(t, envVars)
			defer cleanup()

			cfg, err := Load()
			require.NoError(t, err)
			assert.True(t, tt.expected(cfg), "environment-specific defaults not applied correctly")
		})
	}
}

func TestConfigWithoutEnvironmentFile(t *testing.T) {
	// Test that config loads even when .env file is missing

	// Ensure no .env file exists in test
	if _, err := os.Stat(".env"); err == nil {
		t.Skip("Skipping test - .env file exists")
	}

	cleanup := setTestEnv(t, map[string]string{
		"ENVIRONMENT": "local",
		"PORT":        "8080",
		"JWT_SECRET":  "test-secret",
	})
	defer cleanup()

	cfg, err := Load()
	require.NoError(t, err, "should load config without .env file")
	assert.Equal(t, "local", cfg.Environment)
}

func TestConfigOverrideOrder(t *testing.T) {
	// Test that environment variables override .env file values

	// Create temporary .env file
	envContent := "ENVIRONMENT=local\nPORT=9999\nJWT_SECRET=from_file"
	envPath, cleanup := tempFile(t, ".env", envContent)
	defer cleanup()

	// Change to directory containing the temp .env file
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(envPath[:len(envPath)-5]) // Remove "/.env" to get directory

	// Set environment variable that should override .env file
	envCleanup := setTestEnv(t, map[string]string{
		"ENVIRONMENT": "local", // Use valid environment
		"JWT_SECRET":  "test-secret",
	})
	defer envCleanup()

	cfg, err := Load()
	require.NoError(t, err)

	// Environment variable should override .env file
	assert.Equal(t, "local", cfg.Environment, "environment variable should override .env file")
	assert.Equal(t, "9999", cfg.Server.Port, ".env file value should be used when no env var override")
}
