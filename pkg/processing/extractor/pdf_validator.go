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
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}

	// Accept short payloads – downstream consumers can decide if they need OCR fallback
	if len([]rune(trimmed)) < 80 {
		return false
	}

	printable := 0
	useful := 0
	letters := 0
	nonASCIIPrintable := 0
	replacementRunes := 0
	uniqueRunes := make(map[rune]struct{}, 64)

	for _, r := range trimmed {
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			printable++
			uniqueRunes[r] = struct{}{}
			if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
				useful++
			}
			if unicode.IsLetter(r) {
				letters++
			}
			if r > unicode.MaxASCII {
				nonASCIIPrintable++
			}
			if r == unicode.ReplacementChar {
				replacementRunes++
			}
		} else if r > unicode.MaxASCII {
			nonASCIIPrintable++
		}
	}

	if printable == 0 {
		return true
	}

	if printable > 0 {
		replacementRatio := float64(replacementRunes) / float64(printable)
		if replacementRatio > 0.02 {
			log.Printf("[PDF-EXTRACT] Garbage detected: %.2f%% replacement characters", replacementRatio*100)
			return true
		}
	}

	usefulRatio := float64(useful) / float64(printable)
	if usefulRatio < 0.25 {
		log.Printf("[PDF-EXTRACT] Garbage detected: useful character ratio %.1f%%", usefulRatio*100)
		return true
	}

	// Evaluate word quality
	words := strings.Fields(trimmed)
	if len(words) <= 5 && len([]rune(trimmed)) > 200 {
		// Very low word counts for long payloads usually indicate broken extraction
		return true
	}

	alphaWords := 0
	for _, w := range words {
		if len(w) <= 1 {
			continue
		}
		if strings.IndexFunc(w, unicode.IsLetter) >= 0 {
			alphaWords++
		}
	}

	if len(words) >= 20 {
		alphaRatio := float64(alphaWords) / float64(len(words))
		if alphaRatio < 0.2 {
			log.Printf("[PDF-EXTRACT] Garbage detected: only %.1f%% alphabetic words", alphaRatio*100)
			return true
		}
	}

	// Highly repetitive character sets (e.g. same rune repeated) suggest corruption
	if len(uniqueRunes) < 5 && len(trimmed) > 200 {
		return true
	}

	// Allow mostly non-ASCII text as long as we saw some letters
	if letters == 0 && nonASCIIPrintable > 0 {
		// If we have non-ASCII glyphs but no letters, treat as garbage
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
