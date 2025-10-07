package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "strings"
    "os/signal"
    "syscall"
    "time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"motion-index-fiber/internal/config"
	"motion-index-fiber/internal/handlers"
	"motion-index-fiber/internal/middleware"
)

func main() {
    // Load .env only for local development or when file exists
    if strings.EqualFold(os.Getenv("ENVIRONMENT"), "local") {
        _ = godotenv.Load()
    } else if _, statErr := os.Stat(".env"); statErr == nil {
        _ = godotenv.Load()
    }

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ServerHeader: "Motion-Index-Fiber",
		AppName:      "Motion Index API v1.0",
		ErrorHandler: middleware.ErrorHandler,
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} - ${latency}\n",
	}))
	
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

	// Start queue processing
	queueCtx, queueCancel := context.WithCancel(context.Background())
	defer queueCancel()
	
	if err := h.StartQueueProcessing(queueCtx); err != nil {
		log.Fatalf("Failed to start queue processing: %v", err)
	}
	log.Println("Queue processing started successfully")

	// Ensure queues are stopped on shutdown
	defer func() {
		log.Println("Stopping queue processing...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		
		if err := h.StopQueueProcessing(shutdownCtx); err != nil {
			log.Printf("Error stopping queue processing: %v", err)
		} else {
			log.Println("Queue processing stopped successfully")
		}
	}()

	// Health endpoints
	app.Get("/", h.Health.Root)
	app.Get("/health", h.Health.Health)

	// API routes
	api := app.Group("/api/v1")

	// Public routes
	api.Post("/categorise", h.Processing.UploadDocument)
	api.Post("/analyze-redactions", h.Processing.AnalyzeRedactions)
	api.Post("/redact-document", h.Processing.RedactDocument)
	api.Post("/search", h.Search.SearchDocuments)
	api.Get("/legal-tags", h.Search.GetLegalTags)
	api.Get("/document-types", h.Search.GetDocumentTypes)
	api.Get("/document-stats", h.Search.GetDocumentStats)
	api.Get("/field-options", h.Search.GetFieldOptions)
	api.Get("/all-field-options", h.Search.GetFieldOptions)  // Alias for comprehensive field options
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

	// Batch processing routes
	batch := api.Group("/batch")
	batch.Post("/classify", h.Batch.StartBatchClassification)
	batch.Get("/:job_id/status", h.Batch.GetBatchJobStatus)
	batch.Get("/:job_id/results", h.Batch.GetBatchJobResults)
	batch.Delete("/:job_id", h.Batch.CancelBatchJob)

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
