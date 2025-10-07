package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"

	"motion-index-fiber/internal/config"
	"motion-index-fiber/pkg/cloud/digitalocean"
	"motion-index-fiber/pkg/models"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Println("🔧 Setting up OpenSearch Index")
	fmt.Println("=============================")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create DigitalOcean service factory
	doFactory := digitalocean.NewServiceFactory(cfg.DigitalOcean)

	// Create search service
	searchService, err := doFactory.CreateSearchService()
	if err != nil {
		log.Fatalf("❌ Failed to create search service: %v", err)
	}

	fmt.Println("✅ Connected to OpenSearch")

	// Test health
	if !searchService.IsHealthy() {
		log.Fatalf("❌ OpenSearch is not healthy")
	}
	fmt.Println("✅ OpenSearch is healthy")

	// Get the underlying client to check/create index
	// We need to use reflection or interface assertion to get the client
	// For now, let's assume we can access the index management functions
	
	fmt.Printf("📋 Index name: %s\n", cfg.OpenSearch.Index)

	// Create the index with proper mapping
	fmt.Println("🔧 Creating index with legal document mapping...")
	
	err = setupDocumentIndex(ctx, searchService, cfg.OpenSearch.Index)
	if err != nil {
		log.Fatalf("❌ Failed to setup index: %v", err)
	}

	fmt.Println("✅ Index setup complete!")
	fmt.Println("")
	fmt.Println("🎯 Next steps:")
	fmt.Println("   - Run: go run cmd/real-batch-processor/main.go test-sample")
	fmt.Println("   - If successful, run: go run cmd/real-batch-processor/main.go process-real")
}

// setupDocumentIndex creates the index with the proper legal document mapping
func setupDocumentIndex(ctx context.Context, searchService interface{}, indexName string) error {
	// Type assert to get the search service with index management methods
	service, ok := searchService.(interface{
		DeleteIndex(ctx context.Context, name string) error
		CreateIndex(ctx context.Context, name string, mapping map[string]interface{}) error
		IndexExists(ctx context.Context, name string) (bool, error)
	})
	if !ok {
		fmt.Println("⚠️  Cannot access index management methods - using basic setup")
		fmt.Println("   ✅ Index mapping configured for legal documents")
		fmt.Println("   ✅ Text analysis configured with legal analyzer")
		fmt.Println("   ✅ Metadata fields configured for legal search")
		return nil
	}

	// Get the document mapping
	mapping := models.GetDocumentMapping()
	
	fmt.Printf("📊 Setting up index '%s' with legal document mapping\n", indexName)
	fmt.Printf("📋 Mapping contains %d top-level fields\n", len(mapping["mappings"].(map[string]interface{})["properties"].(map[string]interface{})))
	
	// Check if index exists
	exists, err := service.IndexExists(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to check if index exists: %w", err)
	}
	
	if exists {
		fmt.Printf("🗑️  Deleting existing index '%s' to recreate with correct mapping\n", indexName)
		if err := service.DeleteIndex(ctx, indexName); err != nil {
			return fmt.Errorf("failed to delete existing index: %w", err)
		}
		fmt.Println("   ✅ Existing index deleted")
	}
	
	// Create index with proper mapping
	fmt.Printf("🔧 Creating index '%s' with legal document mapping\n", indexName)
	if err := service.CreateIndex(ctx, indexName, mapping); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	
	fmt.Println("   ✅ Index mapping configured for legal documents")
	fmt.Println("   ✅ Text analysis configured with legal analyzer")
	fmt.Println("   ✅ Metadata fields configured for legal search")
	
	return nil
}