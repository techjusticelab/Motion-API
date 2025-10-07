package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/config"
	"motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/storage"
)

type StorageHandler struct {
	cfg     *config.Config
	storage storage.Service
}

func NewStorageHandler(cfg *config.Config, storage storage.Service) *StorageHandler {
	return &StorageHandler{
		cfg:     cfg,
		storage: storage,
	}
}

// ListDocuments handles GET /api/storage/documents - List documents with pagination
func (h *StorageHandler) ListDocuments(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
	defer cancel()

	// Parse query parameters
	prefix := c.Query("prefix", "documents/")
	limitStr := c.Query("limit", "50")
	cursor := c.Query("cursor", "")
	fileType := c.Query("file_type", "")
	minSizeStr := c.Query("min_size", "0")
	maxSizeStr := c.Query("max_size", "")

	// Parse and validate limit
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 500 {
		limit = 50
	}

	// Parse size filters
	minSize, _ := strconv.ParseInt(minSizeStr, 10, 64)
	var maxSize int64 = -1
	if maxSizeStr != "" {
		maxSize, _ = strconv.ParseInt(maxSizeStr, 10, 64)
	}

	// List documents from storage
	objects, err := h.storage.List(ctx, prefix)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"storage_error",
			"Failed to list documents",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Apply filters and pagination
	filtered := h.filterDocuments(objects, fileType, minSize, maxSize)
	paginatedResult := h.paginateDocuments(filtered, cursor, limit)

	return c.JSON(models.NewSuccessResponse(paginatedResult, "Documents listed successfully"))
}

// GetDocumentsCount handles GET /api/storage/documents/count - Get total document count
func (h *StorageHandler) GetDocumentsCount(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
	defer cancel()

	// Parse query parameters for filtering
	prefix := c.Query("prefix", "documents/")
	fileType := c.Query("file_type", "")
	minSizeStr := c.Query("min_size", "0")
	maxSizeStr := c.Query("max_size", "")

	// Parse size filters
	minSize, _ := strconv.ParseInt(minSizeStr, 10, 64)
	var maxSize int64 = -1
	if maxSizeStr != "" {
		maxSize, _ = strconv.ParseInt(maxSizeStr, 10, 64)
	}

	// List and filter documents
	objects, err := h.storage.List(ctx, prefix)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"storage_error",
			"Failed to count documents",
			map[string]interface{}{"error": err.Error()},
		))
	}

	filtered := h.filterDocuments(objects, fileType, minSize, maxSize)

	response := map[string]interface{}{
		"total_count":    len(filtered),
		"prefix":         prefix,
		"applied_filters": map[string]interface{}{
			"file_type": fileType,
			"min_size":  minSize,
			"max_size":  maxSize,
		},
	}

	return c.JSON(models.NewSuccessResponse(response, "Document count retrieved successfully"))
}

