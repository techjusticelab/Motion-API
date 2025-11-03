package document

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	domainerrors "motion-index-fiber/internal/domain/errors"
)

func TestNewDocumentID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		id, err := NewDocumentID("doc_123")
		assert.NoError(t, err)
		assert.Equal(t, "doc_123", id.String())
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := NewDocumentID("")
		assert.ErrorIs(t, err, domainerrors.ErrEmptyDocumentID)
	})
}

func TestDocumentIDEquals(t *testing.T) {
	first, _ := NewDocumentID("doc_1")
	second, _ := NewDocumentID("doc_1")
	third, _ := NewDocumentID("doc_2")

	assert.True(t, first.Equals(second))
	assert.False(t, first.Equals(third))
}

func TestFilePath(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		path, err := NewFilePath("documents/2024/05/file.pdf")
		assert.NoError(t, err)
		assert.Equal(t, "documents/2024/05/file.pdf", path.String())
		assert.False(t, path.IsEmpty())
	})

	t.Run("empty path", func(t *testing.T) {
		_, err := NewFilePath("")
		assert.ErrorIs(t, err, domainerrors.ErrEmptyFilePath)
	})

	t.Run("missing prefix", func(t *testing.T) {
		_, err := NewFilePath("uploads/file.pdf")
		assert.ErrorIs(t, err, domainerrors.ErrInvalidFilePath)
	})

	t.Run("double slash", func(t *testing.T) {
		_, err := NewFilePath("documents//file.pdf")
		assert.ErrorIs(t, err, domainerrors.ErrInvalidFilePath)
	})

	t.Run("path traversal", func(t *testing.T) {
		_, err := NewFilePath("documents/../file.pdf")
		assert.ErrorIs(t, err, domainerrors.ErrInvalidFilePath)
	})
}

func TestFileName(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		name, err := NewFileName("document.pdf")
		assert.NoError(t, err)
		assert.Equal(t, "document.pdf", name.String())
		assert.Equal(t, ".pdf", name.Extension())
	})

	t.Run("empty", func(t *testing.T) {
		_, err := NewFileName("  ")
		assert.ErrorIs(t, err, domainerrors.ErrEmptyFileName)
	})

	t.Run("invalid characters", func(t *testing.T) {
		_, err := NewFileName("../document.pdf")
		assert.ErrorIs(t, err, domainerrors.ErrInvalidFileName)
	})

	t.Run("unsupported extension", func(t *testing.T) {
		_, err := NewFileName("document.exe")
		assert.ErrorIs(t, err, domainerrors.ErrInvalidFileName)
	})

	t.Run("missing extension", func(t *testing.T) {
		_, err := NewFileName("document")
		assert.ErrorIs(t, err, domainerrors.ErrInvalidFileName)
	})

	t.Run("exceeds max length", func(t *testing.T) {
		longName := strings.Repeat("a", 256) + ".pdf"
		_, err := NewFileName(longName)
		assert.Error(t, err)
	})
}

func TestContentType(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ct, err := NewContentType("Application/PDF")
		assert.NoError(t, err)
		assert.Equal(t, "application/pdf", ct.String())
		assert.True(t, ct.Is("application/pdf"))
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := NewContentType("image/png")
		assert.ErrorIs(t, err, domainerrors.ErrInvalidContentType)
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := NewContentType("  ")
		assert.ErrorIs(t, err, domainerrors.ErrInvalidContentType)
	})
}

func TestFileSize(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		size, err := NewFileSize(1024)
		assert.NoError(t, err)
		assert.Equal(t, int64(1024), size.Bytes())
		assert.InDelta(t, 1.0, size.Kilobytes(), 0.001)
		assert.False(t, size.IsZero())
	})

	t.Run("zero or negative", func(t *testing.T) {
		_, err := NewFileSize(0)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidFileSize)
	})

	t.Run("exceeds max", func(t *testing.T) {
		_, err := NewFileSize(MaxFileSizeBytes + 1)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidFileSize)
	})

	t.Run("zero value helper", func(t *testing.T) {
		var size FileSize
		assert.True(t, size.IsZero())
	})
}

func TestS3URI(t *testing.T) {
	t.Run("NewS3URI success", func(t *testing.T) {
		uri, err := NewS3URI("bucket", "key/path.pdf")
		assert.NoError(t, err)
		assert.Equal(t, "bucket", uri.Bucket())
		assert.Equal(t, "key/path.pdf", uri.Key())
		assert.Equal(t, "s3://bucket/key/path.pdf", uri.String())
	})

	t.Run("NewS3URI invalid", func(t *testing.T) {
		_, err := NewS3URI("", "key")
		assert.ErrorIs(t, err, domainerrors.ErrInvalidS3URI)
	})

	t.Run("ParseS3URI success", func(t *testing.T) {
		uri, err := ParseS3URI("s3://bucket/key.pdf")
		assert.NoError(t, err)
		assert.Equal(t, "bucket", uri.Bucket())
		assert.Equal(t, "key.pdf", uri.Key())
	})

	t.Run("ParseS3URI invalid prefix", func(t *testing.T) {
		_, err := ParseS3URI("https://bucket/key.pdf")
		assert.ErrorIs(t, err, domainerrors.ErrInvalidS3URI)
	})

	t.Run("ParseS3URI missing key", func(t *testing.T) {
		_, err := ParseS3URI("s3://bucket")
		assert.ErrorIs(t, err, domainerrors.ErrInvalidS3URI)
	})
}

