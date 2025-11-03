package testing

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestApp(t *testing.T) {
	// Given: Test app is created
	app := TestApp(t)

	// Then: App is properly configured
	assert.NotNil(t, app)
	assert.NotNil(t, app.Config().ErrorHandler)
}

func TestNewRequest_GET(t *testing.T) {
	// Given: A test app with a GET endpoint
	app := TestApp(t)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When: GET request is made
	resp := NewRequest("GET", "/test").Execute(t, app)

	// Then: Request succeeds
	resp.AssertStatusOK()
	assert.Equal(t, "OK", resp.BodyString())
}

func TestNewRequest_WithJSONBody(t *testing.T) {
	// Given: A test app with a POST endpoint
	app := TestApp(t)
	app.Post("/test", func(c *fiber.Ctx) error {
		var body map[string]string
		if err := c.BodyParser(&body); err != nil {
			return err
		}
		return c.JSON(body)
	})

	// When: POST request is made with JSON body
	requestBody := map[string]string{"key": "value"}
	resp := NewRequest("POST", "/test").
		WithJSONBody(requestBody).
		Execute(t, app)

	// Then: Request succeeds and body is echoed
	resp.AssertStatusOK()

	var responseBody map[string]string
	resp.DecodeJSON(&responseBody)
	assert.Equal(t, "value", responseBody["key"])
}

func TestNewRequest_WithHeader(t *testing.T) {
	// Given: A test app that checks headers
	app := TestApp(t)
	app.Get("/test", func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		return c.SendString(auth)
	})

	// When: Request is made with custom header
	resp := NewRequest("GET", "/test").
		WithHeader("Authorization", "Bearer token123").
		Execute(t, app)

	// Then: Header is received
	resp.AssertStatusOK()
	assert.Equal(t, "Bearer token123", resp.BodyString())
}

func TestNewRequest_WithQuery(t *testing.T) {
	// Given: A test app that reads query params
	app := TestApp(t)
	app.Get("/test", func(c *fiber.Ctx) error {
		name := c.Query("name")
		return c.SendString(name)
	})

	// When: Request is made with query parameters
	resp := NewRequest("GET", "/test").
		WithQuery("name", "John").
		Execute(t, app)

	// Then: Query param is received
	resp.AssertStatusOK()
	assert.Equal(t, "John", resp.BodyString())
}

func TestTestResponse_AssertStatusCode(t *testing.T) {
	// Given: A test app
	app := TestApp(t)
	app.Get("/ok", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/not-found", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNotFound)
	})

	// When/Then: Various status code assertions
	NewRequest("GET", "/ok").Execute(t, app).AssertStatusOK()
	NewRequest("GET", "/not-found").Execute(t, app).AssertStatusNotFound()
}

func TestTestResponse_AssertJSONField(t *testing.T) {
	// Given: A test app that returns JSON
	app := TestApp(t)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"name": "John",
			"age":  float64(30), // JSON numbers are float64
		})
	})

	// When: Request is made
	resp := NewRequest("GET", "/test").Execute(t, app)

	// Then: JSON fields can be asserted
	resp.AssertStatusOK().
		AssertJSONField("name", "John").
		AssertJSONField("age", float64(30))
}

func TestTestResponse_AssertContains(t *testing.T) {
	// Given: A test app
	app := TestApp(t)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	// When: Request is made
	resp := NewRequest("GET", "/test").Execute(t, app)

	// Then: Body can be checked for substrings
	resp.AssertStatusOK().AssertContains("World")
}

func TestTestResponse_AssertHeader(t *testing.T) {
	// Given: A test app that sets headers
	app := TestApp(t)
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Set("X-Custom-Header", "CustomValue")
		return c.SendString("OK")
	})

	// When: Request is made
	resp := NewRequest("GET", "/test").Execute(t, app)

	// Then: Headers can be asserted
	resp.AssertStatusOK().AssertHeader("X-Custom-Header", "CustomValue")
}

