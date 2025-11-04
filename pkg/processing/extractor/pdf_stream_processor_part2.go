package extractor

import (
	"log"
	"regexp"
	"strings"
)

func (p *PDFStreamProcessor) ProcessStreamChunk(streamMatches [][]string, startIndex int) (string, int) {
	var chunkText strings.Builder
	chunkText.Grow(64 * 1024) // Pre-allocate 64KB

	pageCount := 0

	for i, match := range streamMatches {
		if len(match) > 1 {
			streamContent := match[1]

			if p.config.EnableDebugLog {
				log.Printf("[PDF-STREAM] Processing stream %d (length: %d)", startIndex+i+1, len(streamContent))
			}

			// Extract text with immediate cleaning
			text := p.extractTextFromStream(streamContent)

			if text != "" {
				// Apply basic cleaning immediately
				cleanedText := p.textCleaner.BasicStreamCleaning(text)

				if cleanedText != "" {
					if chunkText.Len() > 0 {
						chunkText.WriteString("\n\n")
					}
					chunkText.WriteString(cleanedText)
					pageCount++

					if p.config.EnableDebugLog {
						log.Printf("[PDF-STREAM] Stream %d extracted: %d chars", startIndex+i+1, len(cleanedText))
					}
				}
			}

			// Check chunk size limit
			if chunkText.Len() > p.config.MaxChunkSize {
				if p.config.EnableDebugLog {
					log.Printf("[PDF-STREAM] Chunk size limit reached at stream %d", startIndex+i+1)
				}
				break
			}
		}
	}

	return chunkText.String(), pageCount
}

func (p *PDFStreamProcessor) extractTextFromStream(stream string) string {
	var text strings.Builder

	// Look for text showing commands like (text) Tj, [text] TJ, etc.
	patterns := []string{
		`\((.*?)\)\s*Tj`,        // (text) Tj
		`\[(.*?)\]\s*TJ`,        // [text array] TJ
		`\((.*?)\)\s*'`,         // (text) '
		`\((.*?)\)\s*"`,         // (text) "
		`<([0-9A-Fa-f]+)>\s*Tj`, // <hex> Tj
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(stream, -1)

		for _, match := range matches {
			if len(match) > 1 {
				extractedText := match[1]

				// Handle hex encoded text
				if strings.HasPrefix(pattern, "<") {
					// Convert hex to string
					extractedText = p.decodeHexString(extractedText)
				}

				// Clean escape sequences
				extractedText = strings.ReplaceAll(extractedText, "\\(", "(")
				extractedText = strings.ReplaceAll(extractedText, "\\)", ")")
				extractedText = strings.ReplaceAll(extractedText, "\\\\", "\\")

				extractedText = strings.TrimSpace(extractedText)

				if extractedText != "" {
					extractedText = p.textCleaner.CleanExtractedText(extractedText)
				}

				if extractedText != "" {
					if text.Len() > 0 {
						text.WriteString(" ")
					}
					text.WriteString(extractedText)
				}
			}
		}
	}

	return text.String()
}

func (p *PDFStreamProcessor) decodeHexString(hex string) string {
	// Remove spaces
	hex = strings.ReplaceAll(hex, " ", "")

	// Ensure even number of characters
	if len(hex)%2 != 0 {
		hex += "0"
	}

	result := make([]byte, 0, len(hex)/2)
	for i := 0; i < len(hex); i += 2 {
		if i+1 < len(hex) {
			// Parse hex byte
			var b byte
			for j := 0; j < 2; j++ {
				c := hex[i+j]
				var nibble byte
				if c >= '0' && c <= '9' {
					nibble = c - '0'
				} else if c >= 'a' && c <= 'f' {
					nibble = c - 'a' + 10
				} else if c >= 'A' && c <= 'F' {
					nibble = c - 'A' + 10
				}
				if j == 0 {
					b = nibble << 4
				} else {
					b |= nibble
				}
			}
			result = append(result, b)
		}
	}

	return string(result)
}
