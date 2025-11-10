package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"motion-index-fiber/pkg/processing/extractor"
)

type extractorRunner interface {
	Name() string
	Run(ctx context.Context, data []byte, metadata *extractor.DocumentMetadata) (*extractor.ExtractionResult, error)
}

type currentExtractorRunner struct{}

func (r *currentExtractorRunner) Name() string {
	return "current-ledongthuc"
}

func (r *currentExtractorRunner) Run(ctx context.Context, data []byte, metadata *extractor.DocumentMetadata) (*extractor.ExtractionResult, error) {
	impl := extractor.NewPDFExtractor()
	return impl.Extract(ctx, bytes.NewReader(data), metadata)
}

func main() {
	var (
		filePath = flag.String("file", "", "Path to the PDF file to evaluate")
		mode     = flag.String("mode", "current", "Extraction mode (current, enhanced, placeholder-pdfcpu, placeholder-unidoc)")
		metaJSON = flag.Bool("metadata", false, "Print metadata JSON")
		preview  = flag.Int("preview", 400, "Number of characters to preview from extracted text")
		ocrPages = flag.Int("ocr-pages", 0, "Maximum number of pages to process during OCR (0 = no limit)")
	)

	flag.Parse()

	if *filePath == "" {
		flag.Usage()
		os.Exit(2)
	}

	data, err := os.ReadFile(*filePath)
	if err != nil {
		log.Fatalf("failed to read %s: %v", *filePath, err)
	}

	metadata := &extractor.DocumentMetadata{
		FileName: filepath.Base(*filePath),
		Size:     int64(len(data)),
		Format:   "pdf",
	}

	ctx := context.Background()

	var runner extractorRunner
	switch *mode {
	case "current", "current-ledongthuc":
		runner = &currentExtractorRunner{}
	case "enhanced":
		cfg := extractor.DefaultEnhancedConfig()
		setOCRMaxPages(cfg, *ocrPages)
		runner = newEnhancedRunnerWithConfig(cfg)
	case "placeholder-pdfcpu", "pdfcpu":
		runner = &placeholderRunner{name: "pdfcpu"}
	case "placeholder-unidoc", "unidoc":
		runner = &placeholderRunner{name: "unidoc"}
	default:
		log.Fatalf("unsupported mode %q", *mode)
	}

	start := time.Now()
	result, err := runner.Run(ctx, data, metadata)
	duration := time.Since(start)

	if err != nil {
		log.Printf("[%s] extraction error: %v", runner.Name(), err)
	}

	if result == nil {
		log.Printf("[%s] no result returned", runner.Name())
		os.Exit(1)
	}

	printSummary(runner.Name(), result, duration, *preview, *metaJSON)
}

type placeholderRunner struct {
	name string
}

func (r *placeholderRunner) Name() string {
	return r.name
}

func (r *placeholderRunner) Run(ctx context.Context, data []byte, metadata *extractor.DocumentMetadata) (*extractor.ExtractionResult, error) {
	return &extractor.ExtractionResult{
		Text:    "",
		Success: false,
		Error:   fmt.Sprintf("%s extractor not yet integrated. See PDF_PARSER_EVALUATION.md", r.name),
		Metadata: map[string]interface{}{
			"mode":   r.name,
			"status": "not_implemented",
		},
	}, fmt.Errorf("%s extractor placeholder", r.name)
}

type enhancedRunner struct {
	service extractor.Service
}

func newEnhancedRunner() extractorRunner {
	return newEnhancedRunnerWithConfig(nil)
}

func newEnhancedRunnerWithConfig(config *extractor.EnhancedConfig) extractorRunner {
	if config == nil {
		config = extractor.DefaultEnhancedConfig()
	}
	service := extractor.NewEnhancedService(config)
	return &enhancedRunner{service: service}
}

func (r *enhancedRunner) Name() string {
	return "enhanced"
}

func (r *enhancedRunner) Run(ctx context.Context, data []byte, metadata *extractor.DocumentMetadata) (*extractor.ExtractionResult, error) {
	if metadata == nil {
		metadata = &extractor.DocumentMetadata{}
	}
	if metadata.Properties == nil {
		metadata.Properties = make(map[string]string)
	}
	metadata.Format = "pdf"
	metadata.Size = int64(len(data))

	return r.service.ExtractText(ctx, bytes.NewReader(data), metadata)
}

func printSummary(mode string, result *extractor.ExtractionResult, duration time.Duration, previewLen int, includeMetadata bool) {
	log.Printf("Mode: %s", mode)
	log.Printf("Success: %v", result.Success)
	log.Printf("Error: %s", result.Error)
	log.Printf("Duration: %v", duration)
	log.Printf("Characters: %d | Words: %d | Pages: %d", result.CharCount, result.WordCount, result.PageCount)

	validator := extractor.NewPDFValidator()
	log.Printf("Validator marks garbage: %v", validator.IsGarbageText(result.Text))

	text := strings.TrimSpace(result.Text)
	if previewLen > 0 && len(text) > previewLen {
		text = text[:previewLen] + "…"
	}
	log.Printf("Preview:\n%s", text)

	if includeMetadata && len(result.Metadata) > 0 {
		encoded, err := json.MarshalIndent(result.Metadata, "", "  ")
		if err != nil {
			log.Printf("Metadata (failed to marshal): %v", err)
			return
		}
		log.Printf("Metadata:\n%s", string(encoded))
	}
}
