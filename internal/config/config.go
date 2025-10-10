package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	doConfig "motion-index-fiber/pkg/cloud/digitalocean/config"
)

type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Storage      StorageConfig
	Auth         AuthConfig
	Processing   ProcessingConfig
	OpenSearch   OpenSearchConfig
	OpenAI       OpenAIConfig // Keep for backward compatibility
	AI           AIConfig     // New comprehensive AI config
	Logging      LoggingConfig
	DigitalOcean *doConfig.Config // Comprehensive DigitalOcean configuration
	Environment  string           // Environment: local, staging, production
}

type ServerConfig struct {
	Port           string
	Production     bool
	AllowedOrigins string
	MaxRequestSize int64
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	UseSSL   bool
}

type StorageConfig struct {
	Backend   string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	CDNDomain string
}

type AuthConfig struct {
	JWTSecret       string
	SupabaseURL     string
	SupabaseAnonKey string
	SupabaseAPIKey  string
}

type ProcessingConfig struct {
	MaxFileSize    int64
	MaxWorkers     int
	BatchSize      int
	ProcessTimeout time.Duration
}

type OpenSearchConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	UseSSL   bool
	Index    string
}

type OpenAIConfig struct {
	APIKey string
	Model  string
}

type AIConfig struct {
	// Primary provider (OpenAI)
	OpenAI OpenAIConfig
	
	// Fallback provider (Claude)
	Claude ClaudeConfig
	
	// Local fallback (Ollama)
	Ollama OllamaConfig
	
	// Fallback configuration
	EnableFallback bool
	RetryAttempts  int
	RetryDelay     time.Duration
}

type ClaudeConfig struct {
	APIKey string
	Model  string
	BaseURL string // Optional custom endpoint
}

type OllamaConfig struct {
	BaseURL string
	Model   string
	Timeout time.Duration
}

type LoggingConfig struct {
	Level              string
	Format             string
	EnableRequestLog   bool
	EnableErrorDetails bool
	EnableStackTrace   bool
}