func TestHash(t *testing.T) {
	t.Run("SHA256 success", func(t *testing.T) {
		hashValue := "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3"
		hash, err := NewHash(hashValue, "SHA256")
		assert.NoError(t, err)
		assert.Equal(t, hashValue, hash.String())
		assert.Equal(t, "SHA256", hash.Algorithm())
		assert.False(t, hash.IsZero())
	})

	t.Run("MD5 success", func(t *testing.T) {
		hash, err := NewHash("a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7", "MD5")
		assert.NoError(t, err)
		assert.Equal(t, "MD5", hash.Algorithm())
	})

	t.Run("empty hash", func(t *testing.T) {
		_, err := NewHash("", "SHA256")
		assert.ErrorIs(t, err, domainerrors.ErrEmptyHash)
	})

	t.Run("invalid hash length", func(t *testing.T) {
		_, err := NewHash("abc", "SHA256")
		assert.ErrorIs(t, err, domainerrors.ErrInvalidHash)
	})

	t.Run("invalid algorithm", func(t *testing.T) {
		_, err := NewHash("abc", "")
		assert.ErrorIs(t, err, domainerrors.ErrInvalidHash)
	})

	t.Run("custom algorithm default branch", func(t *testing.T) {
		hash, err := NewHash("custom", "SHA1")
		assert.NoError(t, err)
		assert.Equal(t, "SHA1", hash.Algorithm())
	})

	t.Run("zero value helper", func(t *testing.T) {
		var hash Hash
		assert.True(t, hash.IsZero())
	})
}

func TestText(t *testing.T) {
	t.Run("trimming and helpers", func(t *testing.T) {
		text := NewText("  Hello legal world  ")
		assert.Equal(t, "Hello legal world", text.String())
		assert.False(t, text.IsEmpty())

		estimated := text.WordEstimate()
		assert.Equal(t, 3, estimated.Value())
	})

	t.Run("empty text", func(t *testing.T) {
		text := NewText("   ")
		assert.True(t, text.IsEmpty())
		assert.True(t, text.WordEstimate().IsZero())
	})
}

func TestPageCount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		pages, err := NewPageCount(12)
		assert.NoError(t, err)
		assert.Equal(t, 12, pages.Value())
		assert.False(t, pages.IsZero())
	})

	t.Run("zero", func(t *testing.T) {
		pages, err := NewPageCount(0)
		assert.NoError(t, err)
		assert.True(t, pages.IsZero())
	})

	t.Run("negative", func(t *testing.T) {
		_, err := NewPageCount(-1)
		var validationErr *domainerrors.ValidationError
		assert.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "pageCount", validationErr.Field)
	})
}

func TestWordCount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		words, err := NewWordCount(120)
		assert.NoError(t, err)
		assert.Equal(t, 120, words.Value())
		assert.False(t, words.IsZero())
	})

	t.Run("zero", func(t *testing.T) {
		words, err := NewWordCount(0)
		assert.NoError(t, err)
		assert.True(t, words.IsZero())
	})

	t.Run("negative", func(t *testing.T) {
		_, err := NewWordCount(-5)
		var validationErr *domainerrors.ValidationError
		assert.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "wordCount", validationErr.Field)
	})
}

func TestConfidence(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		confidence, err := NewConfidence(0.75)
		assert.NoError(t, err)
		assert.InDelta(t, 0.75, confidence.Value(), 0.0001)
		assert.True(t, confidence.IsMedium())
		assert.True(t, confidence.IsValid())
	})

	t.Run("high confidence", func(t *testing.T) {
		confidence, err := NewConfidence(0.85)
		assert.NoError(t, err)
		assert.True(t, confidence.IsHigh())
		assert.False(t, confidence.IsLow())
	})

	t.Run("low confidence", func(t *testing.T) {
		confidence, err := NewConfidence(0.25)
		assert.NoError(t, err)
		assert.True(t, confidence.IsLow())
		assert.False(t, confidence.IsHigh())
	})

	t.Run("below range", func(t *testing.T) {
		_, err := NewConfidence(-0.1)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidConfidence)
	})

	t.Run("above range", func(t *testing.T) {
		_, err := NewConfidence(1.1)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidConfidence)
	})
}

func TestSanitizeTags(t *testing.T) {
	assert.Nil(t, sanitizeTags(nil))
	assert.Nil(t, sanitizeTags([]string{"   "}))

	result := sanitizeTags([]string{" Motion ", "motion", "Order"})
	assert.ElementsMatch(t, []string{"Motion", "Order"}, result)
}
