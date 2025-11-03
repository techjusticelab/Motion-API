package processing

import (
	"errors"
	"fmt"
	"time"
)

// Config holds document processing configuration.
type Config struct {
	maxFileSize    int64
	maxWorkers     int
	batchSize      int
	processTimeout time.Duration
}

// NewConfig creates a processing configuration with validation.
func NewConfig(options ...Option) (*Config, error) {
	cfg := &Config{
		maxFileSize:    100 * 1024 * 1024, // 100MB default
		maxWorkers:     10,
		batchSize:      50,
		processTimeout: 5 * time.Minute,
	}

	for _, opt := range options {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Option is a functional option for processing configuration.
type Option func(*Config)

// WithMaxFileSize sets the maximum file size for processing.
func WithMaxFileSize(size int64) Option {
	return func(c *Config) {
		c.maxFileSize = size
	}
}

// WithMaxWorkers sets the maximum number of processing workers.
func WithMaxWorkers(workers int) Option {
	return func(c *Config) {
		c.maxWorkers = workers
	}
}

// WithBatchSize sets the batch size for batch processing.
func WithBatchSize(size int) Option {
	return func(c *Config) {
		c.batchSize = size
	}
}

// WithProcessTimeout sets the processing timeout.
func WithProcessTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.processTimeout = timeout
	}
}

// Validate checks that the processing configuration is valid.
func (c *Config) Validate() error {
	if c.maxFileSize <= 0 {
		return errors.New("max file size must be positive")
	}

	if c.maxWorkers <= 0 {
		return errors.New("max workers must be positive")
	}

	if c.batchSize <= 0 {
		return errors.New("batch size must be positive")
	}

	if c.processTimeout <= 0 {
		return errors.New("process timeout must be positive")
	}

	return nil
}

// MaxFileSize returns the maximum file size.
func (c *Config) MaxFileSize() int64 {
	return c.maxFileSize
}

// MaxWorkers returns the maximum number of workers.
func (c *Config) MaxWorkers() int {
	return c.maxWorkers
}

// BatchSize returns the batch size.
func (c *Config) BatchSize() int {
	return c.batchSize
}

// ProcessTimeout returns the processing timeout.
func (c *Config) ProcessTimeout() time.Duration {
	return c.processTimeout
}

// String returns a human-readable representation of the processing configuration.
func (c *Config) String() string {
	return fmt.Sprintf("Processing Config: maxFileSize=%d, maxWorkers=%d, batchSize=%d, timeout=%v",
		c.maxFileSize, c.maxWorkers, c.batchSize, c.processTimeout)
}
