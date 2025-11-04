package apihelpers

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultHTTPTimeout = 5 * time.Minute

func doRequest(client *http.Client, req *http.Request, expectedStatuses ...int) ([]byte, error) {
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(expectedStatuses) == 0 {
		expectedStatuses = []int{http.StatusOK}
	}

	for _, status := range expectedStatuses {
		if resp.StatusCode == status {
			return body, nil
		}
	}

	return body, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
}