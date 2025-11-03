package presenter

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuccess(t *testing.T) {
	// Given: A Fiber app with a test route
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Success(c, map[string]string{
			"message": "test data",
		})
	})

	// When: Request is made
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Meta)
	assert.Equal(t, "1.0.0", response.Meta.Version)
}

func TestSuccessWithStatus(t *testing.T) {
	// Given: A Fiber app with a test route
	app := fiber.New()
	app.Post("/test", func(c *fiber.Ctx) error {
		return SuccessWithStatus(c, fiber.StatusCreated, map[string]string{
			"id": "123",
		})
	})

	// When: Request is made
	req := httptest.NewRequest("POST", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response has custom status code
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var response Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
}

func TestErrorResponse(t *testing.T) {
	// Given: A Fiber app with a test route
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return ErrorResponse(c, fiber.StatusBadRequest, "TEST_ERROR", "Test error message", map[string]string{
			"field": "invalid",
		})
	})

	// When: Request is made
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is error
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var response Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Nil(t, response.Data)
	assert.NotNil(t, response.Error)
	assert.Equal(t, "TEST_ERROR", response.Error.Code)
	assert.Equal(t, "Test error message", response.Error.Message)
	assert.NotNil(t, response.Error.Details)
}

func TestBadRequest(t *testing.T) {
	// Given: A Fiber app with a test route
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return BadRequest(c, "Invalid input", map[string]string{
			"field": "email",
		})
	})

	// When: Request is made
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 400
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var response Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "BAD_REQUEST", response.Error.Code)
	assert.Equal(t, "Invalid input", response.Error.Message)
}

func TestNotFound(t *testing.T) {
	// Given: A Fiber app with a test route
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return NotFound(c, "Resource not found")
	})

	// When: Request is made
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 404
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	var response Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "NOT_FOUND", response.Error.Code)
	assert.Equal(t, "Resource not found", response.Error.Message)
}

func TestInternalError(t *testing.T) {
	// Given: A Fiber app with a test route
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return InternalError(c, "Something went wrong")
	})

	// When: Request is made
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 500
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var response Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "INTERNAL_ERROR", response.Error.Code)
	assert.Equal(t, "Something went wrong", response.Error.Message)
}

func TestValidationError(t *testing.T) {
	// Given: A Fiber app with a test route
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return ValidationError(c, map[string]string{
			"email":    "required",
			"password": "min_length",
		})
	})

	// When: Request is made
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is validation error
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var response Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	assert.Equal(t, "Request validation failed", response.Error.Message)
	assert.NotNil(t, response.Error.Details)
}

func TestBuildMeta_WithRequestIDInLocals(t *testing.T) {
	// Given: A Fiber app with request ID middleware
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("requestid", "test-request-id-123")
		return c.Next()
	})
	app.Get("/test", func(c *fiber.Ctx) error {
		return Success(c, nil)
	})

	// When: Request is made
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response includes request ID
	var response Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.Equal(t, "test-request-id-123", response.Meta.RequestID)
}

func TestBuildMeta_WithRequestIDInHeader(t *testing.T) {
	// Given: A Fiber app
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Success(c, nil)
	})

	// When: Request is made with X-Request-ID header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "header-request-id-456")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response includes request ID from header
	var response Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.Equal(t, "header-request-id-456", response.Meta.RequestID)
}

func TestBuildMeta_TimestampIsPresent(t *testing.T) {
	// Given: A Fiber app
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Success(c, nil)
	})

	// When: Request is made
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response includes timestamp
	var response Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.Greater(t, response.Meta.Timestamp, int64(0))
}
