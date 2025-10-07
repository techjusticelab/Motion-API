package classifier

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// DateRange represents a range of dates for multi-day events
type DateRange struct {
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
	Type      string  `json:"type"` // "hearing_period", "trial_dates", etc.
}

// DateExtractionResult contains all extracted date information
type DateExtractionResult struct {
	FilingDate   *string     `json:"filing_date,omitempty"`
	EventDate    *string     `json:"event_date,omitempty"`
	HearingDate  *string     `json:"hearing_date,omitempty"`
	DecisionDate *string     `json:"decision_date,omitempty"`
	ServedDate   *string     `json:"served_date,omitempty"`
	DateRanges   []DateRange `json:"date_ranges,omitempty"`
}

// Common date formats found in legal documents
var dateFormats = []string{
	"2006-01-02",           // ISO format
	"01/02/2006",           // MM/DD/YYYY
	"1/2/2006",             // M/D/YYYY
	"January 2, 2006",      // Month D, YYYY
	"Jan 2, 2006",          // Mon D, YYYY
	"January 02, 2006",     // Month DD, YYYY
	"Jan 02, 2006",         // Mon DD, YYYY
	"02 January 2006",      // DD Month YYYY
	"2 January 2006",       // D Month YYYY
	"02-Jan-2006",          // DD-Mon-YYYY
	"2-Jan-2006",           // D-Mon-YYYY
	"2006/01/02",           // YYYY/MM/DD
	"2006-1-2",             // YYYY-M-D
	"01-02-2006",           // MM-DD-YYYY
	"1-2-2006",             // M-D-YYYY
}

// Partial date formats (month/year only)
var partialDateFormats = []string{
	"January 2006",         // Month YYYY
	"Jan 2006",             // Mon YYYY
	"01/2006",              // MM/YYYY
	"1/2006",               // M/YYYY
	"2006-01",              // YYYY-MM
	"2006/01",              // YYYY/MM
}

// Date extraction patterns for different contexts
var datePatterns = map[string]*regexp.Regexp{
	"filing_date": regexp.MustCompile(`(?i)(?:filed|filing|file date|date filed)(?:\s+on)?\s*:?\s*([^,\n\r;]+)`),
	"event_date":  regexp.MustCompile(`(?i)(?:on or about|incident|occurred|violation|event)(?:\s+on)?\s*:?\s*([^,\n\r;]+)`),
	"hearing_date": regexp.MustCompile(`(?i)(?:hearing|scheduled|arraignment|calendar)(?:\s+(?:set|on|for))?\s*:?\s*([^,\n\r;]+)`),
	"decision_date": regexp.MustCompile(`(?i)(?:decided|ruling|ordered|judgment|entered)(?:\s+on)?\s*:?\s*([^,\n\r;]+)`),
	"served_date": regexp.MustCompile(`(?i)(?:served|service)(?:\s+on)?\s*:?\s*([^,\n\r;]+)`),
}

// DateExtractor provides date extraction and validation functionality
type DateExtractor struct {
	currentYear int
	timezone    *time.Location
}

// NewDateExtractor creates a new date extractor
func NewDateExtractor() *DateExtractor {
	loc, _ := time.LoadLocation("America/Los_Angeles") // Pacific timezone for California courts
	return &DateExtractor{
		currentYear: time.Now().Year(),
		timezone:    loc,
	}
}

// ExtractDatesFromText attempts to extract all date types from the given text
func (de *DateExtractor) ExtractDatesFromText(text string) *DateExtractionResult {
	result := &DateExtractionResult{}

	// Extract each type of date
	result.FilingDate = de.extractDateByType(text, "filing_date")
	result.EventDate = de.extractDateByType(text, "event_date")
	result.HearingDate = de.extractDateByType(text, "hearing_date")
	result.DecisionDate = de.extractDateByType(text, "decision_date")
	result.ServedDate = de.extractDateByType(text, "served_date")

	// Extract date ranges (future enhancement)
	result.DateRanges = de.extractDateRanges(text)

	return result
}

// extractDateByType extracts a specific type of date from text
func (de *DateExtractor) extractDateByType(text, dateType string) *string {
	pattern, exists := datePatterns[dateType]
	if !exists {
		return nil
	}

	matches := pattern.FindStringSubmatch(text)
	if len(matches) < 2 {
		return nil
	}

	dateStr := strings.TrimSpace(matches[1])
	if dateStr == "" {
		return nil
	}

	// Parse and validate the date
	parsedDate := de.parseAndValidateDate(dateStr, dateType)
	return parsedDate
}

