package document

import (
	"fmt"
	domainerrors "motion-index-fiber/internal/domain/errors"
	"path/filepath"
	"regexp"
	"strings"
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

type DocumentID struct {
	value string
}

func NewDocumentID(id string) (DocumentID, error) {
	if id == "" {
		return DocumentID{}, domainerrors.ErrEmptyDocumentID
	}
	return DocumentID{value: id}, nil
}

func (d DocumentID) String() string {
	return d.value
}

func (d DocumentID) Equals(other DocumentID) bool {
	return d.value == other.value
}

func (d DocumentID) IsEmpty() bool {
	return d.value == ""
}

type FileName struct {
	value string
}

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

func (f FileName) String() string {
	return f.value
}

func (f FileName) Extension() string {
	return strings.ToLower(filepath.Ext(f.value))
}

func (f FileName) IsEmpty() bool {
	return f.value == ""
}

type FilePath struct {
	value string
}

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

func (f FilePath) String() string {
	return f.value
}

func (f FilePath) IsEmpty() bool {
	return f.value == ""
}

type S3URI struct {
	bucket string
	key    string
}

func NewS3URI(bucket, key string) (S3URI, error) {
	if bucket == "" || key == "" {
		return S3URI{}, domainerrors.ErrInvalidS3URI
	}
	return S3URI{bucket: bucket, key: key}, nil
}

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

func (s S3URI) String() string {
	return fmt.Sprintf("s3://%s/%s", s.bucket, s.key)
}

func (s S3URI) Bucket() string {
	return s.bucket
}

func (s S3URI) Key() string {
	return s.key
}

type ContentType struct {
	value string
}

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

func (c ContentType) String() string {
	return c.value
}

func (c ContentType) Is(other string) bool {
	return c.value == strings.ToLower(strings.TrimSpace(other))
}

type Hash struct {
	value     string
	algorithm string
}

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

func (h Hash) String() string {
	return h.value
}

func (h Hash) Algorithm() string {
	return h.algorithm
}

func (h Hash) IsZero() bool {
	return h.value == ""
}

type FileSize struct {
	value int64
}

func NewFileSize(bytes int64) (FileSize, error) {
	if bytes <= 0 || bytes > MaxFileSizeBytes {
		return FileSize{}, domainerrors.ErrInvalidFileSize
	}
	return FileSize{value: bytes}, nil
}

func (f FileSize) Bytes() int64 {
	return f.value
}

func (f FileSize) Kilobytes() float64 {
	return float64(f.value) / 1024.0
}

func (f FileSize) IsZero() bool {
	return f.value == 0
}

type Confidence float64

func NewConfidence(value float64) (Confidence, error) {
	if value < 0.0 || value > 1.0 {
		return 0, domainerrors.ErrInvalidConfidence
	}
	return Confidence(value), nil
}

func (c Confidence) Value() float64 {
	return float64(c)
}

func (c Confidence) IsHigh() bool {
	return c >= 0.8
}

func (c Confidence) IsMedium() bool {
	return c >= 0.5 && c < 0.8
}

func (c Confidence) IsLow() bool {
	return c < 0.5
}

func (c Confidence) IsValid() bool {
	return c >= 0.0 && c <= 1.0
}

type Text struct {
	value string
}

func NewText(value string) Text {
	return Text{value: strings.TrimSpace(value)}
}

func (t Text) String() string {
	return t.value
}

func (t Text) IsEmpty() bool {
	return t.value == ""
}

func (t Text) WordEstimate() WordCount {
	wc, _ := NewWordCount(len(strings.Fields(t.value)))
	return wc
}

type PageCount struct {
	value int
}

func NewPageCount(value int) (PageCount, error) {
	if value < 0 {
		return PageCount{}, domainerrors.NewValidationError("pageCount", "must be zero or positive", value)
	}
	return PageCount{value: value}, nil
}

func (p PageCount) Value() int {
	return p.value
}

func (p PageCount) IsZero() bool {
	return p.value == 0
}

type WordCount struct {
	value int
}

func NewWordCount(value int) (WordCount, error) {
	if value < 0 {
		return WordCount{}, domainerrors.NewValidationError("wordCount", "must be zero or positive", value)
	}
	return WordCount{value: value}, nil
}

func (w WordCount) Value() int {
	return w.value
}

func (w WordCount) IsZero() bool {
	return w.value == 0
}
