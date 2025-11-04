package main

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Config struct {
	APIBaseURL      string        `json:"api_base_url"`
	RequestTimeout  time.Duration `json:"request_timeout"`
	RetryAttempts   int           `json:"retry_attempts"`
	RetryDelay      time.Duration `json:"retry_delay"`
	ProcessingDelay time.Duration `json:"processing_delay"`
}

type DocumentInfo struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	FileType     string    `json:"file_type"`
	Filename     string    `json:"filename"`
}

type DocumentListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Documents      []DocumentInfo `json:"documents"`
		NextCursor     string         `json:"next_cursor"`
		HasMore        bool           `json:"has_more"`
		TotalReturned  int            `json:"total_returned"`
		TotalEstimated int            `json:"total_estimated"`
	} `json:"data"`
	Message string `json:"message"`
}

type ProcessResult struct {
	DocumentID     string                 `json:"document_id"`
	Classification map[string]interface{} `json:"classification"`
	ExtractedText  string                 `json:"extracted_text"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type ClassificationStats struct {
	TotalDocuments     int64         `json:"total_documents"`
	ProcessedDocuments int64         `json:"processed_documents"`
	SuccessfulDocs     int64         `json:"successful_docs"`
	FailedDocs         int64         `json:"failed_docs"`
	SkippedDocs        int64         `json:"skipped_docs"`
	StartTime          time.Time     `json:"start_time"`
	Duration           time.Duration `json:"duration"`
	Rate               float64       `json:"rate_per_minute"`
}

type ProcessingError struct {
	DocumentPath string    `json:"document_path"`
	Error        string    `json:"error"`
	Timestamp    time.Time `json:"timestamp"`
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Get command line arguments
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Load configuration
	cfg := loadConfig()

	switch command {
	case "classify-all":
		skip := 0
		if len(os.Args) > 2 {
			if s, err := strconv.Atoi(os.Args[2]); err == nil && s >= 0 {
				skip = s
			} else {
				log.Printf("Invalid skip count, using default: %d", skip)
			}
		}
		classifyAllDocuments(cfg, skip)
	case "classify-count":
		count := 10
		if len(os.Args) > 2 {
			if c, err := strconv.Atoi(os.Args[2]); err == nil && c > 0 {
				count = c
			} else {
				log.Printf("Invalid count, using default: %d", count)
			}
		}
		classifyDocumentsCount(cfg, count)
	case "test-connection":
		testAPIConnection(cfg)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Single-Threaded Document Classifier")
	fmt.Println("===================================")
	fmt.Println()
	fmt.Println("Usage: go run cmd/api-classifier/main.go <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  test-connection        - Test API connection and authentication")
	fmt.Println("  classify-count [N]     - Classify first N documents (default: 10)")
	fmt.Println("  classify-all [SKIP]    - Classify ALL documents in storage (sequential)")
	fmt.Println("                          SKIP: Optional number of documents to skip from the beginning")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run cmd/api-classifier/main.go test-connection")
	fmt.Println("  go run cmd/api-classifier/main.go classify-count 50")
	fmt.Println("  go run cmd/api-classifier/main.go classify-all")
	fmt.Println("  go run cmd/api-classifier/main.go classify-all 300    # Skip first 300 documents")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  API_BASE_URL          - Base URL for the Motion Index API (default: http://localhost:8003)")
	fmt.Println("  REQUEST_TIMEOUT       - Request timeout in seconds (default: 120)")
	fmt.Println("  RETRY_ATTEMPTS        - Number of retry attempts (default: 3)")
	fmt.Println("  PROCESSING_DELAY      - Delay between documents in milliseconds (default: 100)")
}

func loadConfig() *Config {
	cfg := &Config{
		APIBaseURL:      getEnv("API_BASE_URL", "http://localhost:8003"),
		RequestTimeout:  time.Duration(getEnvInt("REQUEST_TIMEOUT", 120)) * time.Second,
		RetryAttempts:   getEnvInt("RETRY_ATTEMPTS", 3),
		RetryDelay:      time.Duration(getEnvInt("RETRY_DELAY_SECONDS", 5)) * time.Second,
		ProcessingDelay: time.Duration(getEnvInt("PROCESSING_DELAY_MS", 100)) * time.Millisecond,
	}

	fmt.Printf("🔧 Configuration loaded:\n")
	fmt.Printf("   API Base URL: %s\n", cfg.APIBaseURL)
	fmt.Printf("   Request Timeout: %s\n", cfg.RequestTimeout)
	fmt.Printf("   Retry Attempts: %d\n", cfg.RetryAttempts)
	fmt.Printf("   Processing Delay: %s\n", cfg.ProcessingDelay)
	fmt.Println()

	return cfg
}

func testAPIConnection(cfg *Config) {
	fmt.Println("🔍 Testing API Connection")
	fmt.Println("=========================")

	client := &http.Client{Timeout: cfg.RequestTimeout}

	// Test health endpoint
	fmt.Println("📊 Testing health endpoint...")
	resp, err := client.Get(cfg.APIBaseURL + "/health")
	if err != nil {
		log.Fatalf("❌ Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("❌ Health check failed: HTTP %d", resp.StatusCode)
	}
	fmt.Println("✅ Health endpoint OK")

	// Test document listing
	fmt.Println("📋 Testing document listing...")
	resp, err = client.Get(cfg.APIBaseURL + "/api/v1/storage/documents?limit=5")
	if err != nil {
		log.Fatalf("❌ Document listing failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("❌ Document listing failed: HTTP %d", resp.StatusCode)
	}

	var docResp DocumentListResponse
	if err := json.NewDecoder(resp.Body).Decode(&docResp); err != nil {
		log.Fatalf("❌ Failed to decode document response: %v", err)
	}

	fmt.Printf("✅ Document listing OK - found %d documents\n", docResp.Data.TotalEstimated)

	// Test processing endpoint
	fmt.Println("🔄 Testing processing endpoint availability...")
	testURL := cfg.APIBaseURL + "/api/v1/categorise"
	req, _ := http.NewRequest("POST", testURL, nil)
	req.Header.Set("Content-Type", "multipart/form-data")

	resp, err = client.Do(req)
	if err != nil {
		log.Printf("⚠️  Processing endpoint test failed: %v", err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == 400 {
			fmt.Println("✅ Processing endpoint available (expected 400 for empty request)")
		} else {
			fmt.Printf("⚠️  Processing endpoint returned: HTTP %d\n", resp.StatusCode)
		}
	}

	fmt.Println("✅ API connection test complete!")
}

func classifyAllDocuments(cfg *Config, skip int) {
	if skip > 0 {
		fmt.Printf("🚀 Single-Threaded Classification of All Documents (skipping first %d)\n", skip)
	} else {
		fmt.Println("🚀 Single-Threaded Classification of All Documents")
	}
	fmt.Println("===================================================")

	startTime := time.Now()
	stats := &ClassificationStats{
		StartTime: startTime,
	}

	// Get total document count first
	totalDocs, err := getTotalDocumentCount(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to get document count: %v", err)
	}

	stats.TotalDocuments = int64(totalDocs)
	fmt.Printf("📊 Found %d total documents to process\n", totalDocs)

	if skip > 0 {
		fmt.Printf("📊 Skipping first %d documents, will process %d documents\n", skip, totalDocs-skip)
		if skip >= totalDocs {
			fmt.Printf("⚠️  Skip count (%d) is >= total documents (%d), nothing to process\n", skip, totalDocs)
			return
		}
	}

	if totalDocs == 0 {
		fmt.Println("⚠️  No documents found to classify")
		return
	}

	// Process all documents using pagination
	processAllDocumentsSequentially(cfg, stats, skip)

	// Final statistics
	stats.Duration = time.Since(startTime)
	if stats.Duration.Minutes() > 0 {
		stats.Rate = float64(stats.ProcessedDocuments) / stats.Duration.Minutes()
	}

	printFinalStats(stats)
}

func classifyDocumentsCount(cfg *Config, maxDocuments int) {
	fmt.Printf("🚀 Single-Threaded Classification of %d Documents\n", maxDocuments)
	fmt.Println("===============================================")

	startTime := time.Now()
	stats := &ClassificationStats{
		StartTime: startTime,
	}

	// Create single HTTP client for all operations
	client := newHTTPClient(cfg)

	// Get documents with limit
	documents, err := getDocuments(cfg, "", maxDocuments)
	if err != nil {
		log.Fatalf("❌ Failed to get documents: %v", err)
	}

	stats.TotalDocuments = int64(len(documents))
	fmt.Printf("📊 Retrieved %d documents for processing\n", len(documents))

	if len(documents) == 0 {
		fmt.Println("⚠️  No documents found to classify")
		return
	}

	// Process documents sequentially with shared client
	processDocumentListSequentiallyWithClient(cfg, client, documents, stats)

	// Final statistics
	stats.Duration = time.Since(startTime)
	if stats.Duration.Minutes() > 0 {
		stats.Rate = float64(stats.ProcessedDocuments) / stats.Duration.Minutes()
	}

	printFinalStats(stats)
}

func processAllDocumentsSequentially(cfg *Config, stats *ClassificationStats, skip int) {
	cursor := ""
	totalProcessed := 0
	totalSkipped := 0
	batchSize := 50 // Process documents in batches for memory efficiency

	// Create a single reusable HTTP client for all operations
	client := newHTTPClient(cfg)

	for {
		// Get batch of documents
		documents, nextCursor, hasMore, err := getDocumentsBatch(cfg, cursor, batchSize)
		if err != nil {
			log.Printf("❌ Failed to get document batch: %v", err)
			break
		}

		if len(documents) == 0 {
			break
		}

		cursorDisplay := cursor
		if len(cursorDisplay) > 8 {
			cursorDisplay = cursorDisplay[:8]
		}

		// Filter out documents we want to skip
		var documentsToProcess []DocumentInfo
		for _, doc := range documents {
			if totalSkipped+totalProcessed < skip {
				totalSkipped++
				fmt.Printf("⏭️  Skipping document %d: %s\n", totalSkipped, doc.Path)
			} else {
				documentsToProcess = append(documentsToProcess, doc)
			}
		}

		if len(documentsToProcess) > 0 {
			fmt.Printf("📋 Processing batch of %d documents (skipped %d, cursor: %s)\n",
				len(documentsToProcess), totalSkipped, cursorDisplay)

			// Process this batch sequentially with shared client
			processDocumentListSequentiallyWithClient(cfg, client, documentsToProcess, stats)
			totalProcessed += len(documentsToProcess)
		} else {
			fmt.Printf("📋 Skipped entire batch of %d documents (total skipped: %d)\n",
				len(documents), totalSkipped)
		}

		fmt.Printf("📊 Progress: %d processed, %d skipped, %d/%d total (%.1f%%)\n",
			totalProcessed, totalSkipped, totalSkipped+totalProcessed,
			stats.TotalDocuments, float64(totalSkipped+totalProcessed)/float64(stats.TotalDocuments)*100)

		if !hasMore {
			break
		}
		cursor = nextCursor

		// Small delay between batches
		time.Sleep(1 * time.Second)
	}
}

func processDocumentListSequentially(cfg *Config, documents []DocumentInfo, stats *ClassificationStats) {
	// Create a single client for backward compatibility
	client := newHTTPClient(cfg)
	processDocumentListSequentiallyWithClient(cfg, client, documents, stats)
}