// parseAndValidateDate attempts to parse a date string and validate it
func (de *DateExtractor) parseAndValidateDate(dateStr, dateType string) *string {
	// Clean the date string
	dateStr = de.cleanDateString(dateStr)
	
	// Try full date formats first
	if parsed := de.tryParseFullDate(dateStr); parsed != nil {
		if de.validateDate(*parsed, dateType) {
			return parsed
		}
	}

	// Try partial date formats
	if parsed := de.tryParsePartialDate(dateStr); parsed != nil {
		if de.validateDate(*parsed, dateType) {
			return parsed
		}
	}

	// Try relative dates
	if parsed := de.tryParseRelativeDate(dateStr); parsed != nil {
		if de.validateDate(*parsed, dateType) {
			return parsed
		}
	}

	return nil
}

// cleanDateString cleans and normalizes a date string
func (de *DateExtractor) cleanDateString(dateStr string) string {
	// Remove common prefixes and suffixes
	dateStr = regexp.MustCompile(`(?i)^(?:on\s+|at\s+|the\s+)`).ReplaceAllString(dateStr, "")
	dateStr = regexp.MustCompile(`(?i)\s+(?:at\s+.*|,\s+at\s+.*)$`).ReplaceAllString(dateStr, "")
	
	// Remove time information
	dateStr = regexp.MustCompile(`\s+\d{1,2}:\d{2}(?::\d{2})?(?:\s*[AaPp][Mm])?`).ReplaceAllString(dateStr, "")
	
	// Normalize whitespace
	dateStr = regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(dateStr), " ")
	
	return dateStr
}

// tryParseFullDate attempts to parse a complete date
func (de *DateExtractor) tryParseFullDate(dateStr string) *string {
	for _, format := range dateFormats {
		if t, err := time.Parse(format, dateStr); err == nil {
			formatted := t.Format("2006-01-02")
			return &formatted
		}
	}
	return nil
}

// tryParsePartialDate attempts to parse a partial date (month/year only)
func (de *DateExtractor) tryParsePartialDate(dateStr string) *string {
	for _, format := range partialDateFormats {
		if t, err := time.Parse(format, dateStr); err == nil {
			// Use first day of the month for partial dates
			firstDay := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			formatted := firstDay.Format("2006-01-02")
			return &formatted
		}
	}
	return nil
}

// tryParseRelativeDate attempts to parse relative dates like "tomorrow", "next Monday"
func (de *DateExtractor) tryParseRelativeDate(dateStr string) *string {
	lower := strings.ToLower(dateStr)
	now := time.Now().In(de.timezone)
	
	switch {
	case strings.Contains(lower, "today"):
		formatted := now.Format("2006-01-02")
		return &formatted
		
	case strings.Contains(lower, "tomorrow"):
		tomorrow := now.AddDate(0, 0, 1)
		formatted := tomorrow.Format("2006-01-02")
		return &formatted
		
	case strings.Contains(lower, "yesterday"):
		yesterday := now.AddDate(0, 0, -1)
		formatted := yesterday.Format("2006-01-02")
		return &formatted
		
	case strings.Contains(lower, "next week"):
		nextWeek := now.AddDate(0, 0, 7)
		formatted := nextWeek.Format("2006-01-02")
		return &formatted
		
	case strings.Contains(lower, "last week"):
		lastWeek := now.AddDate(0, 0, -7)
		formatted := lastWeek.Format("2006-01-02")
		return &formatted
		
	case strings.Contains(lower, "next month"):
		nextMonth := now.AddDate(0, 1, 0)
		formatted := nextMonth.Format("2006-01-02")
		return &formatted
		
	default:
		// Try to parse relative weekdays like "next Monday"
		return de.parseRelativeWeekday(dateStr, now)
	}
}

// parseRelativeWeekday parses relative weekday references
func (de *DateExtractor) parseRelativeWeekday(dateStr string, baseDate time.Time) *string {
	lower := strings.ToLower(dateStr)
	
	weekdays := map[string]time.Weekday{
		"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
		"wednesday": time.Wednesday, "thursday": time.Thursday,
		"friday": time.Friday, "saturday": time.Saturday,
		"sun": time.Sunday, "mon": time.Monday, "tue": time.Tuesday,
		"wed": time.Wednesday, "thu": time.Thursday, "fri": time.Friday, "sat": time.Saturday,
	}
	
	for dayName, weekday := range weekdays {
		if strings.Contains(lower, dayName) {
			// Calculate days until the target weekday
			daysUntil := int(weekday - baseDate.Weekday())
			if strings.Contains(lower, "next") {
				if daysUntil <= 0 {
					daysUntil += 7 // Next occurrence
				}
			} else if strings.Contains(lower, "last") {
				if daysUntil >= 0 {
					daysUntil -= 7 // Previous occurrence
				}
			} else {
				// Current week occurrence
				if daysUntil < 0 {
					daysUntil += 7
				}
			}
			
			targetDate := baseDate.AddDate(0, 0, daysUntil)
			formatted := targetDate.Format("2006-01-02")
			return &formatted
		}
	}
	
	return nil
}

