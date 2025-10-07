package spaces

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DOAPIClient handles direct DigitalOcean API operations
type DOAPIClient interface {
	// CDN Management
	CreateCDN(ctx context.Context, origin string, ttl int) (*CDNInfo, error)
	GetCDN(ctx context.Context, cdnID string) (*CDNInfo, error)
	ListCDNs(ctx context.Context) ([]*CDNInfo, error)
	DeleteCDN(ctx context.Context, cdnID string) error
	FlushCDNCache(ctx context.Context, cdnID string, files []string) error

	// Access Key Management
	CreateSpacesKey(ctx context.Context, name string) (*SpacesKey, error)
	GetSpacesKey(ctx context.Context, accessKey string) (*SpacesKey, error)
	ListSpacesKeys(ctx context.Context) ([]*SpacesKey, error)
	UpdateSpacesKey(ctx context.Context, accessKey, name string) (*SpacesKey, error)
	DeleteSpacesKey(ctx context.Context, accessKey string) error
}

// doAPIClientImpl implements DOAPIClient using direct DigitalOcean REST API calls
type doAPIClientImpl struct {
	apiToken   string
	httpClient *http.Client
	baseURL    string
	
	// Cache for CDN information to reduce API calls
	cdnCache map[string]*CDNInfo
	
	// Performance tracking
	lastAPICall time.Time
	apiCalls    int64
}

// NewDOAPIClient creates a new DigitalOcean API client
func NewDOAPIClient(apiToken string) DOAPIClient {
	return &doAPIClientImpl{
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:  "https://api.digitalocean.com/v2",
		cdnCache: make(map[string]*CDNInfo),
		apiCalls: 0,
	}
}

// makeAPIRequest makes a request to the DigitalOcean API
func (c *doAPIClientImpl) makeAPIRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	c.apiCalls++
	c.lastAPICall = time.Now()

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "motion-index-fiber/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	return resp, nil
}

// parseAPIResponse parses an API response into the target structure
func (c *doAPIClientImpl) parseAPIResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			return fmt.Errorf("API error %d: %v", resp.StatusCode, errorResp)
		}
		return fmt.Errorf("API error %d: %s", resp.StatusCode, resp.Status)
	}

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// CDN Management Implementation

func (c *doAPIClientImpl) CreateCDN(ctx context.Context, origin string, ttl int) (*CDNInfo, error) {
	if origin == "" {
		return nil, fmt.Errorf("origin cannot be empty")
	}

	requestBody := map[string]interface{}{
		"type":   "cdn",
		"config": map[string]interface{}{
			"origin": origin,
			"ttl":    ttl,
		},
	}

	resp, err := c.makeAPIRequest(ctx, "POST", "/cdn/endpoints", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create CDN: %w", err)
	}

	var result struct {
		Endpoint struct {
			ID       string    `json:"id"`
			Origin   string    `json:"origin"`
			Endpoint string    `json:"endpoint"`
			TTL      int       `json:"ttl"`
			Created  time.Time `json:"created_at"`
		} `json:"endpoint"`
	}

	if err := c.parseAPIResponse(resp, &result); err != nil {
		return nil, err
	}

	cdnInfo := &CDNInfo{
		ID:        result.Endpoint.ID,
		Origin:    result.Endpoint.Origin,
		Endpoint:  result.Endpoint.Endpoint,
		TTL:       result.Endpoint.TTL,
		CreatedAt: result.Endpoint.Created,
	}

	// Update cache
	c.cdnCache[cdnInfo.ID] = cdnInfo

	return cdnInfo, nil
}

func (c *doAPIClientImpl) GetCDN(ctx context.Context, cdnID string) (*CDNInfo, error) {
	if cdnID == "" {
		return nil, fmt.Errorf("CDN ID cannot be empty")
	}

	// Check cache first
	if cdn, exists := c.getCDNFromCache(cdnID); exists {
		return cdn, nil
	}

	resp, err := c.makeAPIRequest(ctx, "GET", "/cdn/endpoints/"+cdnID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get CDN: %w", err)
	}

	var result struct {
		Endpoint struct {
			ID       string    `json:"id"`
			Origin   string    `json:"origin"`
			Endpoint string    `json:"endpoint"`
			TTL      int       `json:"ttl"`
			Created  time.Time `json:"created_at"`
		} `json:"endpoint"`
	}

	if err := c.parseAPIResponse(resp, &result); err != nil {
		return nil, err
	}

	cdnInfo := &CDNInfo{
		ID:        result.Endpoint.ID,
		Origin:    result.Endpoint.Origin,
		Endpoint:  result.Endpoint.Endpoint,
		TTL:       result.Endpoint.TTL,
		CreatedAt: result.Endpoint.Created,
	}

	// Update cache
	c.cdnCache[cdnInfo.ID] = cdnInfo

	return cdnInfo, nil
}

