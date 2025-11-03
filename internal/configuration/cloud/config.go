package cloud

import (
	"errors"
	"fmt"

	doConfig "motion-index-fiber/pkg/cloud/digitalocean/config"
)

// Config holds cloud service configuration (storage, search, etc.).
type Config struct {
	storage      *StorageConfig
	search       *SearchConfig
	digitalOcean *doConfig.Config
}

// NewConfig creates a cloud configuration.
func NewConfig(options ...Option) (*Config, error) {
	cfg := &Config{}

	for _, opt := range options {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Option is a functional option for cloud configuration.
type Option func(*Config)

// WithStorage sets the storage configuration.
func WithStorage(cfg *StorageConfig) Option {
	return func(c *Config) {
		c.storage = cfg
	}
}

// WithSearch sets the search configuration.
func WithSearch(cfg *SearchConfig) Option {
	return func(c *Config) {
		c.search = cfg
	}
}

// WithDigitalOcean sets the DigitalOcean configuration.
func WithDigitalOcean(cfg *doConfig.Config) Option {
	return func(c *Config) {
		c.digitalOcean = cfg
	}
}

// Validate checks that the cloud configuration is valid.
func (c *Config) Validate() error {
	if c.storage != nil {
		if err := c.storage.Validate(); err != nil {
			return fmt.Errorf("storage config: %w", err)
		}
	}

	if c.search != nil {
		if err := c.search.Validate(); err != nil {
			return fmt.Errorf("search config: %w", err)
		}
	}

	return nil
}

// Storage returns the storage configuration.
func (c *Config) Storage() *StorageConfig {
	return c.storage
}

// Search returns the search configuration.
func (c *Config) Search() *SearchConfig {
	return c.search
}

// DigitalOcean returns the DigitalOcean configuration.
func (c *Config) DigitalOcean() *doConfig.Config {
	return c.digitalOcean
}

// StorageConfig holds object storage configuration.
type StorageConfig struct {
	backend   string
	accessKey string
	secretKey string
	bucket    string
	region    string
	cdnDomain string
}

// NewStorageConfig creates a storage configuration.
func NewStorageConfig(backend string, options ...StorageOption) (*StorageConfig, error) {
	cfg := &StorageConfig{
		backend: backend,
	}

	for _, opt := range options {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// StorageOption is a functional option for storage configuration.
type StorageOption func(*StorageConfig)

// WithStorageCredentials sets the storage access and secret keys.
func WithStorageCredentials(accessKey, secretKey string) StorageOption {
	return func(c *StorageConfig) {
		c.accessKey = accessKey
		c.secretKey = secretKey
	}
}

// WithStorageBucket sets the storage bucket name.
func WithStorageBucket(bucket string) StorageOption {
	return func(c *StorageConfig) {
		c.bucket = bucket
	}
}

// WithStorageRegion sets the storage region.
func WithStorageRegion(region string) StorageOption {
	return func(c *StorageConfig) {
		c.region = region
	}
}

// WithCDNDomain sets the CDN domain.
func WithCDNDomain(domain string) StorageOption {
	return func(c *StorageConfig) {
		c.cdnDomain = domain
	}
}

// Validate checks that the storage configuration is valid.
func (c *StorageConfig) Validate() error {
	if c.backend != "local" && c.backend != "spaces" {
		return fmt.Errorf("storage backend must be 'local' or 'spaces', got '%s'", c.backend)
	}

	// For spaces backend, validate required credentials
	if c.backend == "spaces" {
		if c.accessKey == "" {
			return errors.New("storage access key is required for spaces backend")
		}
		if c.secretKey == "" {
			return errors.New("storage secret key is required for spaces backend")
		}
		if c.bucket == "" {
			return errors.New("storage bucket is required for spaces backend")
		}
		if c.region == "" {
			return errors.New("storage region is required for spaces backend")
		}
	}

	return nil
}

// Backend returns the storage backend.
func (c *StorageConfig) Backend() string {
	return c.backend
}

// AccessKey returns the storage access key.
func (c *StorageConfig) AccessKey() string {
	return c.accessKey
}

// SecretKey returns the storage secret key.
func (c *StorageConfig) SecretKey() string {
	return c.secretKey
}

// Bucket returns the storage bucket name.
func (c *StorageConfig) Bucket() string {
	return c.bucket
}

// Region returns the storage region.
func (c *StorageConfig) Region() string {
	return c.region
}

// CDNDomain returns the CDN domain.
func (c *StorageConfig) CDNDomain() string {
	return c.cdnDomain
}

// Endpoint returns the storage endpoint URL.
func (c *StorageConfig) Endpoint() string {
	if c.backend == "spaces" {
		return fmt.Sprintf("https://%s.%s.digitaloceanspaces.com", c.bucket, c.region)
	}
	return ""
}

// CDNEndpoint returns the CDN endpoint URL.
func (c *StorageConfig) CDNEndpoint() string {
	if c.cdnDomain != "" {
		return fmt.Sprintf("https://%s", c.cdnDomain)
	}
	if c.backend == "spaces" {
		return fmt.Sprintf("https://%s.%s.cdn.digitaloceanspaces.com", c.bucket, c.region)
	}
	return ""
}

// SearchConfig holds search engine configuration (OpenSearch/Elasticsearch).
type SearchConfig struct {
	host     string
	port     int
	username string
	password string
	useSSL   bool
	index    string
}

// NewSearchConfig creates a search configuration.
func NewSearchConfig(host string, port int, options ...SearchOption) (*SearchConfig, error) {
	cfg := &SearchConfig{
		host:   host,
		port:   port,
		useSSL: true,
		index:  "documents",
	}

	for _, opt := range options {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// SearchOption is a functional option for search configuration.
type SearchOption func(*SearchConfig)

// WithSearchCredentials sets the search username and password.
func WithSearchCredentials(username, password string) SearchOption {
	return func(c *SearchConfig) {
		c.username = username
		c.password = password
	}
}

// WithSearchSSL enables or disables SSL.
func WithSearchSSL(useSSL bool) SearchOption {
	return func(c *SearchConfig) {
		c.useSSL = useSSL
	}
}

// WithSearchIndex sets the search index name.
func WithSearchIndex(index string) SearchOption {
	return func(c *SearchConfig) {
		c.index = index
	}
}

// Validate checks that the search configuration is valid.
func (c *SearchConfig) Validate() error {
	if c.host == "" {
		return errors.New("search host is required")
	}

	if c.port < 1 || c.port > 65535 {
		return fmt.Errorf("search port must be between 1 and 65535, got %d", c.port)
	}

	if c.username == "" {
		return errors.New("search username is required")
	}

	if c.password == "" {
		return errors.New("search password is required")
	}

	if c.index == "" {
		return errors.New("search index is required")
	}

	return nil
}

// Host returns the search host.
func (c *SearchConfig) Host() string {
	return c.host
}

// Port returns the search port.
func (c *SearchConfig) Port() int {
	return c.port
}

// Username returns the search username.
func (c *SearchConfig) Username() string {
	return c.username
}

// Password returns the search password.
func (c *SearchConfig) Password() string {
	return c.password
}

// UseSSL returns whether SSL is enabled.
func (c *SearchConfig) UseSSL() bool {
	return c.useSSL
}

// Index returns the search index name.
func (c *SearchConfig) Index() string {
	return c.index
}

// URL returns the full search URL.
func (c *SearchConfig) URL() string {
	protocol := "http"
	if c.useSSL {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s:%d", protocol, c.host, c.port)
}
