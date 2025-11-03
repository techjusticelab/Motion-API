package config

import (
	"fmt"

	"motion-index-fiber/internal/configuration"
	"motion-index-fiber/internal/configuration/ai"
	"motion-index-fiber/internal/configuration/auth"
	"motion-index-fiber/internal/configuration/cloud"
	"motion-index-fiber/internal/configuration/processing"
	"motion-index-fiber/internal/configuration/server"
)

// LoadNew loads configuration using the new modular configuration system.
// This provides backward compatibility for existing code.
func LoadNew() (*Config, error) {
	newCfg, err := configuration.LoadFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to load new configuration: %w", err)
	}

	// Convert new config to old config structure
	return convertToOldConfig(newCfg)
}

// convertToOldConfig converts the new modular config to the old monolithic structure.
func convertToOldConfig(newCfg *configuration.Config) (*Config, error) {
	cfg := &Config{
		Environment: newCfg.Environment(),
	}

	// Convert server config
	if serverCfg := newCfg.Server(); serverCfg != nil {
		cfg.Server = ServerConfig{
			Port:           serverCfg.Port(),
			Production:     serverCfg.Production(),
			AllowedOrigins: serverCfg.AllowedOriginsString(),
			MaxRequestSize: serverCfg.MaxRequestSize(),
		}
	}

	// Convert auth config
	if authCfg := newCfg.Auth(); authCfg != nil {
		cfg.Auth = AuthConfig{
			JWTSecret:       authCfg.JWTSecret(),
			SupabaseURL:     authCfg.SupabaseURL(),
			SupabaseAnonKey: authCfg.SupabaseAnonKey(),
			SupabaseAPIKey:  authCfg.SupabaseAPIKey(),
		}
	}

	// Convert processing config
	if procCfg := newCfg.Processing(); procCfg != nil {
		cfg.Processing = ProcessingConfig{
			MaxFileSize:    procCfg.MaxFileSize(),
			MaxWorkers:     procCfg.MaxWorkers(),
			BatchSize:      procCfg.BatchSize(),
			ProcessTimeout: procCfg.ProcessTimeout(),
		}
	}

	// Convert cloud config
	if cloudCfg := newCfg.Cloud(); cloudCfg != nil {
		// Storage config
		if storageCfg := cloudCfg.Storage(); storageCfg != nil {
			cfg.Storage = StorageConfig{
				Backend:   storageCfg.Backend(),
				AccessKey: storageCfg.AccessKey(),
				SecretKey: storageCfg.SecretKey(),
				Bucket:    storageCfg.Bucket(),
				Region:    storageCfg.Region(),
				CDNDomain: storageCfg.CDNDomain(),
			}
		}

		// Search config
		if searchCfg := cloudCfg.Search(); searchCfg != nil {
			cfg.OpenSearch = OpenSearchConfig{
				Host:     searchCfg.Host(),
				Port:     searchCfg.Port(),
				Username: searchCfg.Username(),
				Password: searchCfg.Password(),
				UseSSL:   searchCfg.UseSSL(),
				Index:    searchCfg.Index(),
			}
		}

		// DigitalOcean config
		cfg.DigitalOcean = cloudCfg.DigitalOcean()
	}

	// Convert AI config
	if aiCfg := newCfg.AI(); aiCfg != nil {
		// OpenAI
		if openaiCfg := aiCfg.OpenAI(); openaiCfg != nil {
			cfg.OpenAI = OpenAIConfig{
				APIKey:  openaiCfg.APIKey(),
				Model:   openaiCfg.Model(),
				Timeout: openaiCfg.Timeout(),
			}
		}

		// Full AI config with all providers
		cfg.AI = AIConfig{
			OpenAI:         cfg.OpenAI,
			EnableFallback: aiCfg.EnableFallback(),
			RetryAttempts:  aiCfg.RetryAttempts(),
			RetryDelay:     aiCfg.RetryDelay(),
		}

		// Claude
		if claudeCfg := aiCfg.Claude(); claudeCfg != nil {
			cfg.AI.Claude = ClaudeConfig{
				APIKey:  claudeCfg.APIKey(),
				Model:   claudeCfg.Model(),
				BaseURL: claudeCfg.BaseURL(),
				Timeout: claudeCfg.Timeout(),
			}
		}

		// Ollama
		if ollamaCfg := aiCfg.Ollama(); ollamaCfg != nil {
			cfg.AI.Ollama = OllamaConfig{
				BaseURL: ollamaCfg.BaseURL(),
				Model:   ollamaCfg.Model(),
				Timeout: ollamaCfg.Timeout(),
			}
		}
	}

	// Set reasonable defaults for Logging if not configured
	cfg.Logging = LoggingConfig{
		Level:              "info",
		Format:             "text",
		EnableRequestLog:   true,
		EnableErrorDetails: false,
		EnableStackTrace:   false,
	}

	// Database config - set empty defaults (not used in new architecture)
	cfg.Database = DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Username: "postgres",
		Password: "",
		Database: "motion_index",
		UseSSL:   false,
	}

	return cfg, nil
}

