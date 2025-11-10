package validation

import (
	"strings"
)

// SanitizeInput sanitizes user input to prevent injection attacks
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Trim whitespace
	input = strings.TrimSpace(input)

	// Remove complete script blocks (case sensitive, only lowercase)
	input = removeScriptBlocks(input)

	// Remove complete iframe blocks (case sensitive, only lowercase)
	input = removeIframeBlocks(input)

	// Remove javascript: protocols and everything after them until whitespace
	input = removeJavaScriptProtocol(input)

	// Remove other dangerous patterns
	dangerousPatterns := []string{
		"vbscript:", "onload=", "onerror=", "<object", "</object>", "<embed", "</embed>",
	}

	for _, pattern := range dangerousPatterns {
		input = strings.ReplaceAll(input, pattern, "")
	}

	return input
}

// removeScriptBlocks removes complete <script>...</script> blocks (case sensitive, lowercase only)
func removeScriptBlocks(input string) string {
	for {
		start := strings.Index(input, "<script")
		if start == -1 {
			break
		}

		// Find the end of the opening tag
		tagEnd := strings.Index(input[start:], ">")
		if tagEnd == -1 {
			// Malformed tag, just remove what we found
			input = input[:start] + input[start+7:] // Remove "<script"
			continue
		}
		tagEnd += start + 1 // Absolute position after ">"

		// Find the closing </script> tag
		end := strings.Index(input[tagEnd:], "</script>")
		if end == -1 {
			// No closing tag, remove from start position to end
			input = input[:start]
			break
		}
		end += tagEnd + 9 // Absolute position after "</script>"

		// Remove the entire script block
		input = input[:start] + input[end:]
	}
	return input
}

// removeIframeBlocks removes complete <iframe>...</iframe> blocks (case sensitive, lowercase only)
func removeIframeBlocks(input string) string {
	for {
		start := strings.Index(input, "<iframe")
		if start == -1 {
			break
		}

		// Find the end of the opening tag
		tagEnd := strings.Index(input[start:], ">")
		if tagEnd == -1 {
			// Malformed tag, just remove what we found
			input = input[:start] + input[start+7:] // Remove "<iframe"
			continue
		}
		tagEnd += start + 1 // Absolute position after ">"

		// Find the closing </iframe> tag
		end := strings.Index(input[tagEnd:], "</iframe>")
		if end == -1 {
			// No closing tag, remove from start position to end
			input = input[:start]
			break
		}
		end += tagEnd + 9 // Absolute position after "</iframe>"

		// Remove the entire iframe block
		input = input[:start] + input[end:]
	}
	return input
}

// removeJavaScriptProtocol removes javascript: and everything following until whitespace
func removeJavaScriptProtocol(input string) string {
	for {
		start := strings.Index(input, "javascript:")
		if start == -1 {
			break
		}

		// Find the end - either whitespace or end of string
		end := start + 11 // Length of "javascript:"
		for end < len(input) && !strings.ContainsRune(" \t\n\r", rune(input[end])) {
			end++
		}

		// Remove the javascript: protocol and its content
		input = input[:start] + input[end:]
	}
	return input
}
