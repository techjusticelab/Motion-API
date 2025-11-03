package configuration

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"motion-index-fiber/internal/configuration/ai"
	"motion-index-fiber/internal/configuration/auth"
	"motion-index-fiber/internal/configuration/cloud"
	"motion-index-fiber/internal/configuration/processing"
	"motion-index-fiber/internal/configuration/server"
	doConfig "motion-index-fiber/pkg/cloud/digitalocean/config"
)

// LoadFromEnvironment loads configuration from environment variables.
func LoadFromEnvironment() (*Config, error) {
	builder := NewBuilder()

	// Determine environment
	environment := getEnv("ENVIRONMENT", "local")
	if getEnvBool("PRODUCTION", false) {
		environment = "production"
	}
	builder.WithEnvironment(environment)

	// Load server configuration
	serverCfg, err := loadServerConfig(environment)
	if err != nil {
		return nil, fmt.Errorf("failed to load server config: %w", err)
	}
	builder.WithServer(serverCfg)

	// Load auth configuration
	authCfg, err := loadAuthConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load auth config: %w", err)
	}
	builder.WithAuth(authCfg)

	// Load cloud configuration (optional)
	cloudCfg, err := loadCloudConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load cloud config: %w", err)
	}
	if cloudCfg != nil {
		builder.WithCloud(cloudCfg)
	}

	// Load AI configuration (optional)
	aiCfg, err := loadAIConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load AI config: %w", err)
	}
	if aiCfg != nil {
		builder.WithAI(aiCfg)
	}

	// Load processing configuration
	processingCfg, err := loadProcessingConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load processing config: %w", err)
	}
	if processingCfg != nil {
		builder.WithProcessing(processingCfg)
	}

	return builder.Build()
}

// loadServerConfig loads server configuration from environment.
func loadServerConfig(environment string) (*server.Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		return nil, fmt.Errorf("PORT environment variable is required")
	}

	// Parse allowed origins
	var allowedOrigins []string
	originsStr := getEnv("ALLOWED_ORIGINS", "")
	if originsStr != "" {
		allowedOrigins = strings.Split(originsStr, ",")
	}

	production := environment == "production" || environment == "staging" || getEnvBool("PRODUCTION", false)

	return server.NewConfig(port,
		server.WithHost(getEnv("HOST", "0.0.0.0")),
		server.WithReadTimeout(getEnvDuration("READ_TIMEOUT", 30*time.Second)),
		server.WithWriteTimeout(getEnvDuration("WRITE_TIMEOUT", 30*time.Second)),
		server.WithAllowedOrigins(allowedOrigins),
		server.WithMaxRequestSize(getEnvInt64("MAX_REQUEST_SIZE", 100*1024*1024)),
		server.WithProduction(production),
	)
}

// loadAuthConfig loads authentication configuration from environment.
func loadAuthConfig() (*auth.Config, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseAnonKey := os.Getenv("SUPABASE_ANON_KEY")
	supabaseAPIKey := os.Getenv("SUPABASE_SERVICE_KEY")

	return auth.NewConfig(jwtSecret, supabaseURL, supabaseAnonKey, supabaseAPIKey)
}

// loadCloudConfig loads cloud configuration from environment.
func loadCloudConfig() (*cloud.Config, error) {
	// Load storage config
	storageCfg, err := loadStorageConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load storage config: %w", err)
	}

	// Load search config
	searchCfg, err := loadSearchConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load search config: %w", err)
	}

	// Load DigitalOcean config
	doCfg, err := doConfig.LoadFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to load DigitalOcean config: %w", err)
	}

	return cloud.NewConfig(
		cloud.WithStorage(storageCfg),
		cloud.WithSearch(searchCfg),
		cloud.WithDigitalOcean(doCfg),
	)
}

// loadStorageConfig loads storage configuration from environment.
func loadStorageConfig() (*cloud.StorageConfig, error) {
	backend := getEnv("STORAGE_BACKEND", "local")

	accessKey := getEnv("STORAGE_ACCESS_KEY", getEnv("DO_SPACES_ACCESS_KEY", getEnv("DO_SPACES_KEY", "")))
	secretKey := getEnv("STORAGE_SECRET_KEY", getEnv("DO_SPACES_SECRET_KEY", getEnv("DO_SPACES_SECRET", "")))
	bucket := getEnv("STORAGE_BUCKET", getEnv("DO_SPACES_BUCKET", "motion-index-docs"))
	region := getEnv("STORAGE_REGION", getEnv("DO_SPACES_REGION", "nyc3"))
	cdnDomain := getEnv("STORAGE_CDN_DOMAIN", getEnv("DO_SPACES_CDN_DOMAIN", ""))

	return cloud.NewStorageConfig(backend,
		cloud.WithStorageCredentials(accessKey, secretKey),
		cloud.WithStorageBucket(bucket),
		cloud.WithStorageRegion(region),
		cloud.WithCDNDomain(cdnDomain),
	)
}

