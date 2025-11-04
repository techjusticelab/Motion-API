package apihelpers

import (
	"fmt"
	"net/http"
	"os"
)

func UploadFile(client *http.Client, endpoint, filePath, fileName string) error {
	fileHandle, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer fileHandle.Close()

	req, err := createMultipartRequest(endpoint, "file", fileName, fileHandle)
	if err != nil {
		return err
	}

	_, err = doRequest(client, req, http.StatusCreated, http.StatusOK, http.StatusPartialContent)
	return err
}
