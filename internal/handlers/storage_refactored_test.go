package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"motion-index-fiber/pkg/storage"
)

// Test Helper: Mock Storage Service
type mockStorageForHandlers struct {
	objects      []*storage.StorageObject
	existsResult bool
	err          error
}

func newMockStorageForHandlers(objects []*storage.StorageObject) *mockStorageForHandlers {
	return &mockStorageForHandlers{
		objects:      objects,
		existsResult: true,
		err:          nil,
	}
}

func (m *mockStorageForHandlers) List(ctx context.Context, prefix string) ([]*storage.StorageObject, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.objects, nil
}

func (m *mockStorageForHandlers) Exists(ctx context.Context, path string) (bool, error) {
	return m.existsResult, m.err
}

func (m *mockStorageForHandlers) GetURL(path string) string {
	return "https://cdn.example.com/" + path
}

func (m *mockStorageForHandlers) GetSignedURL(path string, expiration time.Duration) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "https://cdn.example.com/" + path + "?signed=true", nil
}

func (m *mockStorageForHandlers) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("mock content")), nil
}

func (m *mockStorageForHandlers) Upload(ctx context.Context, path string, content io.Reader, metadata *storage.UploadMetadata) (*storage.UploadResult, error) {
	return &storage.UploadResult{Path: path, Success: true}, nil
}

func (m *mockStorageForHandlers) Delete(ctx context.Context, path string) error {
	return m.err
}

func (m *mockStorageForHandlers) IsHealthy() bool {
	return true
}

func (m *mockStorageForHandlers) GetMetrics() map[string]interface{} {
	return map[string]interface{}{}
}

func TestStorageHandlerRefactored_ListDocuments(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockObjects    []*storage.StorageObject
		expectedStatus int
		validateBody   func(*testing.T, string)
	}{
		{
			name:        "successful list with default parameters",
			queryParams: "",
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/file2.pdf", Size: 2048, LastModified: time.Now()},
			},
			expectedStatus: 200,
			validateBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "success")
				assert.Contains(t, body, "documents")
			},
		},
		{
			name:        "list with pagination",
			queryParams: "?limit=1&cursor=",
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/file2.pdf", Size: 2048, LastModified: time.Now()},
			},
			expectedStatus: 200,
			validateBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "has_more")
				assert.Contains(t, body, "next_cursor")
			},
		},
		{
			name:        "list with file type filter",
			queryParams: "?file_type=.pdf",
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/file2.docx", Size: 2048, LastModified: time.Now()},
			},
			expectedStatus: 200,
		},
		{
			name:        "list with size filters",
			queryParams: "?min_size=1000&max_size=3000",
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1500, LastModified: time.Now()},
				{Path: "documents/file2.pdf", Size: 500, LastModified: time.Now()},
			},
			expectedStatus: 200,
		},
		{
			name:           "list with invalid min_size",
			queryParams:    "?min_size=-100",
			mockObjects:    []*storage.StorageObject{},
			expectedStatus: 400,
			validateBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			mockStorage := newMockStorageForHandlers(tt.mockObjects)
			handler := NewStorageHandlerRefactored(mockStorage)

			app.Get("/api/storage/documents", handler.ListDocuments)

			req := httptest.NewRequest("GET", "/api/storage/documents"+tt.queryParams, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.validateBody != nil {
				body := make([]byte, 4096)
				n, _ := resp.Body.Read(body)
				tt.validateBody(t, string(body[:n]))
			}
		})
	}
}

func TestStorageHandlerRefactored_GetDocumentsCount(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockObjects    []*storage.StorageObject
		expectedStatus int
		expectedCount  int
	}{
		{
			name:        "count all documents",
			queryParams: "",
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/file2.pdf", Size: 2048, LastModified: time.Now()},
				{Path: "documents/file3.pdf", Size: 3072, LastModified: time.Now()},
			},
			expectedStatus: 200,
			expectedCount:  3,
		},
		{
			name:        "count with file type filter",
			queryParams: "?file_type=.pdf",
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/file2.docx", Size: 2048, LastModified: time.Now()},
			},
			expectedStatus: 200,
			expectedCount:  1,
		},
		{
			name:        "count with size range",
			queryParams: "?min_size=1500&max_size=2500",
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/file2.pdf", Size: 2048, LastModified: time.Now()},
				{Path: "documents/file3.pdf", Size: 3072, LastModified: time.Now()},
			},
			expectedStatus: 200,
			expectedCount:  1,
		},
		{
			name:           "count with invalid parameters",
			queryParams:    "?min_size=5000&max_size=1000",
			mockObjects:    []*storage.StorageObject{},
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			mockStorage := newMockStorageForHandlers(tt.mockObjects)
			handler := NewStorageHandlerRefactored(mockStorage)

			app.Get("/api/storage/documents/count", handler.GetDocumentsCount)

			req := httptest.NewRequest("GET", "/api/storage/documents/count"+tt.queryParams, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == 200 {
				body := make([]byte, 4096)
				n, _ := resp.Body.Read(body)
				bodyStr := string(body[:n])
				assert.Contains(t, bodyStr, "total_count")
			}
		})
	}
}

