package extractor

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// docExtractor handles legacy Microsoft Word documents (DOC) and RTF content
type docExtractor struct{}

// NewDOCExtractor creates a new DOC extractor instance
func NewDOCExtractor() Extractor {
	return &docExtractor{}
}

// Extract extracts text from DOC/RTF documents
func (e *docExtractor) Extract(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (*ExtractionResult, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, NewExtractionError("doc", "failed to read DOC document", err)
	}

	text, err := e.extractText(content)
	if err != nil {
		return nil, NewExtractionError("doc", "failed to extract text from DOC document", err)
	}

	cleaner := NewTextCleaner(DefaultCleaningConfig())
	text = cleaner.CleanText(text)

	wordCount := countWords(text)
	charCount := len(text)

	metadataMap := map[string]interface{}{
		"format":     "doc",
		"file_size":  len(content),
		"is_rtf":     bytes.HasPrefix(bytes.TrimSpace(content), []byte("{\\rtf")),
		"char_count": charCount,
	}

	return &ExtractionResult{
		Text:      text,
		WordCount: wordCount,
		CharCount: charCount,
		PageCount: 0,
		Success:   true,
		Metadata:  metadataMap,
	}, nil
}

// SupportedFormats returns the formats handled by this extractor
func (e *docExtractor) SupportedFormats() []string {
	return []string{"doc", "rtf"}
}

// CanExtract checks if the extractor supports the given format
func (e *docExtractor) CanExtract(format string) bool {
	format = strings.ToLower(format)
	for _, supported := range e.SupportedFormats() {
		if format == supported {
			return true
		}
	}
	return false
}

func (e *docExtractor) extractText(content []byte) (string, error) {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return "", nil
	}

	if bytes.HasPrefix(trimmed, []byte("{\\rtf")) {
		return extractTextFromRTF(trimmed), nil
	}

	text := decodeUTF16LE(content)
	if text != "" {
		return text, nil
	}

	if utf8.Valid(content) {
		return string(content), nil
	}

	return "", errors.New("unsupported DOC encoding")
}

func extractTextFromRTF(content []byte) string {
	var builder strings.Builder
	length := len(content)

	for i := 0; i < length; {
		if content[i] == '\\' {
			i++
			if i >= length {
				break
			}

			switch content[i] {
			case '\\', '{', '}':
				builder.WriteByte(content[i])
				i++
			case '\'':
				i++
				if i+1 < length {
					hexCode := content[i : i+2]
					i += 2
					if r, err := decodeHexToRune(hexCode); err == nil {
						builder.WriteRune(r)
					}
				}
			case '~':
				builder.WriteRune(' ')
				i++
			case '-':
				builder.WriteRune('-')
				i++
			case 'u':
				i++
				start := i
				for i < length && (content[i] == '-' || (content[i] >= '0' && content[i] <= '9')) {
					i++
				}
				code := string(content[start:i])
				if val, err := parseUnicodeValue(code); err == nil {
					builder.WriteRune(val)
				}
				if i < length && content[i] == '?' {
					i++
				}
			case 'p':
				if hasControlWord(content[i:], "par") || hasControlWord(content[i:], "page") {
					builder.WriteRune('\n')
				}
				i = skipControlWord(content, i)
			default:
				i = skipControlWord(content, i)
			}
		} else if content[i] == '{' || content[i] == '}' {
			i++
		} else {
			builder.WriteByte(content[i])
			i++
		}
	}

	return strings.TrimSpace(builder.String())
}

func decodeUTF16LE(content []byte) string {
	if len(content) < 2 {
		return ""
	}

	if len(content)%2 != 0 {
		content = content[:len(content)-1]
	}

	words := make([]uint16, len(content)/2)
	for i := 0; i < len(words); i++ {
		words[i] = uint16(content[i*2]) | uint16(content[i*2+1])<<8
	}

	runes := utf16.Decode(words)
	var builder strings.Builder
	for _, r := range runes {
		if r == 0 {
			continue
		}
		if r == '\r' {
			builder.WriteRune('\n')
			continue
		}
		if r < 0x20 && r != '\t' && r != '\n' {
			continue
		}
		builder.WriteRune(r)
	}

	return strings.TrimSpace(builder.String())
}

func decodeHexToRune(hexBytes []byte) (rune, error) {
	decoded := make([]byte, hex.DecodedLen(len(hexBytes)))
	_, err := hex.Decode(decoded, hexBytes)
	if err != nil || len(decoded) == 0 {
		return 0, err
	}
	return rune(decoded[0]), nil
}

func parseUnicodeValue(token string) (rune, error) {
	if token == "" {
		return 0, errors.New("empty unicode token")
	}

	sign := 1
	if token[0] == '-' {
		sign = -1
		token = token[1:]
	}

	var value int
	for i := 0; i < len(token); i++ {
		ch := token[i]
		if ch < '0' || ch > '9' {
			return 0, errors.New("invalid unicode token")
		}
		value = value*10 + int(ch-'0')
	}

	return rune(sign * value), nil
}

func hasControlWord(data []byte, word string) bool {
	if len(data) < len(word) {
		return false
	}
	for i := 0; i < len(word); i++ {
		if data[i] != word[i] {
			return false
		}
	}
	return true
}

func skipControlWord(data []byte, idx int) int {
	for idx < len(data) && ((data[idx] >= 'a' && data[idx] <= 'z') || (data[idx] >= 'A' && data[idx] <= 'Z')) {
		idx++
	}
	for idx < len(data) && (data[idx] == '-' || (data[idx] >= '0' && data[idx] <= '9')) {
		idx++
	}
	if idx < len(data) && data[idx] == ' ' {
		idx++
	}
	return idx
}
