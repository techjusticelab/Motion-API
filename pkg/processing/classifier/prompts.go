package classifier

import (
	"fmt"
	"strings"
)

// PromptConfig contains configuration for prompt generation
type PromptConfig struct {
	Model           string
	MaxTextLength   int
	IncludeContext  bool
	DetailLevel     string // "minimal", "standard", "comprehensive"
}

// DefaultPromptConfigs contains default configurations for different models
var DefaultPromptConfigs = map[string]*PromptConfig{
	"openai": {
		Model:         "gpt-4",
		MaxTextLength: 12000,
		IncludeContext: true,
		DetailLevel:   "comprehensive",
	},
	"claude": {
		Model:         "claude-3-sonnet",
		MaxTextLength: 15000,
		IncludeContext: true,
		DetailLevel:   "comprehensive",
	},
	"ollama": {
		Model:         "llama3",
		MaxTextLength: 8000,
		IncludeContext: false,
		DetailLevel:   "standard",
	},
}

// Core prompt template constants
const (
	BaseAnalysisPrompt = `You are an expert legal document analyzer specializing in California criminal law and civil litigation. 

Analyze the following legal document and provide comprehensive classification and extraction.`

	DateExtractionInstructions = `CRITICAL DATE EXTRACTION REQUIREMENTS:

Extract ALL relevant dates with high precision. Look for:

1. FILING_DATE: When the document was filed with the court
   - Look for: "Filed on", "Date filed", "Filed", timestamps at document header/footer
   - Common patterns: MM/DD/YYYY, Month DD, YYYY, YYYY-MM-DD

2. EVENT_DATE: Key event or action date referenced in the document  
   - Look for: hearing dates, motion deadlines, incident dates, violation dates
   - Context clues: "on or about", "incident occurred", "hearing scheduled"

3. HEARING_DATE: Scheduled court hearing or proceeding date
   - Look for: "hearing set", "scheduled for", "calendar call", "arraignment"
   - May include time information which should be ignored (extract date only)

4. DECISION_DATE: When a court decision, ruling, or order was made
   - Look for: "decided on", "ruling date", "order entered", "judgment rendered"
   - Often found in orders, rulings, and judgments

5. SERVED_DATE: When documents were served to parties
   - Look for: "served on", "service date", "personally served", "mailed on"
   - Important for calculating response deadlines

DATE PARSING RULES:
- Convert all dates to YYYY-MM-DD format
- If only month/year available, use first day of month: YYYY-MM-01
- If year missing but clearly recent, assume current year
- For partial dates, extract what's available and note uncertainty
- If date is clearly invalid or impossible, return null
- Handle relative dates: "tomorrow", "next Monday", etc. (calculate from filing date if available)

DATE VALIDATION:
- Legal documents typically range from 1950-present
- Filing dates should not be in the future
- Event dates can be past, present, or reasonable future
- Hearing dates are typically future dates from filing date`

	EntityExtractionGuidelines = `ENTITY EXTRACTION GUIDELINES:

Extract entities with confidence scores:
- PERSON: Individual names (judges, attorneys, parties, witnesses)
- ORGANIZATION: Law firms, government agencies, courts, companies
- LOCATION: Courts, jurisdictions, addresses, crime scenes
- DATE: All dates (will be processed separately for date fields)
- MONEY: Amounts, fees, damages, bail amounts
- LEGAL_CITATION: Case citations, statutes, regulations
- CASE_NUMBER: Docket numbers, case identifiers
- STATUTE: Specific legal code sections`

	JSONResponseSchema = `{
  "document_type": "<one of the available document types>",
  "legal_category": "<primary legal area>", 
  "subject": "<concise 8-12 word subject line>",
  "summary": "<comprehensive legal summary following document-specific requirements>",
  "confidence": <float between 0 and 1>,
  "keywords": ["<key legal terms and procedural elements>"],
  "legal_tags": ["<relevant legal doctrine tags>"],
  "case_info": {
    "case_number": "<case number if found>",
    "case_name": "<case title if found>",
    "case_type": "<criminal|civil|traffic|family>",
    "docket": "<full docket number>"
  },
  "court_info": {
    "court_name": "<court name>",
    "jurisdiction": "<federal|state|local>",
    "level": "<trial|appellate|supreme>",
    "county": "<county if applicable>"
  },
  "parties": [
    {
      "name": "<party name>",
      "role": "<defendant|plaintiff|appellant|respondent>",
      "party_type": "<individual|corporation|government>"
    }
  ],
  "attorneys": [
    {
      "name": "<attorney name>",
      "role": "<defense|prosecution|counsel>",
      "organization": "<law firm or agency>"
    }
  ],
  "judge": {
    "name": "<judge name>",
    "title": "<title if specified>"
  },
  "charges": [
    {
      "statute": "<statute number>",
      "description": "<charge description>",
      "grade": "<felony|misdemeanor>",
      "class": "<A|B|C>"
    }
  ],
  "authorities": [
    {
      "citation": "<legal citation>",
      "case_title": "<case name>",
      "type": "<case_law|statute|regulation>",
      "precedent": <true|false>
    }
  ],
  "filing_date": "<YYYY-MM-DD format or null>",
  "event_date": "<YYYY-MM-DD format or null>",
  "hearing_date": "<YYYY-MM-DD format or null>",
  "decision_date": "<YYYY-MM-DD format or null>",
  "served_date": "<YYYY-MM-DD format or null>",
  "status": "<filed|granted|denied|pending|served>",
  "entities": [
    {
      "text": "<entity text>",
      "type": "<PERSON|ORGANIZATION|LOCATION|DATE|MONEY|LEGAL_CITATION|CASE_NUMBER|STATUTE>",
      "confidence": <float between 0 and 1>
    }
  ]
}`
)

