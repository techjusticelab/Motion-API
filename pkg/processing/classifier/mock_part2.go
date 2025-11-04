package classifier

func (m *mockClassifier) generateSummary(text, documentType string) string {
	switch documentType {
	case DocumentTypeMotionToSuppress:
		return "Motion requesting exclusion of evidence from trial proceedings."
	case DocumentTypeMotionToDismiss:
		return "Motion seeking dismissal of charges or claims for legal insufficiency."
	case DocumentTypeMotionToCompel:
		return "Motion requesting court order to compel compliance with discovery requests."
	case DocumentTypeMotionInLimine:
		return "Pre-trial motion seeking to exclude prejudicial evidence or testimony."
	case DocumentTypeOrder:
		return "Court order directing parties to take or refrain from specific actions."
	case DocumentTypeRuling:
		return "Judicial decision on a matter before the court."
	case DocumentTypeBrief:
		return "Legal brief presenting arguments and case law analysis."
	case DocumentTypeComplaint:
		return "Initial pleading setting forth plaintiff's claims and requested relief."
	case DocumentTypeAnswer:
		return "Defendant's response to allegations in complaint or petition."
	default:
		return "Legal document containing case-related information and proceedings."
	}
}

func (m *mockClassifier) calculateProcessingTime(metadata *DocumentMetadata) int {
	baseTime := 10 // Base processing time in milliseconds

	if metadata == nil {
		return baseTime
	}

	// Simulate longer processing for larger documents
	wordCount := metadata.WordCount
	switch {
	case wordCount < 500:
		return baseTime
	case wordCount < 2000:
		return baseTime + 20
	case wordCount < 5000:
		return baseTime + 50
	default:
		return baseTime + 100
	}
}

func (m *mockClassifier) generateSubject(documentType string, metadata *DocumentMetadata) string {
	switch documentType {
	case DocumentTypeMotionToSuppress:
		return "Motion to Suppress Evidence - Criminal Case"
	case DocumentTypeMotionToDismiss:
		return "Motion to Dismiss Charges - Legal Insufficiency"
	case DocumentTypeOrder:
		return "Court Order - Judicial Ruling"
	case DocumentTypeBrief:
		return "Legal Brief - Case Analysis"
	default:
		return "Legal Document - Case Proceeding"
	}
}

func (m *mockClassifier) generateMockCaseInfo(text string, metadata *DocumentMetadata) *CaseInfo {
	return &CaseInfo{
		CaseNumber:   "CR-2024-001234",
		CaseName:     "People v. Smith",
		CaseType:     "criminal",
		Docket:       "Superior Court Case CR-2024-001234",
		NatureOfSuit: "Criminal prosecution",
	}
}

func (m *mockClassifier) generateMockCourtInfo(text string, metadata *DocumentMetadata) *CourtInfo {
	return &CourtInfo{
		CourtName:    "Superior Court of California",
		Jurisdiction: "state",
		Level:        "trial",
		County:       "Los Angeles",
	}
}

func (m *mockClassifier) generateMockParties(text string, metadata *DocumentMetadata) []Party {
	return []Party{
		{
			Name:      "John Smith",
			Role:      "defendant",
			PartyType: "individual",
		},
		{
			Name:      "People of the State of California",
			Role:      "plaintiff",
			PartyType: "government",
		},
	}
}

func (m *mockClassifier) generateMockAttorneys(text string, metadata *DocumentMetadata) []Attorney {
	return []Attorney{
		{
			Name:         "Jane Doe",
			Role:         "defense",
			Organization: "Public Defender's Office",
		},
		{
			Name:         "Michael Johnson",
			Role:         "prosecution",
			Organization: "District Attorney's Office",
		},
	}
}

func (m *mockClassifier) generateMockJudge(text string, metadata *DocumentMetadata) *Judge {
	return &Judge{
		Name:  "Hon. Sarah Wilson",
		Title: "Superior Court Judge",
	}
}

func stringPtr(s string) *string {
	return &s
}

func getMetadataInt(metadata *DocumentMetadata, field string) int {
	if metadata == nil {
		return 0
	}
	switch field {
	case "word_count":
		return metadata.WordCount
	case "page_count":
		return metadata.PageCount
	default:
		return 0
	}
}
