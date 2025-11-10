package extractor

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"unicode"
)

var (
	whitespaceRe           = regexp.MustCompile(`\s+`)
	horizontalWhitespaceRe = regexp.MustCompile(`[ \t\f]+`)
	newlineCollapseRe      = regexp.MustCompile(`\n{2,}`)
)

type PDFTextCleaner struct {
	textCleaner *TextCleaner
	config      PDFTextCleanerConfig
}

type PDFTextCleanerConfig struct {
	DebugLogging bool
}

func DefaultPDFTextCleanerConfig() PDFTextCleanerConfig {
	return PDFTextCleanerConfig{
		DebugLogging: true,
	}
}

func NewPDFTextCleaner(config PDFTextCleanerConfig) *PDFTextCleaner {
	cleanConfig := DefaultCleaningConfig()
	cleanConfig.DebugLogging = config.DebugLogging

	return &PDFTextCleaner{
		textCleaner: NewTextCleaner(cleanConfig),
		config:      config,
	}
}

func (c *PDFTextCleaner) Clean(text string) string {
	if text == "" {
		return ""
	}

	log.Printf("[PDF-EXTRACT] 🧹 Before enhanced cleaning: %d chars", len(text))
	text = c.textCleaner.CleanText(text)
	log.Printf("[PDF-EXTRACT] 🧹 After enhanced cleaning: %d chars", len(text))

	text = c.removePDFArtifacts(text)
	text = c.finalTextNormalization(text)

	return text
}

func (c *PDFTextCleaner) BasicStreamCleaning(text string) string {
	if text == "" {
		return ""
	}

	text = normalizePDFEscapes(text)
	text = collapseWhitespace(text, true)
	text = filterPrintable(text, false)
	text = strings.TrimSpace(text)
	if len(text) < 3 {
		return ""
	}

	return text
}

func (c *PDFTextCleaner) StreamCleanText(text string) string {
	if text == "" {
		return ""
	}

	text = c.streamNormalizeText(text)
	text = c.streamRemovePDFArtifacts(text)
	text = c.streamFinalCleanup(text)

	return text
}

func (c *PDFTextCleaner) CleanExtractedText(text string) string {
	if text == "" {
		return ""
	}

	text = normalizePDFEscapes(text)
	text = filterPrintable(text, false)
	return strings.TrimSpace(text)
}

func (c *PDFTextCleaner) BasicPageCleaning(pageText string) string {
	if pageText == "" {
		return ""
	}

	lines := strings.Split(pageText, "\n")
	cleanedLines := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if strings.HasPrefix(line, "obj") && strings.HasSuffix(line, "endobj") {
			continue
		}
		if line == "stream" || line == "endstream" {
			continue
		}

		cleanedLines = append(cleanedLines, line)
	}

	return strings.Join(cleanedLines, "\n")
}

func (c *PDFTextCleaner) streamNormalizeText(text string) string {
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

func (c *PDFTextCleaner) streamRemovePDFArtifacts(text string) string {
	lines := strings.Split(text, "\n")
	cleanedLines := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		lineLower := strings.ToLower(line)
		if lineLower == "endobj" || lineLower == "stream" || lineLower == "endstream" {
			continue
		}
		if c.isNumericLine(line) && len(line) < 10 {
			continue
		}
		if c.isRepeatedCharacterLine(line) {
			continue
		}

		cleanedLines = append(cleanedLines, line)
	}

	return strings.Join(cleanedLines, "\n")
}

func (c *PDFTextCleaner) streamFinalCleanup(text string) string {
	text = filterPrintable(text, true)
	return strings.TrimSpace(text)
}

