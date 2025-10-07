package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Configuration for the single-threaded classifier
type Config struct {
	APIBaseURL      string        `json:"api_base_url"`
	RequestTimeout  time.Duration `json:"request_timeout"`
	RetryAttempts   int           `json:"retry_attempts"`
	RetryDelay      time.Duration `json:"retry_delay"`
	ProcessingDelay time.Duration `json:"processing_delay"`
}

// DocumentInfo represents a document from the storage API
type DocumentInfo struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	FileType     string    `json:"file_type"`
	Filename     string    `json:"filename"`
}

// DocumentListResponse represents the API response for document listing
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

// ProcessResult holds the results from document processing
type ProcessResult struct {
	DocumentID     string                 `json:"document_id"`
	Classification map[string]interface{} `json:"classification"`
	ExtractedText  string                 `json:"extracted_text"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// ClassificationStats tracks overall classification statistics
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

// ProcessingError represents an error during document processing
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

	// Process documents sequentially
	processDocumentListSequentially(cfg, documents, stats)

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

			// Process this batch sequentially
			processDocumentListSequentially(cfg, documentsToProcess, stats)
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
	client := &http.Client{Timeout: cfg.RequestTimeout}
	var errors []ProcessingError

	for i, doc := range documents {
		fmt.Printf("🔄 [%d/%d] Processing: %s\n", i+1, len(documents), doc.Path)

		// Process single document
		success, err := processDocument(cfg, client, doc)
		
		stats.ProcessedDocuments++
		
		if success {
			stats.SuccessfulDocs++
			fmt.Printf("✅ [%d/%d] Successfully processed: %s\n", i+1, len(documents), doc.Path)
		} else {
			// Failed processing
			stats.FailedDocs++
			errorInfo := ProcessingError{
				DocumentPath: doc.Path,
				Error:        fmt.Sprintf("%v", err),
				Timestamp:    time.Now(),
			}
			errors = append(errors, errorInfo)
			fmt.Printf("❌ [%d/%d] Failed to process: %s - %v\n", i+1, len(documents), doc.Path, err)
		}

		// Add delay between documents
		if cfg.ProcessingDelay > 0 {
			time.Sleep(cfg.ProcessingDelay)
		}
	}

	// Log errors if any
	if len(errors) > 0 {
		fmt.Printf("\n⚠️  Processing completed with %d errors:\n", len(errors))
		for _, e := range errors {
			fmt.Printf("   - %s: %s\n", e.DocumentPath, e.Error)
		}
	}
}

func processDocument(cfg *Config, client *http.Client, doc DocumentInfo) (bool, error) {
	// Step 1: Download document content from storage
	fmt.Printf("   📥 Downloading document content...\n")
	docContent, err := downloadDocumentContent(cfg, client, doc.Path)
	if err != nil {
		return false, fmt.Errorf("failed to download document: %w", err)
	}
	defer docContent.Close()

	// Step 2: Process document through the processing API
	fmt.Printf("   🤖 Classifying document...\n")
	_, err = processDocumentWithAPI(cfg, client, doc, docContent)
	if err != nil {
		return false, fmt.Errorf("failed to process document: %w", err)
	}

	// Step 3: Document is automatically indexed by the processing pipeline
	// No need for manual indexing since we set index_document=true

	return true, nil
}

// Duplicate checking functions removed - processing all documents without checks

// downloadDocumentContent downloads the document content from storage
func downloadDocumentContent(cfg *Config, client *http.Client, docPath string) (io.ReadCloser, error) {
	downloadURL := fmt.Sprintf("%s/api/v1/files/%s", cfg.APIBaseURL, docPath)
	
	resp, err := client.Get(downloadURL)
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	
	return resp.Body, nil
}

// processDocumentWithAPI processes the document using the processing API
func processDocumentWithAPI(cfg *Config, client *http.Client, doc DocumentInfo, content io.Reader) (*ProcessResult, error) {
	processURL := cfg.APIBaseURL + "/api/v1/categorise"
	
	// Create multipart form with the file content
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	
	// Add file content
	part, err := writer.CreateFormFile("file", doc.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	
	if _, err := io.Copy(part, content); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}
	
	// Add options as form fields
	if err := writer.WriteField("extract_text", "true"); err != nil {
		return nil, fmt.Errorf("failed to write extract_text field: %w", err)
	}
	if err := writer.WriteField("classify_doc", "true"); err != nil {
		return nil, fmt.Errorf("failed to write classify_doc field: %w", err)
	}
	if err := writer.WriteField("index_document", "true"); err != nil {
		return nil, fmt.Errorf("failed to write index_document field: %w", err)
	}
	if err := writer.WriteField("store_document", "false"); err != nil {
		return nil, fmt.Errorf("failed to write store_document field: %w", err)
	}
	
	writer.Close()
	
	req, err := http.NewRequest("POST", processURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	// Execute request with retries
	var resp *http.Response
	for attempt := 0; attempt <= cfg.RetryAttempts; attempt++ {
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			break
		}
		
		if resp != nil {
			resp.Body.Close()
		}
		
		if attempt < cfg.RetryAttempts {
			fmt.Printf("   ⏳ Retry %d/%d for processing after %v\n", attempt+1, cfg.RetryAttempts, cfg.RetryDelay)
			time.Sleep(cfg.RetryDelay)
		}
	}
	
	if err != nil {
		return nil, fmt.Errorf("request failed after retries: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("processing failed: HTTP %d - %s", resp.StatusCode, string(bodyBytes))
	}
	
	// Parse processing response
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			DocumentID           string `json:"document_id"`
			ExtractionResult     struct {
				Text      string `json:"text"`
				PageCount int    `json:"page_count"`
				Language  string `json:"language"`
			} `json:"extraction_result"`
			ClassificationResult struct {
				Category   string   `json:"category"`
				Confidence float64  `json:"confidence"`
				Tags       []string `json:"tags"`
			} `json:"classification_result"`
		} `json:"data"`
		Message string `json:"message"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	if !response.Success {
		return nil, fmt.Errorf("processing failed: %s", response.Message)
	}
	
	// Convert to ProcessResult
	result := &ProcessResult{
		DocumentID:    response.Data.DocumentID,
		ExtractedText: response.Data.ExtractionResult.Text,
		Classification: map[string]interface{}{
			"category":   response.Data.ClassificationResult.Category,
			"confidence": response.Data.ClassificationResult.Confidence,
			"tags":       response.Data.ClassificationResult.Tags,
		},
		Metadata: map[string]interface{}{
			"page_count": response.Data.ExtractionResult.PageCount,
			"language":   response.Data.ExtractionResult.Language,
			"file_name":  doc.Filename,
			"file_size":  doc.Size,
			"file_type":  doc.FileType,
		},
	}
	
	return result, nil
}

