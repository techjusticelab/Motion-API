package classifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// claudeClassifier implements classification using Claude's API
type claudeClassifier struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// NewClaudeClassifier creates a new Claude-based classifier
func NewClaudeClassifier(config *ClaudeConfig) (Classifier, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Claude API key is required")
	}

	model := config.Model
	if model == "" {
		model = "claude-3-5-sonnet-20241022" // Default model
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	return &claudeClassifier{
		apiKey:  config.APIKey,
		model:   model,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Classify analyzes document text using Claude and returns classification results
func (c *claudeClassifier) Classify(ctx context.Context, text string, metadata *DocumentMetadata) (*ClassificationResult, error) {
	// Create the classification prompt
	prompt := c.buildClassificationPrompt(text, metadata)

	// Make request to Claude
	response, err := c.makeClaudeRequest(ctx, prompt)
	if err != nil {
		return nil, NewClassificationError("claude_request", "failed to classify document", err)
	}

	// Parse the response
	result, err := c.parseClassificationResponse(response)
	if err != nil {
		return nil, NewClassificationError("response_parsing", "failed to parse classification response", err)
	}

	return result, nil
}

// GetSupportedCategories returns the categories this classifier can identify
func (c *claudeClassifier) GetSupportedCategories() []string {
	return GetDefaultCategories()
}

// IsConfigured returns true if the classifier is properly configured
func (c *claudeClassifier) IsConfigured() bool {
	return c.apiKey != "" && c.model != ""
}

// buildClassificationPrompt creates a prompt for Claude classification using unified prompts
func (c *claudeClassifier) buildClassificationPrompt(text string, metadata *DocumentMetadata) string {
	// Use the unified prompt builder with Claude-specific configuration
	config := DefaultPromptConfigs["claude"]
	if config == nil {
		config = &PromptConfig{
			Model:         c.model,
			MaxTextLength: 15000,
			IncludeContext: true,
			DetailLevel:   "comprehensive",
		}
	}
	
	builder := NewPromptBuilder(config)
	return builder.BuildClassificationPrompt(text, metadata)
}

// Claude API request/response structures
type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// makeClaudeRequest sends a request to Claude's API
func (c *claudeClassifier) makeClaudeRequest(ctx context.Context, prompt string) (string, error) {
	reqBody := claudeRequest{
		Model:     c.model,
		MaxTokens: 1500,
		Messages: []claudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Claude API returned status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if claudeResp.Error != nil {
		return "", fmt.Errorf("Claude API error: %s", claudeResp.Error.Message)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("no content returned from Claude")
	}

	return claudeResp.Content[0].Text, nil
}

// parseClassificationResponse parses the Claude response into a ClassificationResult
func (c *claudeClassifier) parseClassificationResponse(response string) (*ClassificationResult, error) {
	// Add debugging logs to see actual response
	log.Printf("[CLAUDE] Raw response length: %d chars", len(response))
	log.Printf("[CLAUDE] Raw response preview: %.200s...", response)
	
	// Try to extract JSON from the response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}") + 1

	log.Printf("[CLAUDE] JSON bounds: start=%d, end=%d", jsonStart, jsonEnd)

	if jsonStart == -1 || jsonEnd <= jsonStart {
		log.Printf("[CLAUDE] ❌ No valid JSON brackets found in response")
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := response[jsonStart:jsonEnd]
	log.Printf("[CLAUDE] Extracted JSON length: %d chars", len(jsonStr))
	log.Printf("[CLAUDE] Extracted JSON preview: %.200s...", jsonStr)

	var result ClassificationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Validate and set defaults
	if result.DocumentType == "" {
		result.DocumentType = DocumentTypeOther
	}
	if result.LegalCategory == "" {
		result.LegalCategory = LegalCategoryCivil
	}
	if result.Confidence == 0 {
		result.Confidence = 0.5 // Default confidence
	}
	
	// Validate and parse dates using date extractor
	dateExtractor := NewDateExtractor()
	
	// Validate each date field if present
	if result.FilingDate != nil {
		if !dateExtractor.validateDate(*result.FilingDate, "filing_date") {
			log.Printf("[CLAUDE] Invalid filing_date: %s, setting to nil", *result.FilingDate)
			result.FilingDate = nil
		}
	}
	if result.EventDate != nil {
		if !dateExtractor.validateDate(*result.EventDate, "event_date") {
			log.Printf("[CLAUDE] Invalid event_date: %s, setting to nil", *result.EventDate)
			result.EventDate = nil
		}
	}
	if result.HearingDate != nil {
		if !dateExtractor.validateDate(*result.HearingDate, "hearing_date") {
			log.Printf("[CLAUDE] Invalid hearing_date: %s, setting to nil", *result.HearingDate)
			result.HearingDate = nil
		}
	}
	if result.DecisionDate != nil {
		if !dateExtractor.validateDate(*result.DecisionDate, "decision_date") {
			log.Printf("[CLAUDE] Invalid decision_date: %s, setting to nil", *result.DecisionDate)
			result.DecisionDate = nil
		}
	}
	if result.ServedDate != nil {
		if !dateExtractor.validateDate(*result.ServedDate, "served_date") {
			log.Printf("[CLAUDE] Invalid served_date: %s, setting to nil", *result.ServedDate)
			result.ServedDate = nil
		}
	}

	// Validate document type against known types
	validTypes := GetDefaultDocumentTypes()
	isValidType := false
	for _, validType := range validTypes {
		if result.DocumentType == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		result.DocumentType = DocumentTypeOther
		result.Confidence = result.Confidence * 0.8 // Reduce confidence for fallback
	}

	// Ensure subject is provided if summary exists but subject doesn't
	if result.Subject == "" && result.Summary != "" {
		// Extract first sentence as subject if available
		sentences := strings.Split(result.Summary, ".")
		if len(sentences) > 0 && len(sentences[0]) > 0 {
			words := strings.Fields(sentences[0])
			if len(words) > 12 {
				result.Subject = strings.Join(words[:12], " ") + "..."
			} else {
				result.Subject = sentences[0]
			}
		}
	}

	// Set success flag
	result.Success = true

	return &result, nil
}