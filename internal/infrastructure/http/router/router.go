package router

import (
	"runtime"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"motion-index-fiber/internal/config"
	"motion-index-fiber/internal/handlers"
	"motion-index-fiber/internal/middleware"
)

// Config holds router configuration
type Config struct {
	Cfg      *config.Config
	Handlers *handlers.Handlers
}

// Setup configures and returns a Fiber app with all routes and middleware
func Setup(cfg *Config) *fiber.App {
	// Create Fiber app with error handling
	app := fiber.New(fiber.Config{
		ServerHeader: "Motion-Index-Fiber",
		AppName:      "Motion Index API v1.0",
		ErrorHandler: middleware.ErrorHandler,
	})

	// Apply middleware
	setupMiddleware(app, cfg.Cfg)

	// Register routes
	registerRoutes(app, cfg.Handlers)

	return app
}

// setupMiddleware configures all middleware
func setupMiddleware(app *fiber.App, cfg *config.Config) {
	// Panic recovery with stack trace
	app.Use(recover.New(recover.Config{
		EnableStackTrace:  true,
		StackTraceHandler: panicHandler(cfg),
	}))

	// Request logging
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} - ${latency}\n",
	}))

	// Memory pressure guard for heavy endpoints
	app.Use(memoryPressureMiddleware())

	// CORS configuration
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Server.AllowedOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "*",
		AllowCredentials: true,
		ExposeHeaders:    "Content-Length,Content-Type,X-Total-Count",
	}))
}

// registerRoutes registers all application routes
func registerRoutes(app *fiber.App, h *handlers.Handlers) {
	// Health endpoints
	app.Get("/", h.Health.Root)
	app.Get("/health", h.Health.Health)

	// API v1 group
	api := app.Group("/api/v1")

	// Document processing routes
	registerProcessingRoutes(api, h)

	// Search routes
	registerSearchRoutes(api, h)

	// Storage routes
	registerStorageRoutes(api, h)

	// Batch processing routes
	registerBatchRoutes(api, h)

	// Indexing routes
	registerIndexingRoutes(api, h)
}

// registerProcessingRoutes registers document processing endpoints
func registerProcessingRoutes(api fiber.Router, h *handlers.Handlers) {
	// Legacy endpoints (to be migrated)
	api.Post("/categorise", h.Processing.UploadDocument)
	api.Post("/analyze-redactions", h.Processing.AnalyzeRedactions)
	api.Post("/redact-document", h.Processing.RedactDocument)
	api.Post("/update-metadata", h.Processing.UpdateMetadata)

	// TODO: Add new clean handlers when use cases are implemented
	// documents := api.Group("/documents")
	// documents.Post("/", documentHandler.ProcessDocument)
	// documents.Post("/batch", documentHandler.BatchProcessDocuments)
	// documents.Post("/:id/index", documentHandler.IndexDocument)
	// documents.Post("/:id/classify", classificationHandler.ClassifyDocument)
	// documents.Put("/:id/classify", classificationHandler.ReclassifyDocument)
	// documents.Put("/:id/metadata", documentHandler.UpdateMetadata)
	//
	// redactions := api.Group("/redactions")
	// redactions.Post("/analyze", redactionHandler.AnalyzeRedactions)
	// redactions.Post("/apply", redactionHandler.ApplyRedactions)
}

// registerSearchRoutes registers search and retrieval endpoints
func registerSearchRoutes(api fiber.Router, h *handlers.Handlers) {
	api.Post("/search", h.Search.SearchDocuments)
	api.Get("/legal-tags", h.Search.GetLegalTags)
	api.Get("/document-types", h.Search.GetDocumentTypes)
	api.Get("/document-stats", h.Search.GetDocumentStats)
	api.Get("/field-options", h.Search.GetFieldOptions)
	api.Get("/all-field-options", h.Search.GetFieldOptions)
	api.Get("/metadata-fields", h.Search.GetMetadataFields)
	api.Get("/metadata-fields/:field", h.Search.GetMetadataFieldValues)
	api.Post("/metadata-field-values", h.Search.PostMetadataFieldValues)
	api.Get("/documents/:id", h.Search.GetDocument)
	api.Get("/documents/:id/redactions", h.Search.GetDocumentRedactions)
	api.Delete("/documents/:id", h.Search.DeleteDocument)
}

// registerStorageRoutes registers file storage endpoints
func registerStorageRoutes(api fiber.Router, h *handlers.Handlers) {
	// File search and serving
	api.Get("/files/search", h.Storage.FindDocumentsByName)

	// File serving with embedding support middleware
	api.Get("/files/*", func(c *fiber.Ctx) error {
		// Allow embedding for development (TODO: Add security for production)
		c.Response().Header.Del("X-Frame-Options")
		c.Response().Header.Del("Cross-Origin-Embedder-Policy")
		c.Response().Header.Del("Cross-Origin-Resource-Policy")
		c.Response().Header.Del("Cross-Origin-Opener-Policy")
		return h.Storage.ServeDocument(c)
	})

	// Storage management endpoints
	storage := api.Group("/storage")
	storage.Get("/documents", h.Storage.ListDocuments)
	storage.Get("/documents/count", h.Storage.GetDocumentsCount)
}

// registerBatchRoutes registers batch processing endpoints
func registerBatchRoutes(api fiber.Router, h *handlers.Handlers) {
	batch := api.Group("/batch")
	batch.Post("/classify", h.Batch.StartBatchClassification)
	batch.Get("/:job_id/status", h.Batch.GetBatchJobStatus)
	batch.Get("/:job_id/results", h.Batch.GetBatchJobResults)
	batch.Delete("/:job_id", h.Batch.CancelBatchJob)
}

// registerIndexingRoutes registers document indexing endpoints
func registerIndexingRoutes(api fiber.Router, h *handlers.Handlers) {
	index := api.Group("/index")
	index.Post("/document", h.Indexing.IndexDocument)
}

// panicHandler creates a panic recovery handler with detailed logging
func panicHandler(cfg *config.Config) func(*fiber.Ctx, interface{}) {
	return func(c *fiber.Ctx, e interface{}) {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		// Log panic details
		log := func(format string, args ...interface{}) {
			// TODO: Use proper logger instead of print
			_ = format
			_ = args
		}

		log("[PANIC-RECOVERY] 💥 Panic recovered: %v", e)
		log("[PANIC-RECOVERY] 📊 Memory at panic: Alloc=%dMB, Sys=%dMB, GC=%d",
			memStats.Alloc/(1024*1024), memStats.Sys/(1024*1024), memStats.NumGC)
		log("[PANIC-RECOVERY] 📍 Request: %s %s", c.Method(), c.Path())

		// Force GC after panic
		runtime.GC()

		// Log stack trace in non-production
		if cfg.Environment != "production" {
			log("[PANIC-RECOVERY] Stack trace: %+v", e)
		}
	}
}

// memoryPressureMiddleware rejects requests when memory usage is too high
func memoryPressureMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Only check heavy processing endpoints
		path := c.Path()
		if path != "/api/v1/categorise" && path != "/api/v1/analyze-redactions" {
			return c.Next()
		}

		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		// Reject if memory usage exceeds 800MB
		const maxMemoryMB = 800
		currentMemoryMB := memStats.Alloc / (1024 * 1024)

		if currentMemoryMB > maxMemoryMB {
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

		return c.Next()
	}
}