// indexDocumentToOpenSearch function removed - documents are now automatically indexed
// by the processing pipeline when index_document=true is set

// Helper functions

func getTotalDocumentCount(cfg *Config) (int, error) {
	client := &http.Client{Timeout: cfg.RequestTimeout}
	resp, err := client.Get(cfg.APIBaseURL + "/api/v1/storage/documents/count")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var response struct {
		Data struct {
			TotalCount int `json:"total_count"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, err
	}

	return response.Data.TotalCount, nil
}

func getDocuments(cfg *Config, cursor string, limit int) ([]DocumentInfo, error) {
	url := fmt.Sprintf("%s/api/v1/storage/documents?limit=%d", cfg.APIBaseURL, limit)
	if cursor != "" {
		url += "&cursor=" + cursor
	}

	client := &http.Client{Timeout: cfg.RequestTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var response DocumentListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Data.Documents, nil
}

func getDocumentsBatch(cfg *Config, cursor string, limit int) ([]DocumentInfo, string, bool, error) {
	url := fmt.Sprintf("%s/api/v1/storage/documents?limit=%d", cfg.APIBaseURL, limit)
	if cursor != "" {
		url += "&cursor=" + cursor
	}

	client := &http.Client{Timeout: cfg.RequestTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", false, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var response DocumentListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, "", false, err
	}

	return response.Data.Documents, response.Data.NextCursor, response.Data.HasMore, nil
}

func printFinalStats(stats *ClassificationStats) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📊 SINGLE-THREADED CLASSIFICATION COMPLETE")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("⏱️  Total Processing Time: %v\n", stats.Duration)
	fmt.Printf("📁 Total Documents Found: %d\n", stats.TotalDocuments)
	fmt.Printf("✅ Successfully Processed: %d\n", stats.SuccessfulDocs)
	fmt.Printf("❌ Failed Documents: %d\n", stats.FailedDocs)
	fmt.Printf("📋 Total Processed: %d\n", stats.ProcessedDocuments)
	if stats.Duration.Minutes() > 0 {
		fmt.Printf("⚡ Average Rate: %.2f documents/minute\n", stats.Rate)
	}
	if stats.TotalDocuments > 0 {
		successRate := float64(stats.SuccessfulDocs) / float64(stats.TotalDocuments) * 100
		fmt.Printf("📈 Success Rate: %.1f%%\n", successRate)
	}
	fmt.Println()
	fmt.Println("💡 IMPLEMENTATION NOTES:")
	fmt.Println("   - This is a sequential, single-threaded processor")
	fmt.Println("   - Documents are processed one at a time for easier debugging")
	fmt.Println("   - Uses /categorise endpoint with index_document=true for integrated processing")
	fmt.Println("   - ⚠️  DUPLICATE CHECKING DISABLED - processes ALL documents without existence checks")
	fmt.Println("   - Supports all enhanced metadata fields (dates, court info, parties, etc.)")
	fmt.Println("   - Use this for controlled processing and detailed error tracking")
	fmt.Println()
	fmt.Println("📈 PERFORMANCE METRICS:")
	if stats.Duration.Minutes() > 0 {
		fmt.Printf("   - Processing Rate: %.2f docs/min\n", stats.Rate)
		fmt.Printf("   - Average Time Per Document: %.2f seconds\n", stats.Duration.Seconds()/float64(stats.ProcessedDocuments))
	}
	if stats.ProcessedDocuments > 0 {
		fmt.Printf("   - Success Rate: %.1f%% (%d/%d)\n", 
			float64(stats.SuccessfulDocs)/float64(stats.ProcessedDocuments)*100, 
			stats.SuccessfulDocs, stats.ProcessedDocuments)
		fmt.Printf("   - Failure Rate: %.1f%% (%d/%d)\n", 
			float64(stats.FailedDocs)/float64(stats.ProcessedDocuments)*100, 
			stats.FailedDocs, stats.ProcessedDocuments)
	}
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper functions for search removed since duplicate checking is disabled