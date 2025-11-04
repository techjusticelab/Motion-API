package spaces

func (c *SpacesClient) GetURLWithFallback(path string, forceFallback bool) (url string, usedCDN bool) {
	path = sanitizePath(path)

	// Force fallback if requested
	if forceFallback {
		return c.s3Client.GetPublicURL(c.bucket, path, true), false
	}

	// Try CDN first if available and healthy
	if c.cdnInfo != nil && c.isCDNHealthy() {
		return c.getCDNURL(path), true
	}

	// Fall back to direct Spaces URL
	return c.s3Client.GetPublicURL(c.bucket, path, true), false
}
