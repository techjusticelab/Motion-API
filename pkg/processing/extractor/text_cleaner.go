package extractor

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"unicode"
)

// TextCleaner provides comprehensive text cleaning functionality for legal documents
type TextCleaner struct {
	config CleaningConfig
}

// CleaningConfig defines the cleaning behavior
type CleaningConfig struct {
	RemoveFilePathArtifacts    bool
	RemoveHTMLContent          bool
	RemovePrinterArtifacts     bool
	RemoveSequentialNumbers    bool
	RemoveDrivePathReferences  bool
	PreserveLegalStructure     bool
	DebugLogging              bool
}

// DefaultCleaningConfig returns the standard cleaning configuration
func DefaultCleaningConfig() CleaningConfig {
	return CleaningConfig{
		RemoveFilePathArtifacts:    true,
		RemoveHTMLContent:          true,
		RemovePrinterArtifacts:     true,
		RemoveSequentialNumbers:    true,
		RemoveDrivePathReferences:  true,
		PreserveLegalStructure:     true,
		DebugLogging:              true,
	}
}

// NewTextCleaner creates a new text cleaner with the specified configuration
func NewTextCleaner(config CleaningConfig) *TextCleaner {
	return &TextCleaner{config: config}
}

// CleanText performs comprehensive text cleaning based on the configuration
func (tc *TextCleaner) CleanText(text string) string {
	if tc.config.DebugLogging {
		log.Printf("[TEXT-CLEANER] Starting text cleaning: %d chars", len(text))
	}

	originalLength := len(text)

	// Apply cleaning steps in order
	if tc.config.RemoveFilePathArtifacts {
		text = tc.removeFilePathArtifacts(text)
	}

	if tc.config.RemoveHTMLContent {
		text = tc.removeHTMLContent(text)
	}

	if tc.config.RemovePrinterArtifacts {
		text = tc.removePrinterArtifacts(text)
	}

	if tc.config.RemoveSequentialNumbers {
		text = tc.removeSequentialNumbers(text)
	}

	if tc.config.RemoveDrivePathReferences {
		text = tc.removeDrivePathReferences(text)
	}

	// Final cleanup
	text = tc.finalCleanup(text)

	if tc.config.DebugLogging {
		log.Printf("[TEXT-CLEANER] Cleaning complete: %d -> %d chars (%.1f%% reduction)",
			originalLength, len(text), float64(originalLength-len(text))/float64(originalLength)*100)
	}

	return text
}

// removeFilePathArtifacts removes file path and timestamp artifacts
func (tc *TextCleaner) removeFilePathArtifacts(text string) string {
	if tc.config.DebugLogging {
		log.Printf("[TEXT-CLEANER] Removing file path artifacts")
	}

	// Pattern for file path with timestamp: data/data/data/filename.txtWed Apr 30 18:55:26 2025
	filePathPattern := regexp.MustCompile(`(?i)^(?:data/)*[^/\s]+\.(?:txt|pdf|doc|docx|rtf|html?)(?:Mon|Tue|Wed|Thu|Fri|Sat|Sun)\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2}\s+\d{1,2}:\d{2}:\d{2}\s+\d{4}`)

	// Pattern for nested data paths: data/data/data/filename.txt
	nestedDataPattern := regexp.MustCompile(`(?i)^(?:data/){2,}[^/\s]+\.(?:txt|pdf|doc|docx|rtf|html?)`)

	// Pattern for file path at start of text
	fileStartPattern := regexp.MustCompile(`(?i)^[^/\s]*(?:/[^/\s]*)*\.(?:txt|pdf|doc|docx|rtf|html?)\s*`)

	lines := strings.Split(text, "\n")
	var cleanedLines []string
	removedCount := 0

	for i, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)

		// Skip empty lines
		if len(line) == 0 {
			cleanedLines = append(cleanedLines, "")
			continue
		}

		// Check for file path artifacts at the beginning of lines
		if filePathPattern.MatchString(line) ||
			nestedDataPattern.MatchString(line) ||
			(i < 3 && fileStartPattern.MatchString(line)) { // Only check first few lines for simple file paths

			if tc.config.DebugLogging && removedCount < 5 {
				log.Printf("[TEXT-CLEANER] Removing file path artifact: %q", line[:min(80, len(line))])
			}
			removedCount++
			continue
		}

		// Clean inline file path references but keep the rest of the line
		cleanedLine := filePathPattern.ReplaceAllString(line, "")
		cleanedLine = nestedDataPattern.ReplaceAllString(cleanedLine, "")
		cleanedLine = strings.TrimSpace(cleanedLine)

		// Only keep non-empty cleaned lines
		if len(cleanedLine) > 0 {
			cleanedLines = append(cleanedLines, cleanedLine)
		} else if len(originalLine) > 0 && !strings.Contains(originalLine, "data/") {
			// Keep original line if it's not empty and doesn't contain data paths
			cleanedLines = append(cleanedLines, originalLine)
		}
	}

	if tc.config.DebugLogging {
		log.Printf("[TEXT-CLEANER] File path artifact removal: %d artifacts removed", removedCount)
	}

	return strings.Join(cleanedLines, "\n")
}

