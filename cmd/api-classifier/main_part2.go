package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func processDocumentListSequentiallyWithClient(cfg *Config, client *http.Client, documents []DocumentInfo, stats *ClassificationStats) {
	var errors []ProcessingError

	for i, doc := range documents {
		fmt.Printf("🔄 [%d/%d] Processing: %s\n", i+1, len(documents), doc.Path)

		// Log memory usage before processing
		logMemoryUsage("before processing", doc.Path)

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

		// Log memory usage after processing and force GC if needed
		logMemoryUsage("after processing", doc.Path)

		// Force garbage collection every 10 documents or if memory usage is high
		if (i+1)%10 == 0 || shouldForceGC() {
			fmt.Printf("🗑️ Forcing garbage collection after document %d\n", i+1)
			runtime.GC()
			runtime.GC() // Double GC to ensure cleanup
			logMemoryUsage("after GC", doc.Path)
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

func processDocument(cfg *Config, client *http.Client, doc DocumentInfo) (success bool, err error) {
	// Add panic recovery to prevent crashes
	defer func() {
		if r := recover(); r != nil {
			log.Printf("💥 Panic recovered during document processing: %v", r)
			success = false
			err = fmt.Errorf("document processing panic: %v", r)
		}
	}()

	// Check file size before download (skip files > 50MB)
	if doc.Size > 50*1024*1024 {
		return false, fmt.Errorf("file too large: %d MB (limit: 50MB)", doc.Size/(1024*1024))
	}

	// Create temporary file first
	tmpDir := os.TempDir()
	// Ensure a reasonably unique, filesystem-safe name
	base := filepath.Base(doc.Filename)
	if base == "." || base == "" {
		base = "document"
	}
	tmpFile, err := os.CreateTemp(tmpDir, fmt.Sprintf("api-classifier-*-%s", base))
	if err != nil {
		return false, fmt.Errorf("failed to create temp file: %w", err)
	}
	// Ensure temp file removed after processing
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	// Step 1: Stream download directly to temporary file with retry
	fmt.Printf("   📥 Streaming document to temp file (%d bytes)...\n", doc.Size)
	err = retryOperation("download", func() error {
		// Reset file position for retries
		tmpFile.Seek(0, 0)
		tmpFile.Truncate(0)
		return streamDocumentToFile(cfg, client, doc.Path, tmpFile)
	}, cfg.RetryAttempts)
	if err != nil {
		return false, fmt.Errorf("failed to download document after retries: %w", err)
	}

	// Step 2: Process document through the processing API with retry
	fmt.Printf("   🤖 Classifying document...\n")
	err = retryOperation("processing", func() error {
		_, processingErr := processDocumentWithAPIFromPath(cfg, client, doc, tmpFile.Name())
		return processingErr
	}, cfg.RetryAttempts)
	if err != nil {
		return false, fmt.Errorf("failed to process document after retries: %w", err)
	}

	// Step 3: Document is automatically indexed by the processing pipeline
	// No need for manual indexing since we set index_document=true

	return true, nil
}

func retryOperation(operationName string, operation func() error, maxAttempts int) error {
	var lastErr error
	baseDelay := 1 * time.Second

	for attempt := 0; attempt <= maxAttempts; attempt++ {
		err := operation()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if we should retry based on error type
		if !shouldRetryError(err) {
			fmt.Printf("   ❌ %s failed with non-retryable error: %v\n", operationName, err)
			return err
		}

		if attempt < maxAttempts {
			// Exponential backoff: 1s, 2s, 4s, 8s, etc.
			delay := time.Duration(1<<attempt) * baseDelay
			fmt.Printf("   ⏳ %s attempt %d/%d failed, retrying in %v: %v\n",
				operationName, attempt+1, maxAttempts+1, delay, err)
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("%s failed after %d attempts: %w", operationName, maxAttempts+1, lastErr)
}

func shouldRetryError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Network-related errors that should be retried
	retryableErrors := []string{
		"connection refused", "connection reset", "timeout", "temporary failure",
		"network is unreachable", "no such host", "i/o timeout",
		"context deadline exceeded", "too many open files",
		"http 502", "http 503", "http 504", // Server errors
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(errStr, retryable) {
			return true
		}
	}

	return false
}

func logMemoryUsage(phase, docPath string) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	allocMB := memStats.Alloc / (1024 * 1024)
	sysMB := memStats.Sys / (1024 * 1024)
	numGC := memStats.NumGC

	docName := filepath.Base(docPath)
	if len(docName) > 30 {
		docName = docName[:27] + "..."
	}

	fmt.Printf("   📊 Memory %s (%s): Alloc=%dMB, Sys=%dMB, GC=%d\n",
		phase, docName, allocMB, sysMB, numGC)
}

func shouldForceGC() bool {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Force GC if using more than 300MB
	allocMB := memStats.Alloc / (1024 * 1024)
	return allocMB > 300
}

func streamDocumentToFile(cfg *Config, client *http.Client, docPath string, tmpFile *os.File) error {
	downloadURL := fmt.Sprintf("%s/api/v1/files/%s", cfg.APIBaseURL, docPath)

	resp, err := client.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Use chunked copying to prevent memory spikes
	chunkSize := int64(32 * 1024) // 32KB chunks
	var totalBytes int64
	buffer := make([]byte, chunkSize)

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := tmpFile.Write(buffer[:n]); writeErr != nil {
				return fmt.Errorf("failed to write to temp file: %w", writeErr)
			}
			totalBytes += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
	}

	// Sync to ensure all data is written to disk
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	fmt.Printf("   ✅ Downloaded %d bytes to temp file\n", totalBytes)
	return nil
}

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
