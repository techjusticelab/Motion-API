package storage

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func sanitizeFilename(filename string) string {
	// Replace problematic characters with underscores
	reg := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	sanitized := reg.ReplaceAllString(filename, "_")

	// Remove multiple consecutive underscores
	reg = regexp.MustCompile(`_+`)
	sanitized = reg.ReplaceAllString(sanitized, "_")

	// Trim underscores from start and end
	sanitized = strings.Trim(sanitized, "_")

	// Ensure filename is not empty
	if sanitized == "" {
		sanitized = "unnamed"
	}

	return sanitized
}

func getContentTypeFromExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".doc":
		return "application/msword"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

func generateDocumentID(name string) string {
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	clean := sanitizeFilename(name)
	if clean == "" {
		clean = "document"
	}
	return fmt.Sprintf("doc_%s_%s", timestamp, clean)
}
