package extractor

import (
	"bytes"
	"compress/flate"
	"compress/lzw"
	"compress/zlib"
	"encoding/ascii85"
	"encoding/hex"
	"io"
	"log"
	"regexp"
	"strings"
	"unicode"
)

type PDFStreamProcessor struct {
	textCleaner *PDFTextCleaner
	config      StreamProcessorConfig
}

type StreamProcessorConfig struct {
	MaxStreamSize    int  // Maximum size of a single stream to process
	MaxChunkSize     int  // Maximum size of a chunk
	EnableDebugLog   bool // Enable debug logging
	CleanImmediately bool // Apply cleaning immediately to reduce memory
}

func DefaultStreamProcessorConfig() StreamProcessorConfig {
	return StreamProcessorConfig{
		MaxStreamSize:    10 * 1024 * 1024, // 10MB per stream
		MaxChunkSize:     256 * 1024,       // 256KB per chunk
		EnableDebugLog:   false,
		CleanImmediately: true,
	}
}

func NewPDFStreamProcessor(textCleaner *PDFTextCleaner, config StreamProcessorConfig) *PDFStreamProcessor {
	if textCleaner == nil {
		textCleaner = NewPDFTextCleaner(DefaultPDFTextCleanerConfig())
	}

	return &PDFStreamProcessor{
		textCleaner: textCleaner,
		config:      config,
	}
}

type streamBoundary struct {
	start   int
	end     int
	filters []string
}

func (p *PDFStreamProcessor) ExtractRawTextStreams(content []byte) (string, int) {
	if p.config.EnableDebugLog {
		log.Printf("[PDF-STREAM] Starting raw stream extraction from %d bytes", len(content))
	}

	// Find all stream boundaries using byte operations
	boundaries := p.findPDFStreamBoundaries(content)
	if len(boundaries) == 0 {
		if p.config.EnableDebugLog {
			log.Printf("[PDF-STREAM] No stream boundaries found")
		}
		return "", 0
	}

	if p.config.EnableDebugLog {
		log.Printf("[PDF-STREAM] Found %d stream boundaries", len(boundaries))
	}

	var result strings.Builder
	result.Grow(64 * 1024) // Pre-allocate 64KB

	processedStreams := 0
	for i, boundary := range boundaries {
		// Check size limit
		streamSize := boundary.end - boundary.start
		if streamSize > p.config.MaxStreamSize {
			if p.config.EnableDebugLog {
				log.Printf("[PDF-STREAM] Skipping oversized stream %d: %d bytes", i+1, streamSize)
			}
			continue
		}

		// Extract and process stream content
		streamData := content[boundary.start:boundary.end]
		if len(boundary.filters) > 0 {
			if decoded, ok := p.decodeStreamData(streamData, boundary.filters); ok {
				streamData = decoded
				if p.config.EnableDebugLog {
					log.Printf("[PDF-STREAM] Decoded stream %d with filters %v", i+1, boundary.filters)
				}
			}
		}

		text := p.extractTextFromStreamBytes(streamData)

		if text != "" {
			if p.config.CleanImmediately {
				text = p.textCleaner.BasicStreamCleaning(text)
			}

			if text != "" {
				if result.Len() > 0 {
					result.WriteString("\n\n")
				}
				result.WriteString(text)
				processedStreams++

				// Check accumulated size
				if result.Len() > p.config.MaxChunkSize {
					if p.config.EnableDebugLog {
						log.Printf("[PDF-STREAM] Chunk size limit reached at stream %d", i+1)
					}
					break
				}
			}
		}
	}

	if p.config.EnableDebugLog {
		log.Printf("[PDF-STREAM] Extracted text from %d/%d streams", processedStreams, len(boundaries))
	}

	return result.String(), processedStreams
}

func (p *PDFStreamProcessor) findPDFStreamBoundaries(content []byte) []streamBoundary {
	var boundaries []streamBoundary

	streamStart := []byte("stream")
	streamEnd := []byte("endstream")

	offset := 0
	for offset < len(content) {
		// Find next stream start
		relIdx := bytes.Index(content[offset:], streamStart)
		if relIdx == -1 {
			break
		}
		streamKeywordIdx := offset + relIdx
		startIdx := streamKeywordIdx

		// Move past "stream" keyword
		startIdx += len(streamStart)

		// Skip whitespace after "stream"
		for startIdx < len(content) && (content[startIdx] == '\n' || content[startIdx] == '\r' || content[startIdx] == ' ') {
			startIdx++
		}

		// Find corresponding endstream
		endIdx := bytes.Index(content[startIdx:], streamEnd)
		if endIdx == -1 {
			break
		}
		endIdx += startIdx

		filters := p.extractStreamFilters(content, streamKeywordIdx)

		boundaries = append(boundaries, streamBoundary{
			start:   startIdx,
			end:     endIdx,
			filters: filters,
		})

		offset = endIdx + len(streamEnd)
	}

	return boundaries
}

