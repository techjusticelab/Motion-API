package middleware

import (
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type ErrorResponse struct {
	Status       int      `json:"status"`
	Message      string   `json:"message"`
	Error        string   `json:"error,omitempty"`
	Suggestions  []string `json:"suggestions,omitempty"`
	RequestedURL string   `json:"requested_url,omitempty"`
}

// Available API routes for suggestions
var availableRoutes = []string{
	"GET /health",
	"GET /api/v1/legal-tags",
	"GET /api/v1/document-types",
	"GET /api/v1/document-stats",
	"GET /api/v1/field-options",
	"GET /api/v1/metadata-fields/{field}",
	"GET /api/v1/documents/{id}",
	"POST /api/v1/categorise",
	"POST /api/v1/analyze-redactions",
	"POST /api/v1/search",
	"POST /api/v1/update-metadata (auth required)",
	"DELETE /api/v1/documents/{id} (auth required)",
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"
	errorDetail := ""

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	} else {
		errorDetail = err.Error()
		log.Printf("Unhandled error: %v", err)
	}

	response := ErrorResponse{
		Status:       code,
		Message:      message,
		RequestedURL: c.OriginalURL(),
	}

	// Include error details in development mode
	if errorDetail != "" {
		response.Error = errorDetail
	}

	// Enhanced 404 handling with route suggestions
	if code == fiber.StatusNotFound {
		response.Message = fmt.Sprintf("Endpoint not found: %s %s", c.Method(), c.Path())
		response.Suggestions = generateRouteSuggestions(c.Method(), c.Path())
		
		// Add helpful message about available endpoints
		if len(response.Suggestions) == 0 {
			response.Error = "This endpoint does not exist. Check the available routes below for valid API endpoints."
		}
	}

	return c.Status(code).JSON(response)
}

// generateRouteSuggestions provides helpful route suggestions for 404 errors
func generateRouteSuggestions(method, path string) []string {
	var suggestions []string
	
	// Normalize path for comparison
	normalizedPath := strings.ToLower(path)
	
	// Look for similar routes based on path segments
	for _, route := range availableRoutes {
		routeParts := strings.Fields(route)
		if len(routeParts) >= 2 {
			routeMethod := routeParts[0]
			routePath := strings.ToLower(routeParts[1])
			
			// If methods match or method is GET, suggest the route
			if routeMethod == method || method == "GET" {
				// Check for partial path matches
				if strings.Contains(normalizedPath, "legal-tag") && strings.Contains(routePath, "legal-tag") {
					suggestions = append(suggestions, route)
				} else if strings.Contains(normalizedPath, "document") && strings.Contains(routePath, "document") {
					suggestions = append(suggestions, route)
				} else if strings.Contains(normalizedPath, "search") && strings.Contains(routePath, "search") {
					suggestions = append(suggestions, route)
				} else if strings.Contains(normalizedPath, "health") && strings.Contains(routePath, "health") {
					suggestions = append(suggestions, route)
				}
			}
		}
	}
	
	// If no specific suggestions found, provide common endpoints
	if len(suggestions) == 0 {
		suggestions = []string{
			"GET /health - Health check endpoint",
			"GET /api/v1/legal-tags - Get available legal tags",
			"POST /api/v1/search - Search documents",
			"GET /api/v1/document-stats - Get document statistics",
		}
	}
	
	return suggestions
}
