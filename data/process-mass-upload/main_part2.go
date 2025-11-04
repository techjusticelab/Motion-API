package main

import (
	"errors"
	"flag"
	"fmt"
	apihelpers "motion-index-fiber/data/process-mass-upload/api-helpers"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

func runUpload(args []string) error {
	defaultEndpoint := buildDefaultEndpoint()
	defaultConcurrency := envInt("MASS_UPLOAD_CONCURRENCY", 10)
	if defaultConcurrency < 1 {
		defaultConcurrency = 2
	}

	defaultRetries := envInt("MASS_UPLOAD_RETRIES", 3)
	if defaultRetries < 0 {
		defaultRetries = 3
	}

	defaultRetryDelay := envDuration("MASS_UPLOAD_RETRY_DELAY", 2*time.Second)
	if defaultRetryDelay <= 0 {
		defaultRetryDelay = 2 * time.Second
	}

	defaultTimeout := envDuration("MASS_UPLOAD_TIMEOUT", 5*time.Minute)
	if defaultTimeout <= 0 {
		defaultTimeout = 5 * time.Minute
	}

	defaultMaxSize := envByteSize("MASS_UPLOAD_MAX_SIZE", 100*1024*1024)
	if defaultMaxSize < 0 {
		defaultMaxSize = 50 * 1024 * 1024
	}

	defaultBatchSize := envInt("MASS_UPLOAD_BATCH_SIZE", 10)
	if defaultBatchSize < 0 {
		defaultBatchSize = 0
	}

	defaultBatchDelay := envDuration("MASS_UPLOAD_BATCH_DELAY", 0)
	if defaultBatchDelay < 0 {
		defaultBatchDelay = 0
	}

	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	endpoint := fs.String("endpoint", defaultEndpoint, "Full upload endpoint (e.g. http://localhost:8003/api/v1/upload/s3)")
	dir := fs.String("dir", "", "Directory or file to upload (defaults to current directory or positional argument)")
	concurrency := fs.Int("concurrency", defaultConcurrency, "Number of concurrent uploads")
	retries := fs.Int("retries", defaultRetries, "Number of retry attempts per file")
	retryDelay := fs.Duration("retry-delay", defaultRetryDelay, "Delay between retries (e.g. 2s, 1m)")
	timeout := fs.Duration("timeout", defaultTimeout, "HTTP timeout per file (e.g. 5m)")
	maxSizeFlag := fs.String("max-size", formatByteSize(defaultMaxSize), "Maximum file size to upload (e.g. 50MB, 100M, 52428800)")
	batchSizeFlag := fs.Int("batch-size", defaultBatchSize, "Number of files to upload per batch (0 = all)")
	batchDelayFlag := fs.Duration("batch-delay", defaultBatchDelay, "Optional delay between batches (e.g. 15s)")

	fs.Usage = func() {
		printUploadUsage(fs)
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("failed to parse upload options: %w", err)
	}

	targetPath := strings.TrimSpace(*dir)
	if targetPath == "" {
		if fs.NArg() > 0 {
			targetPath = fs.Arg(0)
		} else {
			targetPath = "."
		}
	}

	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path %s: %w", targetPath, err)
	}

	candidates, baseDir, skippedUnsupported, err := collectUploadCandidates(absTarget)
	if err != nil {
		return err
	}

	maxSizeBytes, err := parseByteSize(*maxSizeFlag)
	if err != nil {
		return fmt.Errorf("invalid max-size value %q: %w", *maxSizeFlag, err)
	}
	if maxSizeBytes <= 0 {
		maxSizeBytes = defaultMaxSize
	}

	jobs := make([]uploadJob, 0, len(candidates))
	skippedOversize := make([]fileCandidate, 0)
	for _, candidate := range candidates {
		if maxSizeBytes > 0 && candidate.size > maxSizeBytes {
			skippedOversize = append(skippedOversize, candidate)
			continue
		}

		display := filepath.Base(candidate.path)
		if rel, relErr := filepath.Rel(baseDir, candidate.path); relErr == nil {
			display = rel
		}
		jobs = append(jobs, uploadJob{path: candidate.path, display: display})
	}

	if len(jobs) == 0 {
		if len(skippedUnsupported) > 0 || len(skippedOversize) > 0 {
			return fmt.Errorf("no supported files found in %s (%d unsupported, %d oversized)", targetPath, len(skippedUnsupported), len(skippedOversize))
		}
		return fmt.Errorf("no supported files found in %s", targetPath)
	}

	conc := *concurrency
	if conc < 1 {
		conc = 1
	}

	retryCount := *retries
	if retryCount < 0 {
		retryCount = 0
	}

	delay := *retryDelay
	if delay <= 0 {
		delay = defaultRetryDelay
	}

	httpTimeout := *timeout
	if httpTimeout <= 0 {
		httpTimeout = defaultTimeout
	}

	batchSize := *batchSizeFlag
	if batchSize < 0 {
		batchSize = 0
	}

	batchDelay := *batchDelayFlag
	if batchDelay < 0 {
		batchDelay = 0
	}

	endpointURL := strings.TrimSpace(*endpoint)
	if endpointURL == "" {
		endpointURL = defaultEndpoint
	}

	// If endpoint looks like it's targeting the dry-classification route, update it
	if strings.Contains(endpointURL, "/os/dry-classification") && !strings.Contains(endpointURL, "/upload/s3/os/dry-classification") {
		endpointURL = strings.Replace(endpointURL, "/os/dry-classification", "/upload/s3/os/dry-classification", 1)
	}

	client := &http.Client{Timeout: httpTimeout}
	totalFiles := len(jobs)

	filesPerBatch := batchSize
	if filesPerBatch <= 0 || filesPerBatch > totalFiles {
		filesPerBatch = totalFiles
	}

	fmt.Printf("Starting upload of files to API from: %s\n", absTarget)
	fmt.Printf("API Endpoint: %s\n", endpointURL)
	fmt.Println("-------------------------------------------------------")
	fmt.Printf("Found %d supported files to upload", totalFiles)
	if len(skippedUnsupported) > 0 {
		fmt.Printf(" (%d skipped as unsupported)", len(skippedUnsupported))
	}
	if len(skippedOversize) > 0 {
		fmt.Printf(" (%d skipped for size > %s)", len(skippedOversize), formatByteSize(maxSizeBytes))
	}
	fmt.Println()
	if filesPerBatch > 0 && conc > filesPerBatch {
		conc = filesPerBatch
	}
	if conc > totalFiles {
		conc = totalFiles
	}
	if conc < 1 {
		conc = 1
	}

	fmt.Printf("Max concurrency: %d | Retries per file: %d | Timeout: %s\n", conc, retryCount, httpTimeout)
	if filesPerBatch < totalFiles {
		batchDelayText := "0s"
		if batchDelay > 0 {
			batchDelayText = batchDelay.String()
		}
		fmt.Printf("Batch size: %d | Batch delay: %s\n", filesPerBatch, batchDelayText)
	} else {
		fmt.Println("Batch size: unlimited (processing all files in a single run)")
	}
	fmt.Println("-------------------------------------------------------")

	jobsCh := make(chan uploadJob, conc)
	resultsCh := make(chan uploadResult, conc)

	var wg sync.WaitGroup
	maxAttempts := retryCount + 1
	startTime := time.Now()

	worker := func() {
		defer wg.Done()
		for job := range jobsCh {
			var attemptErr error
			attemptsUsed := 0

			for attempt := 1; attempt <= maxAttempts; attempt++ {
				attemptsUsed = attempt
				attemptErr = apihelpers.UploadFile(client, endpointURL, job.path, filepath.Base(job.path))
				if attemptErr == nil {
					resultsCh <- uploadResult{job: job, attempts: attemptsUsed}
					break
				}

				if attempt < maxAttempts {
					fmt.Printf("↻ Retry %d/%d for %s: %v\n", attempt, retryCount, job.display, attemptErr)
					time.Sleep(delay)
				}
			}

			if attemptErr != nil {
				resultsCh <- uploadResult{job: job, attempts: attemptsUsed, err: attemptErr}
			}
		}
	}

	for i := 0; i < conc; i++ {
		wg.Add(1)
		go worker()
	}

	processed := 0
	successful := 0
	failed := 0
	totalRetriesUsed := 0
	var failedDetails []string

	batchCount := (totalFiles + filesPerBatch - 1) / filesPerBatch

	for batchIndex, start := 0, 0; start < totalFiles; batchIndex++ {
		end := start + filesPerBatch
		if end > totalFiles {
			end = totalFiles
		}

		if batchCount > 1 {
			fmt.Printf(">>> Batch %d/%d: uploading %d files\n", batchIndex+1, batchCount, end-start)
		}

		for _, job := range jobs[start:end] {
			jobsCh <- job
		}

		for i := start; i < end; i++ {
			result := <-resultsCh
			processed++
			retriesUsed := 0
			if result.attempts > 0 {
				retriesUsed = result.attempts - 1
			}
			totalRetriesUsed += retriesUsed

			if result.err == nil {
				if retriesUsed > 0 {
					fmt.Printf("[%d/%d] ✓ Uploaded: %s (retries: %d)\n", processed, totalFiles, result.job.display, retriesUsed)
				} else {
					fmt.Printf("[%d/%d] ✓ Uploaded: %s\n", processed, totalFiles, result.job.display)
				}
				successful++
			} else {
				fmt.Printf("[%d/%d] ✗ Failed: %s - %v\n", processed, totalFiles, result.job.display, result.err)
				failed++
				failedDetails = append(failedDetails, fmt.Sprintf("%s: %v", result.job.display, result.err))
			}
		}

		// Force garbage collection after each batch to release accumulated memory
		if batchCount > 1 {
			fmt.Printf("Running garbage collection after batch %d...\n", batchIndex+1)
			runtime.GC()
		}

		if batchDelay > 0 && batchCount > 1 && end < totalFiles {
			fmt.Printf("Waiting %s before next batch...\n", batchDelay)
			time.Sleep(batchDelay)
		} else if batchCount > 1 && end < totalFiles {
			// Even without explicit batch delay, add a small pause for GC to complete
			fmt.Printf("Brief pause for memory cleanup...\n")
			time.Sleep(2 * time.Second)
		}

		start = end
	}

	close(jobsCh)
	wg.Wait()
	close(resultsCh)

	duration := time.Since(startTime)
	fmt.Println("-------------------------------------------------------")
	fmt.Println("Upload Summary:")
	fmt.Printf("  Total files: %d\n", totalFiles)
	fmt.Printf("  Successful: %d\n", successful)
	fmt.Printf("  Failed: %d\n", failed)
	fmt.Printf("  Skipped (unsupported): %d\n", len(skippedUnsupported))
	fmt.Printf("  Skipped (oversized > %s): %d\n", formatByteSize(maxSizeBytes), len(skippedOversize))
	fmt.Printf("  Total retries: %d\n", totalRetriesUsed)
	successRate := float64(successful) / float64(totalFiles) * 100
	fmt.Printf("  Success rate: %.1f%%\n", successRate)
	fmt.Printf("  Duration: %s\n", duration.Round(time.Second))

	if failed > 0 {
		fmt.Println()
		fmt.Println("Failed uploads:")
		for _, detail := range failedDetails {
			fmt.Printf("  ✗ %s\n", detail)
		}
	}

	if len(skippedUnsupported) > 0 {
		fmt.Println()
		fmt.Println("Skipped files (unsupported extensions):")
		limit := len(skippedUnsupported)
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			display := filepath.Base(skippedUnsupported[i])
			if rel, relErr := filepath.Rel(baseDir, skippedUnsupported[i]); relErr == nil {
				display = rel
			}
			fmt.Printf("  - %s\n", display)
		}
		if len(skippedUnsupported) > limit {
			fmt.Printf("  ... and %d more\n", len(skippedUnsupported)-limit)
		}
	}

	if len(skippedOversize) > 0 {
		fmt.Println()
		fmt.Println("Skipped files (exceeded max size):")
		limit := len(skippedOversize)
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			display := filepath.Base(skippedOversize[i].path)
			if rel, relErr := filepath.Rel(baseDir, skippedOversize[i].path); relErr == nil {
				display = rel
			}
			fmt.Printf("  - %s (%s)\n", display, formatByteSize(skippedOversize[i].size))
		}
		if len(skippedOversize) > limit {
			fmt.Printf("  ... and %d more\n", len(skippedOversize)-limit)
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d file(s) failed to upload", failed)
	}

	return nil
}