// removeHTMLContent removes HTML tags and entities
func (tc *TextCleaner) removeHTMLContent(text string) string {
	if tc.config.DebugLogging {
		log.Printf("[TEXT-CLEANER] Removing HTML content")
	}

	// Remove HTML tags
	htmlTagPattern := regexp.MustCompile(`<[^>]*>`)
	text = htmlTagPattern.ReplaceAllString(text, " ")

	// Remove HTML entities
	htmlEntities := map[string]string{
		"&nbsp;":  " ",
		"&amp;":   "&",
		"&lt;":    "<",
		"&gt;":    ">",
		"&quot;":  "\"",
		"&apos;":  "'",
		"&copy;":  "©",
		"&reg;":   "®",
		"&trade;": "™",
		"&mdash;": "—",
		"&ndash;": "–",
		"&hellip;": "...",
	}

	for entity, replacement := range htmlEntities {
		text = strings.ReplaceAll(text, entity, replacement)
	}

	// Remove numeric HTML entities
	numericEntityPattern := regexp.MustCompile(`&#\d+;`)
	text = numericEntityPattern.ReplaceAllString(text, " ")

	// Remove hex HTML entities
	hexEntityPattern := regexp.MustCompile(`&#x[0-9a-fA-F]+;`)
	text = hexEntityPattern.ReplaceAllString(text, " ")

	// Remove common HTML attribute remnants
	attributePattern := regexp.MustCompile(`\b(?:bgcolor|color|style|class|id|width|height|font|size|face)=\w+`)
	text = attributePattern.ReplaceAllString(text, "")

	return text
}

// removePrinterArtifacts removes printer control characters and artifacts
func (tc *TextCleaner) removePrinterArtifacts(text string) string {
	if tc.config.DebugLogging {
		log.Printf("[TEXT-CLEANER] Removing printer artifacts")
	}

	// HP LaserJet specific artifacts
	hpArtifacts := []string{
		"HP LaserJet",
		"HPLASIII.PRS",
		"(HP Roman 8)",
		"(Port)",
		"(FW)",
		"Swiss Roman 11pt",
		"Swiss Bold 11pt",
	}

	for _, artifact := range hpArtifacts {
		text = strings.ReplaceAll(text, artifact, "")
	}

	// Remove complex control sequences - more specific patterns
	controlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\^4<[^>]*>`),                    // ^4<...> patterns
		regexp.MustCompile(`\\[0-9]{4}[a-zA-Z]*`),          // \0808 style sequences
		regexp.MustCompile(`[\\^][<>|,()@'hdxlXtP]{5,}`),   // Long control sequences
		regexp.MustCompile(`\\t'[a-zA-Z@0-9<>]{3,}`),       // \t'pll@8@ style
		regexp.MustCompile(`[\\^@|<>'()]{8,}`),             // Very long sequences
		regexp.MustCompile(`\\x[0-9a-fA-F]{2}`),            // Hex escape sequences
	}

	for _, pattern := range controlPatterns {
		text = pattern.ReplaceAllString(text, " ")
	}

	return text
}

// removeSequentialNumbers removes leading sequential number patterns
func (tc *TextCleaner) removeSequentialNumbers(text string) string {
	if tc.config.DebugLogging {
		log.Printf("[TEXT-CLEANER] Removing sequential numbers")
	}

	lines := strings.Split(text, "\n")
	var cleanedLines []string

	for _, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)

		if len(line) == 0 {
			cleanedLines = append(cleanedLines, "")
			continue
		}

		// Check if line starts with sequential numbers (1 2 3 4 5...)
		if tc.isSequentialNumberLine(line) {
			if tc.config.DebugLogging {
				log.Printf("[TEXT-CLEANER] Removing sequential number line: %q", line[:min(50, len(line))])
			}
			continue
		}

		// Remove sequential numbers from the beginning of the line
		cleanedLine := tc.removeLeadingSequentialNumbers(line)
		if len(strings.TrimSpace(cleanedLine)) > 0 {
			cleanedLines = append(cleanedLines, cleanedLine)
		} else {
			cleanedLines = append(cleanedLines, originalLine)
		}
	}

	return strings.Join(cleanedLines, "\n")
}