// FromNew creates an old Config from a new configuration.Config.
// This is useful for tests and migration scenarios.
func FromNew(newCfg *configuration.Config) (*Config, error) {
	return convertToOldConfig(newCfg)
}

// ToNew converts an old Config to the new modular configuration.
// This is useful for reverse compatibility scenarios.
func (c *Config) ToNew() (*configuration.Config, error) {
	builder := configuration.NewBuilder()

	// Server config
	serverCfg, err := server.NewConfig(c.Server.Port,
		server.WithMaxRequestSize(c.Server.MaxRequestSize),
		server.WithProduction(c.Server.Production),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create server config: %w", err)
	}
	builder.WithServer(serverCfg)

	// Auth config
	authCfg, err := auth.NewConfig(
		c.Auth.JWTSecret,
		c.Auth.SupabaseURL,
		c.Auth.SupabaseAnonKey,
		c.Auth.SupabaseAPIKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth config: %w", err)
	}
	builder.WithAuth(authCfg)

	// Processing config
	procCfg, err := processing.NewConfig(
		processing.WithMaxFileSize(c.Processing.MaxFileSize),
		processing.WithMaxWorkers(c.Processing.MaxWorkers),
		processing.WithBatchSize(c.Processing.BatchSize),
		processing.WithProcessTimeout(c.Processing.ProcessTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create processing config: %w", err)
	}
	builder.WithProcessing(procCfg)

	// Cloud config
	cloudCfg, err := buildCloudConfig(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud config: %w", err)
	}
	if cloudCfg != nil {
		builder.WithCloud(cloudCfg)
	}

	// AI config
	aiCfg, err := buildAIConfig(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI config: %w", err)
	}
	if aiCfg != nil {
		builder.WithAI(aiCfg)
	}

	builder.WithEnvironment(c.Environment)

	return builder.Build()
}

func buildCloudConfig(c *Config) (*cloud.Config, error) {
	var options []cloud.Option

	// Storage
	storageCfg, err := cloud.NewStorageConfig(c.Storage.Backend,
		cloud.WithStorageCredentials(c.Storage.AccessKey, c.Storage.SecretKey),
		cloud.WithStorageBucket(c.Storage.Bucket),
		cloud.WithStorageRegion(c.Storage.Region),
		cloud.WithCDNDomain(c.Storage.CDNDomain),
	)
	if err != nil {
		return nil, err
	}
	options = append(options, cloud.WithStorage(storageCfg))

	// Search
	if c.OpenSearch.Host != "" {
		searchCfg, err := cloud.NewSearchConfig(c.OpenSearch.Host, c.OpenSearch.Port,
			cloud.WithSearchCredentials(c.OpenSearch.Username, c.OpenSearch.Password),
			cloud.WithSearchSSL(c.OpenSearch.UseSSL),
			cloud.WithSearchIndex(c.OpenSearch.Index),
		)
		if err != nil {
			return nil, err
		}
		options = append(options, cloud.WithSearch(searchCfg))
	}

	// DigitalOcean
	if c.DigitalOcean != nil {
		options = append(options, cloud.WithDigitalOcean(c.DigitalOcean))
	}

	return cloud.NewConfig(options...)
}

func buildAIConfig(c *Config) (*ai.Config, error) {
	var options []ai.Option

	// OpenAI
	if c.AI.OpenAI.APIKey != "" {
		openaiCfg, err := ai.NewOpenAIConfig(
			c.AI.OpenAI.APIKey,
			c.AI.OpenAI.Model,
			ai.WithOpenAITimeout(c.AI.OpenAI.Timeout),
		)
		if err != nil {
			return nil, err
		}
		options = append(options, ai.WithOpenAI(openaiCfg))
	}

	// Claude
	if c.AI.Claude.APIKey != "" {
		claudeCfg, err := ai.NewClaudeConfig(
			c.AI.Claude.APIKey,
			c.AI.Claude.Model,
			ai.WithClaudeBaseURL(c.AI.Claude.BaseURL),
			ai.WithClaudeTimeout(c.AI.Claude.Timeout),
		)
		if err != nil {
			return nil, err
		}
		options = append(options, ai.WithClaude(claudeCfg))
	}

	// Ollama
	if c.AI.Ollama.Model != "" {
		ollamaCfg, err := ai.NewOllamaConfig(c.AI.Ollama.Model,
			ai.WithOllamaBaseURL(c.AI.Ollama.BaseURL),
			ai.WithOllamaTimeout(c.AI.Ollama.Timeout),
		)
		if err != nil {
			return nil, err
		}
		options = append(options, ai.WithOllama(ollamaCfg))
	}

	// If no providers configured, return nil
	if len(options) == 0 {
		return nil, nil
	}

	options = append(options,
		ai.WithFallback(c.AI.EnableFallback),
		ai.WithRetryAttempts(c.AI.RetryAttempts),
		ai.WithRetryDelay(c.AI.RetryDelay),
	)

	return ai.NewConfig(options...)
}
