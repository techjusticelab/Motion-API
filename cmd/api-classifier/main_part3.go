package main

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptrace"
	"os"
	"strconv"
	"strings"
	"time"
)

func processDocumentWithAPIFromPath(cfg *Config, client *http.Client, doc DocumentInfo, filePath string) (*ProcessResult, error) {
	processURL := cfg.APIBaseURL + "/api/v1/categorise"

	// Helper to build a new streaming multipart request each attempt
	buildRequest := func() (*http.Request, *multipart.Writer, io.Closer, error) {
		pr, pw := io.Pipe()
		mw := multipart.NewWriter(pw)

		// Write multipart in background
		go func() {
			// Any error from this goroutine should be propagated by failing the pipe
			// Open the file for this attempt
			f, err := os.Open(filePath)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("open file error: %w", err))
				return
			}
			defer f.Close()

			// Create the file part and copy
			part, err := mw.CreateFormFile("file", doc.Filename)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("form file error: %w", err))
				return
			}
			if _, err := io.Copy(part, f); err != nil {
				pw.CloseWithError(fmt.Errorf("copy file error: %w", err))
				return
			}

			// Additional fields
			if err := mw.WriteField("extract_text", "true"); err != nil {
				pw.CloseWithError(err)
				return
			}
			if err := mw.WriteField("classify_doc", "true"); err != nil {
				pw.CloseWithError(err)
				return
			}
			if err := mw.WriteField("index_document", "true"); err != nil {
				pw.CloseWithError(err)
				return
			}
			if err := mw.WriteField("store_document", "false"); err != nil {
				pw.CloseWithError(err)
				return
			}

			// Close multipart then the pipe writer
			if err := mw.Close(); err != nil {
				pw.CloseWithError(err)
				return
			}
			_ = pw.Close()
		}()

		req, err := http.NewRequest("POST", processURL, pr)
		if err != nil {
			// Close writer to unblock goroutine if any
			_ = pw.Close()
			return nil, nil, nil, err
		}
		req.Header.Set("Content-Type", mw.FormDataContentType())
		return req, mw, pr, nil
	}

	var resp *http.Response
	var err error
	for attempt := 0; attempt <= cfg.RetryAttempts; attempt++ {
		req, _, bodyReader, buildErr := buildRequest()
		if buildErr != nil {
			err = buildErr
			break
		}

		// Add lightweight trace to detect connection failures early
		trace := &httptrace.ClientTrace{
			GotConn: func(info httptrace.GotConnInfo) {
				_ = info
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

		resp, err = client.Do(req)
		// Ensure body reader is closed if the request failed to avoid pipe leaks
		if err != nil {
			if bodyReader != nil {
				_ = bodyReader.Close()
			}
		}

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
			DocumentID       string `json:"document_id"`
			ExtractionResult struct {
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

func getTotalDocumentCount(cfg *Config) (int, error) {
	client := newHTTPClient(cfg)
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

	client := newHTTPClient(cfg)
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

	client := newHTTPClient(cfg)
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

func newHTTPClient(cfg *Config) *http.Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          200,               // Increased total connections
		MaxIdleConnsPerHost:   50,                // Increased per-host connections
		IdleConnTimeout:       120 * time.Second, // Longer idle timeout
		ExpectContinueTimeout: 2 * time.Second,   // Slightly longer for large files
		DisableKeepAlives:     false,             // Keep connections alive
		ForceAttemptHTTP2:     false,             // Stick to HTTP/1.1 for stability
		ResponseHeaderTimeout: 30 * time.Second,  // Timeout for headers
		TLSHandshakeTimeout:   10 * time.Second,  // TLS timeout
	}
	return &http.Client{
		Timeout:   cfg.RequestTimeout,
		Transport: transport,
	}
}
