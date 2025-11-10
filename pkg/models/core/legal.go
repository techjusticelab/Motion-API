package core

import "time"

// CaseInfo contains detailed case information
type CaseInfo struct {
	CaseNumber   string `json:"case_number"`
	CaseName     string `json:"case_name"`
	CaseType     string `json:"case_type,omitempty"`     // "criminal", "civil", "traffic"
	Chapter      string `json:"chapter,omitempty"`       // Bankruptcy chapter
	Docket       string `json:"docket,omitempty"`        // Full docket number
	NatureOfSuit string `json:"nature_of_suit,omitempty"`
}

// CourtInfo contains detailed court information
type CourtInfo struct {
	CourtID      string `json:"court_id"`
	CourtName    string `json:"court_name"`
	Jurisdiction string `json:"jurisdiction"`            // "federal", "state", "local"
	Level        string `json:"level"`                   // "trial", "appellate", "supreme"
	District     string `json:"district,omitempty"`
	Division     string `json:"division,omitempty"`
	County       string `json:"county,omitempty"`
}

// Party represents a party to the case
type Party struct {
	Name      string     `json:"name"`
	Role      string     `json:"role"`                    // "defendant", "plaintiff", "appellant"
	PartyType string     `json:"party_type,omitempty"`    // "individual", "corporation", "government"
	Date      *time.Time `json:"date,omitempty"`          // Date associated with this party
}

// Attorney represents legal counsel
type Attorney struct {
	Name         string `json:"name"`
	BarNumber    string `json:"bar_number,omitempty"`
	Role         string `json:"role"`                    // "defense", "prosecution", "counsel"
	Organization string `json:"organization,omitempty"`
	ContactInfo  string `json:"contact_info,omitempty"`
}

// Judge represents the presiding judge
type Judge struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	JudgeID string `json:"judge_id,omitempty"`
}

// Charge represents criminal charges
type Charge struct {
	Statute     string `json:"statute"`
	Description string `json:"description"`
	Grade       string `json:"grade,omitempty"`         // "felony", "misdemeanor"
	Class       string `json:"class,omitempty"`         // "A", "B", "C"
	Count       int    `json:"count,omitempty"`         // Count number
}

// Authority represents legal authorities cited
type Authority struct {
	Citation  string `json:"citation"`
	CaseTitle string `json:"case_title,omitempty"`
	Type      string `json:"type"`                    // "case_law", "statute", "regulation"
	Precedent bool   `json:"precedent"`
	Page      string `json:"page,omitempty"`          // Page or section reference
}

// LegalTag represents a legal classification tag
type LegalTag struct {
	Name        string `json:"name"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
}

// Common legal tag categories
const (
	LegalTagCategoryMotion     = "motion"
	LegalTagCategoryEvidence   = "evidence"
	LegalTagCategoryProcedural = "procedural"
	LegalTagCategorySubstantive = "substantive"
	LegalTagCategoryCriminal   = "criminal"
	LegalTagCategoryCivil      = "civil"
)

// GetCommonLegalTags returns commonly used legal tags
func GetCommonLegalTags() []LegalTag {
	return []LegalTag{
		{Name: "motion to dismiss", Category: LegalTagCategoryMotion},
		{Name: "motion to suppress", Category: LegalTagCategoryMotion},
		{Name: "discovery", Category: LegalTagCategoryProcedural},
		{Name: "sentencing", Category: LegalTagCategoryCriminal},
		{Name: "plea bargain", Category: LegalTagCategoryCriminal},
		{Name: "summary judgment", Category: LegalTagCategoryCivil},
		{Name: "evidence exclusion", Category: LegalTagCategoryEvidence},
		{Name: "constitutional challenge", Category: LegalTagCategorySubstantive},
		{Name: "procedural motion", Category: LegalTagCategoryProcedural},
		{Name: "appeal", Category: LegalTagCategoryProcedural},
	}
}
