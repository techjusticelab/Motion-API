package extractor

import (
	"fmt"
	"regexp"
	"strings"
)

func (c *PDFTextCleaner) isHeaderFooterLine(line string) bool {
	if len(line) == 0 {
		return false
	}

	lineLower := strings.ToLower(strings.TrimSpace(line))

	headerFooterPatterns := []string{
		"page ", "of ", "continued", "confidential", "draft",
		"proprietary", "exhibit ", "attachment ", "schedule ",
		"case no", "docket", "filed", "clerk", "court",
	}

	if regexp.MustCompile(`^page\s+\d+`).MatchString(lineLower) {
		return true
	}
	if regexp.MustCompile(`^\d+\s+of\s+\d+$`).MatchString(lineLower) {
		return true
	}
	if regexp.MustCompile(`^-\s*\d+\s*-$`).MatchString(line) {
		return true
	}
	if regexp.MustCompile(`\d{1,2}[\/\-]\d{1,2}[\/\-]\d{4}`).MatchString(line) && len(line) < 30 {
		return true
	}
	if regexp.MustCompile(`^[A-Z0-9\-]{8,}$`).MatchString(strings.ReplaceAll(line, " ", "")) {
		return true
	}
	if len(line) < 50 {
		for _, pattern := range headerFooterPatterns {
			if strings.Contains(lineLower, pattern) {
				return true
			}
		}
	}

	return false
}

func (c *PDFTextCleaner) cleanTableOfContentsArtifacts(line string) string {
	if strings.Count(line, ".") < 5 {
		return line
	}

	tocPattern := regexp.MustCompile(`^(.+?)(\.{5,})(\s*)(\d+)?\s*$`)
	matches := tocPattern.FindStringSubmatch(line)

	if len(matches) >= 3 {
		title := strings.TrimSpace(matches[1])
		pageNum := ""
		if len(matches) >= 5 && matches[4] != "" {
			pageNum = matches[4]
		}

		if len(title) > 3 && c.hasAlphanumeric(title) {
			if pageNum != "" {
				return fmt.Sprintf("%s (page %s)", title, pageNum)
			}
			return title
		}
	}

	return line
}
