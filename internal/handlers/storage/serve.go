package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/config"
	"motion-index-fiber/pkg/storage"
)

type ServeHandler struct {
	cfg     *config.Config
	storage storage.Service
}

func NewServeHandler(cfg *config.Config, storage storage.Service) *ServeHandler {
	return &ServeHandler{
		cfg:     cfg,
		storage: storage,
	}
}

// ServeDocument handles GET /api/v1/files/* - Serve or redirect to a document
func (h *ServeHandler) ServeDocument(c *fiber.Ctx) error {
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
			"error":      "Invalid URL encoding in document path",
			"details":    fmt.Sprintf("Failed to decode path '%s': %v", rawDocumentPath, err),
			"suggestion": "Ensure the path is properly URL-encoded. Use %2F for forward slashes.",
		})
	}

	// Validate and sanitize the path
	if err := h.validateDocumentPath(documentPath); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid document path",
			"details": err.Error(),
			"path":    documentPath,
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
			"error":      "Failed to check document existence",
			"details":    err.Error(),
			"path":       documentPath,
			"suggestion": "Check storage connectivity and path validity",
		})
	}

	if !exists {
		// Fallback: try to recover the correct path by searching Spaces using ID/filename parts
		if recoveredPath, ok := h.resolveDocumentPathBySearch(ctx, documentPath); ok {
			documentPath = recoveredPath
		} else {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":      "Document not found",
				"path":       documentPath,
				"suggestion": "Verify the document exists in storage and the path is correct",
				"available_endpoints": []string{
					"/api/v1/files/search?name=filename - Search for documents by name",
					"/api/v1/storage/documents - List all documents",
				},
			})
		}
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
				"error":      "Failed to generate signed URL",
				"details":    err.Error(),
				"path":       documentPath,
				"expiration": expiration.String(),
				"suggestion": "Check storage service configuration and credentials",
			})
		}
	} else {
		// Generate public URL (CDN or direct)
		documentURL = h.storage.GetURL(documentPath)
		if documentURL == "" {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":      "Failed to generate document URL",
				"path":       documentPath,
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

// resolveDocumentPathBySearch attempts to recover the correct storage key when the exact path is unknown.
func (h *ServeHandler) resolveDocumentPathBySearch(ctx context.Context, requestedPath string) (string, bool) {
	// Ensure prefix for consistency
	path := requestedPath
	if !strings.HasPrefix(path, "documents/") {
		path = "documents/" + path
	}

	base := filepath.Base(path)
	if base == "" || base == "documents" {
		return "", false
	}

	// Build candidate prefixes
	var prefixes []string
	parts := strings.Split(base, "_")
	if len(parts) >= 3 && strings.HasPrefix(parts[0], "doc") {
		// Example: doc_1759855215756364200_2015-Dependency-...pdf
		docIDPrefix := strings.Join(parts[0:2], "_") // doc_<number>
		name := strings.Join(parts[2:], "_")         // 2015-Dependency-...pdf
		nameNoExt := strings.TrimSuffix(name, filepath.Ext(name))

		prefixes = append(prefixes,
			"documents/"+docIDPrefix,
			"documents/"+name,
		)
		if nameNoExt != "" {
			prefixes = append(prefixes, "documents/"+nameNoExt)
		}
	} else {
		// Fall back to filename-based prefixes
		name := base
		nameNoExt := strings.TrimSuffix(name, filepath.Ext(name))
		prefixes = append(prefixes, "documents/"+name)
		if nameNoExt != "" {
			prefixes = append(prefixes, "documents/"+nameNoExt)
		}
	}

	// Try each prefix and pick the best candidate
	for _, p := range prefixes {
		objects, err := h.storage.List(ctx, p)
		if err != nil || len(objects) == 0 {
			continue
		}

		// 1) Exact basename match wins
		for _, obj := range objects {
			if strings.EqualFold(filepath.Base(obj.Path), base) {
				return obj.Path, true
			}
		}
		// 2) Any object containing the basename
		for _, obj := range objects {
			if strings.Contains(strings.ToLower(obj.Path), strings.ToLower(base)) {
				return obj.Path, true
			}
		}
		// 3) Single object under the prefix is a reasonable guess
		if len(objects) == 1 {
			return objects[0].Path, true
		}
	}

	return "", false
}

// validateDocumentPath validates and sanitizes the document path
func (h *ServeHandler) validateDocumentPath(path string) error {
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

// shouldProxyFile determines if we should proxy the file content instead of redirecting
func (h *ServeHandler) shouldProxyFile(c *fiber.Ctx, ext string) bool {
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
func (h *ServeHandler) proxyFileContent(c *fiber.Ctx, fileURL, contentType, documentPath string) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make request to the file URL
	resp, err := client.Get(fileURL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to fetch document content",
			"details": err.Error(),
			"path":    documentPath,
		})
	}
	defer resp.Body.Close()

	// Check if the remote request was successful
	if resp.StatusCode != http.StatusOK {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":       "Failed to retrieve document from storage",
			"status_code": resp.StatusCode,
			"path":        documentPath,
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
			"error":   "Failed to stream document content",
			"details": err.Error(),
		})
	}

	return nil
}
