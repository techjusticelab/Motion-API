package document

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	domainerrors "motion-index-fiber/internal/domain/errors"
)

const (
	// MaxFileSizeBytes caps legal document uploads at ~100MB.
	MaxFileSizeBytes  int64 = 100 * 1024 * 1024
	maxFileNameLength       = 255
)

var (
	allowedFileExtensions = map[string]struct{}{
		".pdf":  {},
		".doc":  {},
		".docx": {},
		".txt":  {},
		".rtf":  {},
	}

	allowedContentTypes = map[string]struct{}{
		"application/pdf":    {},
		"application/msword": {},
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": {},
		"text/plain": {},
	}
)

// DocumentID represents a unique document identifier.
type DocumentID struct {
	value string
}

// NewDocumentID creates a validated DocumentID.
func NewDocumentID(id string) (DocumentID, error) {
	if id == "" {
		return DocumentID{}, domainerrors.ErrEmptyDocumentID
	}
	return DocumentID{value: id}, nil
}

// String returns the identifier as a string.
func (d DocumentID) String() string {
	return d.value
}

// Equals compares two DocumentID values for equality.
func (d DocumentID) Equals(other DocumentID) bool {
	return d.value == other.value
}

// IsEmpty reports whether the identifier is empty.
func (d DocumentID) IsEmpty() bool {
	return d.value == ""
}

// FileName represents the base name of a stored document.
type FileName struct {
	value string
}

