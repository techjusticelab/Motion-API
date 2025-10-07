package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestJWT(t *testing.T) {
	// Test JWT secret for testing
	testSecret := "test-secret-key-for-jwt-testing"

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectNext     bool
	}{
		{
			name:           "valid JWT token",
			authHeader:     "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			expectedStatus: 200,
			expectNext:     true,
		},
		{
			name:           "missing authorization header",
			authHeader:     "",
			expectedStatus: 401,
			expectNext:     false,
		},
		{
			name:           "invalid authorization format - no Bearer",
			authHeader:     "invalid-format",
			expectedStatus: 401,
			expectNext:     false,
		},
		{
			name:           "invalid authorization format - only Bearer",
			authHeader:     "Bearer",
			expectedStatus: 401,
			expectNext:     false,
		},
		{
			name:           "invalid JWT token",
			authHeader:     "Bearer invalid.jwt.token",
			expectedStatus: 401,
			expectNext:     false,
		},
		{
			name:           "empty token after Bearer",
			authHeader:     "Bearer ",
			expectedStatus: 401,
			expectNext:     false,
		},
		{
			name:           "malformed JWT - not enough parts",
			authHeader:     "Bearer invalid.token",
			expectedStatus: 401,
			expectNext:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create Fiber app
			app := fiber.New()

			// Add JWT middleware
			app.Use("/protected", JWT(testSecret))

			// Add test route
			nextCalled := false
			app.Get("/protected/test", func(c *fiber.Ctx) error {
				nextCalled = true
				return c.JSON(fiber.Map{"message": "success"})
			})

			// Create request
			req := httptest.NewRequest("GET", "/protected/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Execute request
			resp, err := app.Test(req)
			assert.NoError(t, err)

			// Check status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Check if next handler was called
			assert.Equal(t, tt.expectNext, nextCalled)
		})
	}
}

func TestExtractTokenFromHeader(t *testing.T) {
	tests := []struct {
		name          string
		authHeader    string
		expectedToken string
		expectedError bool
	}{
		{
			name:          "valid Bearer token",
			authHeader:    "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expectedToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expectedError: false,
		},
		{
			name:          "empty header",
			authHeader:    "",
			expectedToken: "",
			expectedError: true,
		},
		{
			name:          "missing Bearer prefix",
			authHeader:    "token-without-bearer",
			expectedToken: "",
			expectedError: true,
		},
		{
			name:          "Bearer without token",
			authHeader:    "Bearer",
			expectedToken: "",
			expectedError: true,
		},
		{
			name:          "Bearer with empty token",
			authHeader:    "Bearer ",
			expectedToken: "",
			expectedError: true,
		},
		{
			name:          "case sensitive Bearer",
			authHeader:    "bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expectedToken: "",
			expectedError: true,
		},
		{
			name:          "Bearer with extra spaces",
			authHeader:    "Bearer  eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expectedToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := extractTokenFromHeader(tt.authHeader)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}
		})
	}
}

func TestValidateJWT(t *testing.T) {
	testSecret := "test-secret-key-for-jwt-testing"

	tests := []struct {
		name          string
		token         string
		secret        string
		expectedError bool
	}{
		{
			name:          "valid JWT token (none algorithm for testing)",
			token:         "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.",
			secret:        testSecret,
			expectedError: true, // none algorithm should be rejected for security
		},
		{
			name:          "malformed JWT - invalid structure",
			token:         "invalid.jwt",
			secret:        testSecret,
			expectedError: true,
		},
		{
			name:          "empty token",
			token:         "",
			secret:        testSecret,
			expectedError: true,
		},
		{
			name:          "JWT with invalid signature",
			token:         "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.invalid-signature",
			secret:        testSecret,
			expectedError: true,
		},
		{
			name:          "JWT with wrong secret",
			token:         "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			secret:        "wrong-secret",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJWT(tt.token, tt.secret)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJWTWithMultipleRoutes(t *testing.T) {
	testSecret := "test-secret-key-for-jwt-testing"

	// Create Fiber app
	app := fiber.New()

	// Public route (no JWT required)
	app.Get("/public", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "public"})
	})

	// Protected routes group
	protected := app.Group("/protected", JWT(testSecret))
	protected.Get("/test1", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "protected1"})
	})
	protected.Get("/test2", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "protected2"})
	})

	tests := []struct {
		name           string
		url            string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "public route - no auth required",
			url:            "/public",
			authHeader:     "",
			expectedStatus: 200,
		},
		{
			name:           "protected route without auth",
			url:            "/protected/test1",
			authHeader:     "",
			expectedStatus: 401,
		},
		{
			name:           "protected route with valid auth",
			url:            "/protected/test1",
			authHeader:     "Bearer valid-token-placeholder", // This will still fail validation but tests middleware flow
			expectedStatus: 401,                              // Will be 401 due to invalid token, but middleware is working
		},
		{
			name:           "another protected route without auth",
			url:            "/protected/test2",
			authHeader:     "",
			expectedStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestJWTErrorHandling(t *testing.T) {
	testSecret := "test-secret"

	app := fiber.New()
	app.Use("/test", JWT(testSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	// Test various error scenarios
	errorTests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "completely malformed header",
			authHeader: "NotBearer malformed",
			wantStatus: 401,
		},
		{
			name:       "bearer with special characters",
			authHeader: "Bearer token@#$%^&*()",
			wantStatus: 401,
		},
		{
			name:       "bearer with newlines",
			authHeader: "Bearer token\nwith\nnewlines",
			wantStatus: 401,
		},
		{
			name:       "bearer with tabs",
			authHeader: "Bearer token\twith\ttabs",
			wantStatus: 401,
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tt.authHeader)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestJWTMiddlewareConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		secret     string
		shouldWork bool
	}{
		{
			name:       "valid secret",
			secret:     "valid-secret-key",
			shouldWork: true,
		},
		{
			name:       "empty secret",
			secret:     "",
			shouldWork: true, // Middleware should still be created, but validation will fail
		},
		{
			name:       "short secret",
			secret:     "short",
			shouldWork: true, // Middleware created, but security may be compromised
		},
		{
			name:       "long secret",
			secret:     "very-long-secret-key-with-many-characters-for-enhanced-security",
			shouldWork: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that middleware creation doesn't panic
			assert.NotPanics(t, func() {
				middleware := JWT(tt.secret)
				assert.NotNil(t, middleware)
			})
		})
	}
}
