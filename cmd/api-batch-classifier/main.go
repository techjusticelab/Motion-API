package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"
)

// Configuration for the API-based batch classifier
type Config struct {
	APIBaseURL           string        `json:"api_base_url"`
	MaxConcurrentWorkers int           `json:"max_concurrent_workers"`
	BatchSize            int           `json:"batch_size"`
	RateLimitPerMinute   int           `json:"rate_limit_per_minute"`
	RequestTimeout       time.Duration `json:"request_timeout"`
	RetryAttempts        int           `json:"retry_attempts"`
	RetryDelay           time.Duration `json:"retry_delay"`
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

// BatchClassifyRequest represents a request to the batch classification API
type BatchClassifyRequest struct {
	Documents []BatchDocumentInput   `json:"documents"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// BatchDocumentInput represents a document to be classified
type BatchDocumentInput struct {
	DocumentID   string `json:"document_id"`
	DocumentPath string `json:"document_path,omitempty"`
	Text         string `json:"text,omitempty"`
}

// BatchJobResponse represents the response from starting a batch job
type BatchJobResponse struct {
	Success bool `json:"success"`
	Data    struct {
		JobID          string    `json:"job_id"`
		Status         string    `json:"status"`
		TotalDocuments int       `json:"total_documents"`
		CreatedAt      time.Time `json:"created_at"`
	} `json:"data"`
	Message string `json:"message"`
}

// BatchJobStatusResponse represents the response from checking job status
type BatchJobStatusResponse struct {
	Success bool `json:"success"`
	Data    struct {
		ID          string        `json:"id"`
		Type        string        `json:"type"`
		Status      string        `json:"status"`
		Progress    BatchProgress `json:"progress"`
		CreatedAt   time.Time     `json:"created_at"`
		UpdatedAt   time.Time     `json:"updated_at"`
		CompletedAt *time.Time    `json:"completed_at,omitempty"`
	} `json:"data"`
	Message string `json:"message"`
}

// BatchProgress tracks the progress of a batch job
type BatchProgress struct {
	TotalDocuments    int     `json:"total_documents"`
	ProcessedCount    int     `json:"processed_count"`
	SuccessCount      int     `json:"success_count"`
	ErrorCount        int     `json:"error_count"`
	SkippedCount      int     `json:"skipped_count"`
	IndexedCount      int     `json:"indexed_count"`
	IndexErrorCount   int     `json:"index_error_count"`
	PercentComplete   float64 `json:"percent_complete"`
	EstimatedDuration string  `json:"estimated_duration,omitempty"`
}

// ClassificationStats tracks overall classification statistics
type ClassificationStats struct {
	TotalDocuments     int64         `json:"total_documents"`
	ProcessedDocuments int64         `json:"processed_documents"`
	SuccessfulJobs     int64         `json:"successful_jobs"`
	FailedJobs         int64         `json:"failed_jobs"`
	StartTime          time.Time     `json:"start_time"`
	Duration           time.Duration `json:"duration"`
	Rate               float64       `json:"rate_per_minute"`
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
		classifyAllDocuments(cfg)
	case "classify-batch":
		batchSize := 100
		if len(os.Args) > 2 {
			if bs, err := fmt.Sscanf(os.Args[2], "%d", &batchSize); err != nil || bs != 1 {
				log.Printf("Invalid batch size, using default: %d", batchSize)
			}
		}
		classifyDocumentsBatch(cfg, batchSize)
	case "test-api":
		testAPIConnection(cfg)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("API-Based Batch Document Classifier")
	fmt.Println("==================================")
	fmt.Println()
	fmt.Println("Usage: go run cmd/api-batch-classifier/main.go <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  test-api              - Test API connection and authentication")
	fmt.Println("  classify-batch [size] - Classify specified number of documents")
	fmt.Println("  classify-all          - Classify ALL documents in storage")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run cmd/api-batch-classifier/main.go test-api")
	fmt.Println("  go run cmd/api-batch-classifier/main.go classify-batch 500")
	fmt.Println("  go run cmd/api-batch-classifier/main.go classify-all")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  API_BASE_URL          - Base URL for the Motion Index API (default: http://localhost:6000)")
	fmt.Println("  MAX_WORKERS           - Maximum concurrent workers (default: 5)")
	fmt.Println("  BATCH_SIZE            - Documents per batch (default: 50)")
	fmt.Println("  RATE_LIMIT            - API requests per minute (default: 100)")
}

func loadConfig() *Config {
	cfg := &Config{
		APIBaseURL:           getEnv("API_BASE_URL", "http://localhost:8003"),
		MaxConcurrentWorkers: getEnvInt("MAX_WORKERS", 5),
		BatchSize:            getEnvInt("BATCH_SIZE", 50),
		RateLimitPerMinute:   getEnvInt("RATE_LIMIT", 100),
		RequestTimeout:       time.Duration(getEnvInt("REQUEST_TIMEOUT_SECONDS", 120)) * time.Second,
		RetryAttempts:        getEnvInt("RETRY_ATTEMPTS", 3),
		RetryDelay:           time.Duration(getEnvInt("RETRY_DELAY_SECONDS", 5)) * time.Second,
	}

	fmt.Printf("🔧 Configuration loaded:\n")
	fmt.Printf("   API Base URL: %s\n", cfg.APIBaseURL)
	fmt.Printf("   Max Workers: %d\n", cfg.MaxConcurrentWorkers)
	fmt.Printf("   Batch Size: %d\n", cfg.BatchSize)
	fmt.Printf("   Rate Limit: %d req/min\n", cfg.RateLimitPerMinute)
	fmt.Printf("   Request Timeout: %s\n", cfg.RequestTimeout)
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

	// Test document count
	fmt.Println("🔢 Testing document count...")
	resp, err = client.Get(cfg.APIBaseURL + "/api/v1/storage/documents/count")
	if err != nil {
		log.Printf("⚠️  Document count test failed: %v", err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			fmt.Println("✅ Document count endpoint OK")
		} else {
			fmt.Printf("⚠️  Document count returned HTTP %d\n", resp.StatusCode)
		}
	}

	fmt.Println("✅ API connection test complete!")
}

func classifyAllDocuments(cfg *Config) {
	fmt.Println("🚀 API-Based Classification of All Documents")
	fmt.Println("============================================")

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

	if totalDocs == 0 {
		fmt.Println("⚠️  No documents found to classify")
		return
	}

	// Process all documents using pagination
	processAllDocuments(cfg, stats)

	// Final statistics
	stats.Duration = time.Since(startTime)
	stats.Rate = float64(stats.ProcessedDocuments) / stats.Duration.Minutes()

	printFinalStats(stats)
}

func classifyDocumentsBatch(cfg *Config, maxDocuments int) {
	fmt.Printf("🚀 API-Based Classification of %d Documents\n", maxDocuments)
	fmt.Println("==========================================")

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

	// Process documents in batches
	processDocumentList(cfg, documents, stats)

	// Final statistics
	stats.Duration = time.Since(startTime)
	stats.Rate = float64(stats.ProcessedDocuments) / stats.Duration.Minutes()

	printFinalStats(stats)
}

func processAllDocuments(cfg *Config, stats *ClassificationStats) {
	cursor := ""
	totalProcessed := 0

	for {
		// Get batch of documents
		documents, nextCursor, hasMore, err := getDocumentsBatch(cfg, cursor, cfg.BatchSize*2)
		if err != nil {
			log.Printf("❌ Failed to get document batch: %v", err)
			break
		}

		if len(documents) == 0 {
			break
		}

		fmt.Printf("📋 Processing batch of %d documents (cursor: %s)\n", len(documents), cursor[:min(8, len(cursor))])

		// Process this batch
		processDocumentList(cfg, documents, stats)

		totalProcessed += len(documents)
		fmt.Printf("📊 Progress: %d/%d documents processed (%.1f%%)\n",
			totalProcessed, stats.TotalDocuments, float64(totalProcessed)/float64(stats.TotalDocuments)*100)

		if !hasMore {
			break
		}
		cursor = nextCursor

		// Rate limiting between batches
		time.Sleep(time.Duration(60.0/float64(cfg.RateLimitPerMinute)*float64(cfg.BatchSize)) * time.Second)
	}
}

func processDocumentList(cfg *Config, documents []DocumentInfo, stats *ClassificationStats) {
	// Create worker pool with rate limiting
	semaphore := make(chan struct{}, cfg.MaxConcurrentWorkers)

	// Calculate rate limiter interval - ensure it's at least 1ms
	intervalSeconds := 60.0 / float64(cfg.RateLimitPerMinute)
	if intervalSeconds < 0.001 { // Minimum 1ms
		intervalSeconds = 0.001
	}
	interval := time.Duration(intervalSeconds * float64(time.Second))

	rateLimiter := time.NewTicker(interval)
	defer rateLimiter.Stop()

	var wg sync.WaitGroup
	jobChan := make(chan []DocumentInfo, 10)

	// Start workers
	for i := 0; i < cfg.MaxConcurrentWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			processWorker(cfg, workerID, jobChan, semaphore, rateLimiter, stats)
		}(i)
	}

	// Split documents into batches for processing
	for i := 0; i < len(documents); i += cfg.BatchSize {
		end := i + cfg.BatchSize
		if end > len(documents) {
			end = len(documents)
		}

		batch := documents[i:end]
		jobChan <- batch
	}

	close(jobChan)
	wg.Wait()
}

func processWorker(cfg *Config, workerID int, jobChan <-chan []DocumentInfo, semaphore chan struct{}, rateLimiter *time.Ticker, stats *ClassificationStats) {
	client := &http.Client{Timeout: cfg.RequestTimeout}

	for batch := range jobChan {
		// Acquire semaphore
		semaphore <- struct{}{}

		// Wait for rate limiter
		<-rateLimiter.C

		// Process batch
		jobID, err := submitBatchJob(cfg, client, batch)
		if err != nil {
			log.Printf("❌ Worker %d: Failed to submit batch: %v", workerID, err)
			atomic.AddInt64(&stats.FailedJobs, 1)
			<-semaphore
			continue
		}

		// Wait for completion and track progress
		success := waitForJobCompletion(cfg, client, jobID, workerID)
		if success {
			atomic.AddInt64(&stats.SuccessfulJobs, 1)
		} else {
			atomic.AddInt64(&stats.FailedJobs, 1)
		}

		atomic.AddInt64(&stats.ProcessedDocuments, int64(len(batch)))

		// Release semaphore
		<-semaphore
	}
}

func submitBatchJob(cfg *Config, client *http.Client, documents []DocumentInfo) (string, error) {
	// Prepare batch request
	batchDocs := make([]BatchDocumentInput, len(documents))
	for i, doc := range documents {
		batchDocs[i] = BatchDocumentInput{
			DocumentID:   doc.Path,
			DocumentPath: doc.Path,
		}
	}

	request := BatchClassifyRequest{
		Documents: batchDocs,
		Options: map[string]interface{}{
			"store_results": true,
			"update_index":  true,
		},
	}

	// Submit job
	requestBody, _ := json.Marshal(request)
	resp, err := client.Post(cfg.APIBaseURL+"/api/v1/batch/classify", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to submit batch job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("batch job submission failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	var jobResp BatchJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
		return "", fmt.Errorf("failed to decode job response: %w", err)
	}

	return jobResp.Data.JobID, nil
}

func waitForJobCompletion(cfg *Config, client *http.Client, jobID string, workerID int) bool {
	maxWaitTime := 30 * time.Minute
	startTime := time.Now()
	checkInterval := 10 * time.Second

	for time.Since(startTime) < maxWaitTime {
		// Check job status
		resp, err := client.Get(cfg.APIBaseURL + "/api/v1/batch/" + jobID + "/status")
		if err != nil {
			log.Printf("⚠️  Worker %d: Failed to check job status: %v", workerID, err)
			time.Sleep(checkInterval)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("⚠️  Worker %d: Job status check failed: HTTP %d", workerID, resp.StatusCode)
			time.Sleep(checkInterval)
			continue
		}

		var statusResp BatchJobStatusResponse
		if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
			log.Printf("⚠️  Worker %d: Failed to decode status response: %v", workerID, err)
			time.Sleep(checkInterval)
			continue
		}

		status := statusResp.Data.Status
		progress := statusResp.Data.Progress

		switch status {
		case "completed":
			fmt.Printf("✅ Worker %d: Job %s completed successfully | ✅ %d classified | 📦 %d queued | 🚫 %d classification errors | ❌ %d queue errors\n",
				workerID, jobID[:8], progress.SuccessCount, progress.IndexedCount, progress.ErrorCount, progress.IndexErrorCount)
			return true
		case "failed":
			fmt.Printf("❌ Worker %d: Job %s failed | ✅ %d classified | 📦 %d queued | 🚫 %d classification errors | ❌ %d queue errors\n",
				workerID, jobID[:8], progress.SuccessCount, progress.IndexedCount, progress.ErrorCount, progress.IndexErrorCount)
			return false
		case "cancelled":
			fmt.Printf("⚠️  Worker %d: Job %s was cancelled\n", workerID, jobID[:8])
			return false
		case "running", "queued":
			if progress.TotalDocuments > 0 {
				// Enhanced progress reporting with classification and queue details
				fmt.Printf("⏳ Worker %d: Job %s - %.1f%% complete (%d/%d) | ✅ %d classified | 🚫 %d errors | 📦 %d queued | ❌ %d queue errors\n",
					workerID, jobID[:8], progress.PercentComplete, progress.ProcessedCount, progress.TotalDocuments,
					progress.SuccessCount, progress.ErrorCount, progress.IndexedCount, progress.IndexErrorCount)
			}
		}

		time.Sleep(checkInterval)
	}

	fmt.Printf("⏰ Worker %d: Job %s timed out after %s\n", workerID, jobID[:8], maxWaitTime)
	return false
}

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
	fmt.Println("📊 API-BASED CLASSIFICATION COMPLETE")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("⏱️  Total Processing Time: %v\n", stats.Duration)
	fmt.Printf("📁 Total Documents Found: %d\n", stats.TotalDocuments)
	fmt.Printf("✅ Successful Batches: %d\n", stats.SuccessfulJobs)
	fmt.Printf("❌ Failed Batches: %d\n", stats.FailedJobs)
	fmt.Printf("📋 Documents Processed: %d\n", stats.ProcessedDocuments)
	if stats.Duration.Minutes() > 0 {
		fmt.Printf("⚡ Average Rate: %.2f documents/minute\n", stats.Rate)
	}
	if stats.TotalDocuments > 0 {
		successRate := float64(stats.ProcessedDocuments) / float64(stats.TotalDocuments) * 100
		fmt.Printf("📈 Processing Rate: %.1f%%\n", successRate)
	}
	fmt.Println()
	fmt.Println("💡 FOR DETAILED LOGS:")
	fmt.Println("   - Check server logs for [BATCH-CLASSIFY] entries to see OpenAI API quota hits")
	fmt.Println("   - Look for [BATCH-QUEUE] entries to see indexing queue operations")
	fmt.Println("   - Monitor [BATCH-PROGRESS] for detailed per-document status")
	fmt.Println("   - QUOTA_EXCEEDED and RATE_LIMIT errors indicate OpenAI API limits")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := fmt.Sscanf(value, "%d", &defaultValue); err == nil && intValue == 1 {
			return defaultValue
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
