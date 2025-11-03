package middleware

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDocumentID_Success(t *testing.T) {
	// Given: Fiber app with validation middleware
	app := fiber.New()
	app.Get("/documents/:id", ValidateDocumentID(), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made with valid document ID
	req := httptest.NewRequest("GET", "/documents/doc_123", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request succeeds
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestValidateDocumentID_Missing(t *testing.T) {
	// Given: Fiber app with validation middleware
	app := fiber.New()
	app.Get("/documents/:id", ValidateDocumentID(), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made without document ID
	req := httptest.NewRequest("GET", "/documents/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request is rejected
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode) // Route doesn't match
}

func TestValidateDocumentID_InvalidFormat(t *testing.T) {
	// Given: Fiber app with validation middleware
	app := fiber.New()
	app.Get("/documents/:id", ValidateDocumentID(), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made with invalid document ID (special characters)
	req := httptest.NewRequest("GET", "/documents/doc_123<script>", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request is rejected
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestValidateFileUpload_Success(t *testing.T) {
	// Given: Fiber app with file upload validation
	app := fiber.New()
	app.Post("/upload", ValidateFileUpload("file"), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made with valid file
	body, contentType := createMultipartFormDataWithFilename("test.pdf", []byte("test content"))

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", contentType)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request succeeds
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestValidateFileUpload_Missing(t *testing.T) {
	// Given: Fiber app with file upload validation
	app := fiber.New()
	app.Post("/upload", ValidateFileUpload("file"), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made without file
	req := httptest.NewRequest("POST", "/upload", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request is rejected
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestValidateFileUpload_EmptyFile(t *testing.T) {
	// Given: Fiber app with file upload validation
	app := fiber.New()
	app.Post("/upload", ValidateFileUpload("file"), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made with empty file
	body, contentType := createMultipartFormData(map[string][]byte{
		"file": []byte(""),
	})

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", contentType)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request is rejected
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestValidateContentType_Success(t *testing.T) {
	// Given: Fiber app with content type validation
	app := fiber.New()
	app.Post("/data", ValidateContentType("application/json"), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made with correct content type
	req := httptest.NewRequest("POST", "/data", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request succeeds
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestValidateContentType_Invalid(t *testing.T) {
	// Given: Fiber app with content type validation
	app := fiber.New()
	app.Post("/data", ValidateContentType("application/json"), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made with wrong content type
	req := httptest.NewRequest("POST", "/data", strings.NewReader("plain text"))
	req.Header.Set("Content-Type", "text/plain")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request is rejected
	assert.Equal(t, fiber.StatusUnsupportedMediaType, resp.StatusCode)
}

func TestValidateContentType_Missing(t *testing.T) {
	// Given: Fiber app with content type validation
	app := fiber.New()
	app.Post("/data", ValidateContentType("application/json"), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made without content type
	req := httptest.NewRequest("POST", "/data", strings.NewReader(`{"test":"data"}`))
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request is rejected
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestValidatePagination_Success(t *testing.T) {
	// Given: Fiber app with pagination validation
	app := fiber.New()
	app.Get("/items", ValidatePagination(), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made with valid pagination
	req := httptest.NewRequest("GET", "/items?page=1&size=10", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request succeeds
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestValidatePagination_NegativePage(t *testing.T) {
	// Given: Fiber app with pagination validation
	app := fiber.New()
	app.Get("/items", ValidatePagination(), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made with negative page
	req := httptest.NewRequest("GET", "/items?page=-1&size=10", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request is rejected
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestValidatePagination_ExcessiveSize(t *testing.T) {
	// Given: Fiber app with pagination validation
	app := fiber.New()
	app.Get("/items", ValidatePagination(), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made with size > 100
	req := httptest.NewRequest("GET", "/items?page=1&size=200", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request is rejected
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestValidateQueryParam_Success(t *testing.T) {
	// Given: Fiber app with query param validation
	app := fiber.New()
	app.Get("/search", ValidateQueryParam("query"), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made with required query param
	req := httptest.NewRequest("GET", "/search?query=test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request succeeds
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestValidateQueryParam_Missing(t *testing.T) {
	// Given: Fiber app with query param validation
	app := fiber.New()
	app.Get("/search", ValidateQueryParam("query"), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made without required query param
	req := httptest.NewRequest("GET", "/search", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request is rejected
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestValidateJSONBody_Success(t *testing.T) {
	// Given: Fiber app with JSON body validation
	app := fiber.New()
	app.Post("/data", ValidateJSONBody(), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made with valid JSON
	req := httptest.NewRequest("POST", "/data", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request succeeds
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestValidateJSONBody_WrongContentType(t *testing.T) {
	// Given: Fiber app with JSON body validation
	app := fiber.New()
	app.Post("/data", ValidateJSONBody(), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made with non-JSON content type
	req := httptest.NewRequest("POST", "/data", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "text/plain")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request is rejected
	assert.Equal(t, fiber.StatusUnsupportedMediaType, resp.StatusCode)
}

func TestValidateJSONBody_EmptyBody(t *testing.T) {
	// Given: Fiber app with JSON body validation
	app := fiber.New()
	app.Post("/data", ValidateJSONBody(), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: Request is made with empty body
	req := httptest.NewRequest("POST", "/data", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request is rejected
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestChain_Success(t *testing.T) {
	// Given: Fiber app with chained middleware
	app := fiber.New()

	// Chain returns a slice of handlers that can be used with Fiber's app.Use or inline
	handlers := Chain(
		ValidateContentType("application/json"),
		ValidateJSONBody(),
	)

	app.Post("/data", append(handlers, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})...)

	// When: Request passes all validations
	req := httptest.NewRequest("POST", "/data", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request succeeds
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestChain_FirstMiddlewareFails(t *testing.T) {
	// Given: Fiber app with chained middleware
	app := fiber.New()

	handlers := Chain(
		ValidateContentType("application/json"),
		ValidateJSONBody(),
	)

	app.Post("/data", append(handlers, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})...)

	// When: First middleware fails
	req := httptest.NewRequest("POST", "/data", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "text/plain")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Request is rejected at first middleware
	assert.Equal(t, fiber.StatusUnsupportedMediaType, resp.StatusCode)
}

// Helper function to create multipart form data
func createMultipartFormData(files map[string][]byte) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for filename, content := range files {
		part, _ := writer.CreateFormFile("file", filename)
		part.Write(content)
	}

	writer.Close()
	return body, writer.FormDataContentType()
}

// Helper function to create multipart form data with specific filename
func createMultipartFormDataWithFilename(filename string, content []byte) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("file", filename)
	part.Write(content)

	writer.Close()
	return body, writer.FormDataContentType()
}
