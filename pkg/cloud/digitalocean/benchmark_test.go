package digitalocean

import (
	"os"
	"testing"

	"motion-index-fiber/pkg/cloud/digitalocean/config"
)

// BenchmarkProviderCreation benchmarks provider creation from environment
func BenchmarkProviderCreation(b *testing.B) {
	// Setup test environment
	os.Setenv("ENVIRONMENT", "local")
	defer os.Unsetenv("ENVIRONMENT")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider, err := NewProviderFromEnvironment()
		if err != nil {
			b.Fatal(err)
		}
		_ = provider.Shutdown()
	}
}

// BenchmarkConfigurationValidation benchmarks configuration validation
func BenchmarkConfigurationValidation(b *testing.B) {
	cfg := config.DefaultConfig()
	// Set required fields for validation
	cfg.DigitalOcean.OpenSearch.Port = 25060
	cfg.DigitalOcean.OpenSearch.Index = "documents"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := cfg.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkServiceFactoryCreation benchmarks service factory creation
func BenchmarkServiceFactoryCreation(b *testing.B) {
	cfg := config.DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		factory := NewServiceFactory(cfg)
		_ = factory
	}
}

// BenchmarkEndpointGeneration benchmarks endpoint URL generation
func BenchmarkEndpointGeneration(b *testing.B) {
	cfg := config.DefaultConfig()
	cfg.DigitalOcean.Spaces.Region = "nyc3"
	cfg.DigitalOcean.Spaces.Bucket = "test-bucket"
	cfg.DigitalOcean.OpenSearch.Host = "test-host.db.ondigitalocean.com"
	cfg.DigitalOcean.OpenSearch.Port = 25060
	cfg.DigitalOcean.OpenSearch.UseSSL = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.GetSpacesEndpoint()
		_ = cfg.GetSpacesCDNEndpoint()
		_ = cfg.GetOpenSearchEndpoint()
	}
}

// BenchmarkProviderMethodCalls benchmarks common provider method calls
func BenchmarkProviderMethodCalls(b *testing.B) {
	provider, err := DefaultProvider()
	if err != nil {
		b.Fatal(err)
	}
	defer provider.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.GetConfig()
		_ = provider.GetEnvironment()
		_ = provider.IsLocal()
		_ = provider.IsHealthy()
		_ = provider.GetMetrics()
	}
}