// isSequentialNumberLine checks if a line is primarily sequential numbers
func (tc *TextCleaner) isSequentialNumberLine(line string) bool {
	// Look for pattern like "1 2 3 4 5 6 7 8 9 10 11 12 13 14 15..."
	sequentialPattern := regexp.MustCompile(`^\s*(\d+\s+){5,}`)
	if !sequentialPattern.MatchString(line) {
		return false
	}

	// Extract numbers and check if they're sequential
	numberPattern := regexp.MustCompile(`\d+`)
	numbers := numberPattern.FindAllString(line, -1)

	if len(numbers) < 5 {
		return false
	}

	// Check if first 5-10 numbers are sequential
	checkCount := min(10, len(numbers))
	sequential := 0

	for i := 1; i < checkCount; i++ {
		if len(numbers[i-1]) <= 2 && len(numbers[i]) <= 2 { // Only check single/double digit numbers
			prev, _ := parseInt(numbers[i-1])
			curr, _ := parseInt(numbers[i])
			if curr == prev+1 {
				sequential++
			}
		}
	}

	// If most of the checked numbers are sequential, consider it a sequential line
	return float64(sequential)/float64(checkCount-1) > 0.7
}

// removeLeadingSequentialNumbers removes sequential numbers from the start of a line
func (tc *TextCleaner) removeLeadingSequentialNumbers(line string) string {
	// First check if this line actually has sequential numbers
	if !tc.isSequentialNumberLine(line) {
		return line
	}

	// Pattern to match leading sequential numbers but preserve text after
	leadingNumberPattern := regexp.MustCompile(`^(\s*\d+\s+){5,}`)
	cleaned := leadingNumberPattern.ReplaceAllString(line, "")
	
	// If we removed numbers and there's still content, return it
	if len(strings.TrimSpace(cleaned)) > 0 {
		return cleaned
	}
	
	// If the entire line was sequential numbers, return empty
	return ""
}

// removeDrivePathReferences removes Windows drive path references
func (tc *TextCleaner) removeDrivePathReferences(text string) string {
	if tc.config.DebugLogging {
		log.Printf("[TEXT-CLEANER] Removing drive path references")
	}

	// Windows path patterns - more specific to avoid over-matching
	windowsPathPattern := regexp.MustCompile(`[A-Z]:\\[^\\/:*?"<>|\r\n\s]*(?:\\[^\\/:*?"<>|\r\n\s]*)*`)
	text = windowsPathPattern.ReplaceAllString(text, "")

	// UNC path patterns
	uncPathPattern := regexp.MustCompile(`\\\\[^\s\\]+(?:\\[^\s\\]+)*`)
	text = uncPathPattern.ReplaceAllString(text, "")

	// Clean up any remaining excessive backslash sequences
	backslashPattern := regexp.MustCompile(`\\{3,}`)
	text = backslashPattern.ReplaceAllString(text, "")

	return text
}

// finalCleanup performs final text normalization
func (tc *TextCleaner) finalCleanup(text string) string {
	// Normalize whitespace
	multiSpacePattern := regexp.MustCompile(`\s+`)
	text = multiSpacePattern.ReplaceAllString(text, " ")

	// Normalize line breaks
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Remove excessive line breaks (more than 2 consecutive)
	excessiveNewlinePattern := regexp.MustCompile(`\n{3,}`)
	text = excessiveNewlinePattern.ReplaceAllString(text, "\n\n")

	// Remove lines that are only whitespace
	lines := strings.Split(text, "\n")
	var cleanedLines []string

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			cleanedLines = append(cleanedLines, line)
		} else if len(cleanedLines) > 0 && cleanedLines[len(cleanedLines)-1] != "" {
			// Keep one empty line between sections
			cleanedLines = append(cleanedLines, "")
		}
	}

	// Trim leading and trailing whitespace
	text = strings.TrimSpace(strings.Join(cleanedLines, "\n"))

	return text
}

// Helper functions

// parseInt safely parses an integer
func parseInt(s string) (int, error) {
	result := 0
	for _, r := range s {
		if unicode.IsDigit(r) {
			result = result*10 + int(r-'0')
		} else {
			return 0, fmt.Errorf("invalid integer: %s", s)
		}
	}
	return result, nil
}