func (h *StorageHandler) ServeDocument(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	// Get document path from URL parameters
	rawDocumentPath := c.Params("*")
	if rawDocumentPath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Document path is required",
		})
	}

	// Decode URL-encoded path
	documentPath, err := url.QueryUnescape(rawDocumentPath)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid URL encoding in document path",
			"details": fmt.Sprintf("Failed to decode path '%s': %v", rawDocumentPath, err),
			"suggestion": "Ensure the path is properly URL-encoded. Use %2F for forward slashes.",
		})
	}

	// Validate and sanitize the path
	if err := h.validateDocumentPath(documentPath); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid document path",
			"details": err.Error(),
			"path": documentPath,
		})
	}

	// Clean the path - ensure it starts with documents/
	if !strings.HasPrefix(documentPath, "documents/") {
		documentPath = "documents/" + documentPath
	}

	// Check if document exists
	exists, err := h.storage.Exists(ctx, documentPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check document existence",
			"details": err.Error(),
			"path": documentPath,
			"suggestion": "Check storage connectivity and path validity",
		})
	}

	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
			"path":  documentPath,
			"suggestion": "Verify the document exists in storage and the path is correct",
			"available_endpoints": []string{
				"/api/v1/files/search?name=filename - Search for documents by name",
				"/api/v1/storage/documents - List all documents",
			},
		})
	}

	// Parse query parameters for URL type and expiration
	useSignedURL := c.Query("signed", "true") == "true"
	expirationParam := c.Query("expires", "1h")
	
	// Parse expiration duration (default 1 hour)
	expiration, err := time.ParseDuration(expirationParam)
	if err != nil {
		expiration = time.Hour // Default to 1 hour if parsing fails
	}

	// Limit maximum expiration to 24 hours for security
	if expiration > 24*time.Hour {
		expiration = 24 * time.Hour
	}

	var documentURL string
	
	if useSignedURL {
		// Generate signed URL for secure access
		documentURL, err = h.storage.GetSignedURL(documentPath, expiration)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to generate signed URL",
				"details": err.Error(),
				"path": documentPath,
				"expiration": expiration.String(),
				"suggestion": "Check storage service configuration and credentials",
			})
		}
	} else {
		// Generate public URL (CDN or direct)
		documentURL = h.storage.GetURL(documentPath)
		if documentURL == "" {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to generate document URL",
				"path": documentPath,
				"suggestion": "Check storage service configuration",
			})
		}
	}

	// Get file extension for content type determination
	ext := strings.ToLower(filepath.Ext(documentPath))
	contentType := getContentTypeFromExtension(ext)

	// Determine if we should proxy the file content vs redirect
	shouldProxy := h.shouldProxyFile(c, ext)

	if shouldProxy {
		// Proxy the file content for embedding/display
		return h.proxyFileContent(c, documentURL, contentType, documentPath)
	}

	// For download requests or when redirect is preferred
	if c.Query("download", "false") == "true" {
		c.Set("Content-Type", contentType)
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(documentPath)))
	}

	// Redirect to CDN URL
	return c.Redirect(documentURL, fiber.StatusFound)
}

// filterDocuments applies file type and size filters to the document list
func (h *StorageHandler) filterDocuments(objects []*storage.StorageObject, fileType string, minSize, maxSize int64) []*storage.StorageObject {
	var filtered []*storage.StorageObject

	for _, obj := range objects {
		// Skip directories
		if strings.HasSuffix(obj.Path, "/") {
			continue
		}

		// Skip very small files (likely empty or corrupt)
		if obj.Size < 100 {
			continue
		}

		// Skip system files
		filename := filepath.Base(obj.Path)
		if strings.Contains(filename, "__MACOSX") ||
			strings.Contains(filename, ".DS_Store") ||
			strings.HasSuffix(filename, ".tmp") ||
			strings.HasSuffix(filename, ".log") {
			continue
		}

		// Apply file type filter
		if fileType != "" {
			ext := strings.ToLower(filepath.Ext(obj.Path))
			if !strings.HasSuffix(ext, strings.ToLower(fileType)) {
				continue
			}
		}

		// Apply size filters
		if obj.Size < minSize {
			continue
		}
		if maxSize > 0 && obj.Size > maxSize {
			continue
		}

		filtered = append(filtered, obj)
	}

	return filtered
}

// paginateDocuments implements cursor-based pagination
func (h *StorageHandler) paginateDocuments(objects []*storage.StorageObject, cursor string, limit int) map[string]interface{} {
	startIndex := 0

	// Decode cursor if provided
	if cursor != "" {
		if decoded, err := base64.URLEncoding.DecodeString(cursor); err == nil {
			var cursorData map[string]interface{}
			if json.Unmarshal(decoded, &cursorData) == nil {
				if idx, ok := cursorData["index"].(float64); ok {
					startIndex = int(idx)
				}
			}
		}
	}

	// Apply pagination
	endIndex := startIndex + limit
	if endIndex > len(objects) {
		endIndex = len(objects)
	}

	var paginatedObjects []*storage.StorageObject
	if startIndex < len(objects) {
		paginatedObjects = objects[startIndex:endIndex]
	}

	// Generate next cursor
	var nextCursor string
	hasMore := endIndex < len(objects)
	if hasMore {
		cursorData := map[string]interface{}{
			"index": endIndex,
		}
		if cursorBytes, err := json.Marshal(cursorData); err == nil {
			nextCursor = base64.URLEncoding.EncodeToString(cursorBytes)
		}
	}

	// Convert storage objects to response format
	var documents []map[string]interface{}
	for _, obj := range paginatedObjects {
		documents = append(documents, map[string]interface{}{
			"path":          obj.Path,
			"size":          obj.Size,
			"last_modified": obj.LastModified,
			"file_type":     strings.ToLower(filepath.Ext(obj.Path)),
			"filename":      filepath.Base(obj.Path),
		})
	}

	return map[string]interface{}{
		"documents":        documents,
		"next_cursor":      nextCursor,
		"has_more":         hasMore,
		"total_returned":   len(documents),
		"total_estimated":  len(objects),
	}
}

