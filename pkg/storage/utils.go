package storage

import (
	"crypto/md5"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"
)

// GenerateStorageKey generates a unique storage key for a file
func GenerateStorageKey(filename, category string) string {
	// Clean filename and ensure it's safe
	cleanFilename := filepath.Base(filename)
	cleanFilename = strings.ReplaceAll(cleanFilename, " ", "_")
	cleanFilename = strings.ReplaceAll(cleanFilename, "/", "_")

	// Generate timestamp-based key for uniqueness
	timestamp := time.Now().Format("2006/01/02")

	if category != "" {
		return fmt.Sprintf("documents/%s/%s/%s", category, timestamp, cleanFilename)
	}

	return fmt.Sprintf("documents/uncategorized/%s/%s", timestamp, cleanFilename)
}

// GetContentTypeFromFilename determines content type from file extension
func GetContentTypeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	// Use standard mime type detection first
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType
	}

	// Fallback for specific legal document formats
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".doc":
		return "application/msword"
	case ".txt":
		return "text/plain"
	case ".rtf":
		return "application/rtf"
	case ".html", ".htm":
		return "text/html"
	case ".xml":
		return "application/xml"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

// ValidateFileName checks if filename is safe for storage
func ValidateFileName(filename string) error {
	if filename == "" {
		return NewStorageError("invalid_filename", "filename cannot be empty", "", nil)
	}

	// Check for dangerous characters
	dangerousChars := []string{"..", "/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range dangerousChars {
		if strings.Contains(filename, char) {
			return NewStorageError("invalid_filename", fmt.Sprintf("filename contains dangerous character: %s", char), filename, nil)
		}
	}

	// Check length
	if len(filename) > 255 {
		return NewStorageError("invalid_filename", "filename too long (max 255 characters)", filename, nil)
	}

	return nil
}

// ValidateFileSize checks if file size is within limits
func ValidateFileSize(size int64, maxSize int64) error {
	if size <= 0 {
		return NewStorageError("invalid_size", "file size must be greater than 0", "", nil)
	}

	if size > maxSize {
		return NewStorageError("file_too_large", fmt.Sprintf("file size %d bytes exceeds maximum %d bytes", size, maxSize), "", nil)
	}

	return nil
}

// CalculateHash calculates MD5 hash of content
func CalculateHash(content io.Reader) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, content); err != nil {
		return "", NewStorageError("hash_failed", "failed to calculate hash", "", err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// SanitizeStoragePath ensures storage path is safe and normalized
func SanitizeStoragePath(path string) string {
	// Remove leading/trailing slashes
	path = strings.Trim(path, "/")

	// Replace multiple slashes with single slash
	path = strings.ReplaceAll(path, "//", "/")

	// Ensure it starts with documents/ if it's a document path
	if !strings.HasPrefix(path, "documents/") && !strings.HasPrefix(path, "temp/") && !strings.HasPrefix(path, "uploads/") {
		path = "documents/" + path
	}

	return path
}

// GetFileExtension extracts file extension from filename
func GetFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	return strings.ToLower(ext)
}

// IsLegalDocumentType checks if file type is a supported legal document format
func IsLegalDocumentType(contentType string) bool {
	// Remove charset and other parameters from content type
	baseContentType := strings.Split(contentType, ";")[0]
	baseContentType = strings.TrimSpace(baseContentType)

	supportedTypes := map[string]bool{
		"application/pdf": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/msword": true,
		"text/plain":         true,
		"application/rtf":    true,
		"text/html":          true,
		"application/xml":    true,
		"text/xml":           true, // Also support text/xml
	}

	return supportedTypes[baseContentType]
}

// GenerateUniqueKey generates a unique key with timestamp and optional hash
func GenerateUniqueKey(filename string, useHash bool) string {
	timestamp := time.Now().Format("20060102-150405")
	cleanName := filepath.Base(filename)
	cleanName = strings.ReplaceAll(cleanName, " ", "_")

	if useHash {
		// Generate a short hash based on timestamp and filename
		hash := md5.Sum([]byte(timestamp + cleanName))
		shortHash := fmt.Sprintf("%x", hash)[:8]
		return fmt.Sprintf("%s-%s-%s", timestamp, shortHash, cleanName)
	}

	return fmt.Sprintf("%s-%s", timestamp, cleanName)
}

// ParseStorageKey extracts information from a storage key
type StorageKeyInfo struct {
	Category string
	Date     string
	Filename string
	FullPath string
	IsValid  bool
}

// ParseStorageKey parses a storage key and extracts metadata
func ParseStorageKey(key string) *StorageKeyInfo {
	info := &StorageKeyInfo{
		FullPath: key,
		IsValid:  false,
	}

	// Expected format: documents/{category}/{date}/{filename}
	parts := strings.Split(key, "/")
	if len(parts) < 4 || parts[0] != "documents" {
		return info
	}

	info.Category = parts[1]
	info.Date = parts[2]
	info.Filename = strings.Join(parts[3:], "/") // Handle filenames with slashes
	info.IsValid = true

	return info
}

// FileTypeFromExtension maps file extensions to document types
func FileTypeFromExtension(filename string) string {
	ext := GetFileExtension(filename)

	switch ext {
	case ".pdf":
		return "PDF Document"
	case ".docx":
		return "Word Document (DOCX)"
	case ".doc":
		return "Word Document (DOC)"
	case ".txt":
		return "Text Document"
	case ".rtf":
		return "Rich Text Document"
	case ".html", ".htm":
		return "HTML Document"
	case ".xml":
		return "XML Document"
	case ".json":
		return "JSON Document"
	default:
		return "Unknown Document"
	}
}
