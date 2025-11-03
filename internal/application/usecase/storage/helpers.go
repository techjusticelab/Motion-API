package storage

import (
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"strings"
)

// isDirectory checks if a path represents a directory.
func isDirectory(path string) bool {
	return strings.HasSuffix(path, "/")
}

// isSystemFile checks if a file is a system file that should be excluded.
func isSystemFile(path string) bool {
	filename := filepath.Base(path)
	// Also check the full path for __MACOSX directory
	return strings.Contains(path, "__MACOSX") ||
		strings.Contains(filename, ".DS_Store") ||
		strings.HasSuffix(filename, ".tmp") ||
		strings.HasSuffix(filename, ".log")
}

// matchesFileType checks if a path matches the given file type.
func matchesFileType(path, fileType string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return strings.HasSuffix(ext, strings.ToLower(fileType))
}

// getFileExtension returns the file extension from a path.
func getFileExtension(path string) string {
	return strings.ToLower(filepath.Ext(path))
}

// getFilename returns the filename from a path.
func getFilename(path string) string {
	return filepath.Base(path)
}

// encodeCursor encodes an index into a base64 cursor string.
func encodeCursor(index int) string {
	cursorData := map[string]interface{}{"index": index}
	cursorBytes, err := json.Marshal(cursorData)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(cursorBytes)
}

// decodeCursor decodes a base64 cursor string into an index.
func decodeCursor(cursor string) (int, error) {
	decoded, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}

	var cursorData map[string]interface{}
	if err := json.Unmarshal(decoded, &cursorData); err != nil {
		return 0, err
	}

	if idx, ok := cursorData["index"].(float64); ok {
		return int(idx), nil
	}

	return 0, nil
}

// getContentTypeFromExtension returns the MIME type for a file extension.
func getContentTypeFromExtension(ext string) string {
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
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".html", ".htm":
		return "text/html"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".tiff", ".tif":
		return "image/tiff"
	default:
		return "application/octet-stream"
	}
}
