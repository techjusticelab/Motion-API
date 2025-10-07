package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Environment represents the deployment environment
type Environment string

const (
	EnvLocal      Environment = "local"
	EnvStaging    Environment = "staging"
	EnvProduction Environment = "production"
)

// Config holds all DigitalOcean service configuration
type Config struct {
	Environment Environment `json:"environment" validate:"required,oneof=local staging production"`

	DigitalOcean struct {
		// API Token for DigitalOcean API operations
		APIToken string `json:"api_token"`

		// Spaces configuration for document storage
		Spaces struct {
			AccessKey   string `json:"access_key"`
			SecretKey   string `json:"secret_key"`
			Bucket      string `json:"bucket"`
			Region      string `json:"region"`
			CDNEndpoint string `json:"cdn_endpoint,omitempty"`
			Endpoint    string `json:"endpoint,omitempty"` // Custom endpoint for testing
		} `json:"spaces"`

		// OpenSearch configuration for document search
		OpenSearch struct {
			Host     string `json:"host"`
			Port     int    `json:"port" validate:"min=1,max=65535"`
			Username string `json:"username"`
			Password string `json:"password"`
			UseSSL   bool   `json:"use_ssl"`
			Index    string `json:"index" validate:"required"`
		} `json:"opensearch"`
	} `json:"digitalocean"`

	// Health monitoring configuration
	Health struct {
		CheckInterval  int  `json:"check_interval_seconds" validate:"min=1,max=3600"`
		TimeoutSeconds int  `json:"timeout_seconds" validate:"min=1,max=300"`
		MaxRetries     int  `json:"max_retries" validate:"min=0,max=10"`
		CircuitBreaker bool `json:"circuit_breaker"`
	} `json:"health"`

	// Performance and optimization settings
	Performance struct {
		MaxConcurrentUploads   int  `json:"max_concurrent_uploads" validate:"min=1,max=100"`
		MaxConcurrentDownloads int  `json:"max_concurrent_downloads" validate:"min=1,max=100"`
		ChunkSizeBytes         int  `json:"chunk_size_bytes" validate:"min=1024"`
		EnableCaching          bool `json:"enable_caching"`
		CacheTTLSeconds        int  `json:"cache_ttl_seconds" validate:"min=60"`
	} `json:"performance"`
}

// ConfigError represents configuration validation errors
type ConfigError struct {
	Field   string
	Message string
	Value   interface{}
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("configuration error in field '%s': %s (value: %v)", e.Field, e.Message, e.Value)
}

