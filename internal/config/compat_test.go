package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadNew_Success(t *testing.T) {
	// Set up test environment
	cleanup := setTestEnv(t, map[string]string{
		"PORT":                     "8003",
		"ENVIRONMENT":              "local",
		"JWT_SECRET":               "test-secret",
		"SUPABASE_URL":             "https://test.supabase.co",
		"SUPABASE_ANON_KEY":        "test-anon-key",
		"SUPABASE_SERVICE_KEY":     "test-service-key",
		"OPENSEARCH_HOST":          "localhost",
		"OPENSEARCH_PORT":          "9200",
		"OPENSEARCH_USERNAME":      "admin",
		"OPENSEARCH_PASSWORD":      "admin",
		"STORAGE_BACKEND":          "local",
		"DO_API_TOKEN":             "test-token",
		"DO_SPACES_ACCESS_KEY":     "test-access",
		"DO_SPACES_SECRET_KEY":     "test-secret",
		"DO_SPACES_BUCKET":         "test-bucket",
		"DO_SPACES_REGION":         "nyc3",
		"DO_OPENSEARCH_HOST":       "localhost",
		"DO_OPENSEARCH_PORT":       "9200",
		"DO_OPENSEARCH_USERNAME":   "admin",
		"DO_OPENSEARCH_PASSWORD":   "admin",
		"DO_OPENSEARCH_INDEX":      "documents",
	})
	defer cleanup()

	cfg, err := LoadNew()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify server config
	assert.Equal(t, "8003", cfg.Server.Port)
	assert.Equal(t, "local", cfg.Environment)

	// Verify auth config
	assert.Equal(t, "test-secret", cfg.Auth.JWTSecret)
	assert.Equal(t, "https://test.supabase.co", cfg.Auth.SupabaseURL)

	// Verify OpenSearch config
	assert.Equal(t, "localhost", cfg.OpenSearch.Host)
	assert.Equal(t, 9200, cfg.OpenSearch.Port)

	// Verify storage config
	assert.Equal(t, "local", cfg.Storage.Backend)
}

func TestLoad_UsesNewSystem(t *testing.T) {
	// Set up test environment
	cleanup := setTestEnv(t, map[string]string{
		"PORT":                     "8003",
		"ENVIRONMENT":              "local",
		"JWT_SECRET":               "test-secret",
		"SUPABASE_URL":             "https://test.supabase.co",
		"SUPABASE_ANON_KEY":        "test-anon-key",
		"SUPABASE_SERVICE_KEY":     "test-service-key",
		"OPENSEARCH_HOST":          "localhost",
		"OPENSEARCH_PORT":          "9200",
		"OPENSEARCH_USERNAME":      "admin",
		"OPENSEARCH_PASSWORD":      "admin",
		"STORAGE_BACKEND":          "local",
		"DO_API_TOKEN":             "test-token",
		"DO_SPACES_ACCESS_KEY":     "test-access",
		"DO_SPACES_SECRET_KEY":     "test-secret",
		"DO_SPACES_BUCKET":         "test-bucket",
		"DO_SPACES_REGION":         "nyc3",
		"DO_OPENSEARCH_HOST":       "localhost",
		"DO_OPENSEARCH_PORT":       "9200",
		"DO_OPENSEARCH_USERNAME":   "admin",
		"DO_OPENSEARCH_PASSWORD":   "admin",
		"DO_OPENSEARCH_INDEX":      "documents",
	})
	defer cleanup()

	// Old Load() function should now use new system
	cfg, err := Load()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "8003", cfg.Server.Port)
	assert.Equal(t, "local", cfg.Environment)
}

func TestConvertToOldConfig_WithAI(t *testing.T) {
	cleanup := setTestEnv(t, map[string]string{
		"PORT":                     "8003",
		"ENVIRONMENT":              "local",
		"JWT_SECRET":               "test-secret",
		"SUPABASE_URL":             "https://test.supabase.co",
		"SUPABASE_ANON_KEY":        "test-anon-key",
		"SUPABASE_SERVICE_KEY":     "test-service-key",
		"OPENSEARCH_HOST":          "localhost",
		"OPENSEARCH_PORT":          "9200",
		"OPENSEARCH_USERNAME":      "admin",
		"OPENSEARCH_PASSWORD":      "admin",
		"STORAGE_BACKEND":          "local",
		"DO_API_TOKEN":             "test-token",
		"DO_SPACES_ACCESS_KEY":     "test-access",
		"DO_SPACES_SECRET_KEY":     "test-secret",
		"DO_SPACES_BUCKET":         "test-bucket",
		"DO_SPACES_REGION":         "nyc3",
		"DO_OPENSEARCH_HOST":       "localhost",
		"DO_OPENSEARCH_PORT":       "9200",
		"DO_OPENSEARCH_USERNAME":   "admin",
		"DO_OPENSEARCH_PASSWORD":   "admin",
		"DO_OPENSEARCH_INDEX":      "documents",
		"OPENAI_API_KEY":           "test-openai-key",
		"OPENAI_MODEL":             "gpt-4",
		"CLAUDE_API_KEY":           "test-claude-key",
		"CLAUDE_MODEL":             "claude-3-5-sonnet-20241022",
	})
	defer cleanup()

	cfg, err := LoadNew()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify AI config was converted
	assert.Equal(t, "test-openai-key", cfg.AI.OpenAI.APIKey)
	assert.Equal(t, "gpt-4", cfg.AI.OpenAI.Model)
	assert.Equal(t, "test-claude-key", cfg.AI.Claude.APIKey)
	assert.Equal(t, "claude-3-5-sonnet-20241022", cfg.AI.Claude.Model)
}

func TestConfig_HelperMethods(t *testing.T) {
	cleanup := setTestEnv(t, map[string]string{
		"PORT":                     "8003",
		"ENVIRONMENT":              "production",
		"JWT_SECRET":               "test-secret",
		"SUPABASE_URL":             "https://test.supabase.co",
		"SUPABASE_ANON_KEY":        "test-anon-key",
		"SUPABASE_SERVICE_KEY":     "test-service-key",
		"OPENSEARCH_HOST":          "localhost",
		"OPENSEARCH_PORT":          "9200",
		"OPENSEARCH_USERNAME":      "admin",
		"OPENSEARCH_PASSWORD":      "admin",
		"STORAGE_BACKEND":          "spaces",
		"DO_API_TOKEN":             "test-token",
		"DO_SPACES_ACCESS_KEY":     "test-access",
		"DO_SPACES_SECRET_KEY":     "test-secret",
		"DO_SPACES_BUCKET":         "test-bucket",
		"DO_SPACES_REGION":         "nyc3",
		"DO_OPENSEARCH_HOST":       "localhost",
		"DO_OPENSEARCH_PORT":       "9200",
		"DO_OPENSEARCH_USERNAME":   "admin",
		"DO_OPENSEARCH_PASSWORD":   "admin",
		"DO_OPENSEARCH_USE_SSL":    "true",
		"DO_OPENSEARCH_INDEX":      "documents",
	})
	defer cleanup()

	cfg, err := Load()
	assert.NoError(t, err)

	// Test helper methods still work
	assert.True(t, cfg.IsProduction())
	assert.False(t, cfg.IsLocal())
	assert.Equal(t, "https://localhost:9200", cfg.GetOpenSearchURL())
}