var (
	filterEntryRegexp = regexp.MustCompile(`/Filter\s*(\[[^\]]+\]|/[A-Za-z0-9]+)`)
	filterNameRegexp  = regexp.MustCompile(`/([A-Za-z0-9]+)`)
)

func (p *PDFStreamProcessor) extractStreamFilters(content []byte, streamTokenIndex int) []string {
	if streamTokenIndex <= 0 {
		return nil
	}

	searchEnd := streamTokenIndex
	searchStart := searchEnd - 4096
	if searchStart < 0 {
		searchStart = 0
	}

	segment := content[searchStart:searchEnd]
	lastDictStartRel := bytes.LastIndex(segment, []byte("<<"))
	if lastDictStartRel == -1 {
		return nil
	}

	dict := segment[lastDictStartRel:]
	if end := bytes.Index(dict, []byte(">>")); end != -1 {
		dict = dict[:end+2]
	}

	return parseStreamFilters(dict)
}

func parseStreamFilters(dict []byte) []string {
	if len(dict) == 0 {
		return nil
	}

	matches := filterEntryRegexp.FindSubmatch(dict)
	if len(matches) < 2 {
		return nil
	}

	entry := string(matches[1])
	var filters []string
	if strings.HasPrefix(entry, "[") {
		names := filterNameRegexp.FindAllString(entry, -1)
		for _, name := range names {
			filters = append(filters, strings.TrimPrefix(name, "/"))
		}
	} else {
		filters = append(filters, strings.TrimPrefix(entry, "/"))
	}

	return filters
}

func (p *PDFStreamProcessor) decodeStreamData(data []byte, filters []string) ([]byte, bool) {
	decoded := data
	changed := false

	for _, filter := range filters {
		switch strings.TrimSpace(filter) {
		case "", "None":
			continue
		case "FlateDecode":
			out, err := decodeFlate(decoded)
			if err != nil {
				continue
			}
			decoded = out
			changed = true
		case "ASCII85Decode":
			out, err := decodeASCII85(decoded)
			if err != nil {
				continue
			}
			decoded = out
			changed = true
		case "ASCIIHexDecode":
			out, err := decodeASCIIHex(decoded)
			if err != nil {
				continue
			}
			decoded = out
			changed = true
		case "LZWDecode":
			out, err := decodeLZW(decoded)
			if err != nil {
				continue
			}
			decoded = out
			changed = true
		default:
			continue
		}
	}

	return decoded, changed
}

func decodeFlate(data []byte) ([]byte, error) {
	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err == nil {
		defer zr.Close()
		return io.ReadAll(zr)
	}

	fr := flate.NewReader(bytes.NewReader(data))
	defer fr.Close()
	return io.ReadAll(fr)
}

func decodeASCII85(data []byte) ([]byte, error) {
	decoder := ascii85.NewDecoder(bytes.NewReader(data))
	return io.ReadAll(decoder)
}

func decodeASCIIHex(data []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, nil
	}

	if idx := bytes.IndexByte(trimmed, '>'); idx != -1 {
		trimmed = trimmed[:idx]
	}

	buf := make([]byte, 0, len(trimmed)/2)
	for _, b := range trimmed {
		switch b {
		case ' ', '\n', '\r', '\t', '\f', '\v':
			continue
		}
		buf = append(buf, b)
	}

	if len(buf)%2 == 1 {
		buf = append(buf, '0')
	}

	decoded := make([]byte, hex.DecodedLen(len(buf)))
	n, err := hex.Decode(decoded, buf)
	if err != nil {
		return nil, err
	}
	return decoded[:n], nil
}

func decodeLZW(data []byte) ([]byte, error) {
	reader := lzw.NewReader(bytes.NewReader(data), lzw.LSB, 8)
	defer reader.Close()
	return io.ReadAll(reader)
}

