package classifier

import (
	"strings"
)

func (c *openaiClassifier) generateContextualPrompt(metadata *DocumentMetadata) string {
	if metadata == nil {
		return "CRITICAL INSTRUCTIONS:\n1. Classify document type from available types\n2. Provide substantive legal summary\n3. Extract legal entities with precision"
	}

	wordCount := metadata.WordCount
	pageCount := metadata.PageCount

	contextPrompt := "DOCUMENT ANALYSIS CONTEXT:\n"

	// Add analysis guidance based on document characteristics
	switch {
	case wordCount < 300:
		contextPrompt += "- SHORT DOCUMENT: Focus on key identifying elements and brief classification\n"
		contextPrompt += "- Prioritize document type identification over detailed extraction\n"
	case wordCount > 5000:
		contextPrompt += "- COMPREHENSIVE DOCUMENT: Perform detailed analysis and full entity extraction\n"
		contextPrompt += "- Extract maximum legal detail including all parties, dates, and authorities\n"
	case pageCount > 10:
		contextPrompt += "- MULTI-PAGE DOCUMENT: Analyze structure and extract section-specific information\n"
		contextPrompt += "- Look for procedural progression and case development over multiple sections\n"
	default:
		contextPrompt += "- STANDARD DOCUMENT: Perform balanced analysis with focus on legal substance\n"
	}

	// Add specific guidance based on file type
	fileType := strings.ToLower(metadata.FileType)
	switch {
	case strings.Contains(fileType, "pdf"):
		contextPrompt += "- PDF DOCUMENT: May contain formatted legal text, pay attention to structure\n"
	case strings.Contains(fileType, "docx"):
		contextPrompt += "- WORD DOCUMENT: Likely draft or working document, analyze for intent and completeness\n"
	case strings.Contains(fileType, "txt"):
		contextPrompt += "- TEXT DOCUMENT: May lack formatting, focus on content analysis\n"
	}

	return contextPrompt
}

func (c *openaiClassifier) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Retry on rate limit errors (429)
	if strings.Contains(errStr, "status 429") {
		return true
	}

	// Retry on server errors (5xx)
	if strings.Contains(errStr, "status 5") {
		return true
	}

	// DO NOT retry on timeout errors - LLMs may need more time to respond
	// Retrying on timeout wastes API credits by calling the same document multiple times
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "context deadline exceeded") {
		return false
	}

	// Retry on connection errors (network failures, not timeouts)
	if strings.Contains(errStr, "connection") || strings.Contains(errStr, "network") {
		return true
	}

	// Don't retry on client errors (4xx except 429), authentication errors, etc.
	return false
}
