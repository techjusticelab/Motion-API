package configuration

import (
	"testing"

	"motion-index-fiber/internal/configuration/auth"
	"motion-index-fiber/internal/configuration/server"

	"github.com/stretchr/testify/assert"
)

func TestBuilder_Build_Success(t *testing.T) {
	serverCfg, err := server.NewConfig("8080")
	assert.NoError(t, err)

	authCfg, err := auth.NewConfig(
		"test-secret",
		"https://test.supabase.co",
		"test-anon-key",
		"test-api-key",
	)
	assert.NoError(t, err)

	cfg, err := NewBuilder().
		WithServer(serverCfg).
		WithAuth(authCfg).
		WithEnvironment("local").
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "local", cfg.Environment())
	assert.True(t, cfg.IsLocal())
	assert.False(t, cfg.IsProduction())
}

func TestBuilder_Build_MissingServer(t *testing.T) {
	authCfg, _ := auth.NewConfig(
		"test-secret",
		"https://test.supabase.co",
		"test-anon-key",
		"test-api-key",
	)

	_, err := NewBuilder().
		WithAuth(authCfg).
		Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server configuration is required")
}

func TestBuilder_Build_MissingAuth(t *testing.T) {
	serverCfg, _ := server.NewConfig("8080")

	_, err := NewBuilder().
		WithServer(serverCfg).
		Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth configuration is required")
}

func TestBuilder_Build_InvalidEnvironment(t *testing.T) {
	serverCfg, _ := server.NewConfig("8080")
	authCfg, _ := auth.NewConfig(
		"test-secret",
		"https://test.supabase.co",
		"test-anon-key",
		"test-api-key",
	)

	_, err := NewBuilder().
		WithServer(serverCfg).
		WithAuth(authCfg).
		WithEnvironment("invalid").
		Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid environment")
}

func TestConfig_IsProduction(t *testing.T) {
	serverCfg, _ := server.NewConfig("8080")
	authCfg, _ := auth.NewConfig(
		"test-secret",
		"https://test.supabase.co",
		"test-anon-key",
		"test-api-key",
	)

	// Test production environment
	cfg, err := NewBuilder().
		WithServer(serverCfg).
		WithAuth(authCfg).
		WithEnvironment("production").
		Build()

	assert.NoError(t, err)
	assert.True(t, cfg.IsProduction())
	assert.False(t, cfg.IsLocal())
}
