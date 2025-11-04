package extractor

import (
	"bytes"
	"log"
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
	start int
	end   int
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
		startIdx := bytes.Index(content[offset:], streamStart)
		if startIdx == -1 {
			break
		}
		startIdx += offset

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

		boundaries = append(boundaries, streamBoundary{
			start: startIdx,
			end:   endIdx,
		})

		offset = endIdx + len(streamEnd)
	}

	return boundaries
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
