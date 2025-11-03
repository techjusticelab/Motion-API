package server

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Config holds HTTP server configuration.
type Config struct {
	port           string
	host           string
	readTimeout    time.Duration
	writeTimeout   time.Duration
	allowedOrigins []string
	maxRequestSize int64
	production     bool
}

// NewConfig creates a server configuration with validation.
func NewConfig(port string, options ...Option) (*Config, error) {
	cfg := &Config{
		port:           port,
		host:           "0.0.0.0",
		readTimeout:    30 * time.Second,
		writeTimeout:   30 * time.Second,
		maxRequestSize: 100 * 1024 * 1024, // 100MB default
	}

	for _, opt := range options {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Option is a functional option for server configuration.
type Option func(*Config)

// WithHost sets the server host address.
func WithHost(host string) Option {
	return func(c *Config) {
		c.host = host
	}
}

// WithReadTimeout sets the read timeout.
func WithReadTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.readTimeout = timeout
	}
}

// WithWriteTimeout sets the write timeout.
func WithWriteTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.writeTimeout = timeout
	}
}

// WithAllowedOrigins sets the CORS allowed origins.
func WithAllowedOrigins(origins []string) Option {
	return func(c *Config) {
		c.allowedOrigins = origins
	}
}

// WithMaxRequestSize sets the maximum request size.
func WithMaxRequestSize(size int64) Option {
	return func(c *Config) {
		c.maxRequestSize = size
	}
}

// WithProduction sets production mode.
func WithProduction(production bool) Option {
	return func(c *Config) {
		c.production = production
	}
}

// Validate checks that the server configuration is valid.
func (c *Config) Validate() error {
	if c.port == "" {
		return errors.New("server port is required")
	}

	// Validate port is numeric and within range
	port, err := strconv.Atoi(c.port)
	if err != nil {
		return fmt.Errorf("server port must be a valid number: %w", err)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535, got %d", port)
	}

	if c.readTimeout <= 0 {
		return errors.New("read timeout must be positive")
	}

	if c.writeTimeout <= 0 {
		return errors.New("write timeout must be positive")
	}

	if c.maxRequestSize <= 0 {
		return errors.New("max request size must be positive")
	}

	return nil
}

// Port returns the server port.
func (c *Config) Port() string {
	return c.port
}

// Host returns the server host address.
func (c *Config) Host() string {
	return c.host
}

// ReadTimeout returns the read timeout.
func (c *Config) ReadTimeout() time.Duration {
	return c.readTimeout
}

// WriteTimeout returns the write timeout.
func (c *Config) WriteTimeout() time.Duration {
	return c.writeTimeout
}

// AllowedOrigins returns the CORS allowed origins.
func (c *Config) AllowedOrigins() []string {
	return c.allowedOrigins
}

// AllowedOriginsString returns allowed origins as a comma-separated string.
func (c *Config) AllowedOriginsString() string {
	return strings.Join(c.allowedOrigins, ",")
}

// MaxRequestSize returns the maximum request size.
func (c *Config) MaxRequestSize() int64 {
	return c.maxRequestSize
}

// Production returns whether production mode is enabled.
func (c *Config) Production() bool {
	return c.production
}

// Address returns the full server address (host:port).
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%s", c.host, c.port)
}
