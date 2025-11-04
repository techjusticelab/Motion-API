package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

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
	fmt.Println("  MASS_UPLOAD_BATCH_SIZE        - Default files per batch (0 = all)")
	fmt.Println("  MASS_UPLOAD_BATCH_DELAY       - Default delay between batches (duration or seconds)")
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
