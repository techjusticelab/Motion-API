package models

import (
	"time"
)

type Document struct {
	ID          string            `json:"id"`
	FileName    string            `json:"file_name"`
	FilePath    string            `json:"file_path"`
	FileURL     string            `json:"file_url,omitempty"`
	S3URI       string            `json:"s3_uri,omitempty"`
	Text        string            `json:"text"`
	DocType     string            `json:"doc_type"`
	Category    string            `json:"category,omitempty"`
	Hash        string            `json:"hash"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    *DocumentMetadata `json:"metadata"`
	Size        int64             `json:"size,omitempty"`
	ContentType string            `json:"content_type,omitempty"`

	// For backward compatibility with tests
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
}

type DocumentMetadata struct {
	// Basic Information
	DocumentName string       `json:"document_name"`
	Subject      string       `json:"subject"`
	Summary      string       `json:"summary,omitempty"` // Enhanced legal summary
	DocumentType DocumentType `json:"document_type"`

	// Case Information
	Case *CaseInfo `json:"case,omitempty"`

	// Court Information
	Court *CourtInfo `json:"court,omitempty"`

	// People & Parties
	Parties   []Party    `json:"parties,omitempty"`
	Attorneys []Attorney `json:"attorneys,omitempty"`
	Judge     *Judge     `json:"judge,omitempty"`

	// Dates & Status - Enhanced date fields for legal documents
	FilingDate   *time.Time `json:"filing_date,omitempty"`   // When document was filed with court
	EventDate    *time.Time `json:"event_date,omitempty"`    // Key event or action date
	HearingDate  *time.Time `json:"hearing_date,omitempty"`  // Scheduled court hearing date
	DecisionDate *time.Time `json:"decision_date,omitempty"` // When court decision was made
	ServedDate   *time.Time `json:"served_date,omitempty"`   // When documents were served
	Timestamp    *time.Time `json:"timestamp,omitempty"`     // For backward compatibility
	Status       string     `json:"status,omitempty"`

	// Document Properties
	Language  string `json:"language,omitempty"`
	Pages     int    `json:"pages,omitempty"`
	WordCount int    `json:"word_count,omitempty"`

	// Legal Classification
	LegalTags   []string    `json:"legal_tags,omitempty"`
	Charges     []Charge    `json:"charges,omitempty"`
	Authorities []Authority `json:"authorities,omitempty"`

	// Processing Metadata
	ProcessedAt  time.Time `json:"processed_at"`
	Confidence   float64   `json:"confidence,omitempty"`
	AIClassified bool      `json:"ai_classified"`

	// Legacy fields for backward compatibility
	CaseName   string `json:"case_name,omitempty"`
	CaseNumber string `json:"case_number,omitempty"`
	Author     string `json:"author,omitempty"`
}

func (dm *DocumentMetadata) GetCaseName() string {
	if dm.Case != nil && dm.Case.CaseName != "" {
		return dm.Case.CaseName
	}
	return dm.CaseName
}

func (dm *DocumentMetadata) GetCaseNumber() string {
	if dm.Case != nil && dm.Case.CaseNumber != "" {
		return dm.Case.CaseNumber
	}
	return dm.CaseNumber
}

func (dm *DocumentMetadata) GetCourtName() string {
	if dm.Court != nil {
		return dm.Court.CourtName
	}
	return ""
}

func (dm *DocumentMetadata) GetJudgeName() string {
	if dm.Judge != nil {
		return dm.Judge.Name
	}
	return ""
}

func (dm *DocumentMetadata) HasLegalTag(tag string) bool {
	for _, t := range dm.LegalTags {
		if t == tag {
			return true
		}
	}
	return false
}

func (dm *DocumentMetadata) AddLegalTag(tag string) {
	if !dm.HasLegalTag(tag) {
		dm.LegalTags = append(dm.LegalTags, tag)
	}
}

func (dm *DocumentMetadata) GetPrimaryAttorney() *Attorney {
	if len(dm.Attorneys) > 0 {
		return &dm.Attorneys[0]
	}
	return nil
}

func (dm *DocumentMetadata) GetDefenseAttorneys() []Attorney {
	var defense []Attorney
	for _, attorney := range dm.Attorneys {
		if attorney.Role == "defense" {
			defense = append(defense, attorney)
		}
	}
	return defense
}

func (dm *DocumentMetadata) GetProsecutionAttorneys() []Attorney {
	var prosecution []Attorney
	for _, attorney := range dm.Attorneys {
		if attorney.Role == "prosecution" {
			prosecution = append(prosecution, attorney)
		}
	}
	return prosecution
}

func (dm *DocumentMetadata) IsMotion() bool {
	return dm.DocumentType.IsMotion()
}

func (dm *DocumentMetadata) IsOrder() bool {
	return dm.DocumentType.IsOrder()
}

func (dm *DocumentMetadata) IsPleading() bool {
	return dm.DocumentType.IsPleading()
}

func (dm *DocumentMetadata) GetDocumentCategory() string {
	return dm.DocumentType.GetCategory()
}

func (dm *DocumentMetadata) SetLegacyFields() {
	dm.CaseName = dm.GetCaseName()
	dm.CaseNumber = dm.GetCaseNumber()
	if attorney := dm.GetPrimaryAttorney(); attorney != nil {
		dm.Author = attorney.Name
	}
}

func NewDocumentMetadata() *DocumentMetadata {
	return &DocumentMetadata{
		ProcessedAt:  time.Now(),
		AIClassified: false,
		DocumentType: DocTypeUnknown,
		Language:     "en",
		LegalTags:    make([]string, 0),
		Parties:      make([]Party, 0),
		Attorneys:    make([]Attorney, 0),
		Charges:      make([]Charge, 0),
		Authorities:  make([]Authority, 0),
	}
}

func GetDocumentMapping() map[string]interface{} {
	return map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type": "keyword",
				},
				"file_name": map[string]interface{}{
					"type": "text",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{
							"type": "keyword",
						},
					},
				},
				"file_path": map[string]interface{}{
					"type": "keyword",
				},
				"file_url": map[string]interface{}{
					"type":  "keyword",
					"index": false,
				},
				"s3_uri": map[string]interface{}{
					"type":  "keyword",
					"index": false,
				},
				"text": map[string]interface{}{
					"type":     "text",
					"analyzer": "legal_analyzer",
				},
				"doc_type": map[string]interface{}{
					"type": "keyword",
				},
				"category": map[string]interface{}{
					"type": "keyword",
				},
				"hash": map[string]interface{}{
					"type": "keyword",
				},
				"created_at": map[string]interface{}{
					"type": "date",
				},
				"updated_at": map[string]interface{}{
					"type": "date",
				},
				"size": map[string]interface{}{
					"type": "long",
				},
				"content_type": map[string]interface{}{
					"type": "keyword",
				},
				"metadata": getMetadataMapping(),
			},
		},
		"settings": map[string]interface{}{
			"number_of_shards":   1,
			"number_of_replicas": 0,
			"analysis": map[string]interface{}{
				"analyzer": map[string]interface{}{
					"legal_analyzer": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "standard",
						"filter": []string{
							"lowercase",
							"stop",
							"stemmer",
							"legal_synonyms",
						},
					},
				},
				"filter": map[string]interface{}{
					"legal_synonyms": map[string]interface{}{
						"type": "synonym",
						"synonyms": []string{
							"motion,petition,application",
							"defendant,accused,respondent",
							"plaintiff,petitioner,complainant",
							"attorney,counsel,lawyer",
							"judge,court,magistrate",
							"order,ruling,decision",
							"suppress,exclude,prohibit",
							"dismiss,quash,deny",
						},
					},
				},
			},
		},
	}
}

func getMetadataMapping() map[string]interface{} {
	return map[string]interface{}{
		"properties": map[string]interface{}{
			"document_name": map[string]interface{}{
				"type": "text",
				"fields": map[string]interface{}{
					"keyword": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
			"subject": map[string]interface{}{
				"type":     "text",
				"analyzer": "legal_analyzer",
			},
			"summary": map[string]interface{}{
				"type":     "text",
				"analyzer": "legal_analyzer",
			},
			"document_type": map[string]interface{}{
				"type": "keyword",
			},
			"status": map[string]interface{}{
				"type": "keyword",
			},
			"filing_date": map[string]interface{}{
				"type": "date",
			},
			"event_date": map[string]interface{}{
				"type": "date",
			},
			"hearing_date": map[string]interface{}{
				"type": "date",
			},
			"decision_date": map[string]interface{}{
				"type": "date",
			},
			"served_date": map[string]interface{}{
				"type": "date",
			},
			"processed_at": map[string]interface{}{
				"type": "date",
			},
			"confidence": map[string]interface{}{
				"type": "float",
			},
			"ai_classified": map[string]interface{}{
				"type": "boolean",
			},
			"case":        getCaseMapping(),
			"court":       getCourtMapping(),
			"parties":     getPartiesMapping(),
			"attorneys":   getAttorneysMapping(),
			"judge":       getJudgeMapping(),
			"charges":     getChargesMapping(),
			"authorities": getAuthoritiesMapping(),
			"legal_tags": map[string]interface{}{
				"type": "keyword",
			},
			"language": map[string]interface{}{
				"type": "keyword",
			},
			"pages": map[string]interface{}{
				"type": "integer",
			},
			"word_count": map[string]interface{}{
				"type": "integer",
			},
			// Legacy fields for backward compatibility
			"case_name": map[string]interface{}{
				"type": "text",
				"fields": map[string]interface{}{
					"keyword": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
			"case_number": map[string]interface{}{
				"type": "keyword",
			},
			"author": map[string]interface{}{
				"type": "keyword",
			},
		},
	}
}
