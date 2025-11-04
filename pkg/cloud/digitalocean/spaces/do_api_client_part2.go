package spaces

import (
	"context"
	"fmt"
	"time"
)

func (c *doAPIClientImpl) UpdateSpacesKey(ctx context.Context, accessKey, name string) (*SpacesKey, error) {
	if accessKey == "" {
		return nil, fmt.Errorf("access key cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	requestBody := map[string]interface{}{
		"name": name,
	}

	resp, err := c.makeAPIRequest(ctx, "PUT", "/spaces/keys/"+accessKey, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to update Spaces key: %w", err)
	}

	var result struct {
		AccessKey struct {
			Name      string    `json:"name"`
			AccessKey string    `json:"access_key_id"`
			CreatedAt time.Time `json:"created_at"`
		} `json:"access_key"`
	}

	if err := c.parseAPIResponse(resp, &result); err != nil {
		return nil, err
	}

	return &SpacesKey{
		Name:      result.AccessKey.Name,
		AccessKey: result.AccessKey.AccessKey,
		CreatedAt: result.AccessKey.CreatedAt,
		Grants:    []*KeyGrant{}, // DigitalOcean Spaces keys have full access
	}, nil
}

func (c *doAPIClientImpl) DeleteSpacesKey(ctx context.Context, accessKey string) error {
	if accessKey == "" {
		return fmt.Errorf("access key cannot be empty")
	}

	resp, err := c.makeAPIRequest(ctx, "DELETE", "/spaces/keys/"+accessKey, nil)
	if err != nil {
		return fmt.Errorf("failed to delete Spaces key: %w", err)
	}

	return c.parseAPIResponse(resp, nil)
}

func (c *doAPIClientImpl) getCDNFromCache(cdnID string) (*CDNInfo, bool) {
	cdn, exists := c.cdnCache[cdnID]
	return cdn, exists
}

func (c *doAPIClientImpl) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"api_calls":      c.apiCalls,
		"last_api_call":  c.lastAPICall,
		"cached_cdns":    len(c.cdnCache),
		"base_url":       c.baseURL,
		"client_timeout": c.httpClient.Timeout.Seconds(),
	}
}
