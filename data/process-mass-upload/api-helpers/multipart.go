package apihelpers

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

func createMultipartRequest(endpoint, fieldName, fileName string, reader io.Reader) (*http.Request, error) {
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err = io.Copy(part, reader); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	contentType := writer.FormDataContentType()
	if err = writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	return req, nil
}