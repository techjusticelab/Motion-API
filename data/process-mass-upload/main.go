package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	apihelpers "motion-index-fiber/data/process-mass-upload/api-helpers"
)

var supportedExtensions = map[string]struct{}{
	".doc":  {},
	".docx": {},
	".pdf":  {},
	".ppt":  {},
	".pptx": {},
	".txt":  {},
}

type uploadJob struct {
	path    string
	display string
}

type uploadResult struct {
	job      uploadJob
	attempts int
	err      error
}

type fileCandidate struct {
	path string
	size int64
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "clean":
		fmt.Println("Running clean_unsupported_file_types.go functionality...")
		fmt.Println("Please run: go run clean_unsupported_file_types.go [directory]")

	case "dedupe":
		fmt.Println("Running delete_duplicate_files.go functionality...")
		fmt.Println("Please run: go run delete_duplicate_files.go [directory]")

	case "upload":
		if err := runUpload(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "help", "--help", "-h":
		printUsage()

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func runUpload(args []string) error {
	defaultEndpoint := buildDefaultEndpoint()
	defaultConcurrency := envInt("MASS_UPLOAD_CONCURRENCY", 5)
	if defaultConcurrency < 1 {
		defaultConcurrency = 5
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

	defaultBatchSize := envInt("MASS_UPLOAD_BATCH_SIZE", 50)
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
	batchSizeFlag := fs.Int("batch-size", defaultBatchSize, "Number of files to upload per batch (0 = all)" )
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
	fmt.Printf("Max concurrency: %d | Retries per file: %d | Timeout: %s\n", conc, retryCount, httpTimeout)
	if batchSize > 0 {
		batchDelayText := "0s"
		if batchDelay > 0 {
			batchDelayText = batchDelay.String()
		}
		fmt.Printf("Batch size: %d | Batch delay: %s\n", batchSize, batchDelayText)
	} else {
		fmt.Println("Batch size: unlimited (processing all files in a single run)")
	}
	fmt.Println("-------------------------------------------------------")

	if batchSize > 0 && conc > batchSize {
		conc = batchSize
	}
	if conc > totalFiles {
		conc = totalFiles
	}
	if conc < 1 {
		conc = 1
	}

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

	filesPerBatch := batchSize
	if filesPerBatch <= 0 || filesPerBatch >= totalFiles {
		filesPerBatch = totalFiles
	}

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

		if batchDelay > 0 && batchCount > 1 && end < totalFiles {
			fmt.Printf("Waiting %s before next batch...\n", batchDelay)
			time.Sleep(batchDelay)
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

func collectUploadCandidates(path string) ([]fileCandidate, string, []string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to inspect %s: %w", path, err)
	}

	if info.IsDir() {
		var files []fileCandidate
		var skipped []string
		err = filepath.WalkDir(path, func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if d.IsDir() {
				return nil
			}

			if !d.Type().IsRegular() {
				skipped = append(skipped, p)
				return nil
			}

			info, err := d.Info()
			if err != nil {
				skipped = append(skipped, p)
				return nil
			}

			if isSupportedExtension(p) {
				files = append(files, fileCandidate{path: p, size: info.Size()})
			} else {
				skipped = append(skipped, p)
			}
			return nil
		})
		if err != nil {
			return nil, "", nil, fmt.Errorf("failed to walk %s: %w", path, err)
		}
		return files, path, skipped, nil
	}

	if !isSupportedExtension(path) {
		return nil, "", nil, fmt.Errorf("file %s has unsupported extension %s", path, strings.ToLower(filepath.Ext(path)))
	}

	baseDir := filepath.Dir(path)
	if baseDir == "" {
		baseDir = "."
	}
	return []fileCandidate{{path: path, size: info.Size()}}, baseDir, nil, nil
}

func isSupportedExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := supportedExtensions[ext]
	return ok
}

func buildDefaultEndpoint() string {
	if endpoint := strings.TrimSpace(os.Getenv("MASS_UPLOAD_ENDPOINT")); endpoint != "" {
		return endpoint
	}

	base := strings.TrimSpace(os.Getenv("MASS_UPLOAD_API_BASE_URL"))
	if base == "" {
		base = strings.TrimSpace(os.Getenv("API_BASE_URL"))
	}
	if base == "" {
		base = "http://localhost:8003/api/v1"
	}

	base = strings.TrimRight(base, "/")
	if strings.HasSuffix(base, "/upload/s3/os/dry-classification") {
		return base
	}
	if strings.HasSuffix(base, "/upload/s3") {
		return base + "/os/dry-classification"
	}

	return base + "/upload/s3/os/dry-classification"
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	if v, err := strconv.Atoi(value); err == nil {
		return v
	}

	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	if d, err := time.ParseDuration(value); err == nil {
		return d
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}

	return fallback
}

func envByteSize(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	size, err := parseByteSize(value)
	if err != nil {
		return fallback
	}
	return size
}

func printUsage() {
	fmt.Println("Usage: go run main.go <command> [options]")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  clean    - Remove unsupported file types (run: go run clean_unsupported_file_types.go)")
	fmt.Println("  dedupe   - Remove duplicate files (run: go run delete_duplicate_files.go)")
	fmt.Println("  upload   - Upload files to the API (run: go run main.go upload --dir <path>)")
	fmt.Println()
	fmt.Println("For upload-specific options, run: go run main.go upload -h")
}

func printUploadUsage(fs *flag.FlagSet) {
	fmt.Println("Upload files to the Motion Index API")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run main.go upload [options] [path]")
	fmt.Println()
	fmt.Println("If no path is provided, the current directory is used.")
	fmt.Println()
	fmt.Println("Options:")
	fs.PrintDefaults()
	fmt.Println()
	fmt.Printf("Supported file types: %s\n", strings.Join(supportedExtensionsList(), ", "))
	fmt.Println()
	fmt.Println("Environment overrides:")
	fmt.Println("  MASS_UPLOAD_ENDPOINT          - Full upload endpoint")
	fmt.Println("  MASS_UPLOAD_API_BASE_URL      - Base API URL combined with /upload/s3 or /upload/s3/os/dry-classification")
	fmt.Println("  MASS_UPLOAD_CONCURRENCY       - Default concurrency")
	fmt.Println("  MASS_UPLOAD_RETRIES           - Default retry attempts")
	fmt.Println("  MASS_UPLOAD_RETRY_DELAY       - Default retry delay (duration or seconds)")
	fmt.Println("  MASS_UPLOAD_TIMEOUT           - Default request timeout (duration or seconds)")
	fmt.Println("  MASS_UPLOAD_MAX_SIZE          - Maximum file size (e.g. 50MB, 1048576)")
}

func supportedExtensionsList() []string {
	exts := make([]string, 0, len(supportedExtensions))
	for ext := range supportedExtensions {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	return exts
}

func parseByteSize(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("empty size value")
	}

	// Try plain integer first (bytes)
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return n, nil
	}

	units := []struct {
		suffix string
		factor int64
	}{
		{"TB", 1 << 40},
		{"T", 1 << 40},
		{"GB", 1 << 30},
		{"G", 1 << 30},
		{"MB", 1 << 20},
		{"M", 1 << 20},
		{"KB", 1 << 10},
		{"K", 1 << 10},
		{"B", 1},
	}

	upper := strings.ToUpper(value)
	for _, unit := range units {
		if strings.HasSuffix(upper, unit.suffix) {
			numPart := strings.TrimSpace(upper[:len(upper)-len(unit.suffix)])
			if numPart == "" {
				return 0, fmt.Errorf("invalid size %q", value)
			}
			num, err := strconv.ParseFloat(numPart, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid size %q: %w", value, err)
			}
			return int64(num * float64(unit.factor)), nil
		}
	}

	return 0, fmt.Errorf("invalid size format %q", value)
}

func formatByteSize(size int64) string {
	if size <= 0 {
		return "0B"
	}

	const (
		kb = 1 << 10
		mb = 1 << 20
		gb = 1 << 30
		tb = 1 << 40
	)

	switch {
	case size >= tb:
		return fmt.Sprintf("%.1fTB", float64(size)/tb)
	case size >= gb:
		return fmt.Sprintf("%.1fGB", float64(size)/gb)
	case size >= mb:
		return fmt.Sprintf("%.1fMB", float64(size)/mb)
	case size >= kb:
		return fmt.Sprintf("%.1fKB", float64(size)/kb)
	default:
		return fmt.Sprintf("%dB", size)
	}
}
