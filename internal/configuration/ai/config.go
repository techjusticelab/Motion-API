package ai

import (
	"errors"
	"fmt"
	"time"
)

// Config holds AI provider configuration with fallback support.
type Config struct {
	openAI         *OpenAIConfig
	claude         *ClaudeConfig
	ollama         *OllamaConfig
	enableFallback bool
	retryAttempts  int
	retryDelay     time.Duration
}

// NewConfig creates an AI configuration.
func NewConfig(options ...Option) (*Config, error) {
	cfg := &Config{
		enableFallback: true,
		retryAttempts:  3,
		retryDelay:     5 * time.Second,
	}

	for _, opt := range options {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Option is a functional option for AI configuration.
type Option func(*Config)

// WithOpenAI sets the OpenAI provider configuration.
func WithOpenAI(cfg *OpenAIConfig) Option {
	return func(c *Config) {
		c.openAI = cfg
	}
}

// WithClaude sets the Claude provider configuration.
func WithClaude(cfg *ClaudeConfig) Option {
	return func(c *Config) {
		c.claude = cfg
	}
}

// WithOllama sets the Ollama provider configuration.
func WithOllama(cfg *OllamaConfig) Option {
	return func(c *Config) {
		c.ollama = cfg
	}
}

// WithFallback enables or disables fallback between providers.
func WithFallback(enabled bool) Option {
	return func(c *Config) {
		c.enableFallback = enabled
	}
}

// WithRetryAttempts sets the number of retry attempts.
func WithRetryAttempts(attempts int) Option {
	return func(c *Config) {
		c.retryAttempts = attempts
	}
}

// WithRetryDelay sets the delay between retry attempts.
func WithRetryDelay(delay time.Duration) Option {
	return func(c *Config) {
		c.retryDelay = delay
	}
}

// Validate checks that at least one AI provider is configured.
func (c *Config) Validate() error {
	if c.openAI == nil && c.claude == nil && c.ollama == nil {
		return errors.New("at least one AI provider must be configured")
	}

	if c.retryAttempts < 0 {
		return errors.New("retry attempts must be non-negative")
	}

	if c.retryDelay < 0 {
		return errors.New("retry delay must be non-negative")
	}

	return nil
}

// OpenAI returns the OpenAI configuration.
func (c *Config) OpenAI() *OpenAIConfig {
	return c.openAI
}

// Claude returns the Claude configuration.
func (c *Config) Claude() *ClaudeConfig {
	return c.claude
}

// Ollama returns the Ollama configuration.
func (c *Config) Ollama() *OllamaConfig {
	return c.ollama
}

// EnableFallback returns whether fallback is enabled.
func (c *Config) EnableFallback() bool {
	return c.enableFallback
}

// RetryAttempts returns the number of retry attempts.
func (c *Config) RetryAttempts() int {
	return c.retryAttempts
}

// RetryDelay returns the delay between retries.
func (c *Config) RetryDelay() time.Duration {
	return c.retryDelay
}

// OpenAIConfig holds OpenAI-specific configuration.
type OpenAIConfig struct {
	apiKey  string
	model   string
	timeout time.Duration
}

// NewOpenAIConfig creates an OpenAI configuration.
func NewOpenAIConfig(apiKey, model string, options ...OpenAIOption) (*OpenAIConfig, error) {
	if apiKey == "" {
		return nil, errors.New("OpenAI API key is required")
	}
	if model == "" {
		model = "gpt-4" // Default model
	}

	cfg := &OpenAIConfig{
		apiKey:  apiKey,
		model:   model,
		timeout: 10 * time.Minute, // Default 10 minutes for LLMs
	}

	for _, opt := range options {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// OpenAIOption is a functional option for OpenAI configuration.
type OpenAIOption func(*OpenAIConfig)

// WithOpenAITimeout sets a custom timeout for OpenAI requests.
func WithOpenAITimeout(timeout time.Duration) OpenAIOption {
	return func(c *OpenAIConfig) {
		c.timeout = timeout
	}
}

// Validate checks that the OpenAI configuration is valid.
func (c *OpenAIConfig) Validate() error {
	if c.timeout <= 0 {
		return errors.New("OpenAI timeout must be positive")
	}
	return nil
}

// APIKey returns the OpenAI API key.
func (c *OpenAIConfig) APIKey() string {
	return c.apiKey
}

// Model returns the OpenAI model name.
func (c *OpenAIConfig) Model() string {
	return c.model
}

// Timeout returns the OpenAI request timeout.
func (c *OpenAIConfig) Timeout() time.Duration {
	return c.timeout
}

// ClaudeConfig holds Claude-specific configuration.
type ClaudeConfig struct {
	apiKey  string
	model   string
	baseURL string
	timeout time.Duration
}

// NewClaudeConfig creates a Claude configuration.
func NewClaudeConfig(apiKey, model string, options ...ClaudeOption) (*ClaudeConfig, error) {
	if apiKey == "" {
		return nil, errors.New("Claude API key is required")
	}
	if model == "" {
		model = "claude-3-5-sonnet-20241022" // Default model
	}

	cfg := &ClaudeConfig{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.anthropic.com",
		timeout: 10 * time.Minute, // Default 10 minutes for LLMs
	}

	for _, opt := range options {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// ClaudeOption is a functional option for Claude configuration.
type ClaudeOption func(*ClaudeConfig)

// WithClaudeBaseURL sets a custom base URL for Claude API.
func WithClaudeBaseURL(url string) ClaudeOption {
	return func(c *ClaudeConfig) {
		c.baseURL = url
	}
}

// WithClaudeTimeout sets a custom timeout for Claude requests.
func WithClaudeTimeout(timeout time.Duration) ClaudeOption {
	return func(c *ClaudeConfig) {
		c.timeout = timeout
	}
}

// Validate checks that the Claude configuration is valid.
func (c *ClaudeConfig) Validate() error {
	if c.timeout <= 0 {
		return errors.New("Claude timeout must be positive")
	}
	return nil
}

// APIKey returns the Claude API key.
func (c *ClaudeConfig) APIKey() string {
	return c.apiKey
}

// Model returns the Claude model name.
func (c *ClaudeConfig) Model() string {
	return c.model
}

// BaseURL returns the Claude API base URL.
func (c *ClaudeConfig) BaseURL() string {
	return c.baseURL
}

// Timeout returns the Claude request timeout.
func (c *ClaudeConfig) Timeout() time.Duration {
	return c.timeout
}

// OllamaConfig holds Ollama-specific configuration.
type OllamaConfig struct {
	baseURL string
	model   string
	timeout time.Duration
}

// NewOllamaConfig creates an Ollama configuration.
func NewOllamaConfig(model string, options ...OllamaOption) (*OllamaConfig, error) {
	if model == "" {
		return nil, errors.New("Ollama model is required")
	}

	cfg := &OllamaConfig{
		baseURL: "http://localhost:11434",
		model:   model,
		timeout: 120 * time.Second,
	}

	for _, opt := range options {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// OllamaOption is a functional option for Ollama configuration.
type OllamaOption func(*OllamaConfig)

// WithOllamaBaseURL sets the Ollama server base URL.
func WithOllamaBaseURL(url string) OllamaOption {
	return func(c *OllamaConfig) {
		c.baseURL = url
	}
}

// WithOllamaTimeout sets the Ollama request timeout.
func WithOllamaTimeout(timeout time.Duration) OllamaOption {
	return func(c *OllamaConfig) {
		c.timeout = timeout
	}
}

// Validate checks that the Ollama configuration is valid.
func (c *OllamaConfig) Validate() error {
	if c.baseURL == "" {
		return errors.New("Ollama base URL is required")
	}
	if c.timeout <= 0 {
		return errors.New("Ollama timeout must be positive")
	}
	return nil
}

// BaseURL returns the Ollama base URL.
func (c *OllamaConfig) BaseURL() string {
	return c.baseURL
}

// Model returns the Ollama model name.
func (c *OllamaConfig) Model() string {
	return c.model
}

// Timeout returns the Ollama request timeout.
func (c *OllamaConfig) Timeout() time.Duration {
	return c.timeout
}

// HasProvider checks if a specific provider is configured.
func (c *Config) HasProvider(provider string) bool {
	switch provider {
	case "openai":
		return c.openAI != nil
	case "claude":
		return c.claude != nil
	case "ollama":
		return c.ollama != nil
	default:
		return false
	}
}

// AvailableProviders returns a list of configured provider names.
func (c *Config) AvailableProviders() []string {
	var providers []string
	if c.openAI != nil {
		providers = append(providers, "openai")
	}
	if c.claude != nil {
		providers = append(providers, "claude")
	}
	if c.ollama != nil {
		providers = append(providers, "ollama")
	}
	return providers
}

// String returns a human-readable representation of the AI configuration.
func (c *Config) String() string {
	return fmt.Sprintf("AI Config: providers=%v, fallback=%v, retries=%d",
		c.AvailableProviders(), c.enableFallback, c.retryAttempts)
}