func (c *doAPIClientImpl) ListCDNs(ctx context.Context) ([]*CDNInfo, error) {
	resp, err := c.makeAPIRequest(ctx, "GET", "/cdn/endpoints", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list CDNs: %w", err)
	}

	var result struct {
		Endpoints []struct {
			ID       string    `json:"id"`
			Origin   string    `json:"origin"`
			Endpoint string    `json:"endpoint"`
			TTL      int       `json:"ttl"`
			Created  time.Time `json:"created_at"`
		} `json:"endpoints"`
		Links struct {
			Pages struct {
				Next string `json:"next"`
			} `json:"pages"`
		} `json:"links"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}

	if err := c.parseAPIResponse(resp, &result); err != nil {
		return nil, err
	}

	cdns := make([]*CDNInfo, len(result.Endpoints))
	for i, ep := range result.Endpoints {
		cdnInfo := &CDNInfo{
			ID:        ep.ID,
			Origin:    ep.Origin,
			Endpoint:  ep.Endpoint,
			TTL:       ep.TTL,
			CreatedAt: ep.Created,
		}
		cdns[i] = cdnInfo
		
		// Update cache
		c.cdnCache[cdnInfo.ID] = cdnInfo
	}

	return cdns, nil
}

func (c *doAPIClientImpl) DeleteCDN(ctx context.Context, cdnID string) error {
	if cdnID == "" {
		return fmt.Errorf("CDN ID cannot be empty")
	}

	resp, err := c.makeAPIRequest(ctx, "DELETE", "/cdn/endpoints/"+cdnID, nil)
	if err != nil {
		return fmt.Errorf("failed to delete CDN: %w", err)
	}

	if err := c.parseAPIResponse(resp, nil); err != nil {
		return err
	}

	// Remove from cache
	delete(c.cdnCache, cdnID)

	return nil
}

func (c *doAPIClientImpl) FlushCDNCache(ctx context.Context, cdnID string, files []string) error {
	if cdnID == "" {
		return fmt.Errorf("CDN ID cannot be empty")
	}
	if len(files) == 0 {
		return fmt.Errorf("files list cannot be empty")
	}
	if len(files) > 50 {
		return fmt.Errorf("cannot flush more than 50 files at once")
	}

	requestBody := map[string]interface{}{
		"files": files,
	}

	resp, err := c.makeAPIRequest(ctx, "DELETE", "/cdn/endpoints/"+cdnID+"/cache", requestBody)
	if err != nil {
		return fmt.Errorf("failed to flush CDN cache: %w", err)
	}

	return c.parseAPIResponse(resp, nil)
}

// Access Key Management Implementation

func (c *doAPIClientImpl) CreateSpacesKey(ctx context.Context, name string) (*SpacesKey, error) {
	if name == "" {
		return nil, fmt.Errorf("key name cannot be empty")
	}

	requestBody := map[string]interface{}{
		"name": name,
	}

	resp, err := c.makeAPIRequest(ctx, "POST", "/spaces/keys", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create Spaces key: %w", err)
	}

	var result struct {
		AccessKey struct {
			Name      string    `json:"name"`
			AccessKey string    `json:"access_key_id"`
			SecretKey string    `json:"secret_access_key"`
			CreatedAt time.Time `json:"created_at"`
		} `json:"access_key"`
	}

	if err := c.parseAPIResponse(resp, &result); err != nil {
		return nil, err
	}

	return &SpacesKey{
		Name:      result.AccessKey.Name,
		AccessKey: result.AccessKey.AccessKey,
		SecretKey: result.AccessKey.SecretKey,
		CreatedAt: result.AccessKey.CreatedAt,
		Grants:    []*KeyGrant{}, // DigitalOcean Spaces keys have full access
	}, nil
}

func (c *doAPIClientImpl) GetSpacesKey(ctx context.Context, accessKey string) (*SpacesKey, error) {
	if accessKey == "" {
		return nil, fmt.Errorf("access key cannot be empty")
	}

	resp, err := c.makeAPIRequest(ctx, "GET", "/spaces/keys/"+accessKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Spaces key: %w", err)
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

func (c *doAPIClientImpl) ListSpacesKeys(ctx context.Context) ([]*SpacesKey, error) {
	resp, err := c.makeAPIRequest(ctx, "GET", "/spaces/keys", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list Spaces keys: %w", err)
	}

	var result struct {
		AccessKeys []struct {
			Name      string    `json:"name"`
			AccessKey string    `json:"access_key_id"`
			CreatedAt time.Time `json:"created_at"`
		} `json:"access_keys"`
	}

	if err := c.parseAPIResponse(resp, &result); err != nil {
		return nil, err
	}

	keys := make([]*SpacesKey, len(result.AccessKeys))
	for i, key := range result.AccessKeys {
		keys[i] = &SpacesKey{
			Name:      key.Name,
			AccessKey: key.AccessKey,
			CreatedAt: key.CreatedAt,
			Grants:    []*KeyGrant{}, // DigitalOcean Spaces keys have full access
		}
	}

	return keys, nil
}

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

// Helper methods

// getCDNFromCache retrieves a CDN from cache by ID
func (c *doAPIClientImpl) getCDNFromCache(cdnID string) (*CDNInfo, bool) {
	cdn, exists := c.cdnCache[cdnID]
	return cdn, exists
}

// GetMetrics returns performance metrics for the API client
func (c *doAPIClientImpl) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"api_calls":       c.apiCalls,
		"last_api_call":   c.lastAPICall,
		"cached_cdns":     len(c.cdnCache),
		"base_url":        c.baseURL,
		"client_timeout":  c.httpClient.Timeout.Seconds(),
	}
}