// validateDocumentPath validates and sanitizes the document path
func (h *StorageHandler) validateDocumentPath(path string) error {
	// Check for empty path
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("empty path not allowed")
	}

	// Check for directory traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("directory traversal not allowed - path contains '..'")
	}

	// Check for null bytes and other control characters
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("null bytes not allowed in path")
	}

	// Check for other problematic characters
	invalidChars := []string{"|", "<", ">", ":", "*", "?", "\""}
	for _, char := range invalidChars {
		if strings.Contains(path, char) {
			return fmt.Errorf("invalid character '%s' not allowed in path", char)
		}
	}

	// Check for excessive path length
	if len(path) > 500 {
		return fmt.Errorf("path too long (max 500 characters, got %d)", len(path))
	}

	// Check for paths that start with special characters
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return fmt.Errorf("path cannot start with directory separators")
	}

	// Validate filename if it has an extension
	ext := strings.ToLower(filepath.Ext(path))
	if ext != "" {
		validExtensions := map[string]bool{
			".pdf": true, ".docx": true, ".doc": true, ".txt": true, ".rtf": true,
			".json": true, ".xml": true, ".html": true, ".htm": true,
			".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true,
			".tiff": true, ".tif": true, ".webp": true,
		}

		if !validExtensions[ext] {
			return fmt.Errorf("unsupported file extension: %s (allowed: pdf, docx, doc, txt, rtf, json, xml, html, jpg, jpeg, png, gif, bmp, tiff, webp)", ext)
		}
	}

	// Check for reasonable filename length
	filename := filepath.Base(path)
	if len(filename) > 255 {
		return fmt.Errorf("filename too long (max 255 characters, got %d)", len(filename))
	}

	// Check that filename is not just an extension
	if strings.HasPrefix(filename, ".") && len(strings.TrimPrefix(filename, ".")) < 2 {
		return fmt.Errorf("invalid filename: cannot be just an extension")
	}

	return nil
}

// FindDocumentsByName handles GET /api/v1/files/search - Find documents by filename pattern
func (h *StorageHandler) FindDocumentsByName(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
	defer cancel()

	// Get search parameters
	namePattern := c.Query("name", "")
	prefix := c.Query("prefix", "documents/")
	limitStr := c.Query("limit", "20")
	exactMatch := c.Query("exact", "false") == "true"

	if namePattern == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"missing_parameter",
			"Search pattern is required",
			map[string]interface{}{"parameter": "name"},
		))
	}

	// Parse limit
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	// List all documents from storage
	objects, err := h.storage.List(ctx, prefix)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"storage_error",
			"Failed to search documents",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Filter by name pattern
	var matches []map[string]interface{}
	namePattern = strings.ToLower(namePattern)

	for _, obj := range objects {
		// Skip directories
		if strings.HasSuffix(obj.Path, "/") {
			continue
		}

		filename := strings.ToLower(filepath.Base(obj.Path))
		
		var isMatch bool
		if exactMatch {
			isMatch = filename == namePattern || filename == namePattern+filepath.Ext(filename)
		} else {
			isMatch = strings.Contains(filename, namePattern)
		}

		if isMatch {
			// Generate both direct and CDN URLs
			directURL := h.storage.GetURL(obj.Path)
			signedURL, _ := h.storage.GetSignedURL(obj.Path, time.Hour)

			matches = append(matches, map[string]interface{}{
				"path":          obj.Path,
				"filename":      filepath.Base(obj.Path),
				"size":          obj.Size,
				"last_modified": obj.LastModified,
				"file_type":     strings.ToLower(filepath.Ext(obj.Path)),
				"direct_url":    directURL,
				"signed_url":    signedURL,
				"api_url":       fmt.Sprintf("/api/v1/files/%s", strings.TrimPrefix(obj.Path, "documents/")),
			})

			if len(matches) >= limit {
				break
			}
		}
	}

	response := map[string]interface{}{
		"documents":     matches,
		"total_found":   len(matches),
		"search_pattern": namePattern,
		"exact_match":   exactMatch,
		"limit":         limit,
	}

	return c.JSON(models.NewSuccessResponse(response, "Documents found successfully"))
}