func Load() (*Config, error) {
	// Determine environment
	environment := getEnv("ENVIRONMENT", "local")
	if getEnvBool("PRODUCTION", false) {
		environment = "production"
	}

    // Default origins: require explicit configuration (production-like)
    var defaultOrigins string
    defaultOrigins = ""

	// Parse numeric values with error handling
    // Determine OpenSearch port with broad env fallback (OPENSEARCH_*, DO_OPENSEARCH_*, ES_*)
    opensearchPort, err := parseEnvInt("OPENSEARCH_PORT", getEnvInt("ES_PORT", 9200))
    if os.Getenv("OPENSEARCH_PORT") == "" && os.Getenv("ES_PORT") == "" {
        if v := os.Getenv("DO_OPENSEARCH_PORT"); v != "" {
            if p, pErr := strconv.Atoi(v); pErr == nil {
                opensearchPort = p
            } else {
                return nil, fmt.Errorf("DO_OPENSEARCH_PORT must be a valid number")
            }
        }
    }
	if err != nil {
		return nil, err
	}

	maxRequestSize, err := parseEnvInt64("MAX_REQUEST_SIZE", 100*1024*1024)
	if err != nil {
		return nil, err
	}

	maxFileSize, err := parseEnvInt64("MAX_FILE_SIZE", 100*1024*1024)
	if err != nil {
		return nil, err
	}

	maxWorkers, err := parseEnvInt("MAX_WORKERS", 10)
	if err != nil {
		return nil, err
	}

	batchSize, err := parseEnvInt("BATCH_SIZE", 50)
	if err != nil {
		return nil, err
	}

	processTimeout, err := parseEnvDuration("PROCESS_TIMEOUT", 5*time.Minute)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Environment: environment,
		Server: ServerConfig{
			Port:           os.Getenv("PORT"), // Don't use default to allow validation
			Production:     environment == "production" || environment == "staging" || getEnvBool("PRODUCTION", false),
			AllowedOrigins: getEnv("ALLOWED_ORIGINS", defaultOrigins),
			MaxRequestSize: maxRequestSize,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			Username: getEnv("DB_USERNAME", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Database: getEnv("DB_DATABASE", "motion_index"),
			UseSSL:   getEnvBool("DB_USE_SSL", false),
		},
        Storage: StorageConfig{
            Backend:   getEnv("STORAGE_BACKEND", "local"),
            AccessKey: getEnv("STORAGE_ACCESS_KEY", getEnv("DO_SPACES_ACCESS_KEY", getEnv("DO_SPACES_KEY", ""))),
            SecretKey: getEnv("STORAGE_SECRET_KEY", getEnv("DO_SPACES_SECRET_KEY", getEnv("DO_SPACES_SECRET", ""))),
            Bucket:    getEnv("STORAGE_BUCKET", getEnv("DO_SPACES_BUCKET", "motion-index-docs")),
            Region:    getEnv("STORAGE_REGION", getEnv("DO_SPACES_REGION", "nyc3")),
            CDNDomain: getEnv("STORAGE_CDN_DOMAIN", getEnv("DO_SPACES_CDN_DOMAIN", "")),
        },
		Auth: AuthConfig{
			JWTSecret:       getEnv("JWT_SECRET", ""),
			SupabaseURL:     getEnv("SUPABASE_URL", ""),
			SupabaseAnonKey: getEnv("SUPABASE_ANON_KEY", ""),
			SupabaseAPIKey:  getEnv("SUPABASE_SERVICE_KEY", ""),
		},
		Processing: ProcessingConfig{
			MaxFileSize:    maxFileSize,
			MaxWorkers:     maxWorkers,
			BatchSize:      batchSize,
			ProcessTimeout: processTimeout,
		},
        OpenSearch: OpenSearchConfig{
            Host:     getEnv("OPENSEARCH_HOST", getEnv("ES_HOST", getEnv("DO_OPENSEARCH_HOST", ""))), // Prefer OPENSEARCH_*, then ES_*, then DO_OPENSEARCH_*
            Port:     opensearchPort,
            Username: getEnv("OPENSEARCH_USERNAME", getEnv("ES_USERNAME", getEnv("DO_OPENSEARCH_USERNAME", ""))),
            Password: getEnv("OPENSEARCH_PASSWORD", getEnv("ES_PASSWORD", getEnv("DO_OPENSEARCH_PASSWORD", ""))),
            UseSSL:   getEnvBool("OPENSEARCH_USE_SSL", getEnvBool("ES_USE_SSL", getEnvBool("DO_OPENSEARCH_USE_SSL", true))),
            Index:    getEnv("OPENSEARCH_INDEX", getEnv("ES_INDEX", getEnv("DO_OPENSEARCH_INDEX", "documents"))),
        },
		OpenAI: OpenAIConfig{
			APIKey: getEnv("OPENAI_API_KEY", ""),
			Model:  getEnv("OPENAI_MODEL", "gpt-4"),
		},
		AI: AIConfig{
			OpenAI: OpenAIConfig{
				APIKey: getEnv("OPENAI_API_KEY", ""),
				Model:  getEnv("OPENAI_MODEL", "gpt-4"),
			},
			Claude: ClaudeConfig{
				APIKey:  getEnv("CLAUDE_API_KEY", ""),
				Model:   getEnv("CLAUDE_MODEL", "claude-3-sonnet-20240229"),
				BaseURL: getEnv("CLAUDE_BASE_URL", "https://api.anthropic.com"),
			},
			Ollama: OllamaConfig{
				BaseURL: getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
				Model:   getEnv("OLLAMA_MODEL", "gpt-oss:20b"),
				Timeout: getEnvDuration("OLLAMA_TIMEOUT", 120*time.Second),
			},
			EnableFallback: getEnvBool("AI_ENABLE_FALLBACK", true),
			RetryAttempts:  getEnvInt("AI_RETRY_ATTEMPTS", 3),
			RetryDelay:     getEnvDuration("AI_RETRY_DELAY", 5*time.Second),
		},
        Logging: LoggingConfig{
            Level:              getEnv("LOG_LEVEL", "info"),
            Format:             getEnv("LOG_FORMAT", "text"),
            EnableRequestLog:   getEnvBool("ENABLE_REQUEST_LOGGING", true),
            EnableErrorDetails: getEnvBool("ENABLE_ERROR_DETAILS", false),
            EnableStackTrace:   getEnvBool("ENABLE_STACK_TRACE", false),
        },
	}

    // Initialize DigitalOcean configuration (always require proper configuration; no local defaults)
    doConfigInstance, err := doConfig.LoadFromEnvironment()
    if err != nil {
        return nil, fmt.Errorf("failed to load DigitalOcean configuration: %w", err)
    }
	cfg.DigitalOcean = doConfigInstance

	// Validate required fields
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	// Validate server configuration
	if err := c.validateServer(); err != nil {
		return err
	}

	// Validate storage configuration
	if err := c.validateStorage(); err != nil {
		return err
	}

	// Validate OpenSearch configuration
	if err := c.validateOpenSearch(); err != nil {
		return err
	}

	// Validate auth configuration
	if err := c.validateAuth(); err != nil {
		return err
	}

	// Validate processing configuration
	if err := c.validateProcessing(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateServer() error {
	// PORT validation
	if c.Server.Port == "" {
		return fmt.Errorf("PORT is required")
	}

	// Validate port is numeric and within range
	port, err := strconv.Atoi(c.Server.Port)
	if err != nil {
		return fmt.Errorf("PORT must be a valid number")
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535")
	}

	return nil
}

func (c *Config) validateStorage() error {
	// Validate storage backend
	if c.Storage.Backend != "local" && c.Storage.Backend != "spaces" {
		return fmt.Errorf("STORAGE_BACKEND must be 'local' or 'spaces'")
	}

	// For spaces backend, validate required credentials
	if c.Storage.Backend == "spaces" {
		if c.Storage.AccessKey == "" {
			return fmt.Errorf("STORAGE_ACCESS_KEY is required for spaces backend")
		}
		if c.Storage.SecretKey == "" {
			return fmt.Errorf("STORAGE_SECRET_KEY is required for spaces backend")
		}
		if c.Storage.Bucket == "" {
			return fmt.Errorf("STORAGE_BUCKET is required for spaces backend")
		}
		if c.Storage.Region == "" {
			return fmt.Errorf("STORAGE_REGION is required for spaces backend")
		}
	}

	return nil
}

func (c *Config) validateOpenSearch() error {
    // Always require OpenSearch host (production-like behavior)
    if c.OpenSearch.Host == "" {
        return fmt.Errorf("OPENSEARCH_HOST is required")
    }

	// Validate port is numeric and within range
	if c.OpenSearch.Port < 1 || c.OpenSearch.Port > 65535 {
		return fmt.Errorf("OPENSEARCH_PORT must be between 1 and 65535")
	}

    // Always require authentication
    if c.OpenSearch.Username == "" {
        return fmt.Errorf("OPENSEARCH_USERNAME is required")
    }
    if c.OpenSearch.Password == "" {
        return fmt.Errorf("OPENSEARCH_PASSWORD is required")
    }

	return nil
}

func (c *Config) validateAuth() error {
    // Always require JWT secret and Supabase configuration (production-like)
    if c.Auth.JWTSecret == "" {
        return fmt.Errorf("JWT_SECRET is required")
    }
    if c.Auth.SupabaseURL == "" {
        return fmt.Errorf("SUPABASE_URL is required")
    }
    if !isValidURL(c.Auth.SupabaseURL) {
        return fmt.Errorf("SUPABASE_URL must be a valid URL")
    }
    if c.Auth.SupabaseAnonKey == "" {
        return fmt.Errorf("SUPABASE_ANON_KEY is required")
    }
    if c.Auth.SupabaseAPIKey == "" {
        return fmt.Errorf("SUPABASE_API_KEY is required")
    }

	return nil
}

func (c *Config) validateProcessing() error {
	// Validate max file size
	if c.Processing.MaxFileSize <= 0 {
		return fmt.Errorf("MAX_FILE_SIZE must be positive")
	}

	// Validate max workers
	if c.Processing.MaxWorkers <= 0 {
		return fmt.Errorf("MAX_WORKERS must be positive")
	}

	// Validate batch size
	if c.Processing.BatchSize <= 0 {
		return fmt.Errorf("BATCH_SIZE must be positive")
	}

	// Validate process timeout
	if c.Processing.ProcessTimeout <= 0 {
		return fmt.Errorf("PROCESS_TIMEOUT must be positive")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// parseEnvInt parses an environment variable as an integer with error handling
func parseEnvInt(key string, defaultValue int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid number", key)
	}

	return intValue, nil
}

// parseEnvInt64 parses an environment variable as an int64 with error handling
func parseEnvInt64(key string, defaultValue int64) (int64, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid number", key)
	}

	return intValue, nil
}

// parseEnvDuration parses an environment variable as a duration with error handling
func parseEnvDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration", key)
	}

	return duration, nil
}

// isValidURL validates if a string is a valid URL
func isValidURL(urlStr string) bool {
	if urlStr == "" {
		return false
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// Must have scheme and host
	return parsedURL.Scheme != "" && parsedURL.Host != ""
}

func (c *Config) GetStorageEndpoint() string {
	return fmt.Sprintf("https://%s.%s.digitaloceanspaces.com", c.Storage.Bucket, c.Storage.Region)
}

func (c *Config) GetCDNEndpoint() string {
	if c.Storage.CDNDomain != "" {
		return fmt.Sprintf("https://%s", c.Storage.CDNDomain)
	}
	return fmt.Sprintf("https://%s.%s.cdn.digitaloceanspaces.com", c.Storage.Bucket, c.Storage.Region)
}

func (c *Config) GetOpenSearchURL() string {
	protocol := "http"
	if c.OpenSearch.UseSSL {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s:%d", protocol, c.OpenSearch.Host, c.OpenSearch.Port)
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Environment == "production" || c.Server.Production
}

// IsLocal returns true if running in local development environment
func (c *Config) IsLocal() bool {
	return c.Environment == "local"
}
