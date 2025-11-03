package auth

import (
	"errors"
	"fmt"
	"net/url"
)

// Config holds authentication configuration.
type Config struct {
	jwtSecret       string
	supabaseURL     string
	supabaseAnonKey string
	supabaseAPIKey  string
}

// NewConfig creates an authentication configuration with validation.
func NewConfig(jwtSecret, supabaseURL, supabaseAnonKey, supabaseAPIKey string) (*Config, error) {
	cfg := &Config{
		jwtSecret:       jwtSecret,
		supabaseURL:     supabaseURL,
		supabaseAnonKey: supabaseAnonKey,
		supabaseAPIKey:  supabaseAPIKey,
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that the authentication configuration is valid.
func (c *Config) Validate() error {
	if c.jwtSecret == "" {
		return errors.New("JWT secret is required")
	}

	if c.supabaseURL == "" {
		return errors.New("Supabase URL is required")
	}

	if !isValidURL(c.supabaseURL) {
		return fmt.Errorf("Supabase URL must be a valid URL: %s", c.supabaseURL)
	}

	if c.supabaseAnonKey == "" {
		return errors.New("Supabase anon key is required")
	}

	if c.supabaseAPIKey == "" {
		return errors.New("Supabase API key is required")
	}

	return nil
}

// JWTSecret returns the JWT secret.
func (c *Config) JWTSecret() string {
	return c.jwtSecret
}

// SupabaseURL returns the Supabase URL.
func (c *Config) SupabaseURL() string {
	return c.supabaseURL
}

// SupabaseAnonKey returns the Supabase anonymous key.
func (c *Config) SupabaseAnonKey() string {
	return c.supabaseAnonKey
}

// SupabaseAPIKey returns the Supabase API key.
func (c *Config) SupabaseAPIKey() string {
	return c.supabaseAPIKey
}

// isValidURL validates if a string is a valid URL.
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
