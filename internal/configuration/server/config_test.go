package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig_Success(t *testing.T) {
	cfg, err := NewConfig("8080",
		WithHost("localhost"),
		WithReadTimeout(10*time.Second),
		WithWriteTimeout(10*time.Second),
		WithAllowedOrigins([]string{"http://localhost:3000"}),
		WithMaxRequestSize(50*1024*1024),
		WithProduction(false),
	)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "8080", cfg.Port())
	assert.Equal(t, "localhost", cfg.Host())
	assert.Equal(t, 10*time.Second, cfg.ReadTimeout())
	assert.Equal(t, 10*time.Second, cfg.WriteTimeout())
	assert.Equal(t, []string{"http://localhost:3000"}, cfg.AllowedOrigins())
	assert.Equal(t, int64(50*1024*1024), cfg.MaxRequestSize())
	assert.False(t, cfg.Production())
}

func TestNewConfig_InvalidPort(t *testing.T) {
	_, err := NewConfig("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port is required")
}

func TestNewConfig_InvalidPortNumber(t *testing.T) {
	_, err := NewConfig("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a valid number")
}

func TestNewConfig_PortOutOfRange(t *testing.T) {
	_, err := NewConfig("999999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "between 1 and 65535")
}

func TestConfig_Address(t *testing.T) {
	cfg, err := NewConfig("8080", WithHost("0.0.0.0"))
	assert.NoError(t, err)
	assert.Equal(t, "0.0.0.0:8080", cfg.Address())
}

func TestConfig_AllowedOriginsString(t *testing.T) {
	cfg, err := NewConfig("8080",
		WithAllowedOrigins([]string{"http://localhost:3000", "http://localhost:5173"}),
	)
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:3000,http://localhost:5173", cfg.AllowedOriginsString())
}
