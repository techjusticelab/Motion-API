package spaces

import "time"

// CDNInfo represents DigitalOcean CDN endpoint information
type CDNInfo struct {
	ID        string    `json:"id"`
	Origin    string    `json:"origin"`
	Endpoint  string    `json:"endpoint"`
	TTL       int       `json:"ttl"`
	CreatedAt time.Time `json:"created_at"`
}

// SpacesKey represents DigitalOcean Spaces access key information
type SpacesKey struct {
	Name      string      `json:"name"`
	AccessKey string      `json:"access_key"`
	SecretKey string      `json:"secret_key,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	Grants    []*KeyGrant `json:"grants"`
}

// KeyGrant represents access permissions for a Spaces key
type KeyGrant struct {
	Permission string `json:"permission"`
	Resource   string `json:"resource"`
}