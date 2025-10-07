package redaction

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// service implements the Service interface
type service struct {
	aiEnabled bool
	openaiKey string
}

// NewService creates a new redaction service
func NewService(aiEnabled bool, openaiKey string) Service {
	return &service{
		aiEnabled: aiEnabled,
		openaiKey: openaiKey,
	}
}

// California legal codes and regulations
var CaliforniaCodes = map[string]CaliforniaLegalCode{
	"CCP_1798.3": {
		Code:        "CCP_1798.3",
		Description: "California Civil Code § 1798.3 - Prohibits disclosure of personal information",
	},
	"WIC_827": {
		Code:        "WIC_827",
		Description: "California Welfare and Institutions Code § 827 - Confidentiality of juvenile records",
	},
	"PC_293": {
		Code:        "PC_293",
		Description: "California Penal Code § 293 - Protection of sexual assault victim information",
	},
	"PC_841.5": {
		Code:        "PC_841.5",
		Description: "California Penal Code § 841.5 - Confidentiality of informant information",
	},
	"EC_1040": {
		Code:        "EC_1040",
		Description: "California Evidence Code § 1040 - Privilege for official information",
	},
	"CRC_2.550": {
		Code:        "CRC_2.550",
		Description: "California Rules of Court 2.550 - Sealed records requirements",
	},
	"FC_3042": {
		Code:        "FC_3042",
		Description: "California Family Code § 3042 - Protection of minor's information in custody proceedings",
	},
	"HSC_123100": {
		Code:        "HSC_123100",
		Description: "California Health and Safety Code § 123100 - Medical information confidentiality",
	},
	"CCPA": {
		Code:        "CCPA",
		Description: "California Consumer Privacy Act - Protection of personal information",
	},
	"GOV_6254": {
		Code:        "GOV_6254",
		Description: "California Government Code § 6254 - Exemptions from public records disclosure",
	},
}