func TestStorageHandlerRefactored_FindDocumentsByName(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockObjects    []*storage.StorageObject
		expectedStatus int
		expectedCount  int
		validateBody   func(*testing.T, string)
	}{
		{
			name:        "search by partial name",
			queryParams: "?name=motion",
			mockObjects: []*storage.StorageObject{
				{Path: "documents/motion_to_suppress.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/other_file.pdf", Size: 2048, LastModified: time.Now()},
			},
			expectedStatus: 200,
			expectedCount:  1,
			validateBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "motion")
			},
		},
		{
			name:        "search with exact match",
			queryParams: "?name=motion.pdf&exact=true",
			mockObjects: []*storage.StorageObject{
				{Path: "documents/motion.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/motion_brief.pdf", Size: 2048, LastModified: time.Now()},
			},
			expectedStatus: 200,
			expectedCount:  1,
		},
		{
			name:        "search with limit",
			queryParams: "?name=file&limit=2",
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/file2.pdf", Size: 2048, LastModified: time.Now()},
				{Path: "documents/file3.pdf", Size: 3072, LastModified: time.Now()},
			},
			expectedStatus: 200,
		},
		{
			name:        "search with no matches",
			queryParams: "?name=nonexistent",
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
			},
			expectedStatus: 200,
			validateBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "total_found")
			},
		},
		{
			name:           "search without name parameter",
			queryParams:    "",
			mockObjects:    []*storage.StorageObject{},
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			mockStorage := newMockStorageForHandlers(tt.mockObjects)
			handler := NewStorageHandlerRefactored(mockStorage)

			app.Get("/api/v1/files/search", handler.FindDocumentsByName)

			req := httptest.NewRequest("GET", "/api/v1/files/search"+tt.queryParams, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.validateBody != nil {
				body := make([]byte, 4096)
				n, _ := resp.Body.Read(body)
				tt.validateBody(t, string(body[:n]))
			}
		})
	}
}

func TestStorageHandlerRefactored_ServeDocument(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		queryParams    string
		existsResult   bool
		expectedStatus int
		validateHeader func(*testing.T, *http.Response)
	}{
		{
			name:           "serve existing document",
			path:           "test.pdf",
			queryParams:    "",
			existsResult:   true,
			expectedStatus: 302, // Redirect
			validateHeader: func(t *testing.T, resp *http.Response) {
				location := resp.Header.Get("Location")
				assert.Contains(t, location, "test.pdf")
			},
		},
		{
			name:           "serve with signed URL",
			path:           "test.pdf",
			queryParams:    "?signed=true",
			existsResult:   true,
			expectedStatus: 302,
			validateHeader: func(t *testing.T, resp *http.Response) {
				location := resp.Header.Get("Location")
				assert.Contains(t, location, "signed=true")
			},
		},
		{
			name:           "serve with download flag",
			path:           "test.pdf",
			queryParams:    "?download=true",
			existsResult:   true,
			expectedStatus: 302,
			validateHeader: func(t *testing.T, resp *http.Response) {
				contentDisposition := resp.Header.Get("Content-Disposition")
				assert.Contains(t, contentDisposition, "attachment")
			},
		},
		{
			name:           "document not found",
			path:           "nonexistent.pdf",
			queryParams:    "",
			existsResult:   false,
			expectedStatus: 404,
		},
		{
			name:           "missing path",
			path:           "",
			queryParams:    "",
			existsResult:   true,
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			mockStorage := newMockStorageForHandlers([]*storage.StorageObject{})
			mockStorage.existsResult = tt.existsResult
			handler := NewStorageHandlerRefactored(mockStorage)

			app.Get("/api/v1/files/*", handler.ServeDocument)

			req := httptest.NewRequest("GET", "/api/v1/files/"+tt.path+tt.queryParams, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.validateHeader != nil {
				tt.validateHeader(t, resp)
			}
		})
	}
}

func TestStorageHandlerRefactored_Integration(t *testing.T) {
	// Integration test with multiple operations
	app := fiber.New()
	mockStorage := newMockStorageForHandlers([]*storage.StorageObject{
		{Path: "documents/motion1.pdf", Size: 1024, LastModified: time.Now()},
		{Path: "documents/motion2.pdf", Size: 2048, LastModified: time.Now()},
		{Path: "documents/brief.docx", Size: 3072, LastModified: time.Now()},
	})
	handler := NewStorageHandlerRefactored(mockStorage)

	// Register routes
	app.Get("/api/storage/documents", handler.ListDocuments)
	app.Get("/api/storage/documents/count", handler.GetDocumentsCount)
	app.Get("/api/v1/files/search", handler.FindDocumentsByName)
	app.Get("/api/v1/files/*", handler.ServeDocument)

	t.Run("list all documents", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/storage/documents", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("count documents", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/storage/documents/count", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("search documents", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/files/search?name=motion", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("serve document", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/files/motion1.pdf", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 302, resp.StatusCode)
	})
}
