package testing

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// TestApp creates a test Fiber app with standard configuration
func TestApp(t *testing.T) *fiber.App {
	return fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})
}

// TestRequest represents a test HTTP request builder
type TestRequest struct {
	Method      string
	Path        string
	Body        io.Reader
	Headers     map[string]string
	QueryParams map[string]string
}

// NewRequest creates a new test request builder
func NewRequest(method, path string) *TestRequest {
	return &TestRequest{
		Method:      method,
		Path:        path,
		Headers:     make(map[string]string),
		QueryParams: make(map[string]string),
	}
}

// WithJSONBody sets JSON body for the request
func (r *TestRequest) WithJSONBody(body interface{}) *TestRequest {
	jsonBytes, _ := json.Marshal(body)
	r.Body = bytes.NewReader(jsonBytes)
	r.Headers["Content-Type"] = "application/json"
	return r
}

// WithBody sets raw body for the request
func (r *TestRequest) WithBody(body string) *TestRequest {
	r.Body = strings.NewReader(body)
	return r
}

// WithHeader adds a header to the request
func (r *TestRequest) WithHeader(key, value string) *TestRequest {
	r.Headers[key] = value
	return r
}

// WithQuery adds a query parameter
func (r *TestRequest) WithQuery(key, value string) *TestRequest {
	r.QueryParams[key] = value
	return r
}

// Build builds the HTTP test request
func (r *TestRequest) Build() *http.Request {
	path := r.Path
	if len(r.QueryParams) > 0 {
		path += "?"
		first := true
		for k, v := range r.QueryParams {
			if !first {
				path += "&"
			}
			path += k + "=" + v
			first = false
		}
	}

	req := httptest.NewRequest(r.Method, path, r.Body)
	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}
	return req
}

// Execute executes the request against the app and returns the response
func (r *TestRequest) Execute(t *testing.T, app *fiber.App) *TestResponse {
	req := r.Build()
	resp, err := app.Test(req)
	require.NoError(t, err, "Failed to execute test request")

	return &TestResponse{
		t:        t,
		Response: resp,
	}
}

// TestResponse wraps an HTTP response with assertion helpers
type TestResponse struct {
	t        *testing.T
	Response *http.Response
	bodyRead []byte // Cache body after first read
}

// StatusCode returns the response status code
func (r *TestResponse) StatusCode() int {
	return r.Response.StatusCode
}

// AssertStatusCode asserts the response has the expected status code
func (r *TestResponse) AssertStatusCode(expected int) *TestResponse {
	require.Equal(r.t, expected, r.Response.StatusCode,
		"Expected status %d but got %d", expected, r.Response.StatusCode)
	return r
}

// AssertStatusOK asserts the response status is 200 OK
func (r *TestResponse) AssertStatusOK() *TestResponse {
	return r.AssertStatusCode(fiber.StatusOK)
}

// AssertStatusCreated asserts the response status is 201 Created
func (r *TestResponse) AssertStatusCreated() *TestResponse {
	return r.AssertStatusCode(fiber.StatusCreated)
}

// AssertStatusBadRequest asserts the response status is 400 Bad Request
func (r *TestResponse) AssertStatusBadRequest() *TestResponse {
	return r.AssertStatusCode(fiber.StatusBadRequest)
}

// AssertStatusNotFound asserts the response status is 404 Not Found
func (r *TestResponse) AssertStatusNotFound() *TestResponse {
	return r.AssertStatusCode(fiber.StatusNotFound)
}

// AssertStatusInternalServerError asserts the response status is 500
func (r *TestResponse) AssertStatusInternalServerError() *TestResponse {
	return r.AssertStatusCode(fiber.StatusInternalServerError)
}

// Body returns the response body as bytes (cached after first read)
func (r *TestResponse) Body() []byte {
	if r.bodyRead != nil {
		return r.bodyRead
	}

	body, err := io.ReadAll(r.Response.Body)
	require.NoError(r.t, err, "Failed to read response body")
	r.bodyRead = body
	return body
}

// BodyString returns the response body as string
func (r *TestResponse) BodyString() string {
	return string(r.Body())
}

// DecodeJSON decodes the response body into the provided interface
func (r *TestResponse) DecodeJSON(v interface{}) *TestResponse {
	err := json.Unmarshal(r.Body(), v)
	require.NoError(r.t, err, "Failed to decode JSON response")
	return r
}

// AssertJSONField asserts a specific field exists in JSON response
func (r *TestResponse) AssertJSONField(field string, expectedValue interface{}) *TestResponse {
	var body map[string]interface{}
	r.DecodeJSON(&body)

	value, exists := body[field]
	require.True(r.t, exists, "Expected field '%s' not found in response", field)
	require.Equal(r.t, expectedValue, value, "Field '%s' has unexpected value", field)
	return r
}

// AssertContains asserts the response body contains the substring
func (r *TestResponse) AssertContains(substring string) *TestResponse {
	body := r.BodyString()
	require.Contains(r.t, body, substring,
		"Response body does not contain expected substring")
	return r
}

// AssertHeader asserts a specific header value
func (r *TestResponse) AssertHeader(key, expectedValue string) *TestResponse {
	value := r.Response.Header.Get(key)
	require.Equal(r.t, expectedValue, value,
		"Header '%s' has unexpected value", key)
	return r
}

// MultipartFormBuilder helps build multipart form data
type MultipartFormBuilder struct {
	fields map[string]string
	files  map[string]FileUpload
}

// FileUpload represents a file to upload in multipart form
type FileUpload struct {
	Filename string
	Content  []byte
}

// NewMultipartForm creates a new multipart form builder
func NewMultipartForm() *MultipartFormBuilder {
	return &MultipartFormBuilder{
		fields: make(map[string]string),
		files:  make(map[string]FileUpload),
	}
}

// AddField adds a text field to the form
func (m *MultipartFormBuilder) AddField(key, value string) *MultipartFormBuilder {
	m.fields[key] = value
	return m
}

// AddFile adds a file to the form
func (m *MultipartFormBuilder) AddFile(fieldName, filename string, content []byte) *MultipartFormBuilder {
	m.files[fieldName] = FileUpload{
		Filename: filename,
		Content:  content,
	}
	return m
}

// Build builds the multipart form and returns body and content type
func (m *MultipartFormBuilder) Build() (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add fields
	for key, value := range m.fields {
		_ = writer.WriteField(key, value)
	}

	// Add files
	for fieldName, file := range m.files {
		part, _ := writer.CreateFormFile(fieldName, file.Filename)
		_, _ = part.Write(file.Content)
	}

	_ = writer.Close()
	return body, writer.FormDataContentType()
}

// ToRequest converts the form to a TestRequest
func (m *MultipartFormBuilder) ToRequest(method, path string) *TestRequest {
	body, contentType := m.Build()
	return &TestRequest{
		Method: method,
		Path:   path,
		Body:   body,
		Headers: map[string]string{
			"Content-Type": contentType,
		},
		QueryParams: make(map[string]string),
	}
}