// Default California redaction patterns
var CaliforniaPatterns = []RedactionPattern{
	{
		Name:    "ssn",
		Pattern: `\b\d{3}-\d{2}-\d{4}\b|\b\d{9}\b`,
		Citation: CaliforniaCodes["CCP_1798.3"],
		Reason:  "Social Security numbers must be redacted per California Civil Code § 1798.3",
	},
	{
		Name:    "driver_license",
		Pattern: `\b[A-Z]\d{7}\b`,
		Citation: CaliforniaCodes["GOV_6254"],
		Reason:  "Driver's license numbers are exempt from disclosure under Government Code § 6254(c)",
	},
	{
		Name:    "phone",
		Pattern: `\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`,
		Citation: CaliforniaCodes["CCPA"],
		Reason:  "Phone numbers may constitute personal information under CCPA",
	},
	{
		Name:    "email",
		Pattern: `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
		Citation: CaliforniaCodes["CCPA"],
		Reason:  "Email addresses are personal information protected under CCPA",
	},
	{
		Name:    "credit_card",
		Pattern: `\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`,
		Citation: CaliforniaCodes["CCP_1798.3"],
		Reason:  "Financial account numbers must be redacted per Civil Code § 1798.3",
	},
	{
		Name:    "bank_account",
		Pattern: `\b\d{8,17}\b`,
		Citation: CaliforniaCodes["CCP_1798.3"],
		Reason:  "Bank account numbers are protected financial information",
	},
	{
		Name:    "date_of_birth",
		Pattern: `\b(0[1-9]|1[0-2])/\d{1,2}/\d{2,4}\b`,
		Citation: CaliforniaCodes["GOV_6254"],
		Reason:  "Full dates of birth are exempt from disclosure",
	},
	{
		Name:    "financial_statements",
		Pattern: `\$\s*\d{1,3}(,\d{3})*(\.\d{2})?`,
		Citation: CaliforniaCodes["CCP_1798.3"],
		Reason:  "Hide Financial Numbers from public documents",
	},
}

// RedactPDF redacts a PDF document and returns the redacted PDF and metadata
func (s *service) RedactPDF(ctx context.Context, pdfData io.Reader, options *Options) (*Result, error) {
	// For now, return a placeholder implementation
	// In a full implementation, this would:
	// 1. Extract text from PDF with position information
	// 2. Find patterns to redact using regex and optionally AI
	// 3. Apply redactions to the PDF
	// 4. Return the redacted PDF as base64

	return &Result{
		Success:     false,
		PDFBase64:   "",
		Redactions:  []RedactionItem{},
		TotalCount:  0,
		Error:       "PDF redaction is not yet fully implemented - requires PDF processing library",
	}, nil
}

// AnalyzePDF analyzes a PDF for potential redactions without applying them
func (s *service) AnalyzePDF(ctx context.Context, pdfData io.Reader, options *Options) (*AnalysisResult, error) {
	// Read the PDF data for analysis
	pdfBytes, err := io.ReadAll(pdfData)
	if err != nil {
		return &AnalysisResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to read PDF data: %v", err),
		}, nil
	}

	// For demonstration, convert to string and analyze text patterns
	// In a real implementation, this would use a PDF library to extract positioned text
	pdfText := string(pdfBytes)
	
	var redactions []RedactionItem
	redactionID := 0

	// Apply California patterns if enabled
	if options != nil && options.CaliforniaLaws {
		for _, pattern := range CaliforniaPatterns {
			regex, err := regexp.Compile(pattern.Pattern)
			if err != nil {
				continue
			}

			matches := regex.FindAllStringSubmatch(pdfText, -1)
			for _, match := range matches {
				if len(match) > 0 {
					redactionID++
					redactions = append(redactions, RedactionItem{
						ID:        fmt.Sprintf("redaction_%d", redactionID),
						Page:      0, // Would be calculated from PDF position
						Text:      match[0],
						BBox:      []float64{0, 0, 100, 20}, // Would be calculated from PDF position
						Type:      pattern.Name,
						Citation:  pattern.Citation.Description,
						Reason:    pattern.Reason,
						LegalCode: pattern.Citation.Code,
						Applied:   false,
					})
				}
			}
		}
	}

	// Apply custom patterns if provided
	if options != nil && len(options.IncludePatterns) > 0 {
		for _, customPattern := range options.IncludePatterns {
			regex, err := regexp.Compile(customPattern)
			if err != nil {
				continue
			}

			matches := regex.FindAllStringSubmatch(pdfText, -1)
			for _, match := range matches {
				if len(match) > 0 {
					redactionID++
					redactions = append(redactions, RedactionItem{
						ID:        fmt.Sprintf("custom_redaction_%d", redactionID),
						Page:      0,
						Text:      match[0],
						BBox:      []float64{0, 0, 100, 20},
						Type:      "custom_pattern",
						Citation:  "Custom Pattern",
						Reason:    "Matches custom redaction pattern",
						LegalCode: "CUSTOM",
						Applied:   false,
					})
				}
			}
		}
	}

	// TODO: Add AI-powered redaction detection if enabled and API key is available
	if s.aiEnabled && s.openaiKey != "" && options != nil && options.UseAI {
		// This would call OpenAI API to identify additional sensitive information
		// For now, add a placeholder
		redactionID++
		redactions = append(redactions, RedactionItem{
			ID:        fmt.Sprintf("ai_redaction_%d", redactionID),
			Page:      0,
			Text:      "[AI-detected sensitive info]",
			BBox:      []float64{0, 0, 100, 20},
			Type:      "ai_identified",
			Citation:  "AI Analysis",
			Reason:    "AI identified as potentially sensitive information",
			LegalCode: "AI_DETECTED",
			Applied:   false,
		})
	}

	return &AnalysisResult{
		Redactions: redactions,
		TotalCount: len(redactions),
		Success:    true,
	}, nil
}

// ApplyCustomRedactions applies custom redactions to a PDF
func (s *service) ApplyCustomRedactions(ctx context.Context, pdfData io.Reader, redactions []RedactionItem) (*Result, error) {
	// Read PDF data
	pdfBytes, err := io.ReadAll(pdfData)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("Failed to read PDF data: %v", err),
		}, nil
	}

	// For now, return the original PDF as base64 with metadata
	// In a full implementation, this would apply the redactions to the PDF
	encodedPDF := base64.StdEncoding.EncodeToString(pdfBytes)

	// Mark all redactions as applied
	appliedRedactions := make([]RedactionItem, len(redactions))
	for i, redaction := range redactions {
		appliedRedactions[i] = redaction
		appliedRedactions[i].Applied = true
	}

	return &Result{
		RedactedPDF: pdfBytes,
		PDFBase64:   encodedPDF,
		Redactions:  appliedRedactions,
		TotalCount:  len(appliedRedactions),
		Success:     true,
	}, nil
}

// getReplacementText returns the replacement text for redacted content
func getReplacementText(originalText string, replacementChar string) string {
	if replacementChar == "" {
		replacementChar = "■"
	}
	return strings.Repeat(replacementChar, len(originalText))
}