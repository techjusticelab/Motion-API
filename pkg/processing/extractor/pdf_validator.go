package extractor

import (
	"log"
	"strings"
	"unicode"
)

// PDFValidator contains heuristics for validating extracted PDF text and raw content.
type PDFValidator struct{}

// NewPDFValidator creates a new PDFValidator instance.
func NewPDFValidator() *PDFValidator {
	return &PDFValidator{}
}

// IsGarbageText determines whether the provided text is likely unusable garbage.
func (v *PDFValidator) IsGarbageText(text string) bool {
	if text == "" {
		return true
	}

	if len(text) < 100 {
		return false
	}

	lower := strings.ToLower(text)
	markers := []string{"%pdf-", "%%eof", "endobj\n", "startxref\n", "/flatedecode"}
	markerCount := 0
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			markerCount++
		}
	}

	if markerCount >= 3 {
		log.Printf("[PDF-EXTRACT] Garbage detected: found %d control sequences", markerCount)
		return true
	}

	printable := 0
	useful := 0
	for _, r := range text {
		if unicode.IsPrint(r) {
			printable++
			if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
				useful++
			}
		}
	}

	if printable == 0 {
		return true
	}

	ratio := float64(useful) / float64(printable)
	if ratio < 0.1 {
		log.Printf("[PDF-EXTRACT] Garbage detected: only %.1f%% useful characters", ratio*100)
		return true
	}

	return false
}

// IsLikelyGarbagePDF performs a quick check on raw PDF bytes to determine if they are likely garbage.
func (v *PDFValidator) IsLikelyGarbagePDF(content []byte) bool {
	sampleSize := 50 * 1024
	if len(content) < sampleSize {
		sampleSize = len(content)
	}
	sample := content[:sampleSize]

	sampleStr := strings.ToLower(string(sample))

	controlMarkers := []string{
		"/asciihexdecode",
		"/ascii85decode",
		"/ccittfaxdecode",
	}

	controlCount := 0
	for _, marker := range controlMarkers {
		controlCount += strings.Count(sampleStr, marker)
	}

	threshold := 200
	if controlCount > threshold {
		log.Printf("[PDF-EXTRACT] Early garbage detection: found %d exotic encoding markers (threshold: %d)", controlCount, threshold)
		return true
	}

	return false
}

// IsPotentialTextBytes checks if a byte slice might contain readable text.
func (v *PDFValidator) IsPotentialTextBytes(lineBytes []byte) bool {
	if len(lineBytes) == 0 {
		return false
	}

	if len(lineBytes) < 2 || len(lineBytes) > 5000 {
		return false
	}

	printableCount := 0
	for _, b := range lineBytes {
		if (b >= 32 && b <= 126) || b == '\t' || b == '\r' {
			printableCount++
		}
	}

	return float64(printableCount)/float64(len(lineBytes)) > 0.7
}

// IsPotentialTextLine checks if a string line is likely to contain meaningful text.
func (v *PDFValidator) IsPotentialTextLine(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return false
	}

	printableCount := 0
	for _, r := range line {
		if unicode.IsPrint(r) {
			printableCount++
		}
	}

	return float64(printableCount)/float64(len(line)) > 0.7
}
