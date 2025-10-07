package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"

	"motion-index-fiber/internal/config"
	"motion-index-fiber/pkg/search/client"
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

	fmt.Println("🔍 OpenSearch Index Inspection")
	fmt.Println("==============================")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create OpenSearch client
	osClient, err := client.NewClient(&cfg.OpenSearch)
	if err != nil {
		log.Fatalf("❌ Failed to create OpenSearch client: %v", err)
	}

	fmt.Printf("📋 Inspecting index: %s\n", cfg.OpenSearch.Index)

	// Get index mapping
	mappingReq := opensearchapi.IndicesGetMappingRequest{
		Index: []string{cfg.OpenSearch.Index},
	}

	mappingRes, err := mappingReq.Do(ctx, osClient.GetClient())
	if err != nil {
		log.Fatalf("❌ Failed to get index mapping: %v", err)
	}
	defer mappingRes.Body.Close()

	if mappingRes.IsError() {
		fmt.Printf("⚠️  Failed to get mapping: %s\n", mappingRes.Status())
	} else {
		var mappingResponse map[string]interface{}
		if err := json.NewDecoder(mappingRes.Body).Decode(&mappingResponse); err != nil {
			log.Printf("❌ Failed to decode mapping response: %v", err)
		} else {
			fmt.Println("📊 Current index mapping:")
			mappingJSON, _ := json.MarshalIndent(mappingResponse, "", "  ")
			fmt.Printf("%s\n", mappingJSON)
		}
	}

	// Test indexing with minimal document
	fmt.Println("\n🧪 Testing minimal document indexing...")
	
	minimalDoc := map[string]interface{}{
		"id":         "test-minimal",
		"file_name":  "test.pdf",
		"text":       "test content",
		"doc_type":   "Other",
		"created_at": time.Now().Format(time.RFC3339),
	}

	docJSON, _ := json.Marshal(minimalDoc)
	indexReq := opensearchapi.IndexRequest{
		Index:      cfg.OpenSearch.Index,
		DocumentID: "test-minimal",
		Body:       strings.NewReader(string(docJSON)),
	}

	indexRes, err := indexReq.Do(ctx, osClient.GetClient())
	if err != nil {
		fmt.Printf("❌ Minimal indexing failed: %v\n", err)
	} else {
		defer indexRes.Body.Close()
		if indexRes.IsError() {
			fmt.Printf("❌ Minimal indexing failed: %s\n", indexRes.Status())
			
			// Get error details
			var errorResponse map[string]interface{}
			if err := json.NewDecoder(indexRes.Body).Decode(&errorResponse); err == nil {
				errorJSON, _ := json.MarshalIndent(errorResponse, "", "  ")
				fmt.Printf("📋 Error details:\n%s\n", errorJSON)
			}
		} else {
			fmt.Println("✅ Minimal document indexed successfully")
		}
	}

	fmt.Println("\n💡 Expected document structure:")
	expectedDoc := models.GetDocumentMapping()
	expectedJSON, _ := json.MarshalIndent(expectedDoc, "", "  ")
	fmt.Printf("%s\n", expectedJSON)
}