// validateDate validates that a parsed date is reasonable for the given date type
func (de *DateExtractor) validateDate(dateStr, dateType string) bool {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return false
	}
	
	now := time.Now()
	
	// General validation: reasonable year range for legal documents
	if t.Year() < 1950 || t.Year() > now.Year()+10 {
		return false
	}
	
	// Type-specific validation
	switch dateType {
	case "filing_date":
		// Filing dates should not be in the future (with small tolerance)
		return t.Before(now.AddDate(0, 0, 1))
		
	case "event_date":
		// Event dates can be past, present, or reasonable future
		return t.After(time.Date(1950, 1, 1, 0, 0, 0, 0, time.UTC)) && 
			   t.Before(now.AddDate(10, 0, 0))
		
	case "hearing_date":
		// Hearing dates are typically in the future
		return t.After(now.AddDate(0, 0, -30)) && // Allow some past hearings
			   t.Before(now.AddDate(5, 0, 0))     // Reasonable future limit
		
	case "decision_date":
		// Decision dates should not be far in the future
		return t.Before(now.AddDate(0, 0, 30))
		
	case "served_date":
		// Service dates should not be in the future
		return t.Before(now.AddDate(0, 0, 1))
		
	default:
		return true
	}
}

// extractDateRanges extracts date ranges for multi-day events
func (de *DateExtractor) extractDateRanges(text string) []DateRange {
	ranges := []DateRange{}
	
	// Pattern for date ranges like "January 1-3, 2024" or "1/1/2024 to 1/3/2024"
	rangePattern := regexp.MustCompile(`(?i)(?:from\s+)?(\w+\s+\d{1,2}(?:st|nd|rd|th)?)\s*[-–—]\s*(\d{1,2}(?:st|nd|rd|th)?),?\s+(\d{4})|(\d{1,2}\/\d{1,2}\/\d{4})\s+(?:to|through|thru)\s+(\d{1,2}\/\d{1,2}\/\d{4})`)
	
	matches := rangePattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) >= 6 {
			var startDate, endDate *string
			
			if match[1] != "" && match[2] != "" && match[3] != "" {
				// Format: "January 1-3, 2024"
				start := fmt.Sprintf("%s, %s", match[1], match[3])
				end := fmt.Sprintf("%s %s, %s", strings.Fields(match[1])[0], match[2], match[3])
				
				if parsed := de.parseAndValidateDate(start, "event_date"); parsed != nil {
					startDate = parsed
				}
				if parsed := de.parseAndValidateDate(end, "event_date"); parsed != nil {
					endDate = parsed
				}
			} else if match[4] != "" && match[5] != "" {
				// Format: "1/1/2024 to 1/3/2024"
				if parsed := de.parseAndValidateDate(match[4], "event_date"); parsed != nil {
					startDate = parsed
				}
				if parsed := de.parseAndValidateDate(match[5], "event_date"); parsed != nil {
					endDate = parsed
				}
			}
			
			if startDate != nil && endDate != nil {
				ranges = append(ranges, DateRange{
					StartDate: startDate,
					EndDate:   endDate,
					Type:      de.inferRangeType(text, *startDate, *endDate),
				})
			}
		}
	}
	
	return ranges
}

// inferRangeType attempts to determine the type of date range based on context
func (de *DateExtractor) inferRangeType(text, startDate, endDate string) string {
	lower := strings.ToLower(text)
	
	switch {
	case strings.Contains(lower, "trial"):
		return "trial_dates"
	case strings.Contains(lower, "hearing"):
		return "hearing_period"
	case strings.Contains(lower, "conference"):
		return "conference_period"
	case strings.Contains(lower, "deposition"):
		return "deposition_period"
	default:
		return "event_period"
	}
}

// ValidateAllDates validates all dates in a DateExtractionResult
func (de *DateExtractor) ValidateAllDates(result *DateExtractionResult) error {
	var errors []string
	
	if result.FilingDate != nil && !de.validateDate(*result.FilingDate, "filing_date") {
		errors = append(errors, fmt.Sprintf("invalid filing_date: %s", *result.FilingDate))
	}
	
	if result.EventDate != nil && !de.validateDate(*result.EventDate, "event_date") {
		errors = append(errors, fmt.Sprintf("invalid event_date: %s", *result.EventDate))
	}
	
	if result.HearingDate != nil && !de.validateDate(*result.HearingDate, "hearing_date") {
		errors = append(errors, fmt.Sprintf("invalid hearing_date: %s", *result.HearingDate))
	}
	
	if result.DecisionDate != nil && !de.validateDate(*result.DecisionDate, "decision_date") {
		errors = append(errors, fmt.Sprintf("invalid decision_date: %s", *result.DecisionDate))
	}
	
	if result.ServedDate != nil && !de.validateDate(*result.ServedDate, "served_date") {
		errors = append(errors, fmt.Sprintf("invalid served_date: %s", *result.ServedDate))
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("date validation errors: %s", strings.Join(errors, "; "))
	}
	
	return nil
}

// MergeDates merges two DateExtractionResult objects, preferring non-nil values
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