func (p *PDFStreamProcessor) extractTextFromStreamBytes(streamData []byte) string {
	if len(streamData) == 0 {
		return ""
	}

	var text strings.Builder
	text.Grow(len(streamData) / 4) // Pre-allocate conservatively

	// Pattern 1: Text in parentheses (text) Tj
	i := 0
	for i < len(streamData) {
		// Look for opening parenthesis
		if streamData[i] == '(' {
			// Find matching closing parenthesis
			j := i + 1
			parenDepth := 1
			escaped := false

			for j < len(streamData) && parenDepth > 0 {
				if !escaped {
					switch streamData[j] {
					case '\\':
						escaped = true
					case '(':
						parenDepth++
					case ')':
						parenDepth--
					}
				} else {
					escaped = false
				}
				j++
			}

			// Extract text between parentheses
			if parenDepth == 0 && j > i+1 {
				textContent := streamData[i+1 : j-1]
				cleaned := p.cleanStreamBytes(textContent)
				cleaned = p.textCleaner.CleanExtractedText(cleaned)
				if cleaned != "" {
					if text.Len() > 0 {
						text.WriteString(" ")
					}
					text.WriteString(cleaned)
				}
			}
			i = j
		} else {
			i++
		}
	}

	// Pattern 2: Text arrays [...] TJ
	if text.Len() == 0 {
		tjPattern := []byte("] TJ")
		if idx := bytes.Index(streamData, tjPattern); idx != -1 {
			// Find the opening bracket
			startIdx := idx
			for startIdx > 0 && streamData[startIdx] != '[' {
				startIdx--
			}

			if startIdx >= 0 && streamData[startIdx] == '[' {
				arrayData := streamData[startIdx+1 : idx]
				text.WriteString(p.extractFromTextArray(arrayData))
			}
		}
	}

	return text.String()
}

func (p *PDFStreamProcessor) extractFromTextArray(arrayData []byte) string {
	var text strings.Builder
	text.Grow(len(arrayData) / 2)

	i := 0
	for i < len(arrayData) {
		// Skip whitespace
		for i < len(arrayData) && (arrayData[i] == ' ' || arrayData[i] == '\n' || arrayData[i] == '\r') {
			i++
		}

		if i >= len(arrayData) {
			break
		}

		// Look for text in parentheses
		if arrayData[i] == '(' {
			j := i + 1
			parenDepth := 1
			escaped := false

			for j < len(arrayData) && parenDepth > 0 {
				if !escaped {
					switch arrayData[j] {
					case '\\':
						escaped = true
					case '(':
						parenDepth++
					case ')':
						parenDepth--
					}
				} else {
					escaped = false
				}
				j++
			}

			if parenDepth == 0 && j > i+1 {
				textContent := arrayData[i+1 : j-1]
				if len(textContent) > 0 {
					cleaned := p.cleanStreamBytes(textContent)
					cleaned = p.textCleaner.CleanExtractedText(cleaned)
					if cleaned != "" {
						text.WriteString(cleaned)
					}
				}
				i = j
			} else {
				i++
			}
		} else {
			// Skip non-text elements
			for i < len(arrayData) && arrayData[i] != '(' && arrayData[i] != ' ' && arrayData[i] != '\n' {
				i++
			}
		}
	}

	return text.String()
}

func (p *PDFStreamProcessor) cleanStreamBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// Remove null bytes and control characters
	cleaned := make([]byte, 0, len(data))
	for _, b := range data {
		if b >= 32 && b < 127 || b == '\n' || b == '\r' || b == '\t' {
			cleaned = append(cleaned, b)
		} else if b >= 128 && unicode.IsPrint(rune(b)) {
			cleaned = append(cleaned, b)
		}
	}

	text := string(cleaned)

	// Handle escaped characters
	text = strings.ReplaceAll(text, "\\n", "\n")
	text = strings.ReplaceAll(text, "\\r", "\r")
	text = strings.ReplaceAll(text, "\\t", "\t")
	text = strings.ReplaceAll(text, "\\(", "(")
	text = strings.ReplaceAll(text, "\\)", ")")
	text = strings.ReplaceAll(text, "\\\\", "\\")

	// Normalize spaces
	text = strings.TrimSpace(text)

	// Skip very short text
	if len(text) < 2 {
		return ""
	}

	// Check if text has meaningful content
	hasAlpha := false
	for _, ch := range text {
		if unicode.IsLetter(ch) {
			hasAlpha = true
			break
		}
	}

	if !hasAlpha && len(text) < 10 {
		return ""
	}

	return text
}
