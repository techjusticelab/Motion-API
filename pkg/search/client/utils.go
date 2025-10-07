package client

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
)

// buildRequestBody creates a request body from a map
func buildRequestBody(data map[string]interface{}) io.Reader {
	if data == nil {
		return nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil
	}

	return bytes.NewReader(jsonData)
}

// parseResponse parses an OpenSearch response into the target struct
func parseResponse(res *opensearchapi.Response, target interface{}) error {
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, target)
}

// parseRawResponse reads the raw response body as bytes
func parseRawResponse(res *opensearchapi.Response) ([]byte, error) {
	return io.ReadAll(res.Body)
}