func (c *PDFTextCleaner) finalTextNormalization(text string) string {
	text = collapseWhitespace(text, false)
	text = filterPrintable(text, true)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

func normalizePDFEscapes(text string) string {
	if text == "" {
		return text
	}

	replacer := strings.NewReplacer(
		"\\n", " ",
		"\\r", " ",
		"\\t", " ",
		"\\(", "(",
		"\\)", ")",
		"\\\\", "\\",
	)

	return replacer.Replace(text)
}

func collapseWhitespace(text string, replaceNewlines bool) string {
	if text == "" {
		return text
	}

	if replaceNewlines {
		return whitespaceRe.ReplaceAllString(text, " ")
	}

	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = horizontalWhitespaceRe.ReplaceAllString(text, " ")
	text = newlineCollapseRe.ReplaceAllString(text, "\n")
	return text
}

func filterPrintable(text string, keepNewlines bool) string {
	if text == "" {
		return text
	}

	var builder strings.Builder
	builder.Grow(len(text))

	for _, r := range text {
		if unicode.IsPrint(r) || r == ' ' {
			builder.WriteRune(r)
			continue
		}
		if keepNewlines && (r == '\n' || r == '\t') {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

func (c *PDFTextCleaner) removePDFArtifacts(text string) string {
	artifacts := []string{
		"endobj", "stream", "endstream",
		"xref", "trailer", "startxref",
		"%%EOF", "%%Page:",
	}

	lines := strings.Split(text, "\n")
	cleanedLines := make([]string, 0, len(lines))
	removedCount := 0

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			cleanedLines = append(cleanedLines, line)
			continue
		}

		isArtifact := false
		artifactReason := ""

		lineLower := strings.ToLower(strings.TrimSpace(line))
		for _, artifact := range artifacts {
			artifactLower := strings.ToLower(artifact)
			if lineLower == artifactLower || strings.HasPrefix(lineLower, artifactLower+" ") ||
				(strings.HasPrefix(lineLower, artifactLower) && len(lineLower) <= len(artifactLower)+5) {
				isArtifact = true
				artifactReason = fmt.Sprintf("matches artifact '%s'", artifact)
				break
			}
		}

		if !isArtifact && c.isNumericLine(line) {
			isArtifact = true
			artifactReason = "numeric line"
		}

		if !isArtifact && len(line) < 3 && !c.hasAlphanumeric(line) {
			isArtifact = true
			artifactReason = "short non-alphanumeric"
		}

		if !isArtifact && c.isRepeatedCharacterLine(line) {
			isArtifact = true
			artifactReason = "repeated character pattern"
		}

		if !isArtifact && c.isFormFieldLine(line) {
			isArtifact = true
			artifactReason = "form field line"
		}

		if !isArtifact && c.isHeaderFooterLine(line) {
			isArtifact = true
			artifactReason = "header/footer pattern"
		}

		if isArtifact {
			removedCount++
			if removedCount <= 10 {
				log.Printf("[PDF-ARTIFACTS] Removing line %d (%s): %q", i+1, artifactReason, c.truncate(line, 50))
			}
		} else {
			cleanedLines = append(cleanedLines, c.cleanTableOfContentsArtifacts(line))
		}
	}

	result := strings.Join(cleanedLines, "\n")
	log.Printf("[PDF-ARTIFACTS] Artifact removal complete: %d lines removed, %d lines kept, result length: %d chars",
		removedCount, len(cleanedLines), len(result))
	return result
}

func (c *PDFTextCleaner) truncate(line string, limit int) string {
	if len(line) <= limit {
		return line
	}
	return line[:limit]
}

func (c *PDFTextCleaner) isNumericLine(line string) bool {
	if len(line) == 0 {
		return false
	}

	for _, r := range line {
		if !unicode.IsDigit(r) && r != '.' && r != ',' && r != '-' && r != ' ' {
			return false
		}
	}
	return true
}

func (c *PDFTextCleaner) hasAlphanumeric(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func (c *PDFTextCleaner) isRepeatedCharacterLine(line string) bool {
	if len(line) < 5 {
		return false
	}

	repeatedChars := map[rune]int{
		'.': 0, '-': 0, '_': 0, '=': 0, '*': 0, '+': 0, '~': 0,
	}

	totalChars := 0
	repeatedCount := 0

	for _, r := range line {
		if r == ' ' || r == '\t' {
			continue
		}
		totalChars++
		if _, exists := repeatedChars[r]; exists {
			repeatedChars[r]++
			repeatedCount++
		}
	}

	if totalChars > 0 && float64(repeatedCount)/float64(totalChars) > 0.7 {
		return true
	}

	for char := range repeatedChars {
		pattern := string(char)
		charCount := strings.Count(line, pattern)
		if charCount >= 5 && len(line) > 10 {
			noSpace := strings.ReplaceAll(line, " ", "")
			noSpace = strings.ReplaceAll(noSpace, "\t", "")
			if float64(charCount)/float64(len(noSpace)) > 0.5 {
				return true
			}
		}
	}

	return false
}

func (c *PDFTextCleaner) isFormFieldLine(line string) bool {
	if len(line) < 10 {
		return false
	}

	dotCount := strings.Count(line, ".")
	letterCount := 0
	for _, r := range line {
		if unicode.IsLetter(r) {
			letterCount++
		}
	}

	if dotCount >= 5 && letterCount > 0 && letterCount < 20 {
		formPatterns := []string{
			"name", "address", "date", "phone", "signature", "title",
			"city", "state", "zip", "email", "age", "sex", "occupation",
		}

		lineLower := strings.ToLower(line)
		for _, pattern := range formPatterns {
			if strings.Contains(lineLower, pattern) && dotCount > letterCount {
				return true
			}
		}

		if strings.Contains(line, ":") && dotCount > 10 {
			return true
		}
	}

	return false
}
