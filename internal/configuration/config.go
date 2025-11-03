package configuration

import (
	"errors"
	"fmt"

	"motion-index-fiber/internal/configuration/ai"
	"motion-index-fiber/internal/configuration/auth"
	"motion-index-fiber/internal/configuration/cloud"
	"motion-index-fiber/internal/configuration/processing"
	"motion-index-fiber/internal/configuration/server"
)

// Config is the root configuration object that combines all config modules.
type Config struct {
	server      *server.Config
	auth        *auth.Config
	cloud       *cloud.Config
	ai          *ai.Config
	processing  *processing.Config
	environment string
}

// Builder constructs a Config using the builder pattern.
type Builder struct {
	server      *server.Config
	auth        *auth.Config
	cloud       *cloud.Config
	ai          *ai.Config
	processing  *processing.Config
	environment string
	errors      []error
}

// NewBuilder creates a new configuration builder.
func NewBuilder() *Builder {
	return &Builder{
		environment: "local",
	}
}

// WithServer sets the server configuration.
func (b *Builder) WithServer(cfg *server.Config) *Builder {
	if cfg == nil {
		b.errors = append(b.errors, errors.New("server config cannot be nil"))
		return b
	}
	b.server = cfg
	return b
}

// WithAuth sets the authentication configuration.
func (b *Builder) WithAuth(cfg *auth.Config) *Builder {
	if cfg == nil {
		b.errors = append(b.errors, errors.New("auth config cannot be nil"))
		return b
	}
	b.auth = cfg
	return b
}

// WithCloud sets the cloud configuration.
func (b *Builder) WithCloud(cfg *cloud.Config) *Builder {
	b.cloud = cfg
	return b
}

// WithAI sets the AI configuration.
func (b *Builder) WithAI(cfg *ai.Config) *Builder {
	b.ai = cfg
	return b
}

// WithProcessing sets the processing configuration.
func (b *Builder) WithProcessing(cfg *processing.Config) *Builder {
	b.processing = cfg
	return b
}

// WithEnvironment sets the environment (local, staging, production).
func (b *Builder) WithEnvironment(env string) *Builder {
	b.environment = env
	return b
}

// Build validates and constructs the Config.
func (b *Builder) Build() (*Config, error) {
	// Check for any errors collected during building
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("configuration errors: %v", b.errors)
	}

	// Validate required configurations
	if b.server == nil {
		return nil, errors.New("server configuration is required")
	}

	if b.auth == nil {
		return nil, errors.New("auth configuration is required")
	}

	cfg := &Config{
		server:      b.server,
		auth:        b.auth,
		cloud:       b.cloud,
		ai:          b.ai,
		processing:  b.processing,
		environment: b.environment,
	}

	// Final validation
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate performs final validation on the complete configuration.
func (c *Config) Validate() error {
	if c.server == nil {
		return errors.New("server configuration is required")
	}

	if c.auth == nil {
		return errors.New("auth configuration is required")
	}

	// Validate environment
	switch c.environment {
	case "local", "staging", "production":
		// Valid environments
	default:
		return fmt.Errorf("invalid environment: %s (must be local, staging, or production)", c.environment)
	}

	return nil
}

// Server returns the server configuration.
func (c *Config) Server() *server.Config {
	return c.server
}

// Auth returns the authentication configuration.
func (c *Config) Auth() *auth.Config {
	return c.auth
}

// Cloud returns the cloud configuration.
func (c *Config) Cloud() *cloud.Config {
	return c.cloud
}

// AI returns the AI configuration.
func (c *Config) AI() *ai.Config {
	return c.ai
}

// Processing returns the processing configuration.
func (c *Config) Processing() *processing.Config {
	return c.processing
}

// Environment returns the environment name.
func (c *Config) Environment() string {
	return c.environment
}

// IsProduction returns true if running in production environment.
func (c *Config) IsProduction() bool {
	return c.environment == "production" || (c.server != nil && c.server.Production())
}

// IsLocal returns true if running in local development environment.
func (c *Config) IsLocal() bool {
	return c.environment == "local"
}

// String returns a human-readable representation of the configuration.
func (c *Config) String() string {
	return fmt.Sprintf("Config: env=%s, server=%s, production=%v",
		c.environment, c.server.Address(), c.IsProduction())
}