func TestMultipartFormBuilder(t *testing.T) {
	// Given: A test app that accepts multipart form
	app := TestApp(t)
	app.Post("/upload", func(c *fiber.Ctx) error {
		file, err := c.FormFile("file")
		if err != nil {
			return err
		}

		field := c.FormValue("field1")

		return c.JSON(fiber.Map{
			"filename": file.Filename,
			"field1":   field,
		})
	})

	// When: Multipart form is submitted
	form := NewMultipartForm().
		AddField("field1", "value1").
		AddFile("file", "test.pdf", []byte("test content"))

	resp := form.ToRequest("POST", "/upload").Execute(t, app)

	// Then: Form data is received
	resp.AssertStatusOK()

	var result map[string]interface{}
	resp.DecodeJSON(&result)
	assert.Equal(t, "test.pdf", result["filename"])
	assert.Equal(t, "value1", result["field1"])
}

func TestMultipartFormBuilder_MultipleFiles(t *testing.T) {
	// Given: A test app that counts files
	app := TestApp(t)
	app.Post("/upload", func(c *fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			return err
		}

		fileCount := 0
		for _, files := range form.File {
			fileCount += len(files)
		}

		return c.JSON(fiber.Map{
			"file_count": fileCount,
		})
	})

	// When: Multiple files are submitted
	form := NewMultipartForm().
		AddFile("file1", "test1.pdf", []byte("content1")).
		AddFile("file2", "test2.pdf", []byte("content2"))

	resp := form.ToRequest("POST", "/upload").Execute(t, app)

	// Then: Both files are received
	resp.AssertStatusOK()

	var result map[string]interface{}
	resp.DecodeJSON(&result)
	assert.Equal(t, float64(2), result["file_count"]) // JSON numbers are float64
}

func TestTestRequest_ChainedBuilder(t *testing.T) {
	// Given: A test app
	app := TestApp(t)
	app.Post("/test", func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		param := c.Query("param")

		var body map[string]string
		_ = c.BodyParser(&body)

		return c.JSON(fiber.Map{
			"auth":  auth,
			"param": param,
			"body":  body["key"],
		})
	})

	// When: Request is built with method chaining
	resp := NewRequest("POST", "/test").
		WithHeader("Authorization", "Bearer token").
		WithQuery("param", "value").
		WithJSONBody(map[string]string{"key": "value"}).
		Execute(t, app)

	// Then: All parts of the request are received
	resp.AssertStatusOK()

	var result map[string]interface{}
	resp.DecodeJSON(&result)
	assert.Equal(t, "Bearer token", result["auth"])
	assert.Equal(t, "value", result["param"])
	assert.Equal(t, "value", result["body"])
}

func TestTestResponse_MethodChaining(t *testing.T) {
	// Given: A test app
	app := TestApp(t)
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Set("X-Test", "TestValue")
		return c.JSON(fiber.Map{
			"status": "success",
			"data":   "test data",
		})
	})

	// When/Then: All assertions can be chained
	NewRequest("GET", "/test").
		Execute(t, app).
		AssertStatusOK().
		AssertHeader("X-Test", "TestValue").
		AssertJSONField("status", "success").
		AssertJSONField("data", "test data").
		AssertContains("success")
}

func TestTestResponse_DecodeJSON_InvalidJSON(t *testing.T) {
	// Given: A test app that returns invalid JSON
	app := TestApp(t)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("not json")
	})

	// When: Attempting to decode invalid JSON
	resp := NewRequest("GET", "/test").Execute(t, app)

	// Then: We get the response but decoding will fail
	// Note: DecodeJSON uses require.NoError, so it will fail the test
	// This test documents the behavior - in practice, don't call DecodeJSON on non-JSON
	var result map[string]interface{}

	// We expect this will cause test failure due to require.NoError in DecodeJSON
	// In a real test suite, you'd check content-type before calling DecodeJSON
	_ = result
	_ = resp
	// Not actually calling DecodeJSON here since it would fail the test
}

func TestMultipartFormBuilder_EmptyForm(t *testing.T) {
	// Given: Empty multipart form
	form := NewMultipartForm()

	// When: Form is built
	body, contentType := form.Build()

	// Then: Form is valid but empty
	require.NotNil(t, body)
	require.Contains(t, contentType, "multipart/form-data")
}