// NewFileName creates a validated FileName.
func NewFileName(name string) (FileName, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return FileName{}, domainerrors.ErrEmptyFileName
	}

	if len(trimmed) > maxFileNameLength {
		return FileName{}, domainerrors.ErrInvalidFileName
	}

	if strings.ContainsAny(trimmed, `/\`) {
		return FileName{}, domainerrors.ErrInvalidFileName
	}

	ext := strings.ToLower(filepath.Ext(trimmed))
	if ext == "" {
		return FileName{}, domainerrors.ErrInvalidFileName
	}

	if _, ok := allowedFileExtensions[ext]; !ok {
		return FileName{}, domainerrors.ErrInvalidFileName
	}

	return FileName{value: trimmed}, nil
}

// String returns the file name.
func (f FileName) String() string {
	return f.value
}

// Extension returns the file extension (including leading dot).
func (f FileName) Extension() string {
	return strings.ToLower(filepath.Ext(f.value))
}

// IsEmpty reports whether the file name is empty.
func (f FileName) IsEmpty() bool {
	return f.value == ""
}

// FilePath represents a storage path for a document.
type FilePath struct {
	value string
}

// NewFilePath creates a validated FilePath.
func NewFilePath(path string) (FilePath, error) {
	if path == "" {
		return FilePath{}, domainerrors.ErrEmptyFilePath
	}

	if !isValidStoragePath(path) {
		return FilePath{}, domainerrors.ErrInvalidFilePath
	}

	return FilePath{value: path}, nil
}

func isValidStoragePath(path string) bool {
	if !strings.HasPrefix(path, "documents/") {
		return false
	}

	if strings.Contains(path, "//") || strings.Contains(path, "..") {
		return false
	}

	return true
}

// String returns the storage path.
func (f FilePath) String() string {
	return f.value
}

// IsEmpty reports whether the storage path is empty.
func (f FilePath) IsEmpty() bool {
	return f.value == ""
}

// S3URI represents an S3 URI (s3://bucket/key).
type S3URI struct {
	bucket string
	key    string
}

// NewS3URI creates a validated S3URI.
func NewS3URI(bucket, key string) (S3URI, error) {
	if bucket == "" || key == "" {
		return S3URI{}, domainerrors.ErrInvalidS3URI
	}
	return S3URI{bucket: bucket, key: key}, nil
}

// ParseS3URI parses an S3 URI string.
func ParseS3URI(uri string) (S3URI, error) {
	if !strings.HasPrefix(uri, "s3://") {
		return S3URI{}, domainerrors.ErrInvalidS3URI
	}

	parts := strings.SplitN(uri[5:], "/", 2)
	if len(parts) != 2 {
		return S3URI{}, domainerrors.ErrInvalidS3URI
	}

	return NewS3URI(parts[0], parts[1])
}

// String returns the canonical URI string.
func (s S3URI) String() string {
	return fmt.Sprintf("s3://%s/%s", s.bucket, s.key)
}

// Bucket returns the bucket segment.
func (s S3URI) Bucket() string {
	return s.bucket
}

// Key returns the key segment.
func (s S3URI) Key() string {
	return s.key
}

// ContentType represents the MIME type of a document.
type ContentType struct {
	value string
}

// NewContentType creates a validated ContentType.
func NewContentType(value string) (ContentType, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return ContentType{}, domainerrors.ErrInvalidContentType
	}

	if _, ok := allowedContentTypes[normalized]; !ok {
		return ContentType{}, domainerrors.ErrInvalidContentType
	}

	return ContentType{value: normalized}, nil
}

// String returns the MIME type string.
func (c ContentType) String() string {
	return c.value
}

// Is compares the MIME type to another string (case-insensitive).
func (c ContentType) Is(other string) bool {
	return c.value == strings.ToLower(strings.TrimSpace(other))
}

// Hash represents a document content hash.
type Hash struct {
	value     string
	algorithm string
}

// NewHash creates a validated Hash.
func NewHash(value, algorithm string) (Hash, error) {
	if value == "" {
		return Hash{}, domainerrors.ErrEmptyHash
	}

	if !isValidHashFormat(value, algorithm) {
		return Hash{}, domainerrors.ErrInvalidHash
	}

	return Hash{value: value, algorithm: algorithm}, nil
}

func isValidHashFormat(value, algorithm string) bool {
	switch algorithm {
	case "SHA256":
		return len(value) == 64 && isHexString(value)
	case "MD5":
		return len(value) == 32 && isHexString(value)
	case "":
		return false
	default:
		return len(value) > 0
	}
}

func isHexString(value string) bool {
	hexPattern := regexp.MustCompile(`^[0-9a-fA-F]+$`)
	return hexPattern.MatchString(value)
}

// String returns the hash string.
func (h Hash) String() string {
	return h.value
}

// Algorithm returns the algorithm used to generate the hash.
func (h Hash) Algorithm() string {
	return h.algorithm
}

// IsZero reports whether the hash has been initialised.
func (h Hash) IsZero() bool {
	return h.value == ""
}

// FileSize represents the size of a document in bytes.
type FileSize struct {
	value int64
}

// NewFileSize creates a validated FileSize.
func NewFileSize(bytes int64) (FileSize, error) {
	if bytes <= 0 || bytes > MaxFileSizeBytes {
		return FileSize{}, domainerrors.ErrInvalidFileSize
	}
	return FileSize{value: bytes}, nil
}

// Bytes returns the raw byte count.
func (f FileSize) Bytes() int64 {
	return f.value
}

// Kilobytes returns the size expressed in KB.
func (f FileSize) Kilobytes() float64 {
	return float64(f.value) / 1024.0
}

// IsZero reports whether the file size has been initialised.
func (f FileSize) IsZero() bool {
	return f.value == 0
}

// Confidence represents classification confidence (0.0 - 1.0).
type Confidence float64

// NewConfidence creates a validated Confidence.
func NewConfidence(value float64) (Confidence, error) {
	if value < 0.0 || value > 1.0 {
		return 0, domainerrors.ErrInvalidConfidence
	}
	return Confidence(value), nil
}

// Value returns the float value of the confidence.
func (c Confidence) Value() float64 {
	return float64(c)
}

// IsHigh reports whether the confidence is high (>= 0.8).
func (c Confidence) IsHigh() bool {
	return c >= 0.8
}

// IsMedium reports whether the confidence is medium (>= 0.5 and < 0.8).
func (c Confidence) IsMedium() bool {
	return c >= 0.5 && c < 0.8
}

// IsLow reports whether the confidence is low (< 0.5).
func (c Confidence) IsLow() bool {
	return c < 0.5
}

// IsValid reports whether the confidence sits within the valid range.
func (c Confidence) IsValid() bool {
	return c >= 0.0 && c <= 1.0
}

// Text wraps extracted document text and helper behaviours.
type Text struct {
	value string
}

// NewText constructs Text while trimming surrounding whitespace.
func NewText(value string) Text {
	return Text{value: strings.TrimSpace(value)}
}

// String returns the stored text.
func (t Text) String() string {
	return t.value
}

// IsEmpty reports whether the text is empty.
func (t Text) IsEmpty() bool {
	return t.value == ""
}

// WordEstimate returns a derived word count from the text.
func (t Text) WordEstimate() WordCount {
	wc, _ := NewWordCount(len(strings.Fields(t.value)))
	return wc
}

// PageCount represents the number of pages in a document.
type PageCount struct {
	value int
}

// NewPageCount validates and constructs a PageCount.
func NewPageCount(value int) (PageCount, error) {
	if value < 0 {
		return PageCount{}, domainerrors.NewValidationError("pageCount", "must be zero or positive", value)
	}
	return PageCount{value: value}, nil
}

// Value returns the numeric page count.
func (p PageCount) Value() int {
	return p.value
}

// IsZero reports whether there are zero pages.
func (p PageCount) IsZero() bool {
	return p.value == 0
}

// WordCount represents tokenised word counts for metadata.
type WordCount struct {
	value int
}

// NewWordCount validates and constructs a WordCount.
func NewWordCount(value int) (WordCount, error) {
	if value < 0 {
		return WordCount{}, domainerrors.NewValidationError("wordCount", "must be zero or positive", value)
	}
	return WordCount{value: value}, nil
}

// Value returns the number of words.
func (w WordCount) Value() int {
	return w.value
}

// IsZero reports whether there are zero recorded words.
func (w WordCount) IsZero() bool {
	return w.value == 0
}