// LoadFromEnvironment loads configuration from environment variables
func LoadFromEnvironment() (*Config, error) {
	config := &Config{}

	// Environment detection
	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	if env == "" {
		env = "local" // Default to local development
	}

	switch env {
	case "local":
		config.Environment = EnvLocal
	case "staging":
		config.Environment = EnvStaging
	case "production":
		config.Environment = EnvProduction
	default:
		return nil, &ConfigError{
			Field:   "environment",
			Message: "invalid environment, must be one of: local, staging, production",
			Value:   env,
		}
	}

	// Load DigitalOcean API Token
	config.DigitalOcean.APIToken = os.Getenv("DO_API_TOKEN")

	// Load DigitalOcean Spaces configuration
	config.DigitalOcean.Spaces.AccessKey = os.Getenv("DO_SPACES_ACCESS_KEY")
	config.DigitalOcean.Spaces.SecretKey = os.Getenv("DO_SPACES_SECRET_KEY")
	config.DigitalOcean.Spaces.Bucket = os.Getenv("DO_SPACES_BUCKET")
	config.DigitalOcean.Spaces.Region = os.Getenv("DO_SPACES_REGION")
	config.DigitalOcean.Spaces.CDNEndpoint = os.Getenv("DO_SPACES_CDN_ENDPOINT")
	config.DigitalOcean.Spaces.Endpoint = os.Getenv("DO_SPACES_ENDPOINT")

	// Load DigitalOcean OpenSearch configuration
	config.DigitalOcean.OpenSearch.Host = os.Getenv("DO_OPENSEARCH_HOST")
	config.DigitalOcean.OpenSearch.Username = os.Getenv("DO_OPENSEARCH_USERNAME")
	config.DigitalOcean.OpenSearch.Password = os.Getenv("DO_OPENSEARCH_PASSWORD")
	config.DigitalOcean.OpenSearch.Index = getEnvWithDefault("DO_OPENSEARCH_INDEX", "documents")

	// Parse numeric values with defaults
	if port := os.Getenv("DO_OPENSEARCH_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.DigitalOcean.OpenSearch.Port = p
		} else {
			return nil, &ConfigError{
				Field:   "DO_OPENSEARCH_PORT",
				Message: "invalid port number",
				Value:   port,
			}
		}
	} else {
		config.DigitalOcean.OpenSearch.Port = 25060 // Default DO OpenSearch port
	}

	// Parse boolean values
	config.DigitalOcean.OpenSearch.UseSSL = getEnvBoolWithDefault("DO_OPENSEARCH_USE_SSL", true)

	// Load health configuration with defaults
	config.Health.CheckInterval = getEnvIntWithDefault("HEALTH_CHECK_INTERVAL", 30)
	config.Health.TimeoutSeconds = getEnvIntWithDefault("HEALTH_TIMEOUT_SECONDS", 10)
	config.Health.MaxRetries = getEnvIntWithDefault("HEALTH_MAX_RETRIES", 3)
	config.Health.CircuitBreaker = getEnvBoolWithDefault("HEALTH_CIRCUIT_BREAKER", true)

	// Load performance configuration with defaults
	config.Performance.MaxConcurrentUploads = getEnvIntWithDefault("PERF_MAX_CONCURRENT_UPLOADS", 10)
	config.Performance.MaxConcurrentDownloads = getEnvIntWithDefault("PERF_MAX_CONCURRENT_DOWNLOADS", 20)
	config.Performance.ChunkSizeBytes = getEnvIntWithDefault("PERF_CHUNK_SIZE_BYTES", 8*1024*1024) // 8MB
	config.Performance.EnableCaching = getEnvBoolWithDefault("PERF_ENABLE_CACHING", true)
	config.Performance.CacheTTLSeconds = getEnvIntWithDefault("PERF_CACHE_TTL_SECONDS", 3600) // 1 hour

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates the configuration using struct tags and custom logic
func (c *Config) Validate() error {
	validator := validator.New()

	// First run basic struct validation
	if err := validator.Struct(c); err != nil {
		// Convert validation errors to ConfigError
		return &ConfigError{
			Field:   "validation",
			Message: err.Error(),
			Value:   nil,
		}
	}

	// Custom validation for environment-specific required fields
	if !c.IsLocal() {
		// Validate DigitalOcean Spaces configuration for non-local environments
		if strings.TrimSpace(c.DigitalOcean.Spaces.AccessKey) == "" {
			return &ConfigError{
				Field:   "DigitalOcean.Spaces.AccessKey",
				Message: "required for staging and production environments",
				Value:   c.DigitalOcean.Spaces.AccessKey,
			}
		}

		if strings.TrimSpace(c.DigitalOcean.Spaces.SecretKey) == "" {
			return &ConfigError{
				Field:   "DigitalOcean.Spaces.SecretKey",
				Message: "required for staging and production environments",
				Value:   "[REDACTED]",
			}
		}

		if strings.TrimSpace(c.DigitalOcean.Spaces.Bucket) == "" {
			return &ConfigError{
				Field:   "DigitalOcean.Spaces.Bucket",
				Message: "required for staging and production environments",
				Value:   c.DigitalOcean.Spaces.Bucket,
			}
		}

		if strings.TrimSpace(c.DigitalOcean.Spaces.Region) == "" {
			return &ConfigError{
				Field:   "DigitalOcean.Spaces.Region",
				Message: "required for staging and production environments",
				Value:   c.DigitalOcean.Spaces.Region,
			}
		}

		// Validate DigitalOcean API Token for non-local environments
		if strings.TrimSpace(c.DigitalOcean.APIToken) == "" {
			return &ConfigError{
				Field:   "DigitalOcean.APIToken",
				Message: "required for staging and production environments",
				Value:   "[REDACTED]",
			}
		}

		// Validate DigitalOcean OpenSearch configuration for non-local environments
		if strings.TrimSpace(c.DigitalOcean.OpenSearch.Host) == "" {
			return &ConfigError{
				Field:   "DigitalOcean.OpenSearch.Host",
				Message: "required for staging and production environments",
				Value:   c.DigitalOcean.OpenSearch.Host,
			}
		}

		if strings.TrimSpace(c.DigitalOcean.OpenSearch.Username) == "" {
			return &ConfigError{
				Field:   "DigitalOcean.OpenSearch.Username",
				Message: "required for staging and production environments",
				Value:   c.DigitalOcean.OpenSearch.Username,
			}
		}

		if strings.TrimSpace(c.DigitalOcean.OpenSearch.Password) == "" {
			return &ConfigError{
				Field:   "DigitalOcean.OpenSearch.Password",
				Message: "required for staging and production environments",
				Value:   "[REDACTED]",
			}
		}
	}

	return nil
}

// IsLocal returns true if running in local development environment
func (c *Config) IsLocal() bool {
	return c.Environment == EnvLocal
}

// IsStaging returns true if running in staging environment
func (c *Config) IsStaging() bool {
	return c.Environment == EnvStaging
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Environment == EnvProduction
}

// ShouldUseMockServices returns true if mock services should be used
func (c *Config) ShouldUseMockServices() bool {
	return c.Environment == EnvLocal
}

// GetSpacesEndpoint returns the appropriate Spaces endpoint
func (c *Config) GetSpacesEndpoint() string {
	if c.DigitalOcean.Spaces.Endpoint != "" {
		return c.DigitalOcean.Spaces.Endpoint
	}

	// Default DO Spaces endpoint format
	return fmt.Sprintf("https://%s.digitaloceanspaces.com", c.DigitalOcean.Spaces.Region)
}

// GetSpacesCDNEndpoint returns the CDN endpoint if available
func (c *Config) GetSpacesCDNEndpoint() string {
	if c.DigitalOcean.Spaces.CDNEndpoint != "" {
		return c.DigitalOcean.Spaces.CDNEndpoint
	}

	// Default CDN endpoint format
	return fmt.Sprintf("https://%s.%s.cdn.digitaloceanspaces.com",
		c.DigitalOcean.Spaces.Bucket, c.DigitalOcean.Spaces.Region)
}

// GetOpenSearchEndpoint returns the full OpenSearch endpoint
func (c *Config) GetOpenSearchEndpoint() string {
	protocol := "http"
	if c.DigitalOcean.OpenSearch.UseSSL {
		protocol = "https"
	}

	return fmt.Sprintf("%s://%s:%d",
		protocol, c.DigitalOcean.OpenSearch.Host, c.DigitalOcean.OpenSearch.Port)
}

// Helper functions for environment variable parsing

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBoolWithDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// DefaultConfig returns a configuration with sensible defaults for local development
func DefaultConfig() *Config {
	return &Config{
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
			ChunkSizeBytes:         8 * 1024 * 1024, // 8MB
			EnableCaching:          true,
			CacheTTLSeconds:        3600, // 1 hour
		},
	}
}
