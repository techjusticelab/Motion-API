package classifier

func MergeDates(primary, secondary *DateExtractionResult) *DateExtractionResult {
	if primary == nil {
		return secondary
	}
	if secondary == nil {
		return primary
	}

	result := &DateExtractionResult{}

	if primary.FilingDate != nil {
		result.FilingDate = primary.FilingDate
	} else {
		result.FilingDate = secondary.FilingDate
	}

	if primary.EventDate != nil {
		result.EventDate = primary.EventDate
	} else {
		result.EventDate = secondary.EventDate
	}

	if primary.HearingDate != nil {
		result.HearingDate = primary.HearingDate
	} else {
		result.HearingDate = secondary.HearingDate
	}

	if primary.DecisionDate != nil {
		result.DecisionDate = primary.DecisionDate
	} else {
		result.DecisionDate = secondary.DecisionDate
	}

	if primary.ServedDate != nil {
		result.ServedDate = primary.ServedDate
	} else {
		result.ServedDate = secondary.ServedDate
	}

	// Merge date ranges
	result.DateRanges = append(primary.DateRanges, secondary.DateRanges...)

	return result
}
