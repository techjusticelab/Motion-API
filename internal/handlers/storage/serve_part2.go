package storage

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"
)

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
