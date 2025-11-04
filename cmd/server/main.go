package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"motion-index-fiber/internal/config"
	"motion-index-fiber/internal/handlers"
	"motion-index-fiber/internal/handlers/opensearch"
	"motion-index-fiber/internal/middleware"
	"motion-index-fiber/pkg/cloud/digitalocean"
	pkgextractor "motion-index-fiber/pkg/processing/extractor"
)

func main() {
	// Load .env if present (environment-agnostic)
	if _, statErr := os.Stat(".env"); statErr == nil {
		_ = godotenv.Load()
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	bodyLimit := 0
	if cfg.Server.MaxRequestSize > 0 {
		bodyLimit = int(cfg.Server.MaxRequestSize)
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ServerHeader: "Motion-Index-Fiber",
		AppName:      "Motion Index API v1.0",
		ErrorHandler: middleware.ErrorHandler,
		BodyLimit:    bodyLimit,
	})

	// Global middleware with enhanced panic recovery
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			// Get memory stats for debugging
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			log.Printf("[PANIC-RECOVERY] 💥 Panic recovered: %v", e)
			log.Printf("[PANIC-RECOVERY] 📊 Memory at panic: Alloc=%dMB, Sys=%dMB, GC=%d",
				memStats.Alloc/(1024*1024), memStats.Sys/(1024*1024), memStats.NumGC)
			log.Printf("[PANIC-RECOVERY] 📍 Request: %s %s", c.Method(), c.Path())

			// Force garbage collection after panic
			runtime.GC()

			// Log stack trace for debugging
			if cfg.Environment != "production" {
				log.Printf("[PANIC-RECOVERY] Stack trace: %+v", e)
			}
		},
	}))
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} - ${latency}\n",
	}))

	// Memory pressure middleware - reject requests if memory usage is too high
	app.Use(func(c *fiber.Ctx) error {
		// Only apply to heavy processing endpoints
		path := c.Path()
		if path == "/api/v1/categorise" || path == "/api/v1/analyze-redactions" {
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			// If using more than 800MB, reject new requests
			const maxMemoryMB = 800
			currentMemoryMB := memStats.Alloc / (1024 * 1024)

			if currentMemoryMB > maxMemoryMB {
				log.Printf("[MEMORY-GUARD] 🚫 Rejecting request due to high memory usage: %dMB > %dMB",
					currentMemoryMB, maxMemoryMB)
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"success": false,
					"error": map[string]interface{}{
						"code":    "memory_pressure",
						"message": "Server is under memory pressure, please try again later",
						"details": map[string]interface{}{
							"current_memory_mb": currentMemoryMB,
							"max_memory_mb":     maxMemoryMB,
						},
					},
				})
			}
		}
		return c.Next()
	})

	// Temporarily disabled security middleware for embedding issues
	// TODO: Re-enable with proper configuration for production
	// app.Use(helmet.New())

	// Completely disable helmet for now to allow unrestricted access
	// app.Use(helmet.New(helmet.Config{...}))

	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Server.AllowedOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "*",
		AllowCredentials: true,
		ExposeHeaders:    "Content-Length,Content-Type,X-Total-Count",
	}))

	// Initialize handlers
	h, err := handlers.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize handlers: %v", err)
	}

	if cfg.DigitalOcean == nil {
		log.Fatalf("DigitalOcean configuration is required for OpenSearch handler")
	}

	doFactory := digitalocean.NewServiceFactory(cfg.DigitalOcean)
	opensearchService, err := doFactory.CreateSearchService()
	if err != nil {
		log.Fatalf("Failed to create OpenSearch service: %v", err)
	}

	opensearchHandler := opensearch.NewHandler(opensearchService)

	dryStorageService, err := doFactory.CreateStorageService()
	if err != nil {
		log.Fatalf("Failed to create storage service for dry classification: %v", err)
	}
	dryExtractor := pkgextractor.NewService()
	dryHandler := opensearch.NewDryClassificationHandler(cfg, dryStorageService, opensearchService, dryExtractor)

	// Health endpoints
	app.Get("/", h.Health.Root)
	app.Get("/health", h.Health.Health)

	// API routes
	api := app.Group("/api/v1")

	// Health check under API v1 for consistency
	api.Get("/health", h.Health.Health)

	// File upload endpoint
	api.Post("/upload/s3", h.Storage.UploadDocumentToS3)

	// Nested upload routes for dry classification
	uploadGroup := api.Group("/upload")
	uploadS3Group := uploadGroup.Group("/s3")
	uploadS3OsGroup := uploadS3Group.Group("/os")
	uploadS3OsGroup.Post("/dry-classification", dryHandler.Run)

	// Public routes
	api.Post("/categorise", h.Processing.UploadDocument)
	api.Post("/analyze-redactions", h.Processing.AnalyzeRedactions)
	api.Post("/redact-document", h.Processing.RedactDocument)
	api.Post("/search", h.Search.SearchDocuments)
	osRoutes := api.Group("/os")
	osRoutes.Post("/push", opensearchHandler.PushDocument)
	osRoutes.Post("/dry-classification", dryHandler.Run)
	api.Get("/legal-tags", h.Search.GetLegalTags)
	api.Get("/document-types", h.Search.GetDocumentTypes)
	api.Get("/document-stats", h.Search.GetDocumentStats)
	api.Get("/field-options", h.Search.GetFieldOptions)
	api.Get("/all-field-options", h.Search.GetFieldOptions) // Alias for comprehensive field options
	api.Get("/metadata-fields", h.Search.GetMetadataFields)
	api.Get("/metadata-fields/:field", h.Search.GetMetadataFieldValues)
	api.Post("/metadata-field-values", h.Search.PostMetadataFieldValues)
	api.Get("/documents/:id/redactions", h.Search.GetDocumentRedactions)
	api.Get("/documents/:id", h.Search.GetDocument)

	// File serving routes (separate from document metadata routes)
	api.Get("/files/search", h.Storage.FindDocumentsByName)

	// Add middleware for file serving to allow embedding
	api.Get("/files/*", func(c *fiber.Ctx) error {
		// Remove all restrictions for embedding - TEMPORARY for development
		// TODO: Add proper security controls for production

		// Allow framing from any origin
		c.Set("X-Frame-Options", "")
		c.Response().Header.Del("X-Frame-Options")

		// Remove all restrictive CORS policies
		c.Response().Header.Del("Cross-Origin-Embedder-Policy")
		c.Response().Header.Del("Cross-Origin-Resource-Policy")
		c.Response().Header.Del("Cross-Origin-Opener-Policy")

		// Continue to the actual file serving handler
		return h.Storage.ServeDocument(c)
	})

	// Storage routes for document management
	storage := api.Group("/storage")
	storage.Get("/documents", h.Storage.ListDocuments)
	storage.Get("/documents/count", h.Storage.GetDocumentsCount)

	// Indexing routes
	index := api.Group("/index")
	index.Post("/document", h.Indexing.IndexDocument)

	// TODO: Re-enable authentication for these routes in production
	// Currently disabled for early development - these should be protected
	api.Post("/update-metadata", h.Processing.UpdateMetadata)
	api.Delete("/documents/:id", h.Search.DeleteDocument)

	// COMMENTED OUT: Protected routes (require authentication)
	// TODO: Uncomment and configure JWT authentication before production deployment
	// protected := api.Group("", middleware.JWT(cfg.Auth.JWTSecret))
	// protected.Post("/update-metadata", h.Processing.UpdateMetadata)
	// protected.Delete("/documents/:id", h.Search.DeleteDocument)

	// Start server
	port := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Starting server on port %s", cfg.Server.Port)

	// Graceful shutdown
	go func() {
		if err := app.Listen(port); err != nil {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Gracefully shutdown the server
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