// loadSearchConfig loads search configuration from environment.
func loadSearchConfig() (*cloud.SearchConfig, error) {
	host := getEnv("OPENSEARCH_HOST", getEnv("ES_HOST", getEnv("DO_OPENSEARCH_HOST", "")))
	if host == "" {
		return nil, fmt.Errorf("OPENSEARCH_HOST is required")
	}

	// Determine OpenSearch port with broad env fallback
	port := getEnvInt("OPENSEARCH_PORT", getEnvInt("ES_PORT", 9200))
	if os.Getenv("OPENSEARCH_PORT") == "" && os.Getenv("ES_PORT") == "" {
		if v := os.Getenv("DO_OPENSEARCH_PORT"); v != "" {
			if p, err := strconv.Atoi(v); err == nil {
				port = p
			}
		}
	}

	username := getEnv("OPENSEARCH_USERNAME", getEnv("ES_USERNAME", getEnv("DO_OPENSEARCH_USERNAME", "")))
	password := getEnv("OPENSEARCH_PASSWORD", getEnv("ES_PASSWORD", getEnv("DO_OPENSEARCH_PASSWORD", "")))
	useSSL := getEnvBool("OPENSEARCH_USE_SSL", getEnvBool("ES_USE_SSL", getEnvBool("DO_OPENSEARCH_USE_SSL", true)))
	index := getEnv("OPENSEARCH_INDEX", getEnv("ES_INDEX", getEnv("DO_OPENSEARCH_INDEX", "documents")))

	return cloud.NewSearchConfig(host, port,
		cloud.WithSearchCredentials(username, password),
		cloud.WithSearchSSL(useSSL),
		cloud.WithSearchIndex(index),
	)
}

// loadAIConfig loads AI configuration from environment.
func loadAIConfig() (*ai.Config, error) {
	var options []ai.Option

	// Load OpenAI config
	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey != "" {
		openaiCfg, err := ai.NewOpenAIConfig(
			openaiKey,
			getEnv("OPENAI_MODEL", "gpt-4"),
			ai.WithOpenAITimeout(getEnvDuration("OPENAI_TIMEOUT", 10*time.Minute)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenAI config: %w", err)
		}
		options = append(options, ai.WithOpenAI(openaiCfg))
	}

	// Load Claude config
	if claudeKey := os.Getenv("CLAUDE_API_KEY"); claudeKey != "" {
		claudeCfg, err := ai.NewClaudeConfig(
			claudeKey,
			getEnv("CLAUDE_MODEL", "claude-3-5-sonnet-20241022"),
			ai.WithClaudeBaseURL(getEnv("CLAUDE_BASE_URL", "https://api.anthropic.com")),
			ai.WithClaudeTimeout(getEnvDuration("CLAUDE_TIMEOUT", 10*time.Minute)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create Claude config: %w", err)
		}
		options = append(options, ai.WithClaude(claudeCfg))
	}

	// Load Ollama config
	if ollamaModel := os.Getenv("OLLAMA_MODEL"); ollamaModel != "" {
		ollamaCfg, err := ai.NewOllamaConfig(
			ollamaModel,
			ai.WithOllamaBaseURL(getEnv("OLLAMA_BASE_URL", "http://localhost:11434")),
			ai.WithOllamaTimeout(getEnvDuration("OLLAMA_TIMEOUT", 120*time.Second)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create Ollama config: %w", err)
		}
		options = append(options, ai.WithOllama(ollamaCfg))
	}

	// If no AI providers configured, return nil (not an error - AI is optional)
	if len(options) == 0 {
		return nil, nil
	}

	// Add fallback configuration
	options = append(options,
		ai.WithFallback(getEnvBool("AI_ENABLE_FALLBACK", true)),
		ai.WithRetryAttempts(getEnvInt("AI_RETRY_ATTEMPTS", 3)),
		ai.WithRetryDelay(getEnvDuration("AI_RETRY_DELAY", 5*time.Second)),
	)

	return ai.NewConfig(options...)
}

// loadProcessingConfig loads processing configuration from environment.
func loadProcessingConfig() (*processing.Config, error) {
	return processing.NewConfig(
		processing.WithMaxFileSize(getEnvInt64("MAX_FILE_SIZE", 100*1024*1024)),
		processing.WithMaxWorkers(getEnvInt("MAX_WORKERS", 10)),
		processing.WithBatchSize(getEnvInt("BATCH_SIZE", 50)),
		processing.WithProcessTimeout(getEnvDuration("PROCESS_TIMEOUT", 5*time.Minute)),
	)
}

// Helper functions for environment variable parsing

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
