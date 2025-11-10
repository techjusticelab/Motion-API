package core

// DocumentType represents specific legal document categories
type DocumentType string

const (
	// Motions
	DocTypeMotionToSuppress         DocumentType = "motion_to_suppress"
	DocTypeMotionToDismiss          DocumentType = "motion_to_dismiss"
	DocTypeMotionToCompel           DocumentType = "motion_to_compel"
	DocTypeMotionInLimine           DocumentType = "motion_in_limine"
	DocTypeMotionForSummaryJudgment DocumentType = "motion_summary_judgment"
	DocTypeMotionToStrike           DocumentType = "motion_to_strike"
	DocTypeMotionForReconsideration DocumentType = "motion_for_reconsideration"
	DocTypeMotionToAmend            DocumentType = "motion_to_amend"
	DocTypeMotionForContinuance     DocumentType = "motion_for_continuance"

	// Orders and Rulings
	DocTypeOrder      DocumentType = "order"
	DocTypeRuling     DocumentType = "ruling"
	DocTypeJudgment   DocumentType = "judgment"
	DocTypeSentence   DocumentType = "sentence"
	DocTypeInjunction DocumentType = "injunction"

	// Briefs and Pleadings
	DocTypeBrief     DocumentType = "brief"
	DocTypeComplaint DocumentType = "complaint"
	DocTypeAnswer    DocumentType = "answer"
	DocTypePlea      DocumentType = "plea"
	DocTypeReply     DocumentType = "reply"

	// Administrative
	DocTypeDocketEntry    DocumentType = "docket_entry"
	DocTypeNotice         DocumentType = "notice"
	DocTypeStipulation    DocumentType = "stipulation"
	DocTypeCorrespondence DocumentType = "correspondence"
	DocTypeTranscript     DocumentType = "transcript"
	DocTypeEvidence       DocumentType = "evidence"
	DocTypeOther          DocumentType = "other"
	DocTypeUnknown        DocumentType = "unknown"
)

// String returns the string representation of DocumentType
func (dt DocumentType) String() string {
	return string(dt)
}

// IsMotion returns true if the document type is a motion
func (dt DocumentType) IsMotion() bool {
	switch dt {
	case DocTypeMotionToSuppress, DocTypeMotionToDismiss, DocTypeMotionToCompel,
		 DocTypeMotionInLimine, DocTypeMotionForSummaryJudgment, DocTypeMotionToStrike,
		 DocTypeMotionForReconsideration, DocTypeMotionToAmend, DocTypeMotionForContinuance:
		return true
	default:
		return false
	}
}

// IsOrder returns true if the document type is an order or ruling
func (dt DocumentType) IsOrder() bool {
	switch dt {
	case DocTypeOrder, DocTypeRuling, DocTypeJudgment, DocTypeSentence, DocTypeInjunction:
		return true
	default:
		return false
	}
}

// IsPleading returns true if the document type is a pleading
func (dt DocumentType) IsPleading() bool {
	switch dt {
	case DocTypeBrief, DocTypeComplaint, DocTypeAnswer, DocTypePlea, DocTypeReply:
		return true
	default:
		return false
	}
}

// GetCategory returns the general category for the document type
func (dt DocumentType) GetCategory() string {
	if dt.IsMotion() {
		return "motion"
	}
	if dt.IsOrder() {
		return "order"
	}
	if dt.IsPleading() {
		return "pleading"
	}
	return "administrative"
}

// GetAllDocumentTypes returns all available document types
func GetAllDocumentTypes() []DocumentType {
	return []DocumentType{
		DocTypeMotionToSuppress, DocTypeMotionToDismiss, DocTypeMotionToCompel,
		DocTypeMotionInLimine, DocTypeMotionForSummaryJudgment, DocTypeMotionToStrike,
		DocTypeMotionForReconsideration, DocTypeMotionToAmend, DocTypeMotionForContinuance,
		DocTypeOrder, DocTypeRuling, DocTypeJudgment, DocTypeSentence, DocTypeInjunction,
		DocTypeBrief, DocTypeComplaint, DocTypeAnswer, DocTypePlea, DocTypeReply,
		DocTypeDocketEntry, DocTypeNotice, DocTypeStipulation, DocTypeCorrespondence,
		DocTypeTranscript, DocTypeEvidence, DocTypeOther, DocTypeUnknown,
	}
}

// ParseDocumentType safely parses a string to DocumentType
func ParseDocumentType(s string) DocumentType {
	dt := DocumentType(s)
	for _, validType := range GetAllDocumentTypes() {
		if dt == validType {
			return dt
		}
	}
	return DocTypeUnknown
}
