package apihelpers

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
)

func ExtractTextFromFile(client *http.Client, endpoint, filePath, fileName string) (string, error) {
	fileHandle, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for extraction: %w", err)
	}
	defer fileHandle.Close()

	req, err := createMultipartRequest(endpoint, "file", fileName, fileHandle)
	if err != nil {
		return "", err
	}

	body, err := doRequest(client, req, http.StatusOK)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func ExtractTextFromBytes(client *http.Client, endpoint, fileName string, data []byte) (string, error) {
	reader := bytes.NewReader(data)
	req, err := createMultipartRequest(endpoint, "file", fileName, reader)
	if err != nil {
		return "", err
	}

	body, err := doRequest(client, req, http.StatusOK)
	if err != nil {
		return "", err
	}

	return string(body), nil
}