// Document-specific summarization requirements
var DocumentSummarizationRules = map[string]string{
	"motion": `FOR MOTIONS (motion_to_suppress, motion_to_dismiss, etc.):
- Motion type and specific relief sought (3-4 sentences)
- Key legal arguments and constitutional/statutory authorities cited (2-3 sentences)
- Factual basis and procedural posture (2 sentences)
- Potential impact on case progression (1 sentence)`,

	"order": `FOR ORDERS/RULINGS (order, ruling, judgment):
- Court's holding and primary reasoning (3-4 sentences)
- Key legal precedents and statutes applied (2 sentences)
- Impact on pending motions and case status (2 sentences)
- Practical implications for parties (1-2 sentences)`,

	"brief": `FOR BRIEFS (brief, reply):
- Main legal arguments and theory of the case (4-5 sentences)
- Factual background and procedural history (2-3 sentences)
- Authorities relied upon and distinguishing cases (2-3 sentences)
- Relief requested and strategic positioning (1-2 sentences)`,

	"pleading": `FOR PLEADINGS (complaint, answer, plea):
- Claims/charges and factual allegations (3-4 sentences)
- Legal theories and causes of action (2-3 sentences)
- Defenses raised and procedural responses (2 sentences)
- Stakes and potential outcomes (1 sentence)`,
}

// PromptBuilder provides methods to build classification prompts
type PromptBuilder struct {
	config *PromptConfig
}

// NewPromptBuilder creates a new prompt builder with the given configuration
func NewPromptBuilder(config *PromptConfig) *PromptBuilder {
	if config == nil {
		config = DefaultPromptConfigs["openai"] // Default to OpenAI config
	}
	return &PromptBuilder{config: config}
}

// BuildClassificationPrompt creates a complete classification prompt
func (pb *PromptBuilder) BuildClassificationPrompt(text string, metadata *DocumentMetadata) string {
	// Truncate text if necessary
	if len(text) > pb.config.MaxTextLength {
		text = text[:pb.config.MaxTextLength] + "..."
	}

	// Build metadata section
	metadataSection := pb.buildMetadataSection(metadata)

	// Build context section
	contextSection := ""
	if pb.config.IncludeContext && metadata != nil {
		contextSection = pb.buildContextSection(metadata)
	}

	// Get summarization rules
	summaryRules := pb.buildSummarizationRules()

	// Build the complete prompt
	prompt := fmt.Sprintf(`%s

Document metadata:
%s

%s

%s

CRITICAL INSTRUCTIONS:
1. Classify document type from: %s
2. Provide SUBSTANTIVE legal summary based on document type
3. Extract ALL legal entities with high precision
4. Identify case information, parties, and procedural context

%s

%s

DOCUMENT-SPECIFIC SUMMARIZATION REQUIREMENTS:

%s

Document text:
%s

Respond with ONLY a JSON object in this exact format:
%s

Use null for any field that cannot be determined from the document text.`,
		BaseAnalysisPrompt,
		metadataSection,
		contextSection,
		DateExtractionInstructions,
		strings.Join(GetDefaultDocumentTypes(), ", "),
		EntityExtractionGuidelines,
		pb.getModelSpecificInstructions(),
		summaryRules,
		text,
		JSONResponseSchema,
	)

	return prompt
}

// buildMetadataSection creates the metadata section of the prompt
func (pb *PromptBuilder) buildMetadataSection(metadata *DocumentMetadata) string {
	if metadata == nil {
		return "- No metadata available"
	}

	return fmt.Sprintf(`- File name: %s
- File type: %s
- Word count: %d words
- Page count: %d pages
- Source system: %s`,
		getStringValue(metadata, "file_name"),
		getStringValue(metadata, "file_type"),
		getIntValue(metadata, "word_count"),
		getIntValue(metadata, "page_count"),
		getStringValue(metadata, "source_system"),
	)
}

// buildContextSection creates contextual analysis instructions based on document characteristics
func (pb *PromptBuilder) buildContextSection(metadata *DocumentMetadata) string {
	if metadata == nil {
		return ""
	}

	wordCount := getIntValue(metadata, "word_count")
	pageCount := getIntValue(metadata, "page_count")
	fileType := strings.ToLower(getStringValue(metadata, "file_type"))

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

// buildSummarizationRules creates the document-specific summarization requirements
func (pb *PromptBuilder) buildSummarizationRules() string {
	rules := []string{}
	for _, rule := range DocumentSummarizationRules {
		rules = append(rules, rule)
	}
	return strings.Join(rules, "\n\n")
}

// getModelSpecificInstructions returns model-specific optimization instructions
func (pb *PromptBuilder) getModelSpecificInstructions() string {
	switch pb.config.Model {
	case "gpt-4", "gpt-3.5-turbo":
		return `OPENAI OPTIMIZATION:
- Use detailed contextual reasoning for date extraction
- Provide comprehensive entity analysis
- Focus on legal precedent identification`
	case "claude-3-sonnet", "claude-3-haiku":
		return `CLAUDE OPTIMIZATION:
- Apply structured legal reasoning approach
- Emphasize statutory and constitutional analysis
- Provide detailed procedural context`
	default: // Ollama/local models
		return `LOCAL MODEL OPTIMIZATION:
- Use clear, direct analysis approach
- Focus on essential legal elements
- Provide concise but complete extraction`
	}
}