// shouldProxyFile determines if we should proxy the file content instead of redirecting
func (h *StorageHandler) shouldProxyFile(c *fiber.Ctx, ext string) bool {
	// Check if explicitly requested to proxy
	if c.Query("proxy", "") == "true" {
		return true
	}

	// Check if redirect is explicitly disabled
	if c.Query("redirect", "true") == "false" {
		return true
	}

	// Always proxy for browser embedding requests
	secFetchDest := c.Get("Sec-Fetch-Dest")
	secFetchMode := c.Get("Sec-Fetch-Mode")
	
	// Browser is trying to embed the content (like in an iframe, object tag, or embed element)
	if secFetchDest == "embed" || secFetchDest == "object" || secFetchDest == "iframe" {
		return true
	}
	
	// For navigate mode with displayable content, check if it's likely for display
	if secFetchMode == "navigate" {
		// Always proxy PDFs and images for navigation requests
		if ext == ".pdf" || ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" || ext == ".bmp" {
			return true
		}
	}

	// Check referer to see if it's coming from a web application
	referer := c.Get("Referer")
	if referer != "" && (strings.Contains(referer, "localhost:5173") || strings.Contains(referer, "localhost:3000")) {
		// Request from local development frontend - likely needs proxying
		if ext == ".pdf" || ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" {
			return true
		}
	}

	// Check accept header for content type preferences
	accept := c.Get("Accept")
	if accept != "" {
		// If specifically requesting PDF or image content
		if strings.Contains(accept, "application/pdf") && ext == ".pdf" {
			return true
		}
		if strings.Contains(accept, "image/") && strings.HasPrefix(ext, ".") {
			imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff"}
			for _, imgExt := range imageExts {
				if ext == imgExt {
					return true
				}
			}
		}
	}

	return false
}

// proxyFileContent fetches the file from storage and streams it to the client
func (h *StorageHandler) proxyFileContent(c *fiber.Ctx, fileURL, contentType, documentPath string) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make request to the file URL
	resp, err := client.Get(fileURL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch document content",
			"details": err.Error(),
			"path": documentPath,
		})
	}
	defer resp.Body.Close()

	// Check if the remote request was successful
	if resp.StatusCode != http.StatusOK {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve document from storage",
			"status_code": resp.StatusCode,
			"path": documentPath,
		})
	}

	// Set response headers
	c.Set("Content-Type", contentType)
	c.Set("Content-Length", resp.Header.Get("Content-Length"))
	c.Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	c.Set("ETag", resp.Header.Get("ETag"))
	
	// Remove all embedding restrictions - TEMPORARY for development
	// TODO: Add proper security controls for production
	
	// Allow framing from any origin
	c.Response().Header.Del("X-Frame-Options")
	
	// Remove all restrictive security headers for embedded content
	c.Response().Header.Del("Cross-Origin-Embedder-Policy")
	c.Response().Header.Del("Cross-Origin-Resource-Policy") 
	c.Response().Header.Del("Cross-Origin-Opener-Policy")
	
	// Handle range requests for partial content (useful for large PDFs)
	if rangeHeader := c.Get("Range"); rangeHeader != "" {
		c.Set("Accept-Ranges", "bytes")
		// Note: Full range request handling would require more complex logic
		// For now, we'll serve the full content
	}

	// Set filename for download (always inline for now since we removed embedding checks)
	// TODO: Re-add embedding detection when security is re-enabled
	filename := filepath.Base(documentPath)
	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))

	// Stream the content
	_, err = io.Copy(c.Response().BodyWriter(), resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to stream document content",
			"details": err.Error(),
		})
	}

	return nil
}

// getContentTypeFromExtension returns the MIME type for a file extension
func getContentTypeFromExtension(ext string) string {
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".doc":
		return "application/msword"
	case ".txt":
		return "text/plain"
	case ".rtf":
		return "application/rtf"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".html", ".htm":
		return "text/html"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}
