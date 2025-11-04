package apihelpers

import (
	"fmt"
	"net/http"
	"os"
)

func ConvertToPDF(client *http.Client, endpoint, filePath, fileName string) ([]byte, error) {
	fileHandle, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for conversion: %w", err)
	}
	defer fileHandle.Close()

	req, err := createMultipartRequest(endpoint, "file", fileName, fileHandle)
	if err != nil {
		return nil, err
	}

	return doRequest(client, req, http.StatusOK)
}